package records

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

// ReadSorter returns the pieces classified from from up to until (the zero
// time for no end) in a copy of one sorter's own records, its
// local_state.sqlite: the ones not lost (dead) and not marked wrong by a
// person, each with the category it was sorted to. A piece's color is a
// person's correction if there is one, else the newest color a model gave
// it after the sort (PredictedColors), else the classifier's own.
func ReadSorter(path, name string, from, until time.Time) (*Records, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	predicted, err := PredictedColors(db)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	rows, err := db.Query(`
		select part_id, coalesce(part_name, ''), coalesce(color_id, ''), coalesce(color_corrected_id, ''),
		       coalesce(color_name, ''), coalesce(confidence, 0), seen_at, coalesce(category_id, ''),
		       coalesce(uuid, '')
		from piece_records
		where seen_at >= ? and seen_at < ? and classification_status = 'classified' and dead = 0
		  and coalesce(part_correct, 1) != 0 and part_id is not null
		order by seen_at`, from.Unix(), End(until))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer rows.Close()
	r := &Records{Machines: []string{name}, CopiedAt: st.ModTime().Unix()}
	for rows.Next() {
		var p Piece
		var color, corrected, uuid string
		var seen float64
		if err := rows.Scan(&p.BrickLinkPart, &p.PartName, &color, &corrected, &p.ColorName, &p.Confidence, &seen, &p.Category, &uuid); err != nil {
			return nil, err
		}
		switch c, ok := predicted[uuid]; {
		case corrected != "":
			color, p.ColorBy = corrected, CorrectedColor
		case ok:
			color, p.ColorName, p.ColorBy = c.ID, c.Name, c.Model
		}
		p.BrickLinkColor, p.ColorKnown = Color(color)
		p.SeenAt = int64(seen)
		r.Add(p)
	}
	return r, rows.Err()
}

// PredictedColor is a color a model gave a piece after it was sorted: a
// BrickLink color id, its name, and the model.
type PredictedColor struct{ ID, Name, Model string }

// PredictedColors is, for each piece (by uuid), the top color of the newest
// request for one that the sorter did not make while sorting (source
// "live"): a model run over its pictures later, imported into the sorter's
// piece_prediction_requests and piece_predictions. None when the copy has
// no such tables.
func PredictedColors(db *sql.DB) (map[string]PredictedColor, error) {
	var n int
	if err := db.QueryRow(`select count(*) from sqlite_master where type = 'table'
		and name in ('piece_prediction_requests', 'piece_predictions')`).Scan(&n); err != nil {
		return nil, err
	}
	out := map[string]PredictedColor{}
	if n < 2 {
		return out, nil
	}
	rows, err := db.Query(`
		select p.piece_uuid, p.value_id, coalesce(p.value_name, ''), coalesce(r.model_name, r.request)
		from piece_predictions p join piece_prediction_requests r on r.id = p.request_id
		where r.source != 'live' and p.kind = 'color' and p.rank = 0
		order by r.created_at, r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uuid string
		var c PredictedColor
		if err := rows.Scan(&uuid, &c.ID, &c.Name, &c.Model); err != nil {
			return nil, err
		}
		out[uuid] = c
	}
	return out, rows.Err()
}

// SorterLot returns when a sorter's runs added pieces to one of its own lots
// (by name), from the lots and sorting_runs tables of a copy of its
// local_state.sqlite. A run lasts from its start until the next run starts,
// and a run that is another pass over pieces the lot already counted adds
// none.
func SorterLot(path, lot string) ([]Span, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`
		select coalesce(l.name, ''), r.adds_to_lot, r.started_at
		from sorting_runs r left join lots l on l.id = r.lot_id
		order by r.started_at`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer rows.Close()
	var spans []Span
	for rows.Next() {
		var name string
		var adds bool
		var started float64
		if err := rows.Scan(&name, &adds, &started); err != nil {
			return nil, err
		}
		if n := len(spans); n > 0 && spans[n-1].Until == 0 {
			spans[n-1].Until = int64(started)
		}
		if name == lot && adds {
			spans = append(spans, Span{From: int64(started)})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(spans) == 0 {
		return nil, fmt.Errorf("%s: no run adds to the lot %q", path, lot)
	}
	return spans, nil
}

// Color reads a sorter's color id: a BrickLink color, or empty or "any" when
// the classifier did not say.
func Color(s string) (int32, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return int32(n), true
}

// End is until as a unix time, far in the future for the zero time.
func End(t time.Time) int64 {
	if t.IsZero() {
		return 1 << 62
	}
	return t.Unix()
}
