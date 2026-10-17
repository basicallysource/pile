package match

import (
	"slices"
	"testing"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/changes"
)

// testIndex is an index of these parts (number to name and shape), with
// ids in the order given, and their shapes read.
func testIndex(parts ...catalog.Part) *Index {
	cat := &catalog.Catalog{Parts: map[string]*catalog.Part{}}
	ix := &Index{cat: cat, parts: map[string]int32{}}
	for _, p := range parts {
		cat.Parts[p.Num] = &p
		ix.Part(p.Num)
	}
	ix.readShapes(nil)
	return ix
}

func plate(num string, w, l int32) catalog.Part {
	return catalog.Part{Num: num, Shape: &catalog.Shape{Kind: "Plate", Height: 1, W: w, L: l}}
}

// A whole color is changed, one part of it changed back; another part
// takes any color; LineOf still finds each line by the set's own part and
// color; and two lines made one part in one color share its pieces.
func TestRecolor(t *testing.T) {
	ix := testIndex(catalog.Part{Num: "a"}, catalog.Part{Num: "b"})
	e := &Entry{Lines: []Line{{Part: 0, Color: 4, Quantity: 2}, {Part: 1, Color: 4, Quantity: 3}, {Part: 1, Color: 7, Quantity: 1}}}
	got := ix.WithChanges(e, []changes.Change{
		{Color: 4, Colors: &changes.Colors{Use: []int32{5}}},
		{Part: "b", Color: 4, Colors: &changes.Colors{Use: []int32{7}}},
	})
	if got.Lines[0].Color != 5 || got.Lines[1].Color != 7 || got.Lines[2].Color != 7 {
		t.Fatalf("lines %v", got.Lines)
	}
	if i := ix.LineOf(got, "b", 4); i != 1 {
		t.Fatalf("b in 4 is line %d", i)
	}
	if cs, _ := got.Takes(2); cs != nil {
		t.Fatalf("b in 7 takes %v, want its own", cs)
	}
	p := pileOf(map[[2]int32]int32{{1, 7}: 3})
	if f := p.find(got, Colors{}); f.Exact[1]+f.Exact[2] != 3 {
		t.Fatalf("found %v of 3 pieces for two lines", f.Exact)
	}
	anyB := ix.WithChanges(e, []changes.Change{{Part: "b", Color: 7, Colors: &changes.Colors{Any: true}}})
	if _, anyColor := anyB.Takes(2); !anyColor {
		t.Fatal("b in 7 takes any color")
	}
}

// A line that mixes several colors takes its first, then the others, the
// most plentiful first, and no color it was not given.
func TestSeveralColors(t *testing.T) {
	ix := testIndex(catalog.Part{Num: "2454a"})
	e := &Entry{Lines: []Line{{Part: 0, Color: 14, Quantity: 4}}}
	got := ix.WithChanges(e, []changes.Change{{Part: "2454a", Color: 14, Colors: &changes.Colors{Use: []int32{14, 25, 226}}}})
	p := pileOf(map[[2]int32]int32{{0, 14}: 1, {0, 25}: 1, {0, 226}: 5, {0, 4}: 9})
	f := p.find(got, Colors{})
	if f.Exact[0] != 1 || f.Other[0] != 3 {
		t.Fatalf("exact %d other %d, want 1 and 3", f.Exact[0], f.Other[0])
	}
	d := p.draw(got, Colors{})
	if slices.ContainsFunc(d, func(x drawn) bool { return x.key == key(0, 4) || (x.key == key(0, 25) && x.n > 0) }) {
		t.Fatalf("drew %v: a color not given, or the scarcer one before the plentiful", d)
	}
	if cs, _ := got.Takes(0); !slices.Equal(cs, []int32{14, 25, 226}) {
		t.Fatalf("takes %v", cs)
	}
}

