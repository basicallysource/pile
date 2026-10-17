package server

import (
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

// ListChanges is every set's changes, by set number, each with its words.
func (s *Service) ListChanges(_ context.Context, _ *connect.Request[pilev1.ListChangesRequest]) (*connect.Response[pilev1.ListChangesResponse], error) {
	all := s.changes.All()
	nums := make([]string, 0, len(all))
	for num := range all {
		nums = append(nums, num)
	}
	slices.Sort(nums)
	out := &pilev1.ListChangesResponse{}
	for _, num := range nums {
		out.Sets = append(out.Sets, s.setChanges(num, all[num]))
	}
	return connect.NewResponse(out), nil
}

// SetChange makes a change to a set the catalog has (for a line it has, in
// colors, parts and a footprint the catalog has), or takes one back, and
// keeps it in the data folder before it answers. A swap with no stand-ins
// takes the line's swap back.
func (s *Service) SetChange(_ context.Context, req *connect.Request[pilev1.SetChangeRequest]) (*connect.Response[pilev1.SetChangeResponse], error) {
	num := req.Msg.SetNum
	set := s.cat.Sets[num]
	if set == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no set "+num))
	}
	if req.Msg.Change == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no change"))
	}
	c := changeOf(req.Msg.Change)
	if req.Msg.Undo || (c.Kind() == changes.Swap && len(c.Swap) == 0) {
		if err := s.changes.Undo(num, c); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&pilev1.SetChangeResponse{Set: s.setChanges(num, s.changes.Of(num))}), nil
	}
	if err := s.checkChange(set.Lines, c); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.changes.Make(num, c); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pilev1.SetChangeResponse{Set: s.setChanges(num, s.changes.Of(num))}), nil
}

// checkChange says what is wrong with a change to a set of these lines, if
// anything.
func (s *Service) checkChange(lines []catalog.Line, c changes.Change) error {
	if err := c.Valid(); err != nil {
		return err
	}
	if !slices.ContainsFunc(lines, func(l catalog.Line) bool { return l.Color == c.Color && (c.Part == "" || l.Part == c.Part) }) {
		if c.Part == "" {
			return fmt.Errorf("the set has nothing in color %d", c.Color)
		}
		return fmt.Errorf("the set has no %s in color %d", c.Part, c.Color)
	}
	if c.Colors != nil {
		for _, id := range c.Colors.Use {
			if s.cat.Colors[id] == nil {
				return fmt.Errorf("no color %d", id)
			}
		}
	}
	for _, si := range c.Swap {
		for _, p := range si.Parts {
			if s.cat.Parts[p.Part] == nil {
				return errors.New("no part " + p.Part)
			}
		}
		if si.Fits {
			id, ok := s.index.PartID(c.Part)
			if _, shaped := s.index.Shape(id); !ok || !shaped {
				return errors.New(c.Part + " is not a plain plate, tile or brick")
			}
		}
	}
	return nil
}

// setChanges is a set's changes as the contract has them.
func (s *Service) setChanges(num string, cs []changes.Change) *pilev1.SetChanges {
	out := &pilev1.SetChanges{SetNum: num}
	for _, c := range cs {
		out.Changes = append(out.Changes, s.changeMessage(c))
	}
	return out
}

// changeOf is a change from the contract's.
func changeOf(m *pilev1.Change) changes.Change {
	c := changes.Change{Part: m.PartNum, Color: m.Color}
	switch w := m.What.(type) {
	case *pilev1.Change_Colors:
		c.Colors = &changes.Colors{Use: slices.Clone(w.Colors.GetColors()), Any: w.Colors.GetAny()}
	case *pilev1.Change_Swap:
		c.Swap = []changes.StandIn{}
		for _, si := range w.Swap.GetStandIns() {
			c.Swap = append(c.Swap, standInOf(si))
		}
	case *pilev1.Change_Counted:
		counted := w.Counted
		c.Counted = &counted
	}
	return c
}

func standInOf(m *pilev1.StandIn) changes.StandIn {
	si := changes.StandIn{Fits: m.GetFits()}
	for _, p := range m.GetParts() {
		si.Parts = append(si.Parts, changes.Piece{Part: p.PartNum, Quantity: p.Quantity})
	}
	return si
}

func standInMessage(si changes.StandIn) *pilev1.StandIn {
	m := &pilev1.StandIn{Fits: si.Fits}
	for _, p := range si.Parts {
		m.Parts = append(m.Parts, &pilev1.PartQuantity{PartNum: p.Part, Quantity: p.Quantity})
	}
	return m
}

// changeMessage is a change as the contract has it, with its words.
func (s *Service) changeMessage(c changes.Change) *pilev1.Change {
	m := &pilev1.Change{PartNum: c.Part, Color: c.Color, Text: s.changeText(c)}
	switch c.Kind() {
	case changes.Recolor:
		m.What = &pilev1.Change_Colors{Colors: &pilev1.ColorTake{Colors: slices.Clone(c.Colors.Use), Any: c.Colors.Any}}
	case changes.Swap:
		swap := &pilev1.PartSwap{}
		for _, si := range c.Swap {
			swap.StandIns = append(swap.StandIns, standInMessage(si))
		}
		m.What = &pilev1.Change_Swap{Swap: swap}
	case changes.Count:
		m.What = &pilev1.Change_Counted{Counted: *c.Counted}
	}
	return m
}

// changeText is a change in words, part numbers and color names, short
// enough for a list: "2454a Yellow → Tan", "All Red → any color",
// "60470b Yellow: or 2 x 4085", "3069bpr0001 White counted".
func (s *Service) changeText(c changes.Change) string {
	line := s.colorName(c.Color)
	if c.Part != "" {
		line = c.Part + " " + line
	} else {
		line = "All " + line
	}
	switch c.Kind() {
	case changes.Recolor:
		switch {
		case c.Colors.Any:
			return line + " → any color"
		case len(c.Colors.Use) == 0:
			return line + " → its own color"
		}
		names := make([]string, len(c.Colors.Use))
		for i, id := range c.Colors.Use {
			names[i] = s.colorName(id)
		}
		return line + " → " + orList(names)
	case changes.Swap:
		var ways []string
		for _, si := range c.Swap {
			if si.Fits {
				ways = append(ways, "any that fit")
				continue
			}
			var ps []string
			for _, p := range si.Parts {
				ps = append(ps, fmt.Sprintf("%d x %s", p.Quantity, p.Part))
			}
			ways = append(ways, strings.Join(ps, " + "))
		}
		return line + ": or " + orList(ways)
	}
	if *c.Counted {
		return line + " counted"
	}
	return line + " left out"
}

func (s *Service) colorName(id int32) string {
	if c := s.cat.Colors[id]; c != nil {
		return c.Name
	}
	return fmt.Sprintf("color %d", id)
}

// orList is "a", "a or b", "a, b or c".
func orList(xs []string) string {
	if len(xs) < 2 {
		return strings.Join(xs, "")
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " or " + xs[len(xs)-1]
}
