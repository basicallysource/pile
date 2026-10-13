package records

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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

// A piece's color is a person's correction, else the newest color a model
// gave it after the sort, else the classifier's own; the sorter's own
// predictions while sorting (live) do not count.
func TestSorterColors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local_state.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		create table piece_records (uuid text, seen_at real, classification_status text, part_id text,
			part_name text, color_id text, color_name text, category_id text, confidence real,
			dead integer not null default 0, part_correct integer, color_corrected_id text);
		create table piece_prediction_requests (id integer primary key, piece_uuid text, source text,
			request text, provider text, model_name text, created_at real);
		create table piece_predictions (id integer primary key, request_id integer, piece_uuid text,
			kind text, rank integer, value_id text, value_name text);
		insert into piece_records (uuid, seen_at, classification_status, part_id, color_id, color_name, color_corrected_id) values
			('a', 1, 'classified', '3001', '86', 'Light Bluish Gray', null),
			('b', 2, 'classified', '3001', '86', 'Light Bluish Gray', null),
			('c', 3, 'classified', '3001', '86', 'Light Bluish Gray', '5'),
			('d', 4, 'classified', '3001', '86', 'Light Bluish Gray', null);
		insert into piece_prediction_requests values
			(1, 'a', 'live', 'live', 'basically', 'colorpred-v3', 10),
			(2, 'a', 'colorpred-v4 rerun', 'colorpred-v4', 'basically', 'colorpred-v4', 20),
			(3, 'a', 'colorpred-v5 rerun', 'colorpred-v5', 'basically', 'colorpred-v5', 30),
			(4, 'c', 'colorpred-v5 rerun', 'colorpred-v5', 'basically', 'colorpred-v5', 30),
			(5, 'd', 'live', 'live', 'basically', 'colorpred-v3', 10);
		insert into piece_predictions (request_id, piece_uuid, kind, rank, value_id, value_name) values
			(1, 'a', 'color', 0, '11', 'Black'),
			(2, 'a', 'color', 0, '85', 'Dark Bluish Gray'),
			(3, 'a', 'color', 0, '85', 'Dark Bluish Gray'),
			(3, 'a', 'color', 1, '86', 'Light Bluish Gray'),
			(3, 'a', 'part', 0, '3001', 'Brick 2 x 4'),
			(4, 'c', 'color', 0, '11', 'Black'),
			(5, 'd', 'color', 0, '11', 'Black')`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	r, err := ReadSorter(path, "kitbash", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	type got struct {
		color int32
		by    string
	}
	var have []got
	for _, p := range r.Pieces {
		have = append(have, got{p.BrickLinkColor, p.ColorBy})
	}
	want := []got{{85, "colorpred-v5"}, {86, ""}, {5, CorrectedColor}, {86, ""}}
	if !reflect.DeepEqual(have, want) {
		t.Errorf("colors %v, want %v", have, want)
	}
}
