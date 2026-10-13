package server

import (
	"errors"
	"sort"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/match"
)

// view is a stack of lots as a View sees it: the sort-out queue taken out,
// drawn from the first lot that has its pieces, and every set and custom
// model matched against what is left.
type view struct {
	stack    *stack
	anyColor bool
	filter   *pilev1.Filter
	queue    []*match.Match // the sort-out queue, each against what came before it
	found    map[string]match.Found
	drawn    map[string]match.Drawn // where each queued set's pieces come from
	place    map[string]int         // set number to its place in the queue, from 1
	left     *match.Stack
	scores   []*match.Match // every shown entry with a piece left, most interesting first
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
	w := &view{stack: st, anyColor: v.AnyColor, filter: filter, place: map[string]int{}, found: map[string]match.Found{}, drawn: map[string]match.Drawn{}, left: st.piles}
	if len(v.SortOut) > 0 {
		w.left = st.piles.Clone()
	}
	for _, num := range v.SortOut {
		e := s.index.Entry(num)
		if e == nil || w.place[num] > 0 {
			continue
		}
		m, f := s.index.Score(e, w.left.Sum, w.anyColor)
		w.found[num] = f
		w.drawn[num] = w.left.Take(e, w.anyColor)
		w.queue = append(w.queue, m)
		w.place[num] = len(w.queue)
	}
	w.scores = s.index.ScoreAll(w.left.Sum, w.anyColor)
	for _, m := range w.scores {
		m.Pick = st.picks[m.Set.Num]
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
		{(f.Kind == pilev1.Kind_KIND_SET && m.Set.Custom) || (f.Kind == pilev1.Kind_KIND_CUSTOM && !m.Set.Custom), pilev1.Rule_RULE_KIND},
		{f.LicensedOnly && !m.Set.Licensed, pilev1.Rule_RULE_LICENSED},
		{m.Need < f.MinPieces, pilev1.Rule_RULE_SIZE},
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
	case m.Set.Custom:
		if m.Share() >= customShare {
			return pilev1.Place_PLACE_CUSTOM
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
