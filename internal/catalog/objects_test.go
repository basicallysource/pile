package catalog

import (
	"testing"

	"github.com/basicallysource/pile/internal/ldraw"
)

func box(x0, y0, z0, x1, y1, z1 float64) ldraw.Box {
	return ldraw.Box{Min: [3]float64{x0, y0, z0}, Max: [3]float64{x1, y1, z1}}
}

// A brick with a brick on its studs holds; a brick beside it, touching, and
// a part resting on a smooth tile do not.
func TestHolding(t *testing.T) {
	parts := []modelPart{
		{body: box(0, 0, 0, 40, 24, 20)},    // a brick, its top at y 0
		{body: box(0, -24, 0, 40, 0, 20)},   // a brick on top of it
		{body: box(40, 0, 0, 80, 24, 20)},   // a brick beside the first, touching
		{body: box(40, -8, 0, 80, 0, 20)},   // a tile on that one: no studs of its own
		{body: box(45, -28, 2, 75, -8, 18)}, // resting on the tile
	}
	studs := []modelStud{
		{at: [3]float64{10, -2, 10}, part: 0}, {at: [3]float64{30, -2, 10}, part: 0},
		{at: [3]float64{10, -26, 10}, part: 1}, {at: [3]float64{30, -26, 10}, part: 1},
		{at: [3]float64{50, -2, 10}, part: 2}, {at: [3]float64{70, -2, 10}, part: 2},
	}
	uf := newUnion(len(parts))
	holdByBodies(parts, uf)
	holdByStuds(parts, studs, uf)
	held := func(a, b int) bool { return uf.find(a) == uf.find(b) }
	for _, c := range []struct {
		a, b int
		want bool
		what string
	}{
		{0, 1, true, "a brick on studs"},
		{0, 2, false, "bricks side by side"},
		{2, 3, true, "a tile on studs"},
		{3, 4, false, "a part resting on a tile"},
	} {
		if held(c.a, c.b) != c.want {
			t.Errorf("%s: held %v, want %v", c.what, !c.want, c.want)
		}
	}
}

func TestTelling(t *testing.T) {
	for name, want := range map[string]bool{
		"1234 - Front Porch.ldr": true, "Car": true, "step84-16": false, "137-210": false,
		"Step 3": false, "flexHose2-2": false, "main.ldr": false, "m-1d": false,
	} {
		if telling(name) != want {
			t.Errorf("telling(%q) = %v, want %v", name, !want, want)
		}
	}
	if got := cleanName("1234 - Ground Floor - Minifig with Pram.ldr"); got != "Minifig with Pram" {
		t.Errorf("cleanName = %q", got)
	}
}
