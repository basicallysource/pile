package hive

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/basicallysource/pile/internal/records"
)

// RecordsFile is the records store in the data folder: the machines lots
// read through Hive's records API, each as its last pull read it.
const RecordsFile = "records.sqlite"

// Each machine's pieces (BrickLink ids, unix times; people's color labels
// and the color models' answers as JSON lists, as Hive gave them), its lots,
// and its sorting runs. A machine's digest is of what its last pull read, to
// tell whether the next one changed anything.
const recordsSchema = `
create table if not exists machines (
	id        text primary key,
	name      text not null,
	pulled_at integer not null,
	digest    text not null
);
create table if not exists pieces (
	machine_id            text not null,
	piece_uuid            text not null,
	local_id              integer not null,
	seen_at               real,
	dead                  integer not null,
	classification_status text,
	part_id               text,
	part_name             text,
	part_correct          integer,
	color_id              text,
	color_name            text,
	confidence            real,
	category_id           text,
	color_corrected_id    text,
	correction_updated_at real,
	color_labels          text not null,
	color_predictions     text not null,
	primary key (machine_id, piece_uuid)
);
create index if not exists pieces_machine_seen on pieces (machine_id, seen_at);
create table if not exists lots (
	machine_id text not null,
	id         text not null,
	name       text not null,
	primary key (machine_id, id)
);
create table if not exists runs (
	machine_id  text not null,
	id          text not null,
	lot_id      text,
	adds_to_lot integer not null,
	started_at  real not null,
	-- When the next run started; null for the newest.
	ends_at     real,
	primary key (machine_id, id)
);
`

// recordsStore is records.sqlite open for a pull.
type recordsStore struct{ db *sql.DB }

