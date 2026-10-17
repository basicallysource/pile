package match

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"regexp"
	"slices"
	"sort"
	"sync"

	"github.com/basicallysource/pile/internal/catalog"
)

// Line is one line of a set: a part (by index id) in a color, and its class.
type Line struct {
	Part, Color, Quantity int32
	Class                 catalog.Class
}

// Counting is which classes of part count toward a set's completeness, a
// plain part always. A part that does not count is still found, and sorted
// out with its set.
type Counting struct{ Printed, Figure, Accessory bool }

// DefaultCounting counts accessories (often part of the build), not printed
// parts, stickers or minifigure body parts.
var DefaultCounting = Counting{Accessory: true}

// Counts is whether c counts a part of class cl.
func (c Counting) Counts(cl catalog.Class) bool {
	switch cl {
	case catalog.Printed:
		return c.Printed
	case catalog.FigurePart:
		return c.Figure
	case catalog.Accessory:
		return c.Accessory
	}
	return true
}

// Entry is a set, custom model or build as matching sees it.
type Entry struct {
	Set *catalog.Set
	// Every line, one per part and color.
	Lines []Line
	// The pieces DefaultCounting counts, which what is judged once per set
	// rests on (its rarity, whether it is basic or loose bricks); a Match
	// counts by its view.
	Need int32
	// Pieces of each class but plain, and whole minifigures.
	Printed, FigureParts, Accessories, Minifigures int32
	// Basically a minifigure: its figures (about four pieces each) and loose
	// figure parts outweigh its counted pieces.
	MinifigureSet bool
	// Other sets with exactly these lines; Hidden on those others, which
	// point back at this one in SameAs.
	SameContents []string
	SameAs       *Entry
	Hidden       bool
	// How rare its pieces are on average across all sets (exact colors), and
	// how many different parts it has: high for a set that is a model of
	// something, low for a box of bricks.
	Rarity        float64
	DistinctParts int32
	// Loose bricks rather than a model: in a bulk theme, named as a bucket,
	// tub or brick box, or big and made of few different parts.
	Bulk bool
	// Basic: almost all plain bricks, plates, tiles and slopes; few
	// different parts; or nothing unusual (its rarest pieces as common as
	// those of the most common tenth of sets). See plain.go.
	MostlyBasic, FewParts, NothingUnusual bool
	// Of an entry with a set's changes (changes.go): the entry as sold,
	// whose lines are these lines in the same order, and how each line
	// changed.
	base *Entry
	how  []how
}

// Index is every set, custom model and build with a counted line, and the weights
// of their parts.
type Index struct {
	cat     *catalog.Catalog
	parts   map[string]int32 // canonical part number to id
	nums    []string         // id to canonical part number
	Entries []*Entry
	byNum   map[string]*Entry
	// Weight of a part in a color (exact matching) and of a part (any color).
	wExact map[uint64]float64
	wAny   map[int32]float64
	wMax   float64
	// Each color's near colors (colors.go).
	near map[int32][]int32
	// The plain plates, tiles and bricks (footprints.go): each part's shape,
	// the part of each shape, and the ways to cover each one's footprint,
	// worked out when first asked for.
	shapes map[int32]catalog.Shape
	sized  map[catalog.Shape]int32
	cover  sync.Map
}

func key(part, color int32) uint64 { return uint64(uint32(part))<<32 | uint64(uint32(color)) }

