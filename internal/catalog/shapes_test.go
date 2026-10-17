package catalog

import "testing"

// Plain plates, tiles and bricks have a shape, shorter side first; anything
// with more to it does not.
func TestShapeOf(t *testing.T) {
	for name, want := range map[string]*Shape{
		"Plate 2 x 4":            {Kind: "Plate", Height: 1, W: 2, L: 4},
		"Plate 4 x 2":            {Kind: "Plate", Height: 1, W: 2, L: 4},
		"Tile 1 x 2 with Groove": {Kind: "Tile", Height: 1, W: 1, L: 2},
		"Brick 1 x 2 x 3":        {Kind: "Brick", Height: 3, W: 1, L: 2},
		"Brick 2 x 2":            {Kind: "Brick", Height: 1, W: 2, L: 2},
		"Plate Special 1 x 2 with Clips Horizontal": nil,
		"Plate 1 x 2 with 1 Stud":                   nil,
		"Plate 2 x 2 x 2":                           nil,
		"Brick Round 2 x 2":                         nil,
	} {
		got := shapeOf(name)
		if (got == nil) != (want == nil) || (got != nil && *got != *want) {
			t.Fatalf("%s: %+v, want %+v", name, got, want)
		}
	}
}
