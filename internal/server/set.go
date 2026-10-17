package server

import (
	"context"
	"errors"
	"slices"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/match"
)

const (
	// A line's part shows at most this many of its colors in the lots.
	partColorsMax = 12
	// A set's color shows at most this many colors to take instead, and the
	// colors near it besides.
	swapsMax = 8
)

// GetSet is one set, custom model or build against the view: how complete
// it is, its lines with what was found of each and where (made of other
// parts too), its builds, its colors with what could stand in for each, and
// its changes.
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
		e = w.entry(e)
		out.Set = setMatch(w.queue[w.place[set.Num]-1])
		found, drawn = f, w.drawn[set.Num]
	} else {
		e = w.entry(e)
		m, f := s.index.Score(e, w.left.Sum, w.colors, w.counting)
		m.Pick = w.stack.picks[set.Num]
		out.Set, found, drawn = setMatch(m), f, w.left.Sources(e, w.colors)
		out.Place = w.placeOf(m)
		other := s.index.Colors(match.Any)
		if w.colors.Mode == match.Any {
			other = s.index.Colors(match.Exact)
		}
		o, _ := s.index.Score(e, w.left.Sum, other, w.counting)
		o.Pick = m.Pick
		out.OtherColors = setMatch(o)
		out.OtherPlace = (&view{colors: other, counting: w.counting, filter: w.filter, starred: w.starred, place: w.place}).placeOf(o)
		out.HiddenBy = w.hiddenBy(m)
	}
	if w.place[set.Num] > 0 {
		out.Place = pilev1.Place_PLACE_SORT_OUT
	}
	lots := w.stack.piles.Sum
	// Lines of one part (or its variants) and color share what was found for
	// it, made of other parts for it, and the lots it is in, first come
	// first served.
	exactLeft := map[int]int32{}
	otherLeft := map[int]int32{}
	madeLeft := map[int]int32{}
	lotsLeft := map[int][]int32{}
	for i := range found.Exact {
		exactLeft[i], otherLeft[i], lotsLeft[i] = found.Exact[i], found.Other[i], drawn[i]
		if found.Made != nil {
			madeLeft[i] = found.Made[i]
		}
	}
	usedGiven := map[int]bool{}
	for _, l := range set.Lines {
		line := &pilev1.SetLine{PartNum: l.Part, Color: color(s.cat.Colors[l.Color]), Need: l.Quantity, ImageUrl: l.ImageURL}
		var class catalog.Class
		if part := s.cat.Parts[l.Part]; part != nil {
			line.Name, class = part.Name, part.Class
		}
		line.Class, line.Counted = partClass[class], w.counting.Counts(class)
		if id, ok := s.index.PartID(l.Part); ok {
			line.PartColors = s.colorCounts(lots, id, partColorsMax)
			_, line.HasFootprint = s.index.Shape(id)
		}
		i := -1
		if e != nil {
			i = s.index.LineOf(e, l.Part, l.Color)
		}
		if i >= 0 {
			line.Counted = e.Counts(i, w.counting)
			use, anyColor := e.Takes(i)
			line.AnyColor = anyColor
			for _, c := range use {
				line.UseColors = append(line.UseColors, color(s.cat.Colors[c]))
			}
			// Pictured in the color it takes, when there is a picture of it.
			if img := s.cat.Images[catalog.PartColor{Part: l.Part, Color: e.Lines[i].Color}]; len(use) > 0 && img != "" {
				line.ImageUrl = img
			}
			x := min(exactLeft[i], l.Quantity)
			o := min(otherLeft[i], l.Quantity-x)
			m := min(madeLeft[i], l.Quantity-x-o)
			exactLeft[i] -= x
			otherLeft[i] -= o
			madeLeft[i] -= m
			line.Found, line.WrongColor, line.Made = x+o+m, o, m
			taken := x + o
			if m > 0 && !usedGiven[i] {
				usedGiven[i] = true
				for _, u := range found.Used {
					if u.Line == i {
						line.MadeFrom = append(line.MadeFrom, s.usedPiece(u))
						taken += u.N
					}
				}
			}
			from := make([]int32, len(w.stack.lots))
			for j, have := range lotsLeft[i] {
				t := min(have, taken)
				from[j], lotsLeft[i][j], taken = t, have-t, taken-t
			}
			line.Lots = w.stack.lotCounts(from)
		}
		out.Lines = append(out.Lines, line)
	}
	out.Colors = s.setColors(w, set, out.Lines, lots)
	for _, c := range w.changes[set.Num] {
		out.Changes = append(out.Changes, s.changeMessage(c))
	}
	for _, f := range set.Minifigures {
		out.Minifigures = append(out.Minifigures, &pilev1.Minifigure{FigNum: f.Num, Name: f.Name, Quantity: f.Quantity, ImageUrl: f.ImageURL})
	}
	for _, b := range set.Builds {
		if be := s.index.Entry(b.Num); be != nil {
			m, _ := s.index.Score(w.entry(be), w.left.Sum, w.colors, w.counting)
			out.Builds = append(out.Builds, setMatch(m))
		}
	}
	if src := set.Source; src != nil {
		out.SourcePlaced, out.SourceUnplaced = src.Placed, src.Unplaced
	}
	return connect.NewResponse(out), nil
}

