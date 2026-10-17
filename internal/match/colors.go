package match

import (
	"slices"

	"github.com/basicallysource/pile/internal/catalog"
)

// ColorMode is which colors a set's line takes once its own color runs out:
// none, the colors near its own (a darker or older shade a builder would
// take for it), or any color of its part.
type ColorMode int

const (
	Exact ColorMode = iota
	Similar
	Any
)

// Colors is a mode with the near colors it reads, from the index.
type Colors struct {
	Mode ColorMode
	near map[int32][]int32
}

// Colors is the mode m, ready to match with.
func (ix *Index) Colors(m ColorMode) Colors { return Colors{Mode: m, near: ix.near} }

// Near is the colors near c, nearest first.
func (ix *Index) Near(c int32) []int32 { return ix.near[c] }

// others is the colors l takes beside its own, from those the pile has of
// a part (have), in the order they are drawn: with any color (the view's,
// or its set's change), every one, the most plentiful (left) first;
// otherwise the colors its set's change mixes in (h), the most plentiful
// first, then in similar colors the colors near its own by nearness.
func (cs Colors) others(l Line, h *how, have []int32, left func(int32) int32) []int32 {
	plenty := func(a, b int32) int {
		if d := left(b) - left(a); d != 0 {
			return int(d)
		}
		return int(a - b)
	}
	if cs.Mode == Any || (h != nil && h.any) {
		out := make([]int32, 0, len(have))
		for _, c := range have {
			if c != l.Color {
				out = append(out, c)
			}
		}
		slices.SortFunc(out, plenty)
		return out
	}
	var out []int32
	if h != nil {
		for _, c := range h.also {
			if c != l.Color && slices.Contains(have, c) && !slices.Contains(out, c) {
				out = append(out, c)
			}
		}
		slices.SortFunc(out, plenty)
	}
	if cs.Mode == Similar {
		for _, c := range cs.near[l.Color] {
			if slices.Contains(have, c) && !slices.Contains(out, c) {
				out = append(out, c)
			}
		}
	}
	return out
}

// Pairs of colors a builder takes for one another, by Rebrickable's names:
// shades of one color (red and dark red), a color and its old or new
// version (light gray and light bluish gray), one color under two names
// (trans-black and trans-brown). Pairs, not groups, so nearness never
// chains: red is near dark red, and dark red near reddish brown, but red
// is not near reddish brown. The first pairs of a color are its nearest.
var nearColors = [][2]string{
	{"Red", "Dark Red"},
	{"Dark Red", "Reddish Brown"},
	{"Reddish Brown", "Brown"},
	{"Reddish Brown", "Dark Brown"},
	{"Brown", "Dark Brown"},
	{"Tan", "Dark Tan"},
	{"Tan", "Warm Tan"},
	{"Light Bluish Gray", "Light Gray"},
	{"Light Bluish Gray", "Pearl Light Gray"},
	{"Light Bluish Gray", "Flat Silver"},
	{"Light Gray", "Pearl Light Gray"},
	{"Dark Bluish Gray", "Dark Gray"},
	{"Dark Bluish Gray", "Pearl Dark Gray"},
	{"Dark Gray", "Pearl Dark Gray"},
	{"Flat Silver", "Pearl Light Gray"},
	{"Flat Silver", "Metallic Silver"},
	{"Flat Silver", "Chrome Silver"},
	{"White", "Very Light Gray"},
	{"White", "Very Light Bluish Gray"},
	{"Very Light Gray", "Very Light Bluish Gray"},
	{"Blue", "Dark Blue"},
	{"Medium Blue", "Bright Light Blue"},
	{"Dark Azure", "Medium Azure"},
	{"Green", "Dark Green"},
	{"Green", "Bright Green"},
	{"Sand Green", "Olive Green"},
	{"Yellow", "Bright Light Orange"},
	{"Yellow", "Bright Light Yellow"},
	{"Yellow", "Vibrant Yellow"},
	{"Orange", "Dark Orange"},
	{"Orange", "Medium Orange"},
	{"Bright Pink", "Dark Pink"},
	{"Magenta", "Dark Pink"},
	{"Purple", "Dark Purple"},
	{"Lavender", "Medium Lavender"},
	{"Pearl Gold", "Metallic Gold"},
	{"Pearl Gold", "Chrome Gold"},
	{"Pearl Gold", "Warm Gold"},
	{"Trans-Black", "Trans-Brown"},
	{"Trans-Clear", "Opal Trans-Clear"},
	{"Trans-Light Blue", "Trans-Medium Blue"},
	{"Trans-Light Blue", "Trans-Very Lt Blue"},
	{"Trans-Green", "Trans-Bright Green"},
	{"Trans-Orange", "Trans-Neon Orange"},
	{"Trans-Yellow", "Trans-Neon Yellow"},
}

// nearTable is nearColors in the catalog's color ids; a name the catalog
// lacks is left out.
func nearTable(cat *catalog.Catalog) map[int32][]int32 {
	id := map[string]int32{}
	for _, c := range cat.Colors {
		id[c.Name] = c.ID
	}
	out := map[int32][]int32{}
	for _, p := range nearColors {
		a, ok1 := id[p[0]]
		b, ok2 := id[p[1]]
		if ok1 && ok2 {
			out[a] = append(out[a], b)
			out[b] = append(out[b], a)
		}
	}
	return out
}
