package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/changes"
)

// A line's stand-ins list at most this many ways besides those chosen.
const standInsShown = 12

// GetStandIns is what could make one of a set's line's part instead: any
// smaller plates, tiles or bricks that fit its footprint, each way to cover
// it, the part it is a print of, its prints, the parts it is assembled
// from, and the stand-ins the line has, each with how many the lots could
// make of what the queue before the set and the set itself leave.
func (s *Service) GetStandIns(_ context.Context, req *connect.Request[pilev1.GetStandInsRequest]) (*connect.Response[pilev1.GetStandInsResponse], error) {
	w, err := s.view(req.Msg.View)
	if err != nil {
		return nil, err
	}
	num, partNum, col := req.Msg.SetNum, req.Msg.PartNum, req.Msg.Color
	base := s.index.Entry(num)
	if base == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no set "+num))
	}
	i := s.index.LineOf(base, partNum, col)
	if i < 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("%s has no %s in color %d", num, partNum, col))
	}
	id := base.Lines[i].Part
	isSwap := func(c changes.Change) bool { return c.Kind() == changes.Swap && c.Part == partNum && c.Color == col }
	kept := w.changes[num]
	var chosen []changes.StandIn
	if j := slices.IndexFunc(kept, isSwap); j >= 0 {
		chosen = kept[j].Swap
	}
	// The set without this line's stand-ins, taken out of what it is
	// matched against: what is left for them.
	e := s.index.WithChanges(base, slices.DeleteFunc(slices.Clone(kept), isSwap))
	left := w.against(num).Clone()
	left.Take(e, w.colors)
	need := base.Lines[i].Quantity
	option := func(si changes.StandIn, src pilev1.StandInSource) *pilev1.StandInOption {
		o := &pilev1.StandInOption{StandIn: standInMessage(si), Source: src, Makes: left.Makes(e, i, w.colors, s.index.Recipes(si, id), need)}
		for _, p := range si.Parts {
			sp := &pilev1.StandInPart{PartNum: p.Part, Quantity: p.Quantity, ImageUrl: s.partImage(p.Part, []int32{e.Lines[i].Color})}
			if cp := s.cat.Parts[p.Part]; cp != nil {
				sp.Name = cp.Name
			}
			if pid, ok := s.index.PartID(p.Part); ok {
				sp.Pieces = left.HasFor(e, i, w.colors, pid)
			}
			o.Parts = append(o.Parts, sp)
		}
		return o
	}

	out := &pilev1.GetStandInsResponse{}
	if _, ok := s.index.Shape(id); ok {
		out.Fits = option(changes.StandIn{Fits: true}, pilev1.StandInSource_STAND_IN_SOURCE_FOOTPRINT)
		out.Fits.Chosen = slices.ContainsFunc(chosen, func(si changes.StandIn) bool { return si.Fits })
	}
	ways := map[string]pilev1.StandInSource{}
	var order []changes.StandIn
	add := func(si changes.StandIn, src pilev1.StandInSource) {
		if k := standInKey(si); ways[k] == 0 {
			ways[k] = src
			order = append(order, si)
		}
	}
	for _, r := range s.index.Footprints(id) {
		si := changes.StandIn{}
		for _, p := range r {
			si.Parts = append(si.Parts, changes.Piece{Part: s.index.PartNum(p.Part), Quantity: p.N})
		}
		add(si, pilev1.StandInSource_STAND_IN_SOURCE_FOOTPRINT)
	}
	var sub []changes.Piece
	for _, r := range s.related(partNum) {
		if rid, ok := s.index.PartID(r.Part); ok && rid == id {
			continue
		}
		one := changes.StandIn{Parts: []changes.Piece{{Part: r.Part, Quantity: 1}}}
		switch r.Rel {
		case catalog.PrintOf:
			add(one, pilev1.StandInSource_STAND_IN_SOURCE_UNPRINTED)
		case catalog.PrintedAs:
			add(one, pilev1.StandInSource_STAND_IN_SOURCE_PRINTED)
		case catalog.AssembledFrom:
			sub = append(sub, one.Parts[0])
		}
	}
	if len(sub) > 0 {
		add(changes.StandIn{Parts: sub}, pilev1.StandInSource_STAND_IN_SOURCE_SUB_PARTS)
	}

	// Those chosen first, in the line's order, then the rest.
	taken := map[string]bool{}
	for _, si := range chosen {
		if si.Fits {
			continue
		}
		k := standInKey(si)
		src := ways[k]
		if src == 0 {
			src = pilev1.StandInSource_STAND_IN_SOURCE_PICKED
		}
		o := option(si, src)
		o.Chosen = true
		out.Options = append(out.Options, o)
		taken[k] = true
	}
	var rest []*pilev1.StandInOption
	for _, si := range order {
		if taken[standInKey(si)] {
			continue
		}
		if o := option(si, ways[standInKey(si)]); o.Makes > 0 {
			rest = append(rest, o)
		}
	}
	slices.SortStableFunc(rest, func(a, b *pilev1.StandInOption) int {
		return cmp.Or(cmp.Compare(b.Makes, a.Makes), cmp.Compare(optionPieces(a), optionPieces(b)))
	})
	out.Options = append(out.Options, rest[:min(len(rest), standInsShown)]...)
	return connect.NewResponse(out), nil
}

// related is the catalog's other parts related to a part or the part
// standing for its variants, each once.
func (s *Service) related(num string) []catalog.Related {
	out := slices.Clone(s.cat.Related[num])
	if c := s.cat.Canon(num); c != num {
		for _, r := range s.cat.Related[c] {
			if !slices.Contains(out, r) {
				out = append(out, r)
			}
		}
	}
	return out
}

// standInKey is a stand-in in words, the same for the same parts.
func standInKey(si changes.StandIn) string {
	if si.Fits {
		return "fits"
	}
	ps := make([]string, len(si.Parts))
	for i, p := range si.Parts {
		ps[i] = fmt.Sprintf("%d x %s", p.Quantity, p.Part)
	}
	slices.Sort(ps)
	return strings.Join(ps, " + ")
}

// optionPieces is how many pieces make one of the line's part.
func optionPieces(o *pilev1.StandInOption) int32 {
	var n int32
	for _, p := range o.Parts {
		n += p.Quantity
	}
	return n
}
