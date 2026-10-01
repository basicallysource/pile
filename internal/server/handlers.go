package server

import (
	"context"
	"errors"
	"sort"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/match"
)

const (
	// Almost: at least this share found.
	almostShare = 0.8
	// Custom models shown: at least this share found.
	customShare = 0.6
	// Complete sets with fewer counted pieces are tiny (key chains, gear).
	tinyNeed = 5
	// A section sends at most this many; its total says how many it has.
	sectionMax = 150
)

func section(ms []*match.Match) *pilev1.Section {
	return &pilev1.Section{Sets: setMatches(ms[:min(len(ms), sectionMax)]), Total: int32(len(ms))}
}

func (s *Service) ListLots(_ context.Context, _ *connect.Request[pilev1.ListLotsRequest]) (*connect.Response[pilev1.ListLotsResponse], error) {
	out := &pilev1.ListLotsResponse{CatalogUnix: s.cat.DownloadedAt}
	for _, e := range s.index.Entries {
		if e.Hidden {
			continue
		}
		if e.Set.Custom {
			out.CustomModelsChecked++
		} else {
			out.SetsChecked++
		}
	}
	for _, t := range s.order {
		out.Lots = append(out.Lots, s.lotMessage(t))
	}
	return connect.NewResponse(out), nil
}

func (s *Service) GetLot(_ context.Context, req *connect.Request[pilev1.GetLotRequest]) (*connect.Response[pilev1.GetLotResponse], error) {
	w, err := s.view(req.Msg.View)
	if err != nil {
		return nil, err
	}
	out := &pilev1.GetLotResponse{Lot: s.lotMessage(w.lot), PiecesLeft: w.left.Pieces()}
	for _, c := range w.lot.cats {
		out.Categories = append(out.Categories, &pilev1.CategoryShare{Name: c.name, Pieces: c.n})
	}
	for _, m := range w.queue {
		out.SortOut = append(out.SortOut, setMatch(m))
	}
	var complete, tiny, almost, likely, custom, figures []*match.Match
	for _, m := range w.scores {
		if w.place[m.Set.Num] > 0 {
			continue
		}
		switch {
		case m.Set.Custom:
			if m.Share() >= customShare {
				custom = append(custom, m)
			}
		case m.MinifigureSet:
			if m.Complete() || m.Share() >= almostShare || m.Pick > 0 {
				figures = append(figures, m)
			}
		case m.Complete() && m.Need < tinyNeed:
			tiny = append(tiny, m)
		case m.Complete():
			complete = append(complete, m)
		case m.Share() >= almostShare:
			almost = append(almost, m)
		case m.Pick > 0:
			likely = append(likely, m)
		}
	}
	byShare := func(ms []*match.Match) {
		sort.SliceStable(ms, func(i, j int) bool {
			if ms[i].Share() != ms[j].Share() {
				return ms[i].Share() > ms[j].Share()
			}
			return ms[i].Need > ms[j].Need
		})
	}
	byRarity := func(ms []*match.Match) {
		sort.SliceStable(ms, func(i, j int) bool { return ms[i].Rarity() > ms[j].Rarity() })
	}
	byRarity(complete)
	byRarity(tiny)
	byRarity(almost)
	byShare(custom)
	byShare(figures)
	sort.SliceStable(likely, func(i, j int) bool { return likely[i].Pick < likely[j].Pick })
	out.Complete, out.Tiny, out.Almost = section(complete), section(tiny), section(almost)
	out.Likely, out.Custom, out.Minifigure = section(likely), section(custom), section(figures)
	return connect.NewResponse(out), nil
}

func (s *Service) ListSets(_ context.Context, req *connect.Request[pilev1.ListSetsRequest]) (*connect.Response[pilev1.ListSetsResponse], error) {
	w, err := s.view(req.Msg.View)
	if err != nil {
		return nil, err
	}
	var list []*match.Match
	for _, m := range w.scores {
		switch req.Msg.Kind {
		case pilev1.Kind_KIND_SET:
			if m.Set.Custom {
				continue
			}
		case pilev1.Kind_KIND_CUSTOM:
			if !m.Set.Custom {
				continue
			}
		}
		list = append(list, m)
	}
	switch req.Msg.Order {
	case pilev1.Order_ORDER_FOUND:
		sort.SliceStable(list, func(i, j int) bool { return list[i].Have > list[j].Have })
	case pilev1.Order_ORDER_COMPLETE:
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Share() != list[j].Share() {
				return list[i].Share() > list[j].Share()
			}
			return list[i].Need > list[j].Need
		})
	}
	return connect.NewResponse(&pilev1.ListSetsResponse{Sets: setMatches(list)}), nil
}

