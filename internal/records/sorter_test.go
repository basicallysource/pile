package records

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSorterLot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local_state.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		create table lots (id text primary key, name text not null);
		create table sorting_runs (id text primary key, lot_id text, adds_to_lot integer not null, started_at real not null);
		insert into lots values ('a', 'Box'), ('b', 'Other');
		insert into sorting_runs values
			('1', 'a', 1, 100.5),
			('2', 'a', 0, 200),
			('3', 'a', 1, 300),
			('4', 'b', 1, 400),
			('5', 'a', 1, 500)`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}

	spans, err := SorterLot(path, "Box")
	if err != nil {
		t.Fatal(err)
	}
	// The pass (run 2) and the other lot's run (4) add nothing; the last run
	// has no end.
	want := []Span{{100, 200}, {300, 400}, {500, 0}}
	if !reflect.DeepEqual(spans, want) {
		t.Errorf("spans %v, want %v", spans, want)
	}
	if _, err := SorterLot(path, "Missing"); err == nil {
		t.Error("a lot no run adds to read without an error")
	}
}
