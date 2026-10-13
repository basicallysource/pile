package match

import (
	"slices"

	"github.com/basicallysource/pile/internal/catalog"
)

// Pile is a lot's pieces by part and color, the part standing for its mold
// variants and alternates.
type Pile struct {
	exact  map[uint64]int32
	byPart map[int32]int32
	// Each part's colors in the pile.
	colors map[int32][]int32
}

// NewPile counts pieces given in Rebrickable's parts and colors.
func (ix *Index) NewPile(counts map[catalog.PartColor]int32) *Pile {
	p := &Pile{exact: map[uint64]int32{}, byPart: map[int32]int32{}, colors: map[int32][]int32{}}
	for pc, n := range counts {
		part := ix.Part(pc.Part)
		k := key(part, pc.Color)
		if p.exact[k] == 0 {
			p.colors[part] = append(p.colors[part], pc.Color)
		}
		p.exact[k] += n
		p.byPart[part] += n
	}
	return p
}

// Sum is the piles' pieces together.
func Sum(piles []*Pile) *Pile {
	s := &Pile{exact: map[uint64]int32{}, byPart: map[int32]int32{}, colors: map[int32][]int32{}}
	for _, p := range piles {
		for k, n := range p.exact {
			if s.exact[k] == 0 {
				part := int32(k >> 32)
				s.colors[part] = append(s.colors[part], int32(uint32(k)))
			}
			s.exact[k] += n
		}
		for part, n := range p.byPart {
			s.byPart[part] += n
		}
	}
	return s
}

func (p *Pile) Clone() *Pile {
	c := &Pile{exact: make(map[uint64]int32, len(p.exact)), byPart: make(map[int32]int32, len(p.byPart)), colors: p.colors}
	for k, v := range p.exact {
		c.exact[k] = v
	}
	for k, v := range p.byPart {
		c.byPart[k] = v
	}
	return c
}

// Pieces is how many pieces are in the pile.
func (p *Pile) Pieces() int32 {
	var n int32
	for _, v := range p.byPart {
		n += v
	}
	return n
}

// Found is, for each of e's lines, the pieces the pile has for it: Exact in
// the line's color, and with anyColor, Other in other colors of its part
// once every line has had its own color. Lines of one part share its pieces.
type Found struct {
	Exact, Other []int32
}

func (p *Pile) find(e *Entry, anyColor bool) Found {
	f := Found{Exact: make([]int32, len(e.Lines)), Other: make([]int32, len(e.Lines))}
	for i, l := range e.Lines {
		f.Exact[i] = min(p.exact[key(l.Part, l.Color)], l.Quantity)
	}
	if !anyColor {
		return f
	}
	// What is left of each part once its lines took their own colors.
	left := map[int32]int32{}
	for i, l := range e.Lines {
		if _, ok := left[l.Part]; !ok {
			left[l.Part] = p.byPart[l.Part]
		}
		left[l.Part] -= f.Exact[i]
	}
	for i, l := range e.Lines {
		o := min(max(left[l.Part], 0), l.Quantity-f.Exact[i])
		f.Other[i] = o
		left[l.Part] -= o
	}
	return f
}

// drawn is some of a pile's pieces of one part in one color (key), taken
// for one of an entry's lines.
type drawn struct {
	line int
	key  uint64
	n    int32
}

// draw is what taking e's pieces out of the pile takes, without taking it:
// every line's own color first, then (with anyColor) the part's other colors
// left once every line has had its own, most plentiful first.
func (p *Pile) draw(e *Entry, anyColor bool) []drawn {
	f := p.find(e, anyColor)
	var out []drawn
	used := map[uint64]int32{}
	for i, l := range e.Lines {
		if f.Exact[i] > 0 {
			k := key(l.Part, l.Color)
			out = append(out, drawn{i, k, f.Exact[i]})
			used[k] += f.Exact[i]
		}
	}
	for i, l := range e.Lines {
		rest := f.Other[i]
		if rest == 0 {
			continue
		}
		left := func(c int32) int32 { k := key(l.Part, c); return p.exact[k] - used[k] }
		cs := slices.Clone(p.colors[l.Part])
		slices.SortFunc(cs, func(a, b int32) int {
			if d := left(b) - left(a); d != 0 {
				return int(d)
			}
			return int(a - b)
		})
		for _, c := range cs {
			if rest == 0 {
				break
			}
			if t := min(left(c), rest); t > 0 {
				k := key(l.Part, c)
				out = append(out, drawn{i, k, t})
				used[k] += t
				rest -= t
			}
		}
	}
	return out
}

func (p *Pile) remove(k uint64, n int32) {
	p.exact[k] -= n
	p.byPart[int32(k>>32)] -= n
}

// Take takes e's pieces out of the pile, as draw says.
func (p *Pile) Take(e *Entry, anyColor bool) {
	for _, d := range p.draw(e, anyColor) {
		p.remove(d.key, d.n)
	}
}