func (s *Service) GetSet(_ context.Context, req *connect.Request[pilev1.GetSetRequest]) (*connect.Response[pilev1.GetSetResponse], error) {
	w, err := s.view(req.Msg.View)
	if err != nil {
		return nil, err
	}
	set := s.cat.Sets[req.Msg.SetNum]
	if set == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no set "+req.Msg.SetNum))
	}
	out := &pilev1.GetSetResponse{SortOutPlace: int32(w.place[set.Num])}
	e := s.index.Entry(set.Num)
	var found match.Found
	if e == nil {
		out.Set = setMatch(&match.Match{Entry: &match.Entry{Set: set}})
	} else if f, ok := w.found[set.Num]; ok {
		// In the queue: against what the sets before it left.
		out.Set = setMatch(w.queue[w.place[set.Num]-1])
		found = f
	} else {
		m, f := s.index.Score(e, w.left, w.anyColor)
		m.Pick = w.lot.picks[set.Num]
		out.Set, found = setMatch(m), f
	}
	// Lines of one part (or its variants) and color share what was found for
	// it, first come first served.
	exactLeft := map[int]int32{}
	otherLeft := map[int]int32{}
	for i := range found.Exact {
		exactLeft[i], otherLeft[i] = found.Exact[i], found.Other[i]
	}
	for _, l := range set.Lines {
		line := &pilev1.SetLine{PartNum: l.Part, Color: color(s.cat.Colors[l.Color]), Need: l.Quantity, ImageUrl: l.ImageURL, Counting: pilev1.Counting_COUNTING_COUNTED}
		if part := s.cat.Parts[l.Part]; part != nil {
			line.Name = part.Name
			switch part.Counting {
			case catalog.Printed:
				line.Counting = pilev1.Counting_COUNTING_PRINTED
			case catalog.FigurePart:
				line.Counting = pilev1.Counting_COUNTING_MINIFIGURE
			}
		}
		if line.Counting == pilev1.Counting_COUNTING_COUNTED && e != nil {
			if i := s.index.LineOf(e, l.Part, l.Color); i >= 0 {
				x := min(exactLeft[i], l.Quantity)
				o := min(otherLeft[i], l.Quantity-x)
				exactLeft[i] -= x
				otherLeft[i] -= o
				line.Found, line.WrongColor = x+o, o
			}
		}
		out.Lines = append(out.Lines, line)
	}
	for _, f := range set.Minifigures {
		out.Minifigures = append(out.Minifigures, &pilev1.Minifigure{FigNum: f.Num, Name: f.Name, Quantity: f.Quantity, ImageUrl: f.ImageURL})
	}
	return connect.NewResponse(out), nil
}

func (s *Service) ListParts(_ context.Context, req *connect.Request[pilev1.ListPartsRequest]) (*connect.Response[pilev1.ListPartsResponse], error) {
	t := s.lots[req.Msg.LotId]
	if t == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no lot "+req.Msg.LotId))
	}
	out := &pilev1.ListPartsResponse{}
	for _, p := range t.parts {
		out.Parts = append(out.Parts, &pilev1.PartCount{
			PartNum:        p.part.Num,
			BricklinkId:    p.bricklinkPart,
			Name:           p.part.Name,
			Color:          color(p.color),
			Count:          p.count,
			ImageUrl:       s.cat.Images[catalog.PartColor{Part: p.part.Num, Color: p.color.ID}],
			Category:       p.part.Category,
			MeanConfidence: float32(p.confidence / float64(p.count)),
		})
	}
	for _, u := range t.unmatched {
		out.Unmatched = append(out.Unmatched, &pilev1.UnmatchedPart{BricklinkId: u.bricklinkPart, Name: u.name, ColorName: u.colorName, Count: u.count})
	}
	return connect.NewResponse(out), nil
}
