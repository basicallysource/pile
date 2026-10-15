package lots

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/basicallysource/pile/internal/hive"
	"github.com/basicallysource/pile/internal/hive/hivetest"
)

// A sorterLot of a hiveMachine is the pieces of the runs that add to it: a
// pass over pieces the lot already counted adds none, and neither does a run
// of another lot between its runs.
func TestSorterLot(t *testing.T) {
	m := &hivetest.Machine{ID: "m-1", Name: "mysorter",
		Lots: []hivetest.Lot{{ID: "a", Name: "Box"}, {ID: "b", Name: "Other"}},
		Runs: hivetest.Runs(
			hivetest.Begun{Lot: "a", Adds: true, At: 100},
			hivetest.Begun{Lot: "a", Adds: false, At: 200},
			hivetest.Begun{Lot: "b", Adds: true, At: 300},
			hivetest.Begun{Lot: "a", Adds: true, At: 400},
		),
	}
	for _, at := range []int64{50, 150, 250, 350, 450} {
		m.Pieces = append(m.Pieces, hivetest.Classified(at, "3001", "11"))
	}
	h := hivetest.New(t, m)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, File), []byte(`[
		{"id": "box", "name": "Box", "description": "", "hiveMachine": "MySorter", "sorterLot": "Box"},
		{"id": "other", "name": "Other", "description": "", "hiveMachine": "m-1", "sorterLot": "Other"}
	]`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, _ := hive.NewKeyClient(h.URL, hivetest.Key)
	pull := func() {
		t.Helper()
		ls, err := Parse(dir)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := hive.PullMachines(context.Background(), c, filepath.Join(dir, hive.RecordsFile), HiveMachines(ls)); err != nil {
			t.Fatal(err)
		}
	}
	seen := func(l *Lot) []int64 {
		var out []int64
		for _, p := range l.Records.Pieces {
			out = append(out, p.SeenAt)
		}
		return out
	}
	pull()
	ls, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := seen(ls[0]), []int64{150, 450}; !reflect.DeepEqual(got, want) {
		t.Errorf("Box: pieces seen at %v, want %v", got, want)
	}
	if got, want := seen(ls[1]), []int64{350}; !reflect.DeepEqual(got, want) {
		t.Errorf("Other: pieces seen at %v, want %v", got, want)
	}

	// The lot's last run goes on; only the lots reading the machine named
	// are read again.
	m.Pieces = append(m.Pieces, hivetest.Classified(500, "3001", "11"))
	h.Set(m)
	pull()
	fresh, err := Reread(dir, ls, []string{"MySorter"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := seen(fresh[0]), []int64{150, 450, 500}; !reflect.DeepEqual(got, want) || fresh[0] == ls[0] {
		t.Errorf("Box read again: pieces seen at %v, want %v in a new lot", got, want)
	}
	if fresh[1] != ls[1] {
		t.Error("a lot reading a machine not named was read again")
	}
}

// Every lot names exactly one source, and sorterLot goes with hiveMachine.
func TestParseSources(t *testing.T) {
	for _, lot := range []string{
		`{"id": "x", "name": "X"}`,
		`{"id": "x", "name": "X", "hiveMachine": "mysorter", "hive": "mine"}`,
		`{"id": "x", "name": "X", "hive": "mine", "sorterLot": "Box"}`,
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, File), []byte("["+lot+"]"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(dir); err == nil {
			t.Errorf("%s parsed without an error", lot)
		}
	}
}
