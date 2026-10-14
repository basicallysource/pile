package match

import (
	"testing"

	"github.com/basicallysource/pile/internal/catalog"
)

// In similar colors a line takes its own color first, then the colors near
// it, nearest first, and never one that is not near.
func TestSimilarTakesNearColorsOnly(t *testing.T) {
	p := pileOf(map[[2]int32]int32{{1, 4}: 1, {1, 5}: 2, {1, 6}: 9})
	e := &Entry{Lines: []Line{{Part: 1, Color: 4, Quantity: 4}}}
	cs := Colors{Mode: Similar, near: map[int32][]int32{4: {5}}}
	f := p.find(e, cs)
	if f.Exact[0] != 1 || f.Other[0] != 2 {
		t.Fatalf("exact %d other %d, want 1 and 2", f.Exact[0], f.Other[0])
	}
	if f := p.find(e, Colors{Mode: Any}); f.Other[0] != 3 {
		t.Fatalf("any color: other %d, want 3", f.Other[0])
	}
}

// A view's choices recolor a set: a whole color, then one part of it back,
// and a part that takes any color; LineOf still finds each line by the
// set's own part and color.
func TestChoices(t *testing.T) {
	ix := &Index{parts: map[string]int32{"a": 1, "b": 2}, cat: &catalog.Catalog{}}
	e := &Entry{Lines: []Line{{Part: 1, Color: 4, Quantity: 2}, {Part: 2, Color: 4, Quantity: 3}, {Part: 2, Color: 7, Quantity: 1}}}
	got := ix.WithChoices(e, []Choice{{Color: 4, Use: 5}, {Color: 4, Part: "b", Use: 7}})
	want := map[[2]int32]int32{{1, 5}: 2, {2, 7}: 4}
	if len(got.Lines) != len(want) {
		t.Fatalf("lines %v", got.Lines)
	}
	for _, l := range got.Lines {
		if want[[2]int32{l.Part, l.Color}] != l.Quantity {
			t.Fatalf("line %v, want %v", l, want)
		}
	}
	if i := got.was[key(2, 4)]; got.Lines[i].Color != 7 {
		t.Fatalf("b in 4 became %v", got.Lines[i])
	}
	anyB := ix.WithChoices(e, []Choice{{Color: 7, Part: "b", Any: true}})
	if i := anyB.was[key(2, 7)]; !anyB.Lines[i].Any || !anyB.anyLines {
		t.Fatalf("b in 7 takes any color: %v", anyB.Lines[i])
	}
}
