package match

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"slices"
	"sort"

	"github.com/basicallysource/pile/internal/catalog"
)

// Line is one counted line of a set: a part (by index id) in a color.
type Line struct {
	Part, Color, Quantity int32
}

// Entry is a set or custom model as matching sees it.
type Entry struct {
	Set *catalog.Set
	// Counted lines, one per part and color.
	Lines []Line
	Need  int32
	// Left out of Need.
	Printed, FigureParts, Minifigures int32
	// Basically a minifigure: its figures (about four pieces each) and loose
	// figure parts outweigh its counted pieces.
	MinifigureSet bool
	// Other sets with exactly these lines; Hidden on those others.
	SameContents []string
	Hidden       bool
}

// Index is every set and custom model with a counted line, and the weights
// of their parts.
type Index struct {
	cat     *catalog.Catalog
	parts   map[string]int32 // canonical part number to id
	Entries []*Entry
	byNum   map[string]*Entry
	// Weight of a part in a color (exact matching) and of a part (any color).
	wExact map[uint64]float64
	wAny   map[int32]float64
	wMax   float64
}

func key(part, color int32) uint64 { return uint64(uint32(part))<<32 | uint64(uint32(color)) }

func NewIndex(cat *catalog.Catalog) *Index {
	ix := &Index{cat: cat, parts: map[string]int32{}, byNum: map[string]*Entry{}, wExact: map[uint64]float64{}, wAny: map[int32]float64{}}
	for _, s := range cat.Sets {
		e := &Entry{Set: s}
		qty := map[uint64]int32{}
		for _, l := range s.Lines {
			if p := cat.Parts[l.Part]; p != nil {
				switch p.Counting {
				case catalog.Printed:
					e.Printed += l.Quantity
					continue
				case catalog.FigurePart:
					e.FigureParts += l.Quantity
					continue
				}
			}
			qty[key(ix.Part(l.Part), l.Color)] += l.Quantity
		}
		for _, f := range s.Minifigures {
			e.Minifigures += f.Quantity
		}
		if len(qty) == 0 {
			continue
		}
		for k, q := range qty {
			e.Lines = append(e.Lines, Line{Part: int32(k >> 32), Color: int32(uint32(k)), Quantity: q})
			e.Need += q
		}
		slices.SortFunc(e.Lines, func(a, b Line) int {
			if a.Part != b.Part {
				return int(a.Part - b.Part)
			}
			return int(a.Color - b.Color)
		})
		figure := 4*e.Minifigures + e.FigureParts
		e.MinifigureSet = figure > 0 && figure >= e.Need
		ix.Entries = append(ix.Entries, e)
		ix.byNum[s.Num] = e
	}
	ix.dedupe()

	// Weights come from the sets LEGO sold, each distinct contents once.
	dfExact := map[uint64]int{}
	dfAny := map[int32]int{}
	n := 0
	for _, e := range ix.Entries {
		if e.Hidden || e.Set.Custom {
			continue
		}
		n++
		seen := map[int32]bool{}
		for _, l := range e.Lines {
			dfExact[key(l.Part, l.Color)]++
			if !seen[l.Part] {
				seen[l.Part] = true
				dfAny[l.Part]++
			}
		}
	}
	for k, d := range dfExact {
		ix.wExact[k] = math.Log(float64(n) / float64(d))
	}
	for p, d := range dfAny {
		ix.wAny[p] = math.Log(float64(n) / float64(d))
	}
	// A part no set has (only custom models) is as rare as can be.
	ix.wMax = math.Log(float64(n))
	return ix
}

// Part is the id of the part standing for p's mold variants and alternates.
func (ix *Index) Part(p string) int32 {
	p = ix.cat.Canon(p)
	id, ok := ix.parts[p]
	if !ok {
		id = int32(len(ix.parts))
		ix.parts[p] = id
	}
	return id
}

// Entry is the set or custom model with this number, if it has counted lines.
func (ix *Index) Entry(num string) *Entry { return ix.byNum[num] }

func (ix *Index) weight(l Line, anyColor bool) float64 {
	var w float64
	var ok bool
	if anyColor {
		w, ok = ix.wAny[l.Part]
	} else {
		w, ok = ix.wExact[key(l.Part, l.Color)]
	}
	if !ok {
		return ix.wMax
	}
	return w
}

// dedupe hides every set whose lines are exactly an earlier set's (the same
// box sold under several numbers), listing it on that earlier set.
func (ix *Index) dedupe() {
	sort.Slice(ix.Entries, func(i, j int) bool { return ix.Entries[i].Set.Num < ix.Entries[j].Set.Num })
	first := map[[32]byte]*Entry{}
	for _, e := range ix.Entries {
		if e.Set.Custom {
			continue
		}
		h := sha256.New()
		var b [12]byte
		for _, l := range e.Lines {
			binary.LittleEndian.PutUint32(b[0:], uint32(l.Part))
			binary.LittleEndian.PutUint32(b[4:], uint32(l.Color))
			binary.LittleEndian.PutUint32(b[8:], uint32(l.Quantity))
			h.Write(b[:])
		}
		var sum [32]byte
		copy(sum[:], h.Sum(nil))
		if f, ok := first[sum]; ok {
			f.SameContents = append(f.SameContents, e.Set.Num)
			e.Hidden = true
			continue
		}
		first[sum] = e
	}
}

// LineOf is the index in e.Lines of this part (or a variant) in this color,
// or -1.
func (ix *Index) LineOf(e *Entry, part string, color int32) int {
	id := ix.Part(part)
	for i, l := range e.Lines {
		if l.Part == id && l.Color == color {
			return i
		}
	}
	return -1
}
