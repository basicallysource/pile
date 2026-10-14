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
// it is, its lines with what was found of each and where, its builds, and
// its colors with what could stand in for each.
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
	// it, and the lots it is in, first come first served.
	exactLeft := map[int]int32{}
	otherLeft := map[int]int32{}
	lotsLeft := map[int][]int32{}
	for i := range found.Exact {
		exactLeft[i], otherLeft[i], lotsLeft[i] = found.Exact[i], found.Other[i], drawn[i]
	}
	for _, l := range set.Lines {
		line := &pilev1.SetLine{PartNum: l.Part, Color: color(s.cat.Colors[l.Color]), Need: l.Quantity, ImageUrl: l.ImageURL}
		var class catalog.Class
		if part := s.cat.Parts[l.Part]; part != nil {
			line.Name, class = part.Name, part.Class
		}
		line.Class, line.Counted = partClass[class], w.counting.Counts(class)
		use, anyColor := w.choice(set.Num, l.Part, l.Color)
		line.UseColor, line.AnyColor = s.colorOf(use), anyColor
		// Pictured in the color it takes, when there is a picture of it.
		if img := s.cat.Images[catalog.PartColor{Part: l.Part, Color: use}]; use >= 0 && img != "" {
			line.ImageUrl = img
		}
		if id, ok := s.index.PartID(l.Part); ok {
			line.PartColors = s.colorCounts(lots, id, partColorsMax)
		}
		if e != nil {
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
	out.Colors = s.setColors(w, set, out.Lines, lots)
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

// choice is the view's choice for a set's line (a part in its color): the
// color it takes instead (-1: its own), or any color. A part's choice is
// made after its color's, as matching makes it.
func (w *view) choice(set, part string, c int32) (int32, bool) {
	use, anyColor := int32(-1), false
	for _, ch := range w.choices[set] {
		if ch.Part == "" && ch.Color == c {
			use, anyColor = ch.Use, ch.Any
		}
	}
	for _, ch := range w.choices[set] {
		if ch.Part == part && ch.Color == c {
			use, anyColor = ch.Use, ch.Any
		}
	}
	if anyColor {
		use = -1
	}
	return use, anyColor
}

// colorOf is the contract's color for an id; nil for none (-1).
func (s *Service) colorOf(id int32) *pilev1.Color {
	if id < 0 {
		return nil
	}
	return color(s.cat.Colors[id])
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
// found (lines as drawn above), the view's choice for it, and the other
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
			use, anyColor := w.choice(set.Num, "", l.Color)
			sc.UseColor, sc.AnyColor = s.colorOf(use), anyColor
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
