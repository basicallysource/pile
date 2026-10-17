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
// the line's color, Other in the other colors the line takes (cs, and its
// set's changes), once every line has had its own color, and Made from
// other parts (its set's stand-ins), once every line has had its own part.
// Lines of one part share its pieces. Made and Used are nil for an entry
// with no stand-ins.
type Found struct {
	Exact, Other, Made []int32
	// The other parts' pieces Made takes, line by line.
	Used []Used
}

// Used is pieces of another part (index id), in a color, taken to make one
// of a line's part.
type Used struct {
	Line           int
	Part, Color, N int32
}

func (f Found) made(i int) int32 {
	if f.Made == nil {
		return 0
	}
	return f.Made[i]
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
// line by line, then other parts that make one of it (its stand-ins); with
// picks, which pieces each took.
func (p *Pile) allot(e *Entry, cs Colors, picks bool) (Found, []drawn) {
	f := Found{Exact: make([]int32, len(e.Lines)), Other: make([]int32, len(e.Lines))}
	var out []drawn
	// A changed entry's lines can share a part and color (two colors made
	// one), and take other parts, so what is used is kept as it goes.
	changed := e.how != nil
	alternatives := cs.Mode != Exact || changed
	var used map[uint64]int32
	if alternatives {
		used = map[uint64]int32{}
	}
	left := func(k uint64) int32 { return p.exact[k] - used[k] }
	for i, l := range e.Lines {
		k := key(l.Part, l.Color)
		x := min(left(k), l.Quantity)
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
		var h *how
		if changed {
			h = &e.how[i]
		}
		leftIn := func(c int32) int32 { return left(key(l.Part, c)) }
		for _, c := range cs.others(l, h, p.colors[l.Part], leftIn) {
			k := key(l.Part, c)
			t := min(left(k), rest)
			if t <= 0 {
				continue
			}
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
	if !changed {
		return f, out
	}
	for i, l := range e.Lines {
		rest := l.Quantity - f.Exact[i] - f.Other[i]
		if rest == 0 || len(e.how[i].standIns) == 0 {
			continue
		}
		if f.Made == nil {
			f.Made = make([]int32, len(e.Lines))
		}
		for _, s := range e.how[i].standIns {
			n := p.make(e, i, cs, s, rest, used, func(part, c, n int32) {
				f.Used = append(f.Used, Used{i, part, c, n})
				if picks {
					out = append(out, drawn{i, key(part, c), n})
				}
			})
			f.Made[i] += n
			rest -= n
			if rest == 0 {
				break
			}
		}
	}
	return f, out
}

// colorsFor is the colors e's line i takes a part in, from those the pile
// has of it, in the order they are drawn: the line's color first, then the
// others (Colors.others) by what is left after used.
func (p *Pile) colorsFor(e *Entry, i int, cs Colors, part int32, used map[uint64]int32) []int32 {
	l := e.Lines[i]
	var h *how
	if e.how != nil {
		h = &e.how[i]
	}
	left := func(c int32) int32 { k := key(part, c); return p.exact[k] - used[k] }
	return append([]int32{l.Color}, cs.others(l, h, p.colors[part], left)...)
}

// make makes up to most of e's line i's part from a stand-in, recipe by
// recipe, each as many times as what is left after used allows, in the
// colors the line takes; it marks what it takes as used, tells take, and
// says how many it made.
func (p *Pile) make(e *Entry, i int, cs Colors, s standIn, most int32, used map[uint64]int32, take func(part, color, n int32)) int32 {
	var made int32
	for _, r := range s {
		if made == most {
			break
		}
		n := most - made
		for _, pc := range r {
			var have int32
			for _, c := range p.colorsFor(e, i, cs, pc.Part, used) {
				have += max(p.exact[key(pc.Part, c)]-used[key(pc.Part, c)], 0)
			}
			n = min(n, have/pc.N)
		}
		if n <= 0 {
			continue
		}
		for _, pc := range r {
			want := n * pc.N
			for _, c := range p.colorsFor(e, i, cs, pc.Part, used) {
				k := key(pc.Part, c)
				t := min(max(p.exact[k]-used[k], 0), want)
				if t <= 0 {
					continue
				}
				used[k] += t
				want -= t
				take(pc.Part, c, t)
				if want == 0 {
					break
				}
			}
		}
		made += n
	}
	return made
}

// Makes is how many of e's line i's part (up to most) the pile could make
// from a stand-in's recipes, in the colors the line takes, without taking
// them.
func (p *Pile) Makes(e *Entry, i int, cs Colors, recipes [][]Piece, most int32) int32 {
	return p.make(e, i, cs, recipes, most, map[uint64]int32{}, func(int32, int32, int32) {})
}

// HasFor is how many pieces of a part (index id) the pile has in the colors
// e's line i takes.
func (p *Pile) HasFor(e *Entry, i int, cs Colors, part int32) int32 {
	var n int32
	for _, c := range p.colorsFor(e, i, cs, part, nil) {
		n += p.exact[key(part, c)]
	}
	return n
}

// Has is how many pieces of a part (index id) in a color the pile has.
func (p *Pile) Has(part, color int32) int32 { return p.exact[key(part, color)] }

// ColorsOf is the colors the pile has of a part (index id).
func (p *Pile) ColorsOf(part int32) []int32 { return p.colors[part] }

// HasPart is how many pieces of a part (index id) the pile has, in every
// color.
func (p *Pile) HasPart(part int32) int32 { return p.byPart[part] }

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
