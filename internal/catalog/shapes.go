package catalog

import (
	"regexp"
	"strconv"
)

// Shape is a part that is a plain rectangle of whole studs: a plate, tile or
// brick named as Rebrickable names the plain ones ("Plate 2 x 4", "Tile 1 x
// 2 with Groove", "Brick 1 x 2 x 3"). Smaller ones of its kind and height
// can cover its footprint (match's footprints).
type Shape struct {
	// Plate, Tile or Brick.
	Kind string
	// A brick's height in bricks (its third number, else 1); 1 for a plate
	// or tile.
	Height int32
	// Its sides in studs, the shorter first.
	W, L int32
}

var shapeName = regexp.MustCompile(`^(Plate|Tile|Brick) (\d+) x (\d+)(?: x (\d+))?(?: with(?:out)? Groove)?$`)

// shapeOf is the shape a part's name says, if it is a plain rectangle.
func shapeOf(name string) *Shape {
	m := shapeName.FindStringSubmatch(name)
	if m == nil || (m[4] != "" && m[1] != "Brick") {
		return nil
	}
	n := func(s string) int32 { v, _ := strconv.Atoi(s); return int32(v) }
	s := &Shape{Kind: m[1], Height: 1, W: n(m[2]), L: n(m[3])}
	if m[4] != "" {
		s.Height = n(m[4])
	}
	if s.W > s.L {
		s.W, s.L = s.L, s.W
	}
	if s.W < 1 || s.Height < 1 {
		return nil
	}
	return s
}

// Relation is how one part stands for another (Rebrickable's
// part_relationships), from the first part's side.
type Relation int

const (
	// It is a print or pattern of the other.
	PrintOf Relation = iota
	// The other is a print or pattern of it.
	PrintedAs
	// The other is one of the parts it is assembled from.
	AssembledFrom
)

// Related is another part, and how it stands for a part.
type Related struct {
	Part string
	Rel  Relation
}
