package server

import (
	"sort"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/match"
)

// lot is one lot's pieces in Rebrickable's terms, ready to stack.
type lot struct {
	*lots.Lot
	// Most first.
	parts     []*partCount
	unmatched []*unmatched
	pieces    int32
	pile      *match.Pile
	// Pieces by who gave them their color (records.Piece.ColorBy).
	colorBy map[string]int32
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

func newLot(l *lots.Lot, cat *catalog.Catalog, ix *match.Index) *lot {
	t := &lot{Lot: l, colorBy: map[string]int32{}}
	byPC := map[catalog.PartColor]*partCount{}
	byUn := map[[3]string]*unmatched{}
	counts := map[catalog.PartColor]int32{}
	for _, pc := range l.Records.Pieces {
		t.pieces++
		t.colorBy[pc.ColorBy]++
		k, ok := cat.FromBrickLink(pc.BrickLinkPart, pc.BrickLinkColor, pc.ColorKnown)
		if !ok {
			k := [3]string{pc.BrickLinkPart, pc.PartName, pc.ColorName}
			u := byUn[k]
			if u == nil {
				u = &unmatched{bricklinkPart: k[0], name: k[1], colorName: k[2]}
				byUn[k] = u
				t.unmatched = append(t.unmatched, u)
			}
			u.count++
			continue
		}
		p := byPC[k]
		if p == nil {
			p = &partCount{part: cat.Parts[k.Part], bricklinkPart: pc.BrickLinkPart, color: cat.Colors[k.Color]}
			byPC[k] = p
			t.parts = append(t.parts, p)
		}
		p.count++
		p.confidence += pc.Confidence
		counts[k]++
	}
	sort.Slice(t.parts, func(i, j int) bool {
		if t.parts[i].count != t.parts[j].count {
			return t.parts[i].count > t.parts[j].count
		}
		return t.parts[i].part.Num < t.parts[j].part.Num
	})
	sort.Slice(t.unmatched, func(i, j int) bool { return t.unmatched[i].count > t.unmatched[j].count })
	t.pile = ix.NewPile(counts)
	return t
}
