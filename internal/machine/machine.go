// Package machine reads the pieces a sorter classified, from a copy of the
// sorter's own records (its local_state.sqlite).
package machine

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

// Piece is one physical piece the machine saw and classified, in BrickLink's ids.
type Piece struct {
	BrickLinkPart  string
	PartName       string
	BrickLinkColor int32
	ColorName      string
	// Zero when the color is unknown (the classifier's "any color").
	ColorKnown bool
	Confidence float64
	SeenAt     int64
}

type Records struct {
	Pieces    []Piece
	CopiedAt  int64
	FirstSeen int64
	LastSeen  int64
}

// Read returns the pieces classified from from up to until (the zero time
// for no end): the ones not lost (dead) and not marked wrong by a person,
// with a person's color correction in place of the classifier's color.
func Read(path string, from, until time.Time) (*Records, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`
		select part_id, coalesce(part_name, ''), coalesce(nullif(color_corrected_id, ''), color_id),
		       coalesce(color_name, ''), coalesce(confidence, 0), seen_at
		from piece_records
		where seen_at >= ? and seen_at < ? and classification_status = 'classified' and dead = 0
		  and coalesce(part_correct, 1) != 0 and part_id is not null
		order by seen_at`, from.Unix(), end(until))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer rows.Close()
	r := &Records{CopiedAt: st.ModTime().Unix()}
	for rows.Next() {
		var p Piece
		var color sql.NullString
		var seen float64
		if err := rows.Scan(&p.BrickLinkPart, &p.PartName, &color, &p.ColorName, &p.Confidence, &seen); err != nil {
			return nil, err
		}
		if n, err := strconv.Atoi(color.String); err == nil {
			p.BrickLinkColor, p.ColorKnown = int32(n), true
		}
		p.SeenAt = int64(seen)
		if r.FirstSeen == 0 {
			r.FirstSeen = p.SeenAt
		}
		r.LastSeen = p.SeenAt
		r.Pieces = append(r.Pieces, p)
	}
	return r, rows.Err()
}

func end(t time.Time) int64 {
	if t.IsZero() {
		return 1 << 62
	}
	return t.Unix()
}
