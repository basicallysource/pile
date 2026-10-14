package ldraw

import (
	"strings"
	"testing"
)

// A submodel placed in a color gives that color to its parts in color 16,
// and its place to everything in it.
func TestWalk(t *testing.T) {
	m, err := Read(strings.NewReader(`0 FILE main.ldr
1 4 100 0 0 1 0 0 0 1 0 0 0 1 Sub.ldr
1 1 0 0 0 1 0 0 0 1 0 0 0 1 3001.dat
0 NOFILE
0 FILE sub.ldr
1 16 0 -24 0 0 0 1 0 1 0 -1 0 0 3001.dat
1 15 0 0 0 1 0 0 0 1 0 0 0 1 3003.dat
`))
	if err != nil {
		t.Fatal(err)
	}
	type seen struct {
		name  string
		color int32
		at    [3]float64
	}
	var got []seen
	m.Walk(m.Main(), Inherit, Identity, func(name string, color int32, at Transform) {
		got = append(got, seen{name, color, at.Pos})
	})
	want := []seen{{"3001", 4, [3]float64{100, -24, 0}}, {"3003", 15, [3]float64{100, 0, 0}}, {"3001", 1, [3]float64{0, 0, 0}}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("part %d: got %v, want %v", i, got[i], want[i])
		}
	}
	if m.Titles["sub.ldr"] != "Sub.ldr" && m.Titles["sub.ldr"] != "sub.ldr" {
		t.Errorf("title %q", m.Titles["sub.ldr"])
	}
}
