package match

// Stack is the piles of several lots in priority order, and their Sum,
// which sets are matched against. Whatever a set takes from the sum is
// drawn from the first lot that has those pieces, then the next.
type Stack struct {
	Lots []*Pile
	Sum  *Pile
}

// NewStack stacks the lots' piles, first first.
func NewStack(lots []*Pile) *Stack {
	if len(lots) == 1 {
		return &Stack{Lots: lots, Sum: lots[0]}
	}
	return &Stack{Lots: lots, Sum: Sum(lots)}
}

func (s *Stack) Clone() *Stack {
	if len(s.Lots) == 1 {
		c := s.Sum.Clone()
		return &Stack{Lots: []*Pile{c}, Sum: c}
	}
	c := &Stack{Sum: s.Sum.Clone()}
	for _, p := range s.Lots {
		c.Lots = append(c.Lots, p.Clone())
	}
	return c
}

// Drawn is how many of each of an entry's lines' pieces come from each lot:
// Drawn[line][lot], the lots in the stack's order.
type Drawn [][]int32

// Lots is how many of the entry's pieces come from each lot.
func (d Drawn) Lots(lots int) []int32 {
	out := make([]int32, lots)
	for _, line := range d {
		for j, n := range line {
			out[j] += n
		}
	}
	return out
}

// Take takes e's pieces out of the stack, and says which lots they came from.
func (s *Stack) Take(e *Entry, anyColor bool) Drawn {
	ds := s.Sum.draw(e, anyColor)
	out := s.attribute(e, ds, true)
	if len(s.Lots) > 1 {
		for _, d := range ds {
			s.Sum.remove(d.key, d.n)
		}
	}
	return out
}

// Sources is which lots e's pieces would come from, were it taken.
func (s *Stack) Sources(e *Entry, anyColor bool) Drawn {
	return s.attribute(e, s.Sum.draw(e, anyColor), false)
}

// attribute draws each of ds from the lots, first first: taking it out of
// them with take, or else only counting it.
func (s *Stack) attribute(e *Entry, ds []drawn, take bool) Drawn {
	out := make(Drawn, len(e.Lines))
	for i := range out {
		out[i] = make([]int32, len(s.Lots))
	}
	used := make([]map[uint64]int32, len(s.Lots))
	for _, d := range ds {
		n := d.n
		for j, p := range s.Lots {
			if n == 0 {
				break
			}
			if used[j] == nil {
				used[j] = map[uint64]int32{}
			}
			t := min(p.exact[d.key]-used[j][d.key], n)
			if t <= 0 {
				continue
			}
			out[d.line][j] += t
			n -= t
			if take {
				p.remove(d.key, t)
			} else {
				used[j][d.key] += t
			}
		}
	}
	return out
}
