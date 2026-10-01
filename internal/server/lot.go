package server

import (
	"sort"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/machine"
	"github.com/basicallysource/pile/internal/match"
)

// lot is one lot's pieces in Rebrickable's terms, ready to match.
type lot struct {
	*lots.Lot
	parts     []*partCount
	unmatched []*unmatched
	pieces    int32
	pile      *match.Pile
	colors    []share
	cats      []share
	// The likely order, judged in exact colors (any color would let every big
	// set claim a pile of common bricks): set number to place, from 1.
	picks map[string]int
}

type partCount struct {
	part          *catalog.Part
	bricklinkPart string
	color         *catalog.Color
	count         int32
	confidence    float64
}

type unmatched struct {
	bricklinkPart, name, colorName string
	count                          int32
}

type share struct {
	color *catalog.Color
	name  string
	n     int32
}

func newLot(l *lots.Lot, cat *catalog.Catalog, ix *match.Index) *lot {
	t := &lot{Lot: l}
	byPC := map[catalog.PartColor]*partCount{}
	byUn := map[[3]string]*unmatched{}
	counts := map[catalog.PartColor]int32{}
	for _, pc := range l.Records.Pieces {
		t.pieces++
		part := cat.Parts[cat.BrickLinkParts[pc.BrickLinkPart]]
		color, colorOK := cat.BrickLinkColors[pc.BrickLinkColor]
		if part == nil || !pc.ColorKnown || !colorOK {
			k := [3]string{pc.BrickLinkPart, pc.PartName, pc.ColorName}
			u := byUn[k]
			if u == nil {
				u = &unmatched{bricklinkPart: k[0], name: k[1], colorName: k[2]}
				byUn[k] = u
			}
			u.count++
			continue
		}
		k := catalog.PartColor{Part: part.Num, Color: color}
		p := byPC[k]
		if p == nil {
			p = &partCount{part: part, bricklinkPart: pc.BrickLinkPart, color: cat.Colors[color]}
			byPC[k] = p
		}
		p.count++
		p.confidence += pc.Confidence
		counts[k]++
	}
	colors := map[int32]*share{}
	cats := map[string]*share{}
	for _, p := range byPC {
		t.parts = append(t.parts, p)
		c := colors[p.color.ID]
		if c == nil {
			c = &share{color: p.color}
			colors[p.color.ID] = c
		}
		c.n += p.count
		g := cats[p.part.Category]
		if g == nil {
			g = &share{name: p.part.Category}
			cats[p.part.Category] = g
		}
		g.n += p.count
	}
	sort.Slice(t.parts, func(i, j int) bool {
		if t.parts[i].count != t.parts[j].count {
			return t.parts[i].count > t.parts[j].count
		}
		return t.parts[i].part.Num < t.parts[j].part.Num
	})
	for _, u := range byUn {
		t.unmatched = append(t.unmatched, u)
	}
	sort.Slice(t.unmatched, func(i, j int) bool { return t.unmatched[i].count > t.unmatched[j].count })
	for _, c := range colors {
		t.colors = append(t.colors, *c)
	}
	for _, g := range cats {
		t.cats = append(t.cats, *g)
	}
	byCount := func(s []share) { sort.Slice(s, func(i, j int) bool { return s[i].n > s[j].n }) }
	byCount(t.colors)
	byCount(t.cats)

	t.pile = ix.NewPile(counts)
	t.picks = map[string]int{}
	for n, e := range ix.Explain(t.pile, false) {
		t.picks[e.Set.Num] = n + 1
	}
	return t
}

// unmatchedPieces is how many pieces have no Rebrickable match.
func (t *lot) unmatchedPieces() int32 {
	var n int32
	for _, u := range t.unmatched {
		n += u.count
	}
	return n
}

// whole is every lot's records as one lot: the collection.
func whole(ls []*lots.Lot) *lots.Lot {
	w := &lots.Lot{ID: "all", Name: "Whole collection", Description: "Every lot together.", Records: &machine.Records{}}
	for _, l := range ls {
		r := l.Records
		w.Records.Pieces = append(w.Records.Pieces, r.Pieces...)
		if w.Records.FirstSeen == 0 || (r.FirstSeen != 0 && r.FirstSeen < w.Records.FirstSeen) {
			w.Records.FirstSeen = r.FirstSeen
		}
		w.Records.LastSeen = max(w.Records.LastSeen, r.LastSeen)
		w.Records.CopiedAt = max(w.Records.CopiedAt, r.CopiedAt)
	}
	return w
}
