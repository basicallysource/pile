package storage

import (
	"reflect"
	"testing"
)

func TestCheck(t *testing.T) {
	lots := func(id string) bool { return id == "box" || id == "tub" }
	for _, c := range []struct {
		name string
		bins []Bin
		ok   bool
	}{
		{"categories and a rest bin", []Bin{
			{Address: "1A-01", Lot: "box", Categories: []string{"bricks_red", "plates_red"}},
			{Address: "1A-02", Lot: "box"},
			{Address: "1A-03", Lot: "tub", Categories: []string{"bricks_red"}},
		}, true},
		{"no address", []Bin{{Address: " ", Lot: "box"}}, false},
		{"one address twice, in any case", []Bin{{Address: "1a-01", Lot: "box"}, {Address: "1A-01", Lot: "tub"}}, false},
		{"no such lot", []Bin{{Address: "1A-01", Lot: "crate"}}, false},
		{"a category in two bins", []Bin{
			{Address: "1A-01", Lot: "box", Categories: []string{"snot"}},
			{Address: "1A-02", Lot: "box", Categories: []string{"bricks_red", "snot"}},
		}, false},
		{"two rest bins for a lot", []Bin{{Address: "1A-01", Lot: "box"}, {Address: "1A-02", Lot: "box"}}, false},
	} {
		err := (&Storage{Bins: c.bins}).Check(lots)
		if (err == nil) != c.ok {
			t.Errorf("%s: error %v", c.name, err)
		}
	}
}

func TestHolder(t *testing.T) {
	s := &Storage{Bins: []Bin{
		{Address: "1A-01", Lot: "box", Categories: []string{"snot"}},
		{Address: "1A-02", Lot: "box"},
		{Address: "1A-03", Lot: "tub", Categories: []string{"snot"}},
	}}
	holder := s.Holder()
	type got struct {
		bin int
		ok  bool
	}
	for _, c := range []struct {
		lot, category string
		want          got
	}{
		{"box", "snot", got{0, true}},
		{"box", "wedges", got{1, true}}, // the rest of the lot
		{"box", "", got{1, true}},
		{"tub", "snot", got{2, true}},
		{"tub", "wedges", got{0, false}}, // the tub has no rest bin
	} {
		bin, ok := holder(c.lot, c.category)
		if g := (got{bin, ok}); !ok && !c.want.ok || reflect.DeepEqual(g, c.want) {
			continue
		}
		t.Errorf("%s %q: bin %d %v, want %v", c.lot, c.category, bin, ok, c.want)
	}
}
