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
	queue    []*match.Match // the sort-out queue, each against what came before it
	found    map[string]match.Found
	place    map[string]int // set number to its place in the queue, from 1
	left     *match.Pile
	scores   []*match.Match // every shown entry with a piece left, best evidence first
}

func (s *Service) view(v *pilev1.View) (*view, error) {
	if v == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no view"))
	}
	t := s.lots[v.LotId]
	if t == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no lot "+v.LotId))
	}
	w := &view{lot: t, anyColor: v.AnyColor, place: map[string]int{}, found: map[string]match.Found{}, left: t.pile.Clone()}
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
