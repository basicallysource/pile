package server

import (
	"errors"
	"sort"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/match"
)

// view is a stack of lots as a View sees it: the sort-out queue taken out,
// drawn from the first lot that has its pieces, and every set, custom model
// and build matched against what is left.
type view struct {
	stack    *stack
	colors   match.Colors
	counting match.Counting
	filter   *pilev1.Filter
	starred  map[string]bool // with the filter's starred only
	queue    []*match.Match  // the sort-out queue, each against what came before it
	found    map[string]match.Found
	drawn    map[string]match.Drawn // where each queued set's pieces come from
	place    map[string]int         // set number to its place in the queue, from 1
	left     *match.Stack
	scores   []*match.Match // every shown entry with a piece left, most interesting first
	byNum    map[string]*match.Match
	// The view's color choices, by set, and the sets as they change them.
	choices map[string][]match.Choice
	changed map[string]*match.Entry
}

// entry is e as the view's color choices change it.
func (w *view) entry(e *match.Entry) *match.Entry {
	if c, ok := w.changed[e.Set.Num]; ok {
		return c
	}
	return e
}

// colorModes are the contract's color matching as matching's.
var colorModes = map[pilev1.ColorMatch]match.ColorMode{
	pilev1.ColorMatch_COLOR_MATCH_UNSPECIFIED: match.Exact,
	pilev1.ColorMatch_COLOR_MATCH_EXACT:       match.Exact,
	pilev1.ColorMatch_COLOR_MATCH_SIMILAR:     match.Similar,
	pilev1.ColorMatch_COLOR_MATCH_ANY:         match.Any,
}

func (s *Service) view(v *pilev1.View) (*view, error) {
	if v == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no view"))
	}
	st, err := s.stacks.get(v.Lots)
	if err != nil {
		return nil, err
	}
	filter := v.Filter
	if filter == nil {
		filter = &pilev1.Filter{}
	}
	counting := match.DefaultCounting
	if c := v.Counting; c != nil {
		counting = match.Counting{Printed: c.Printed, Figure: c.FigureParts, Accessory: c.Accessories}
	}
	w := &view{stack: st, colors: s.index.Colors(colorModes[v.Colors]), counting: counting, filter: filter, starred: map[string]bool{}, place: map[string]int{}, found: map[string]match.Found{}, drawn: map[string]match.Drawn{}, left: st.piles, choices: map[string][]match.Choice{}, changed: map[string]*match.Entry{}}
	for _, c := range v.ColorChoices {
		// One that takes nothing would read as color 0, black.
		if c.Take != nil {
			w.choices[c.SetNum] = append(w.choices[c.SetNum], match.Choice{Color: c.Color, Part: c.PartNum, Use: c.GetUseColor(), Any: c.GetAnyColor()})
		}
	}
	for num, cs := range w.choices {
		if e := s.index.Entry(num); e != nil {
			w.changed[num] = s.index.WithChoices(e, cs)
		}
	}
	if filter.StarredOnly {
		w.starred = s.stars.Has()
	}
	if len(v.SortOut) > 0 {
		w.left = st.piles.Clone()
	}
	for _, num := range v.SortOut {
		e := s.index.Entry(num)
		if e == nil || w.place[num] > 0 {
			continue
		}
		e = w.entry(e)
		m, f := s.index.Score(e, w.left.Sum, w.colors, w.counting)
		w.found[num] = f
		w.drawn[num] = w.left.Take(e, w.colors)
		w.queue = append(w.queue, m)
		w.place[num] = len(w.queue)
	}
	w.scores = s.index.ScoreAll(w.left.Sum, w.colors, w.counting)
	if len(w.changed) > 0 {
		// The sets the choices change, scored as changed.
		kept := w.scores[:0]
		for _, m := range w.scores {
			if w.changed[m.Set.Num] == nil {
				kept = append(kept, m)
			}
		}
		for _, e := range w.changed {
			if m, _ := s.index.Score(e, w.left.Sum, w.colors, w.counting); !e.Hidden && m.Have > 0 && m.Need > 0 {
				kept = append(kept, m)
			}
		}
		sort.SliceStable(kept, func(i, j int) bool { return kept[i].Interest() > kept[j].Interest() })
		w.scores = kept
	}
	w.byNum = make(map[string]*match.Match, len(w.scores))
	for _, m := range w.scores {
		m.Pick = st.picks[m.Set.Num]
		w.byNum[m.Set.Num] = m
	}
	for _, m := range w.queue {
		m.Pick = st.picks[m.Set.Num]
	}
	return w, nil
}

