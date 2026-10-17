package match

import (
	"cmp"
	"slices"

	"github.com/basicallysource/pile/internal/catalog"
)

// The plain plates, tiles and bricks of whole studs, and the ways smaller
// ones cover a bigger one's footprint: for a set's line of one that may be
// made of any that fit, regardless of how it would hold together.
//
// A footprint is covered by cutting it straight across, and each side again
// and again, down to sizes that are parts of its kind and height (a plate's
// with plates, a 1-high brick's with 1-high bricks). That finds every way of
// a few pieces; the ways only a pinwheel of five or more can make are left
// out. A way has at most coverPieces pieces, and a footprint keeps the
// coverWays ways with the fewest.
const (
	coverPieces = 8
	coverWays   = 32
)

// readShapes notes each plain plate, tile and brick's shape, and which part
// is each shape: of several, the one in the most sets LEGO sold (df).
func (ix *Index) readShapes(df map[int32]int) {
	ix.shapes = map[int32]catalog.Shape{}
	ix.sized = map[catalog.Shape]int32{}
	nums := make([]string, 0, len(ix.cat.Parts))
	for num, p := range ix.cat.Parts {
		if p.Shape != nil {
			nums = append(nums, num)
		}
	}
	slices.Sort(nums)
	for _, num := range nums {
		sh := *ix.cat.Parts[num].Shape
		id := ix.Part(num)
		ix.shapes[id] = sh
		if cur, ok := ix.sized[sh]; !ok || df[id] > df[cur] {
			ix.sized[sh] = id
		}
	}
}

// Shape is a part's (index id) shape, when it is a plain plate, tile or
// brick.
func (ix *Index) Shape(part int32) (catalog.Shape, bool) {
	sh, ok := ix.shapes[part]
	return sh, ok
}

// Footprints is the ways to cover a part's (index id) footprint with other
// plates, tiles or bricks of its kind and height, the fewest pieces first;
// none when it is not a plain one. Each part's are worked out once.
func (ix *Index) Footprints(part int32) [][]Piece {
	sh, ok := ix.shapes[part]
	if !ok {
		return nil
	}
	if v, ok := ix.cover.Load(part); ok {
		return v.([][]Piece)
	}
	memo := map[[2]int32][]way{}
	var ways func(w, l int32) []way
	ways = func(w, l int32) []way {
		if w > l {
			w, l = l, w
		}
		if v, ok := memo[[2]int32{w, l}]; ok {
			return v
		}
		seen := map[way]bool{}
		var out []way
		add := func(r way) {
			if !seen[r] {
				seen[r] = true
				out = append(out, r)
			}
		}
		if id, ok := ix.sized[catalog.Shape{Kind: sh.Kind, Height: sh.Height, W: w, L: l}]; ok {
			add(way{{id, 1}})
		}
		// Both sides' ways come fewest pieces first, so a pair over the
		// limit ends the row.
		cut := func(a, b []way) {
			for _, x := range a {
				for _, y := range b {
					r, ok := x.with(y)
					if !ok {
						break
					}
					add(r)
				}
			}
		}
		for c := int32(1); c <= w/2; c++ {
			cut(ways(c, l), ways(w-c, l))
		}
		for c := int32(1); c <= l/2; c++ {
			cut(ways(w, c), ways(w, l-c))
		}
		slices.SortFunc(out, func(a, b way) int {
			return cmp.Or(cmp.Compare(a.pieces(), b.pieces()), slices.Compare(a.key(), b.key()))
		})
		out = out[:min(len(out), coverWays)]
		memo[[2]int32{w, l}] = out
		return out
	}
	var all [][]Piece
	for _, r := range ways(sh.W, sh.L) {
		// Not the part itself.
		if r != (way{{part, 1}}) {
			all = append(all, r.recipe())
		}
	}
	ix.cover.Store(part, all)
	return all
}

// way is a recipe while footprints are worked out: its pieces by part, in
// order of part, with room for coverPieces of them; the rest are none.
type way [coverPieces]Piece

func (r way) pieces() int32 {
	var n int32
	for _, p := range r {
		n += p.N
	}
	return n
}

// with is two ways' pieces together, if they are within coverPieces.
func (r way) with(o way) (way, bool) {
	if r.pieces()+o.pieces() > coverPieces {
		return way{}, false
	}
	out := r
	n := 0
	for n < len(out) && out[n].N > 0 {
		n++
	}
	for _, p := range o {
		if p.N == 0 {
			break
		}
		if i := slices.IndexFunc(out[:n], func(q Piece) bool { return q.Part == p.Part }); i >= 0 {
			out[i].N += p.N
		} else {
			out[n] = p
			n++
		}
	}
	slices.SortFunc(out[:n], func(x, y Piece) int { return cmp.Compare(x.Part, y.Part) })
	return out, true
}

func (r way) recipe() []Piece {
	var out []Piece
	for _, p := range r {
		if p.N > 0 {
			out = append(out, p)
		}
	}
	return out
}

// key orders ways: by their pieces' parts and counts.
func (r way) key() []int32 {
	out := make([]int32, 0, 2*len(r))
	for _, p := range r {
		out = append(out, p.Part, p.N)
	}
	return out
}
