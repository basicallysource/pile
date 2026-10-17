package changes

import (
	"os"
	"path/filepath"
	"testing"
)

// A change is kept in the file, replaces the change of its line and kind,
// leaves the line's other kinds alone, and is taken back on its own; a set
// left with none is gone from the file.
func TestKept(t *testing.T) {
	dir := t.TempDir()
	k, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	yes := true
	steps := []Change{
		{Part: "2454a", Color: 14, Colors: &Colors{Use: []int32{19}}},
		{Part: "60470b", Color: 14, Swap: []StandIn{{Parts: []Piece{{Part: "4085", Quantity: 2}}}}},
		{Part: "2454a", Color: 14, Colors: &Colors{Use: []int32{19, 2}}},
		{Part: "2454a", Color: 14, Counted: &yes},
	}
	for _, c := range steps {
		if err := k.Make("1234-1", c); err != nil {
			t.Fatal(err)
		}
	}
	again, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	cs := again.Of("1234-1")
	if len(cs) != 3 || len(cs[0].Colors.Use) != 2 || cs[1].Swap[0].Parts[0].Quantity != 2 || !*cs[2].Counted {
		t.Fatalf("kept %+v", cs)
	}
	if err := again.Undo("1234-1", Change{Part: "2454a", Color: 14, Colors: &Colors{}}); err != nil {
		t.Fatal(err)
	}
	if cs := again.Of("1234-1"); len(cs) != 2 || cs[0].Kind() != Swap || cs[1].Kind() != Count {
		t.Fatalf("after undo %+v", cs)
	}
	for _, c := range again.Of("1234-1") {
		if err := again.Undo("1234-1", c); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil || string(b) != "{}\n" {
		t.Fatalf("file %q, %v", b, err)
	}
}

// A change that changes nothing, or two things, or a swap for a whole
// color, is refused.
func TestValid(t *testing.T) {
	no := false
	for _, c := range []Change{
		{Color: 14},
		{Part: "3001", Color: 14, Colors: &Colors{}, Counted: &no},
		{Color: 14, Swap: []StandIn{{Fits: true}}},
		{Part: "3001", Color: 14, Swap: []StandIn{{}}},
		{Part: "3001", Color: 14, Swap: []StandIn{{Parts: []Piece{{Part: "3003", Quantity: 0}}}}},
		{Part: "3001", Color: 14, Colors: &Colors{Use: []int32{1}, Any: true}},
	} {
		if c.Valid() == nil {
			t.Fatalf("%+v passed", c)
		}
	}
}
