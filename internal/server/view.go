package server

import (
	"errors"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/match"
)

// view is a lot as a View sees it: the sort-out queue taken out, and every
// set and custom model matched against what is left.
type view struct {
	lot      *lot
	anyColor bool
	showBulk bool
	queue    []*match.Match // the sort-out queue, each against what came before it
	found    map[string]match.Found
	place    map[string]int // set number to its place in the queue, from 1
	left     *match.Pile
	scores   []*match.Match // every shown entry with a piece left, most interesting first
}

func (s *Service) view(v *pilev1.View) (*view, error) {
	if v == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no view"))
	}
	t := s.lots[v.LotId]
	if t == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no lot "+v.LotId))
	}
	w := &view{lot: t, anyColor: v.AnyColor, showBulk: v.ShowBulk, place: map[string]int{}, found: map[string]match.Found{}, left: t.pile.Clone()}
	for _, num := range v.SortOut {
		e := s.index.Entry(num)
		if e == nil || w.place[num] > 0 {
			continue
		}
		m, f := s.index.Score(e, w.left, w.anyColor)
		w.found[num] = f
		w.left.Take(e, w.anyColor)
		w.queue = append(w.queue, m)
		w.place[num] = len(w.queue)
	}
	w.scores = s.index.ScoreAll(w.left, w.anyColor)
	for _, m := range w.scores {
		m.Pick = t.picks[m.Set.Num]
	}
	for _, m := range w.queue {
		m.Pick = t.picks[m.Set.Num]
	}
	return w, nil
}

// shown is whether the view lists m: sets of loose bricks only when asked.
func (w *view) shown(m *match.Match) bool { return w.showBulk || !m.Bulk }

// placeOf is which of the lot's overview sections shows m in this view.
func (w *view) placeOf(m *match.Match) pilev1.Place {
	switch {
	case w.place[m.Set.Num] > 0:
		return pilev1.Place_PLACE_SORT_OUT
	case !w.shown(m):
		return pilev1.Place_PLACE_BULK_HIDDEN
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
