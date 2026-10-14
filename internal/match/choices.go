package match

import "slices"

// Choice is a change a view makes to a set's colors: its lines in Color
// take Use instead (every line of that color, or only Part's when Part is
// set), or, with Any, any color of their part. A part's choice is made
// after its color's, so it can undo it for one part.
type Choice struct {
	Color int32
	Part  string
	Use   int32
	Any   bool
}

// WithChoices is e with the choices made: lines recolored (two lines that
// become one part in one color are one line), lines that take any color
// marked, and which line each of the set's own lines became kept for
// LineOf. Without choices it is e itself.
func (ix *Index) WithChoices(e *Entry, cs []Choice) *Entry {
	if len(cs) == 0 {
		return e
	}
	c := *e
	c.Lines = nil
	c.was = map[uint64]int{}
	at := map[uint64]int{}
	for _, l := range e.Lines {
		color, anyColor := l.Color, false
		for _, ch := range cs {
			if ch.Part == "" && ch.Color == l.Color {
				color, anyColor = ch.Use, ch.Any
			}
		}
		for _, ch := range cs {
			if id, ok := ix.PartID(ch.Part); ok && ch.Part != "" && id == l.Part && ch.Color == l.Color {
				color, anyColor = ch.Use, ch.Any
			}
		}
		if anyColor {
			color = l.Color
		}
		k := key(l.Part, color)
		i, ok := at[k]
		if !ok {
			i = len(c.Lines)
			at[k] = i
			c.Lines = append(c.Lines, Line{Part: l.Part, Color: color, Class: l.Class})
		}
		c.Lines[i].Quantity += l.Quantity
		c.Lines[i].Any = c.Lines[i].Any || anyColor
		c.anyLines = c.anyLines || anyColor
		c.was[key(l.Part, l.Color)] = i
	}
	// Lines in order, as the index keeps them; the map follows.
	order := make([]int, len(c.Lines))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int {
		la, lb := c.Lines[a], c.Lines[b]
		if la.Part != lb.Part {
			return int(la.Part - lb.Part)
		}
		return int(la.Color - lb.Color)
	})
	moved := make([]int, len(order))
	lines := make([]Line, len(order))
	for to, from := range order {
		moved[from] = to
		lines[to] = c.Lines[from]
	}
	c.Lines = lines
	for k, i := range c.was {
		c.was[k] = moved[i]
	}
	return &c
}
