package match

import (
	"slices"

	"github.com/basicallysource/pile/internal/changes"
)

// how is how a set's changes change one of its lines.
type how struct {
	// The colors it takes beside its color (several to mix), or any color.
	also []int32
	any  bool
	// 1: it counts toward the set's completeness, -1: it does not, whatever
	// the view counts; 0: as the view counts its class.
	count int8
	// Other parts that make one of it, each tried in turn once its own part
	// runs out.
	standIns []standIn
}

// standIn is the ways one stand-in makes one of a part: one recipe of
// other parts, or, for any that fit, every way to cover its footprint
// (footprints.go), the fewest pieces first.
type standIn [][]Piece

// Piece is n of a part (index id).
type Piece struct{ Part, N int32 }

// WithChanges is e as a set's changes make it: each line in the colors it
// takes (its color's change, then its own), the other parts that make one
// of it, and whether it counts whatever the view counts. Its lines are e's,
// in the same order, so a line of the set is found by LineOf either way.
// Without changes it is e itself.
func (ix *Index) WithChanges(e *Entry, cs []changes.Change) *Entry {
	if len(cs) == 0 {
		return e
	}
	c := *e
	c.base = e
	c.Lines = slices.Clone(e.Lines)
	c.how = make([]how, len(e.Lines))
	for i, l := range e.Lines {
		h := &c.how[i]
		var take *changes.Colors
		for _, ch := range cs {
			if ch.Part == "" && ch.Color == l.Color && ch.Colors != nil {
				take = ch.Colors
			}
		}
		for _, ch := range cs {
			if ch.Part == "" || ch.Color != l.Color {
				continue
			}
			if id, ok := ix.PartID(ch.Part); !ok || id != l.Part {
				continue
			}
			switch ch.Kind() {
			case changes.Recolor:
				take = ch.Colors
			case changes.Swap:
				h.standIns = ix.standIns(ch.Swap, l.Part)
			case changes.Count:
				h.count = -1
				if *ch.Counted {
					h.count = 1
				}
			}
		}
		switch {
		case take == nil:
		case take.Any:
			h.any = true
		case len(take.Use) > 0:
			c.Lines[i].Color = take.Use[0]
			h.also = take.Use[1:]
		}
	}
	return &c
}

// standIns is a swap's stand-ins for a part, as matching makes them: those
// whose parts the index has, and any that fit when the part has a
// footprint.
func (ix *Index) standIns(ss []changes.StandIn, part int32) []standIn {
	var out []standIn
	for _, s := range ss {
		if rs := ix.Recipes(s, part); len(rs) > 0 {
			out = append(out, rs)
		}
	}
	return out
}

// Recipes is the ways a stand-in makes one of a part (index id): its parts
// as one recipe (none when the index lacks one of them), or the ways to
// cover the part's footprint.
func (ix *Index) Recipes(s changes.StandIn, part int32) [][]Piece {
	if s.Fits {
		return ix.Footprints(part)
	}
	var r []Piece
	for _, p := range s.Parts {
		id, ok := ix.PartID(p.Part)
		if !ok {
			return nil
		}
		// Two numbers of one part are one piece of the recipe.
		if j := slices.IndexFunc(r, func(q Piece) bool { return q.Part == id }); j >= 0 {
			r[j].N += p.Quantity
		} else {
			r = append(r, Piece{id, p.Quantity})
		}
	}
	return [][]Piece{r}
}

// Takes is the colors e's line i takes instead of its set's own color (one,
// or several to mix), or any color; nil and false for its own.
func (e *Entry) Takes(i int) ([]int32, bool) {
	if e.how == nil {
		return nil, false
	}
	h := e.how[i]
	if h.any {
		return nil, true
	}
	if len(h.also) == 0 && e.Lines[i].Color == e.base.Lines[i].Color {
		return nil, false
	}
	return append([]int32{e.Lines[i].Color}, h.also...), false
}

// Counts is whether e's line i counts toward its completeness: as its
// set's change says, or as c counts its class.
func (e *Entry) Counts(i int, c Counting) bool {
	if e.how != nil {
		switch e.how[i].count {
		case 1:
			return true
		case -1:
			return false
		}
	}
	return c.Counts(e.Lines[i].Class)
}

// anyColor is whether e's line i takes any color of its part by its set's
// change.
func (e *Entry) anyColor(i int) bool { return e.how != nil && e.how[i].any }
