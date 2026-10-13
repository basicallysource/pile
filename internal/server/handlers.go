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
	// Half: at least this share found, of a set of at least halfNeed.
	halfShare = 0.5
	halfNeed  = 50
	// Custom models shown: at least this share found.
	customShare = 0.6
	// Complete sets with fewer counted pieces are tiny (key chains, gear).
	tinyNeed = 5
	// A section sends at most this many; its total says how many it has.
	sectionMax = 600
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

func (s *Service) GetOverview(_ context.Context, req *connect.Request[pilev1.GetOverviewRequest]) (*connect.Response[pilev1.GetOverviewResponse], error) {
	w, err := s.view(req.Msg.View)
	if err != nil {
		return nil, err
	}
	st := w.stack
	out := &pilev1.GetOverviewResponse{Totals: st.totals, Categories: st.cats, PiecesLeft: w.left.Sum.Pieces()}
	for _, t := range st.lots {
		out.Lots = append(out.Lots, s.lotMessage(t))
	}
	for _, m := range w.queue {
		out.SortOut = append(out.SortOut, &pilev1.QueuedSet{Set: setMatch(m), Lots: st.lotCounts(w.drawn[m.Set.Num].Lots(len(st.lots)))})
	}
	listed, themes := w.listed()
	out.Themes = themes
	in := map[pilev1.Place][]*match.Match{}
	for _, m := range listed {
		if p := w.placeOf(m); p != pilev1.Place_PLACE_NONE {
			in[p] = append(in[p], m)
		}
	}
	complete, tiny, almost, half := in[pilev1.Place_PLACE_COMPLETE], in[pilev1.Place_PLACE_TINY], in[pilev1.Place_PLACE_ALMOST], in[pilev1.Place_PLACE_HALF]
	likely, custom, figures := in[pilev1.Place_PLACE_LIKELY], in[pilev1.Place_PLACE_CUSTOM], in[pilev1.Place_PLACE_MINIFIGURE]
	byShare := func(ms []*match.Match) {
		sort.SliceStable(ms, func(i, j int) bool {
			if ms[i].Share() != ms[j].Share() {
				return ms[i].Share() > ms[j].Share()
			}
			return ms[i].Need > ms[j].Need
		})
	}
	byInterest := func(ms []*match.Match) {
		sort.SliceStable(ms, func(i, j int) bool { return ms[i].Interest() > ms[j].Interest() })
	}
	byInterest(complete)
	byInterest(tiny)
	byInterest(almost)
	sort.SliceStable(half, func(i, j int) bool { return half[i].Have > half[j].Have })
	byShare(custom)
	byShare(figures)
	sort.SliceStable(likely, func(i, j int) bool { return likely[i].Pick < likely[j].Pick })
	out.Complete, out.Tiny, out.Almost = section(complete), section(tiny), section(almost)
	out.Likely, out.Custom, out.Minifigure = section(likely), section(custom), section(figures)
	out.Half = section(half)
	return connect.NewResponse(out), nil
}

func (s *Service) ListSets(_ context.Context, req *connect.Request[pilev1.ListSetsRequest]) (*connect.Response[pilev1.ListSetsResponse], error) {
	w, err := s.view(req.Msg.View)
	if err != nil {
		return nil, err
	}
	q := req.Msg
	list, themes := w.listed()
	out := &pilev1.ListSetsResponse{Themes: themes}
	by := map[pilev1.Order]func(a, b *match.Match) bool{
		pilev1.Order_ORDER_FOUND:   func(a, b *match.Match) bool { return a.Have > b.Have },
		pilev1.Order_ORDER_BIGGEST: func(a, b *match.Match) bool { return a.Need > b.Need },
		pilev1.Order_ORDER_COMPLEX: func(a, b *match.Match) bool { return a.DistinctParts > b.DistinctParts },
		pilev1.Order_ORDER_COMPLETE: func(a, b *match.Match) bool {
			if a.Share() != b.Share() {
				return a.Share() > b.Share()
			}
			return a.Need > b.Need
		},
	}
	// The scores come most interesting first, which breaks every tie.
	if less, ok := by[q.Order]; ok {
		sort.SliceStable(list, func(i, j int) bool { return less(list[i], list[j]) })
	}
	out.Sets = setMatches(list)
	return connect.NewResponse(out), nil
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
	out := &pilev1.GetSetResponse{}
	e := s.index.Entry(set.Num)
	var found match.Found
	var drawn match.Drawn
	if e == nil {
		out.Set = setMatch(&match.Match{Entry: &match.Entry{Set: set}})
	} else if f, ok := w.found[set.Num]; ok {
		// In the queue: against what the sets before it left.
		out.Set = setMatch(w.queue[w.place[set.Num]-1])
		found, drawn = f, w.drawn[set.Num]
	} else {
		m, f := s.index.Score(e, w.left.Sum, w.anyColor)
		m.Pick = w.stack.picks[set.Num]
		out.Set, found, drawn = setMatch(m), f, w.left.Sources(e, w.anyColor)
		out.Place = w.placeOf(m)
		o, _ := s.index.Score(e, w.left.Sum, !w.anyColor)
		o.Pick = m.Pick
		out.OtherColors = setMatch(o)
		out.OtherPlace = (&view{anyColor: !w.anyColor, filter: w.filter, place: w.place}).placeOf(o)
		out.HiddenBy = w.hiddenBy(m)
	}
	if w.place[set.Num] > 0 {
		out.Place = pilev1.Place_PLACE_SORT_OUT
	}
	// Lines of one part (or its variants) and color share what was found for
	// it, and the lots it is in, first come first served.
	exactLeft := map[int]int32{}
	otherLeft := map[int]int32{}
	lotsLeft := map[int][]int32{}
	for i := range found.Exact {
		exactLeft[i], otherLeft[i], lotsLeft[i] = found.Exact[i], found.Other[i], drawn[i]
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
				from := make([]int32, len(w.stack.lots))
				n := x + o
				for j, have := range lotsLeft[i] {
					t := min(have, n)
					from[j], lotsLeft[i][j], n = t, have-t, n-t
				}
				line.Lots = w.stack.lotCounts(from)
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
	st, err := s.stacks.get(req.Msg.Lots)
	if err != nil {
		return nil, err
	}
	out := &pilev1.ListPartsResponse{}
	for _, p := range st.parts {
		out.Parts = append(out.Parts, &pilev1.PartCount{
			PartNum:        p.part.Num,
			BricklinkId:    p.bricklinkPart,
			Name:           p.part.Name,
			Color:          color(p.color),
			Count:          p.count,
			ImageUrl:       s.cat.Images[catalog.PartColor{Part: p.part.Num, Color: p.color.ID}],
			Category:       p.part.Category,
			MeanConfidence: float32(p.confidence / float64(p.count)),
			Lots:           st.lotCounts(p.lots),
		})
	}
	for _, u := range st.unmatched {
		out.Unmatched = append(out.Unmatched, &pilev1.UnmatchedPart{BricklinkId: u.bricklinkPart, Name: u.name, ColorName: u.colorName, Count: u.count})
	}
	return connect.NewResponse(out), nil
}
