// Package match works out which LEGO sets a pile of sorted pieces makes up.
//
// Every set is scored by the pieces of the pile it could claim, line by
// line, capped at what the set needs. A plain fraction would rank every
// small box of 2x4 bricks first, so each part and color is weighted by how
// rare it is across all sets (its inverse document frequency): a set's
// unusual pieces are the evidence that it is there.
//
// Two rankings come out of it. Alone scores each set against the whole pile.
// Explained picks sets one at a time, best evidence first, taking each
// pick's pieces out of the pile before the next, so a bucket of basic bricks
// is picked once and the themed sets underneath it surface on their own.
package match

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"slices"
	"sort"

	"github.com/basicallysource/pile/internal/catalog"
)

const (
	// Sets smaller than this are not ranked: a polybag of a few common parts
	// is complete in any pile and says nothing.
	minSetPieces = 10
	// A pick needs this weighted completeness...
	minPickWeighted = 0.3
	// ...and this much evidence (the rarity-weighted pieces it claims).
	minPickEvidence = 40
	maxPicks        = 120
)

// Key is a part (standing for its mold variants and alternates) in a color.
type Key int32

type line struct {
	key Key
	qty int32
}

type setLines struct {
	set      *catalog.Set
	lines    []line // counted lines, one per key
	need     int32
	printed  int32
	minifigs int32
	same     []string // other sets with exactly these lines
}

type Match struct {
	Set          *catalog.Set
	Have, Need   int32
	Weighted     float64
	Evidence     float64
	Printed      int32
	Minifigures  int32
	SameContents []string
	Pick         int // in the explained ranking, from 1
	claimed      map[Key]int32
}

type Result struct {
	Alone     []*Match // best first
	Explained []*Match // in the order picked
	Explains  int32    // pieces the picks claim
	Ranked    int      // sets considered

	keys  map[catalog.PartColor]Key
	pile  map[Key]int32
	alone map[string]*Match
	first map[string]*Match // a set's first pick
	cat   *catalog.Catalog
}

// Run scores every set in cat against pile (Rebrickable part and color → count).
func Run(cat *catalog.Catalog, pile map[catalog.PartColor]int32) *Result {
	r := &Result{keys: map[catalog.PartColor]Key{}, pile: map[Key]int32{}, alone: map[string]*Match{}, first: map[string]*Match{}, cat: cat}
	for pc, n := range pile {
		r.pile[r.key(pc)] += n
	}

	// Every set's counted lines, and how many sets hold each key.
	var sets []*setLines
	df := map[Key]int32{}
	for _, s := range cat.Sets {
		sl := &setLines{set: s}
		qty := map[Key]int32{}
		for _, l := range s.Lines {
			if p := cat.Parts[l.Part]; p != nil && p.Printed {
				sl.printed += l.Quantity
				continue
			}
			qty[r.key(catalog.PartColor{Part: l.Part, Color: l.Color})] += l.Quantity
		}
		for _, f := range s.Minifigures {
			sl.minifigs += f.Quantity
		}
		for k, q := range qty {
			sl.lines = append(sl.lines, line{k, q})
			sl.need += q
			df[k]++
		}
		if len(sl.lines) == 0 {
			continue
		}
		slices.SortFunc(sl.lines, func(a, b line) int { return int(a.key - b.key) })
		sets = append(sets, sl)
	}
	weight := map[Key]float64{}
	n := float64(len(sets))
	for k, d := range df {
		weight[k] = math.Log(n / float64(d))
	}

	// One entry per distinct contents: the same box sold under several numbers.
	sets = dedupe(sets)

	score := func(sl *setLines, pile map[Key]int32) *Match {
		m := &Match{Set: sl.set, Need: sl.need, Printed: sl.printed, Minifigures: sl.minifigs, SameContents: sl.same}
		var wNeed float64
		for _, l := range sl.lines {
			h := min(pile[l.key], l.qty)
			w := weight[l.key]
			m.Have += h
			wNeed += float64(l.qty) * w
			m.Evidence += float64(h) * w
		}
		if wNeed > 0 {
			m.Weighted = m.Evidence / wNeed
		}
		return m
	}
	rank := func(m *Match) float64 { return m.Evidence * m.Weighted }

	var pool []*setLines
	for _, sl := range sets {
		if sl.need < minSetPieces {
			continue
		}
		r.Ranked++
		m := score(sl, r.pile)
		if m.Have == 0 {
			continue
		}
		r.Alone = append(r.Alone, m)
		r.alone[sl.set.Num] = m
		if m.Evidence >= minPickEvidence && m.Weighted >= minPickWeighted {
			pool = append(pool, sl)
		}
	}
	sort.Slice(r.Alone, func(i, j int) bool { return rank(r.Alone[i]) > rank(r.Alone[j]) })

	left := map[Key]int32{}
	for k, v := range r.pile {
		left[k] = v
	}
	for len(r.Explained) < maxPicks {
		var best *Match
		var bestLines *setLines
		for _, sl := range pool {
			m := score(sl, left)
			if m.Evidence < minPickEvidence || m.Weighted < minPickWeighted {
				continue
			}
			if best == nil || rank(m) > rank(best) {
				best, bestLines = m, sl
			}
		}
		if best == nil {
			break
		}
		best.Pick = len(r.Explained) + 1
		best.claimed = map[Key]int32{}
		for _, l := range bestLines.lines {
			h := min(left[l.key], l.qty)
			left[l.key] -= h
			best.claimed[l.key] = h
		}
		r.Explains += best.Have
		r.Explained = append(r.Explained, best)
		if _, ok := r.first[best.Set.Num]; !ok {
			r.first[best.Set.Num] = best
		}
	}
	return r
}