// hiddenBy is the filter's rules that hide m; none when the view lists it.
// The theme counts for none, so a page can offer every theme the rest lets
// through.
func (w *view) hiddenBy(m *match.Match) []pilev1.Rule {
	f := w.filter
	var out []pilev1.Rule
	for _, r := range []struct {
		hides bool
		rule  pilev1.Rule
	}{
		{f.HideLooseBricks && m.Bulk, pilev1.Rule_RULE_LOOSE_BRICKS},
		{f.HideMostlyBasic && m.MostlyBasic, pilev1.Rule_RULE_MOSTLY_BASIC},
		{f.HideFewParts && m.FewParts, pilev1.Rule_RULE_FEW_PARTS},
		{f.HideNothingUnusual && m.NothingUnusual, pilev1.Rule_RULE_NOTHING_UNUSUAL},
		{f.Kind != pilev1.Kind_KIND_UNSPECIFIED && f.Kind != kinds[m.Set.Kind], pilev1.Rule_RULE_KIND},
		{f.LicensedOnly && !m.Set.Licensed, pilev1.Rule_RULE_LICENSED},
		{m.Need < f.MinPieces, pilev1.Rule_RULE_SIZE},
		{f.StarredOnly && !w.starred[m.Set.Num], pilev1.Rule_RULE_NOT_STARRED},
	} {
		if r.hides {
			out = append(out, r.rule)
		}
	}
	if f.ThemeGroup != "" && m.Set.ThemeGroup != f.ThemeGroup {
		out = append(out, pilev1.Rule_RULE_THEME)
	}
	return out
}

// listed is the scores the view lists, and the theme groups of those the
// filter lets through but for its theme, most first.
func (w *view) listed() ([]*match.Match, []*pilev1.ThemeCount) {
	var out []*match.Match
	themes := map[string]int32{}
	for _, m := range w.scores {
		h := w.hiddenBy(m)
		if len(h) == 0 || (len(h) == 1 && h[0] == pilev1.Rule_RULE_THEME) {
			themes[m.Set.ThemeGroup]++
		}
		if len(h) == 0 {
			out = append(out, m)
		}
	}
	var ts []*pilev1.ThemeCount
	for name, n := range themes {
		ts = append(ts, &pilev1.ThemeCount{Name: name, Sets: n})
	}
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].Sets != ts[j].Sets {
			return ts[i].Sets > ts[j].Sets
		}
		return ts[i].Name < ts[j].Name
	})
	return out, ts
}

// placeOf is which of the overview's sections shows m in this view.
func (w *view) placeOf(m *match.Match) pilev1.Place {
	switch {
	case w.place[m.Set.Num] > 0:
		return pilev1.Place_PLACE_SORT_OUT
	case len(w.hiddenBy(m)) > 0:
		return pilev1.Place_PLACE_HIDDEN
	case m.Set.Collection != "":
		return pilev1.Place_PLACE_COLLECTION
	case m.Set.Kind == catalog.CustomModel:
		if m.Share() >= customShare {
			return pilev1.Place_PLACE_CUSTOM
		}
	case m.Set.Kind == catalog.Build:
		// A complete set's builds are complete too, and say nothing more.
		if of := w.byNum[m.Set.Of.Num]; m.Share() >= buildShare && (of == nil || !of.Complete()) {
			return pilev1.Place_PLACE_BUILD
		}
	case m.MinifigureSet:
		if m.Complete() || m.Share() >= almostShare || m.Pick > 0 {
			return pilev1.Place_PLACE_MINIFIGURE
		}
	case m.Complete() && m.Need < tinyNeed:
		return pilev1.Place_PLACE_TINY
	case m.Complete():
		return pilev1.Place_PLACE_COMPLETE
	case m.Share() >= almostShare:
		return pilev1.Place_PLACE_ALMOST
	case m.Pick > 0:
		return pilev1.Place_PLACE_LIKELY
	case m.Share() >= halfShare && m.Need >= halfNeed:
		return pilev1.Place_PLACE_HALF
	}
	return pilev1.Place_PLACE_NONE
}
