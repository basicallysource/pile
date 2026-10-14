package match

import (
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
// the line's color, and Other in the other colors the line takes (cs), once
// every line has had its own color. Lines of one part share its pieces.
type Found struct {
	Exact, Other []int32
}

// drawn is some of a pile's pieces of one part in one color (key), taken
// for one of an entry's lines.
type drawn struct {
	line int
	key  uint64
	n    int32
}

func (p *Pile) find(e *Entry, cs Colors) Found {
	f, _ := p.allot(e, cs, false)
	return f
}

// draw is what taking e's pieces out of the pile takes, without taking it.
func (p *Pile) draw(e *Entry, cs Colors) []drawn {
	_, d := p.allot(e, cs, true)
	return d
}

// allot gives each of e's lines the pile's pieces: every line its own color
// first, then the other colors it takes (Colors.others) from what is left,
// line by line; with picks, which pieces each took.
func (p *Pile) allot(e *Entry, cs Colors, picks bool) (Found, []drawn) {
	f := Found{Exact: make([]int32, len(e.Lines)), Other: make([]int32, len(e.Lines))}
	var out []drawn
	alternatives := cs.Mode != Exact || e.anyLines
	var used map[uint64]int32
	if alternatives {
		used = map[uint64]int32{}
	}
	for i, l := range e.Lines {
		k := key(l.Part, l.Color)
		x := min(p.exact[k], l.Quantity)
		f.Exact[i] = x
		if alternatives {
			used[k] += x
		}
		if picks && x > 0 {
			out = append(out, drawn{i, k, x})
		}
	}
	if !alternatives {
		return f, out
	}
	for i, l := range e.Lines {
		rest := l.Quantity - f.Exact[i]
		if rest == 0 {
			continue
		}
		left := func(c int32) int32 { k := key(l.Part, c); return p.exact[k] - used[k] }
		for _, c := range cs.others(l, p.colors[l.Part], left) {
			t := min(left(c), rest)
			if t <= 0 {
				continue
			}
			k := key(l.Part, c)
			f.Other[i] += t
			used[k] += t
			rest -= t
			if picks {
				out = append(out, drawn{i, k, t})
			}
			if rest == 0 {
				break
			}
		}
	}
	return f, out
}

// Has is how many pieces of a part (index id) in a color the pile has.
func (p *Pile) Has(part, color int32) int32 { return p.exact[key(part, color)] }

// ColorsOf is the colors the pile has of a part (index id).
func (p *Pile) ColorsOf(part int32) []int32 { return p.colors[part] }

func (p *Pile) remove(k uint64, n int32) {
	p.exact[k] -= n
	p.byPart[int32(k>>32)] -= n
}

// Take takes e's pieces out of the pile, as draw says.
func (p *Pile) Take(e *Entry, cs Colors) {
	for _, d := range p.draw(e, cs) {
		p.remove(d.key, d.n)
	}
}