func NewIndex(cat *catalog.Catalog) *Index {
	ix := &Index{cat: cat, parts: map[string]int32{}, byNum: map[string]*Entry{}, wExact: map[uint64]float64{}, wAny: map[int32]float64{}, near: nearTable(cat)}
	for _, s := range cat.Sets {
		e := &Entry{Set: s}
		qty := map[uint64]int32{}
		class := map[uint64]catalog.Class{}
		for _, l := range s.Lines {
			var cl catalog.Class
			if p := cat.Parts[l.Part]; p != nil {
				cl = p.Class
			}
			switch cl {
			case catalog.Printed:
				e.Printed += l.Quantity
			case catalog.FigurePart:
				e.FigureParts += l.Quantity
			case catalog.Accessory:
				e.Accessories += l.Quantity
			}
			k := key(ix.Part(l.Part), l.Color)
			if _, ok := class[k]; !ok {
				class[k] = cl
			}
			qty[k] += l.Quantity
		}
		for _, f := range s.Minifigures {
			e.Minifigures += f.Quantity
		}
		for k, q := range qty {
			e.Lines = append(e.Lines, Line{Part: int32(k >> 32), Color: int32(uint32(k)), Quantity: q, Class: class[k]})
			if DefaultCounting.Counts(class[k]) {
				e.Need += q
			}
		}
		if e.Need == 0 {
			continue
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

	// Weights come from the sets LEGO sold, each distinct contents once,
	// every line of them, so a part counted only by some views has one too.
	dfExact := map[uint64]int{}
	dfAny := map[int32]int{}
	n := 0
	for _, e := range ix.Entries {
		if e.Hidden || !e.Set.Sold() {
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
	// A part no sold set has (only models beside them) is as rare as can be.
	ix.wMax = math.Log(float64(n))
	for _, e := range ix.Entries {
		parts := map[int32]bool{}
		var w float64
		for _, l := range e.Lines {
			if !DefaultCounting.Counts(l.Class) {
				continue
			}
			parts[l.Part] = true
			w += float64(l.Quantity) * ix.weight(l, ix.Colors(Exact), false)
		}
		e.DistinctParts = int32(len(parts))
		e.Rarity = w / float64(max(e.Need, 1))
		e.Bulk = e.Set.Sold() && (e.Set.BulkTheme || bulkName.MatchString(e.Set.Name) ||
			(!e.Set.Licensed && e.Need >= bulkSize && e.Need >= bulkRepeat*e.DistinctParts))
	}
	ix.judgePlain()
	ix.readShapes(dfAny)
	return ix
}

const (
	// A set this big with this many pieces per different part is a box of
	// bricks whatever its theme (Creator's brick boxes run 13 to 27; models
	// 3 to 7). Licensed sets are models even so (a big bust, a mosaic of a
	// character).
	bulkSize   = 100
	bulkRepeat = 12
)

var bulkName = regexp.MustCompile(`(?i)\b(bucket|tub|canister|box of bricks|brick box|bricks box)\b`)

// Part is the id of the part standing for p's mold variants and alternates.
func (ix *Index) Part(p string) int32 {
	p = ix.cat.Canon(p)
	id, ok := ix.parts[p]
	if !ok {
		id = int32(len(ix.parts))
		ix.parts[p] = id
		ix.nums = append(ix.nums, p)
	}
	return id
}

// PartID is the id of the part standing for p, if the index has it. Unlike
// Part it never adds one, so it is safe while views are served.
func (ix *Index) PartID(p string) (int32, bool) {
	id, ok := ix.parts[ix.cat.Canon(p)]
	return id, ok
}

// PartNum is the canonical part number of a part id.
func (ix *Index) PartNum(id int32) string { return ix.nums[id] }

// Entry is the set or custom model with this number, if it has counted lines.
func (ix *Index) Entry(num string) *Entry { return ix.byNum[num] }

// weight is how rare l's part is: in its color, or in any color when the
// line takes any (anyColor).
func (ix *Index) weight(l Line, cs Colors, anyColor bool) float64 {
	var w float64
	var ok bool
	if cs.Mode == Any || anyColor {
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
// box sold under several numbers), listing it on that earlier set, and the
// builds of such a set when the earlier one has builds too.
func (ix *Index) dedupe() {
	sort.Slice(ix.Entries, func(i, j int) bool { return ix.Entries[i].Set.Num < ix.Entries[j].Set.Num })
	first := map[[32]byte]*Entry{}
	for _, e := range ix.Entries {
		if !e.Set.Sold() {
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
			e.Hidden, e.SameAs = true, f
			continue
		}
		first[sum] = e
	}
	// A hidden set's builds are its twin's, when the twin has its own.
	for _, e := range ix.Entries {
		if e.Set.Kind != catalog.Build {
			continue
		}
		if of := ix.byNum[e.Set.Of.Num]; of != nil && of.SameAs != nil && len(of.SameAs.Set.Builds) > 0 {
			e.Hidden = true
		}
	}
}

// LineOf is the index in e.Lines of this part (or a variant) in this color,
// as the set has it (before its changes), or -1.
func (ix *Index) LineOf(e *Entry, part string, color int32) int {
	id, ok := ix.PartID(part)
	if !ok {
		return -1
	}
	if e.base != nil {
		e = e.base
	}
	for i, l := range e.Lines {
		if l.Part == id && l.Color == color {
			return i
		}
	}
	return -1
}