func openRecords(path string) (*recordsStore, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(60000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(recordsSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &recordsStore{db: db}, nil
}

func (s *recordsStore) Close() error { return s.db.Close() }

// put keeps what a pull read of a machine, in one transaction: its rows are
// replaced when the digest differs from the last pull's, which it says.
func (s *recordsStore) put(m recordsMachine, pieces []record, runs sortingRuns, digest string) (changed bool, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var last string
	if err := tx.QueryRow(`select digest from machines where id = ?`, m.ID).Scan(&last); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if changed = last != digest; changed {
		if err := replace(tx, m.ID, pieces, runs); err != nil {
			return false, err
		}
	}
	if _, err := tx.Exec(`insert or replace into machines (id, name, pulled_at, digest) values (?, ?, ?, ?)`,
		m.ID, m.Name, time.Now().Unix(), digest); err != nil {
		return false, err
	}
	return changed, tx.Commit()
}

// replace puts a machine's rows in place of the ones it had.
func replace(tx *sql.Tx, machine string, pieces []record, runs sortingRuns) error {
	for _, t := range []string{"pieces", "lots", "runs"} {
		if _, err := tx.Exec(`delete from `+t+` where machine_id = ?`, machine); err != nil {
			return err
		}
	}
	st, err := tx.Prepare(`insert or replace into pieces
		(machine_id, piece_uuid, local_id, seen_at, dead, classification_status, part_id, part_name, part_correct,
		 color_id, color_name, confidence, category_id, color_corrected_id, correction_updated_at, color_labels, color_predictions)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer st.Close()
	for _, p := range pieces {
		labels, err := json.Marshal(nonNil(p.Labels))
		if err != nil {
			return err
		}
		predictions, err := json.Marshal(nonNil(p.Predictions))
		if err != nil {
			return err
		}
		if _, err := st.Exec(machine, p.UUID, p.LocalID, column(p.SeenAt), p.Dead, p.Status, p.Part, p.PartName, p.PartCorrect,
			p.Color, p.ColorName, p.Confidence, p.Category, p.Corrected, column(p.CorrectedAt), string(labels), string(predictions)); err != nil {
			return err
		}
	}
	for _, l := range runs.Lots {
		if _, err := tx.Exec(`insert or replace into lots (machine_id, id, name) values (?, ?, ?)`, machine, l.ID, l.Name); err != nil {
			return err
		}
	}
	for _, r := range runs.Runs {
		started, ok := seconds(&r.StartedAt)
		if !ok {
			return fmt.Errorf("run %s: no start", r.ID)
		}
		if _, err := tx.Exec(`insert or replace into runs (machine_id, id, lot_id, adds_to_lot, started_at, ends_at) values (?, ?, ?, ?, ?, ?)`,
			machine, r.ID, r.LotID, r.AddsToLot, started, column(r.EndsAt)); err != nil {
			return err
		}
	}
	return nil
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// ReadMachine returns one machine's classified pieces in the records store at
// path, from from up to until (the zero time for no end): the ones not lost
// (dead) and not marked the wrong part by a person, each in the color
// pieceColor gives it and with the category it was sorted to. The machine is
// named by its Hive name, in any case, or its id; the records are as of its
// last pull.
func ReadMachine(path, machine string, from, until time.Time) (*records.Records, error) {
	db, err := openRead(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	id, name, pulled, err := find(db, machine)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	rows, err := db.Query(`
		select part_id, coalesce(part_name, ''), coalesce(color_id, ''), coalesce(color_name, ''),
		       coalesce(confidence, 0), seen_at, coalesce(category_id, ''), coalesce(color_corrected_id, ''),
		       coalesce(correction_updated_at, 0), color_labels, color_predictions
		from pieces
		where machine_id = ? and seen_at >= ? and seen_at < ?
		  and classification_status = 'classified' and dead = 0 and coalesce(part_correct, 1) != 0 and part_id is not null
		order by seen_at`, id, float64(from.Unix()), float64(records.End(until)))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer rows.Close()
	r := &records.Records{Machines: []string{name}, CopiedAt: pulled}
	for rows.Next() {
		var p records.Piece
		var own, ownName, corrected, labels, predictions string
		var seen, correctedAt float64
		if err := rows.Scan(&p.BrickLinkPart, &p.PartName, &own, &ownName, &p.Confidence, &seen, &p.Category,
			&corrected, &correctedAt, &labels, &predictions); err != nil {
			return nil, err
		}
		var ls []colorLabel
		var ps []colorPrediction
		if err := json.Unmarshal([]byte(labels), &ls); err != nil {
			return nil, fmt.Errorf("%s: color labels: %w", path, err)
		}
		if err := json.Unmarshal([]byte(predictions), &ps); err != nil {
			return nil, fmt.Errorf("%s: color predictions: %w", path, err)
		}
		var color string
		color, p.ColorName, p.ColorBy = pieceColor(own, ownName, corrected, correctedAt, ls, ps)
		p.BrickLinkColor, p.ColorKnown = records.Color(color)
		p.SeenAt = int64(seen)
		r.Add(p)
	}
	return r, rows.Err()
}

// MachineLot is when one machine's runs (in the records store at path) added
// pieces to one of its own lots, by name: each run that adds to it, from its
// start until the next run started. A run that is another pass over pieces
// the lot already counted adds none.
func MachineLot(path, machine, lot string) ([]records.Span, error) {
	db, err := openRead(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	id, name, _, err := find(db, machine)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	rows, err := db.Query(`
		select r.started_at, coalesce(r.ends_at, 0)
		from runs r join lots l on l.machine_id = r.machine_id and l.id = r.lot_id
		where r.machine_id = ? and l.name = ? and r.adds_to_lot = 1
		order by r.started_at`, id, lot)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer rows.Close()
	var spans []records.Span
	for rows.Next() {
		var started, ends float64
		if err := rows.Scan(&started, &ends); err != nil {
			return nil, err
		}
		spans = append(spans, records.Span{From: int64(started), Until: int64(ends)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(spans) == 0 {
		return nil, fmt.Errorf("%s: no run of %s adds to the lot %q", path, name, lot)
	}
	return spans, nil
}

func openRead(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return sql.Open("sqlite", "file:"+path+"?mode=ro")
}

// find is the machine a name or id stands for: its id, its Hive name, and
// when it was pulled.
func find(db *sql.DB, machine string) (id, name string, pulled int64, err error) {
	rows, err := db.Query(`select id, name, pulled_at from machines where lower(name) = lower(?) or id = ?`, machine, machine)
	if err != nil {
		return "", "", 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		if err := rows.Scan(&id, &name, &pulled); err != nil {
			return "", "", 0, err
		}
		n++
	}
	switch {
	case rows.Err() != nil:
		return "", "", 0, rows.Err()
	case n == 0:
		return "", "", 0, fmt.Errorf("no machine %q pulled", machine)
	case n > 1:
		return "", "", 0, fmt.Errorf("more than one machine is named %q: name it by id", machine)
	}
	return id, name, pulled, nil
}
