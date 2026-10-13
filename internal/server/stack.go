package server

import (
	"errors"
	"slices"
	"sort"
	"strings"
	"sync"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/match"
)

// stack is the lots a view looks at, in priority order, taken together:
// their parts in each color with each lot's count, their totals, the piles
// sets are matched against, and the likely order of the sets in them all.
type stack struct {
	lots []*lot
	// Most first.
	parts     []*stackPart
	unmatched []*unmatched
	totals    *pilev1.Totals
	cats      []*pilev1.CategoryShare
	piles     *match.Stack
	// The likely order, judged in exact colors (any color would let every
	// big set claim a pile of common bricks): set number to place, from 1.
	picks map[string]int
}

// stackPart is a part in a color across the lots: count is theirs together.
type stackPart struct {
	partCount
	// Each lot's count, in the stack's order.
	lots []int32
}

func newStack(ls []*lot, ix *match.Index) *stack {
	st := &stack{lots: ls}
	byPC := map[catalog.PartColor]*stackPart{}
	byUn := map[[3]string]*unmatched{}
	var piles []*match.Pile
	var machines []string
	totals := &pilev1.Totals{}
	colorBy := map[string]*pilev1.ColorSource{}
	for j, t := range ls {
		piles = append(piles, t.pile)
		totals.Pieces += t.pieces
		for by, n := range t.colorBy {
			c := colorBy[by]
			if c == nil {
				c = &pilev1.ColorSource{By: by}
				colorBy[by] = c
				totals.ColorSources = append(totals.ColorSources, c)
			}
			c.Pieces += n
		}
		for _, m := range t.Records.Machines {
			if !slices.Contains(machines, m) {
				machines = append(machines, m)
			}
		}
		if f := t.Records.FirstSeen; f != 0 && (totals.FirstSeenUnix == 0 || f < totals.FirstSeenUnix) {
			totals.FirstSeenUnix = f
		}
		totals.LastSeenUnix = max(totals.LastSeenUnix, t.Records.LastSeen)
		for _, p := range t.parts {
			k := catalog.PartColor{Part: p.part.Num, Color: p.color.ID}
			sp := byPC[k]
			if sp == nil {
				sp = &stackPart{partCount: partCount{part: p.part, bricklinkPart: p.bricklinkPart, color: p.color}, lots: make([]int32, len(ls))}
				byPC[k] = sp
				st.parts = append(st.parts, sp)
			}
			sp.count += p.count
			sp.confidence += p.confidence
			sp.lots[j] += p.count
		}
		for _, u := range t.unmatched {
			k := [3]string{u.bricklinkPart, u.name, u.colorName}
			su := byUn[k]
			if su == nil {
				su = &unmatched{bricklinkPart: u.bricklinkPart, name: u.name, colorName: u.colorName}
				byUn[k] = su
				st.unmatched = append(st.unmatched, su)
			}
			su.count += u.count
			totals.UnmatchedPieces += u.count
		}
	}
	sort.Slice(st.parts, func(i, j int) bool {
		if st.parts[i].count != st.parts[j].count {
			return st.parts[i].count > st.parts[j].count
		}
		return st.parts[i].part.Num < st.parts[j].part.Num
	})
	sort.SliceStable(st.unmatched, func(i, j int) bool { return st.unmatched[i].count > st.unmatched[j].count })
	totals.PartColors = int32(len(st.parts))
	totals.MachineName = machineName(machines)

	colors := map[int32]*pilev1.ColorShare{}
	cats := map[string]*pilev1.CategoryShare{}
	for _, p := range st.parts {
		c := colors[p.color.ID]
		if c == nil {
			c = &pilev1.ColorShare{Color: color(p.color)}
			colors[p.color.ID] = c
			totals.Colors = append(totals.Colors, c)
		}
		c.Pieces += p.count
		g := cats[p.part.Category]
		if g == nil {
			g = &pilev1.CategoryShare{Name: p.part.Category}
			cats[p.part.Category] = g
			st.cats = append(st.cats, g)
		}
		g.Pieces += p.count
	}
	sort.SliceStable(totals.Colors, func(i, j int) bool { return totals.Colors[i].Pieces > totals.Colors[j].Pieces })
	sort.Slice(totals.ColorSources, func(i, j int) bool {
		a, b := totals.ColorSources[i], totals.ColorSources[j]
		if a.Pieces != b.Pieces {
			return a.Pieces > b.Pieces
		}
		return a.By < b.By
	})
	sort.SliceStable(st.cats, func(i, j int) bool { return st.cats[i].Pieces > st.cats[j].Pieces })
	st.totals = totals

	st.piles = match.NewStack(piles)
	st.picks = map[string]int{}
	for n, e := range ix.Explain(st.piles.Sum, false) {
		st.picks[e.Set.Num] = n + 1
	}
	return st
}

// lotCounts is some count of each of the stack's lots, as messages, leaving
// out the lots with none.
func (st *stack) lotCounts(ns []int32) []*pilev1.LotCount {
	var out []*pilev1.LotCount
	for j, n := range ns {
		if n > 0 {
			out = append(out, &pilev1.LotCount{LotId: st.lots[j].ID, Pieces: n})
		}
	}
	return out
}

// stacks keeps the stacks views ask for: each lot alone, made at start, and
// the last few of several lots, made when first asked for (working out their
// likely order takes a while).
type stacks struct {
	ix   *match.Index
	lots map[string]*lot
	one  map[string]*stack

	mu      sync.Mutex
	several map[string]*making
	recent  []string
}

// The stacks of several lots kept at a time.
const severalKept = 8

type making struct {
	done chan struct{}
	st   *stack
}

func newStacks(ix *match.Index, order []*lot, byID map[string]*lot) *stacks {
	k := &stacks{ix: ix, lots: byID, one: map[string]*stack{}, several: map[string]*making{}}
	for _, t := range order {
		k.one[t.ID] = newStack([]*lot{t}, ix)
	}
	return k
}

// get is the stack of these lots, in this order.
func (k *stacks) get(ids []string) (*stack, error) {
	if len(ids) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no lots"))
	}
	ls := make([]*lot, len(ids))
	for i, id := range ids {
		t := k.lots[id]
		if t == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("no lot "+id))
		}
		if slices.Contains(ids[:i], id) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("lot "+id+" twice"))
		}
		ls[i] = t
	}
	if len(ls) == 1 {
		return k.one[ids[0]], nil
	}
	key := strings.Join(ids, ",")
	k.mu.Lock()
	m := k.several[key]
	if m == nil {
		m = &making{done: make(chan struct{})}
		k.several[key] = m
		k.recent = append(k.recent, key)
		if len(k.recent) > severalKept {
			delete(k.several, k.recent[0])
			k.recent = k.recent[1:]
		}
		k.mu.Unlock()
		m.st = newStack(ls, k.ix)
		close(m.done)
		return m.st, nil
	}
	// Asked for again: the last to go.
	k.recent = append(slices.DeleteFunc(k.recent, func(r string) bool { return r == key }), key)
	k.mu.Unlock()
	<-m.done
	return m.st, nil
}
