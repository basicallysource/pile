package match

import "testing"

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