// A part made of other parts: its own first, then two of the stand-in
// for each one missing, in the line's color; taking it takes them.
func TestStandInParts(t *testing.T) {
	ix := testIndex(catalog.Part{Num: "60470b"}, catalog.Part{Num: "4085"})
	e := &Entry{Lines: []Line{{Part: 0, Color: 14, Quantity: 3}}}
	got := ix.WithChanges(e, []changes.Change{{Part: "60470b", Color: 14, Swap: []changes.StandIn{{Parts: []changes.Piece{{Part: "4085", Quantity: 2}}}}}})
	p := pileOf(map[[2]int32]int32{{0, 14}: 1, {1, 14}: 3, {1, 4}: 8})
	m, f := ix.Score(got, p, Colors{}, DefaultCounting)
	if f.Exact[0] != 1 || f.Made[0] != 1 || m.Have != 2 {
		t.Fatalf("exact %d made %v have %d, want 1, 1 and 2", f.Exact[0], f.Made, m.Have)
	}
	if len(f.Used) != 1 || f.Used[0] != (Used{0, 1, 14, 2}) {
		t.Fatalf("used %v, want two 4085 in 14", f.Used)
	}
	p.Take(got, Colors{})
	if p.Has(1, 14) != 1 || p.Has(1, 4) != 8 {
		t.Fatalf("left %d in 14 and %d in 4, want 1 and 8", p.Has(1, 14), p.Has(1, 4))
	}
}

// A 2 x 4 plate's footprint is covered by smaller plates in every way of a
// few pieces, not by itself; any that fit makes it from what is there.
func TestFootprints(t *testing.T) {
	ix := testIndex(plate("3024", 1, 1), plate("3023", 1, 2), plate("3022", 2, 2), plate("3710", 1, 4), plate("3020", 2, 4))
	id := func(n string) int32 { i, _ := ix.PartID(n); return i }
	ways := ix.Footprints(id("3020"))
	has := func(r ...Piece) bool {
		slices.SortFunc(r, func(a, b Piece) int { return int(a.Part - b.Part) })
		return slices.ContainsFunc(ways, func(w []Piece) bool { return slices.Equal(w, r) })
	}
	for _, r := range [][]Piece{
		{{id("3022"), 2}},
		{{id("3710"), 2}},
		{{id("3023"), 4}},
		{{id("3022"), 1}, {id("3023"), 2}},
		{{id("3024"), 8}},
	} {
		if !has(r...) {
			t.Fatalf("no way %v in %v", r, ways)
		}
	}
	if first := ways[0]; has(Piece{id("3020"), 1}) || len(first) != 1 || first[0].N != 2 {
		t.Fatalf("ways %v: itself, or not fewest first", ways)
	}
	e := &Entry{Lines: []Line{{Part: id("3020"), Color: 1, Quantity: 2}}}
	got := ix.WithChanges(e, []changes.Change{{Part: "3020", Color: 1, Swap: []changes.StandIn{{Fits: true}}}})
	p := pileOf(map[[2]int32]int32{{id("3022"), 1}: 1, {id("3023"), 1}: 3, {id("3023"), 2}: 9})
	if f := p.find(got, Colors{}); f.Made[0] != 1 {
		t.Fatalf("made %v, want one (a 2 x 2 and two 1 x 2 in its color)", f.Made)
	}
}

// A line the view leaves out can be counted for one set, and one it
// counts left out.
func TestCounted(t *testing.T) {
	ix := testIndex(catalog.Part{Num: "p"}, catalog.Part{Num: "q"})
	e := &Entry{Lines: []Line{{Part: 0, Color: 1, Quantity: 2, Class: catalog.Printed}, {Part: 1, Color: 1, Quantity: 3}}}
	yes, no := true, false
	got := ix.WithChanges(e, []changes.Change{{Part: "p", Color: 1, Counted: &yes}, {Part: "q", Color: 1, Counted: &no}})
	m, _ := ix.Score(got, pileOf(map[[2]int32]int32{{0, 1}: 2}), Colors{}, DefaultCounting)
	if m.Need != 2 || m.Have != 2 {
		t.Fatalf("need %d have %d, want the printed part's 2 of 2", m.Need, m.Have)
	}
}