// usedPiece is pieces of another part taken to make one of a line's, as
// the contract has them.
func (s *Service) usedPiece(u match.Used) *pilev1.PartCount {
	num := s.index.PartNum(u.Part)
	pc := &pilev1.PartCount{PartNum: num, Color: color(s.cat.Colors[u.Color]), Count: u.N, ImageUrl: s.cat.Images[catalog.PartColor{Part: num, Color: u.Color}]}
	if p := s.cat.Parts[num]; p != nil {
		pc.Name = p.Name
	}
	return pc
}

// colorCounts is a part's colors in a pile, the most pieces first, at most n.
func (s *Service) colorCounts(p *match.Pile, part int32, n int) []*pilev1.ColorCount {
	cs := slices.Clone(p.ColorsOf(part))
	slices.SortFunc(cs, func(a, b int32) int {
		if d := p.Has(part, b) - p.Has(part, a); d != 0 {
			return int(d)
		}
		return int(a - b)
	})
	var out []*pilev1.ColorCount
	for _, c := range cs[:min(len(cs), n)] {
		if k := p.Has(part, c); k > 0 {
			out = append(out, &pilev1.ColorCount{Color: color(s.cat.Colors[c]), Pieces: k})
		}
	}
	return out
}

// setColors is a set's colors, the most pieces first: what of each was
// found (lines as drawn above), the set's change for it, and the other
// colors the lots have its parts in, each with how many of the color's
// pieces it would find (each line capped at what it needs).
func (s *Service) setColors(w *view, set *catalog.Set, lines []*pilev1.SetLine, lots *match.Pile) []*pilev1.SetColor {
	byColor := map[int32]*pilev1.SetColor{}
	swaps := map[int32]map[int32]int32{}
	var order []int32
	for i, l := range set.Lines {
		sc := byColor[l.Color]
		if sc == nil {
			sc = &pilev1.SetColor{Color: color(s.cat.Colors[l.Color])}
			for _, ch := range w.changes[set.Num] {
				if ch.Part == "" && ch.Color == l.Color && ch.Colors != nil {
					sc.AnyColor = ch.Colors.Any
					for _, c := range ch.Colors.Use {
						sc.UseColors = append(sc.UseColors, color(s.cat.Colors[c]))
					}
				}
			}
			byColor[l.Color] = sc
			swaps[l.Color] = map[int32]int32{}
			order = append(order, l.Color)
		}
		sc.Need += l.Quantity
		sc.Found += lines[i].Found
		id, ok := s.index.PartID(l.Part)
		if !ok {
			continue
		}
		for _, c := range lots.ColorsOf(id) {
			if c != l.Color {
				swaps[l.Color][c] += min(l.Quantity, lots.Has(id, c))
			}
		}
	}
	var out []*pilev1.SetColor
	for _, c := range order {
		sc := byColor[c]
		var cs []int32
		for to := range swaps[c] {
			cs = append(cs, to)
		}
		slices.SortFunc(cs, func(a, b int32) int {
			if d := swaps[c][b] - swaps[c][a]; d != 0 {
				return int(d)
			}
			return int(a - b)
		})
		// The colors near it always, the rest as far as there is room.
		shown := cs[:min(len(cs), swapsMax)]
		for _, n := range s.index.Near(c) {
			if swaps[c][n] > 0 && !slices.Contains(shown, n) {
				shown = append(slices.Clip(shown), n)
			}
		}
		for _, to := range shown {
			sc.Swaps = append(sc.Swaps, &pilev1.ColorCount{Color: color(s.cat.Colors[to]), Pieces: swaps[c][to]})
		}
		out = append(out, sc)
	}
	slices.SortStableFunc(out, func(a, b *pilev1.SetColor) int { return int(b.Need - a.Need) })
	return out
}
