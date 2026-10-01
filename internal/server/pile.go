package server

import (
	"sort"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/machine"
	"github.com/basicallysource/pile/internal/match"
)

// Pile is the machine's pieces put into Rebrickable's terms, and the sets they match.
type Pile struct {
	Catalog   *catalog.Catalog
	Records   *machine.Records
	Machine   string
	Lots      []*Lot
	Unmatched []*Unmatched
	Pieces    int32
	Result    *match.Result
}

type Lot struct {
	Part          *catalog.Part
	BrickLinkPart string
	Color         *catalog.Color
	Count         int32
	confidence    float64
}

type Unmatched struct {
	BrickLinkPart, Name, ColorName string
	Count                          int32
}

func Build(cat *catalog.Catalog, rec *machine.Records, machineName string) *Pile {
	p := &Pile{Catalog: cat, Records: rec, Machine: machineName}
	lots := map[catalog.PartColor]*Lot{}
	unmatched := map[[3]string]*Unmatched{}
	counts := map[catalog.PartColor]int32{}
	for _, pc := range rec.Pieces {
		p.Pieces++
		part := cat.Parts[cat.BrickLinkParts[pc.BrickLinkPart]]
		color, colorOK := cat.BrickLinkColors[pc.BrickLinkColor]
		if part == nil || !pc.ColorKnown || !colorOK {
			k := [3]string{pc.BrickLinkPart, pc.PartName, pc.ColorName}
			u := unmatched[k]
			if u == nil {
				u = &Unmatched{BrickLinkPart: k[0], Name: k[1], ColorName: k[2]}
				unmatched[k] = u
			}
			u.Count++
			continue
		}
		key := catalog.PartColor{Part: part.Num, Color: color}
		l := lots[key]
		if l == nil {
			l = &Lot{Part: part, BrickLinkPart: pc.BrickLinkPart, Color: cat.Colors[color]}
			lots[key] = l
		}
		l.Count++
		l.confidence += pc.Confidence
		counts[key]++
	}
	for _, l := range lots {
		p.Lots = append(p.Lots, l)
	}
	sort.Slice(p.Lots, func(i, j int) bool {
		if p.Lots[i].Count != p.Lots[j].Count {
			return p.Lots[i].Count > p.Lots[j].Count
		}
		return p.Lots[i].Part.Num < p.Lots[j].Part.Num
	})
	for _, u := range unmatched {
		p.Unmatched = append(p.Unmatched, u)
	}
	sort.Slice(p.Unmatched, func(i, j int) bool { return p.Unmatched[i].Count > p.Unmatched[j].Count })
	p.Result = match.Run(cat, counts)
	return p
}

func (p *Pile) UnmatchedPieces() int32 {
	var n int32
	for _, u := range p.Unmatched {
		n += u.Count
	}
	return n
}