func (r *Result) key(pc catalog.PartColor) Key {
	pc.Part = r.cat.Canon(pc.Part)
	k, ok := r.keys[pc]
	if !ok {
		k = Key(len(r.keys))
		r.keys[pc] = k
	}
	return k
}

// dedupe keeps one set per distinct contents, listing the others on it.
func dedupe(sets []*setLines) []*setLines {
	sort.Slice(sets, func(i, j int) bool { return sets[i].set.Num < sets[j].set.Num })
	byHash := map[[32]byte]*setLines{}
	var out []*setLines
	for _, sl := range sets {
		h := sha256.New()
		var b [8]byte
		for _, l := range sl.lines {
			binary.LittleEndian.PutUint32(b[:4], uint32(l.key))
			binary.LittleEndian.PutUint32(b[4:], uint32(l.qty))
			h.Write(b[:])
		}
		var sum [32]byte
		copy(sum[:], h.Sum(nil))
		if first, ok := byHash[sum]; ok {
			first.same = append(first.same, sl.set.Num)
			continue
		}
		byHash[sum] = sl
		out = append(out, sl)
	}
	return out
}

// Alone is the set's match against the whole pile (nil if it has no pieces
// there, is too small to rank, or shares its contents with another set).
func (r *Result) AloneMatch(setNum string) *Match { return r.alone[setNum] }

// Picked is the set's first pick in the explained ranking, if any.
func (r *Result) Picked(setNum string) *Match { return r.first[setNum] }

// InPile is how many pieces of this part (or a variant) in this color the pile has.
func (r *Result) InPile(pc catalog.PartColor) int32 {
	pc.Part = r.cat.Canon(pc.Part)
	k, ok := r.keys[pc]
	if !ok {
		return 0
	}
	return r.pile[k]
}

// Claimed is how many pieces of this part and color a pick took from the pile.
func (m *Match) Claimed(r *Result, pc catalog.PartColor) int32 {
	if m == nil || m.claimed == nil {
		return 0
	}
	pc.Part = r.cat.Canon(pc.Part)
	k, ok := r.keys[pc]
	if !ok {
		return 0
	}
	return m.claimed[k]
}
