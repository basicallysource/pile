package match

import (
	"sort"

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

// Take takes e's pieces out of the pile: each line's own color first, then
// (with anyColor) the part's other colors, most plentiful first.
func (p *Pile) Take(e *Entry, anyColor bool) {
	f := p.find(e, anyColor)
	for i, l := range e.Lines {
		k := key(l.Part, l.Color)
		p.exact[k] -= f.Exact[i]
		p.byPart[l.Part] -= f.Exact[i]
		rest := f.Other[i]
		if rest == 0 {
			continue
		}
		cs := append([]int32(nil), p.colors[l.Part]...)
		sort.Slice(cs, func(a, b int) bool { return p.exact[key(l.Part, cs[a])] > p.exact[key(l.Part, cs[b])] })
		for _, c := range cs {
			if rest == 0 {
				break
			}
			kc := key(l.Part, c)
			t := min(p.exact[kc], rest)
			p.exact[kc] -= t
			p.byPart[l.Part] -= t
			rest -= t
		}
	}
}
