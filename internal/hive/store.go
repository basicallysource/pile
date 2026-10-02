package hive

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/basicallysource/pile/internal/records"
)

// The store is hive.sqlite in the data folder: every machine pulled and every
// piece it reported, as Hive gave them (BrickLink ids, unix times).
const schema = `
create table if not exists machines (
	id        text primary key,
	name      text not null,
	-- The signed-in user's own machine.
	mine      integer not null,
	pulled_at integer not null,
	pieces    integer not null
);
create table if not exists pieces (
	machine_id            text not null,
	piece_uuid            text not null,
	local_id              integer not null,
	seen_at               real,
	classification_status text,
	part_id               text,
	part_name             text,
	color_id              text,
	color_name            text,
	confidence            real,
	dead                  integer not null,
	primary key (machine_id, piece_uuid)
);
create index if not exists pieces_machine_local on pieces (machine_id, local_id);
`

// Store is hive.sqlite open for a pull.
type Store struct{ db *sql.DB }

// Open opens (making it if need be) the store at path.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(300000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// held is the highest and lowest local id stored for a machine (0 for
// none), and whether a pull of it ever finished (it then has a machines row).
func (s *Store) held(machine string) (newest, oldest int64, done bool, err error) {
	var hi, lo sql.NullInt64
	if err = s.db.QueryRow(`select max(local_id), min(local_id) from pieces where machine_id = ?`, machine).Scan(&hi, &lo); err != nil {
		return
	}
	err = s.db.QueryRow(`select count(*) > 0 from machines where id = ?`, machine).Scan(&done)
	return hi.Int64, lo.Int64, done, err
}

func (s *Store) putPieces(machine string, ps []piece) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	st, err := tx.Prepare(`insert or replace into pieces
		(machine_id, piece_uuid, local_id, seen_at, classification_status, part_id, part_name, color_id, color_name, confidence, dead)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer st.Close()
	for _, p := range ps {
		if _, err := st.Exec(machine, p.UUID, p.LocalID, p.seen(), p.Status, p.Part, p.PartName, p.Color, p.ColorName, p.Confidence, p.Dead); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) putMachine(m machine, mine bool) error {
	var n int64
	if err := s.db.QueryRow(`select count(*) from pieces where machine_id = ?`, m.ID).Scan(&n); err != nil {
		return err
	}
	_, err := s.db.Exec(`insert or replace into machines (id, name, mine, pulled_at, pieces) values (?, ?, ?, ?, ?)`,
		m.ID, m.Name, mine, time.Now().Unix(), n)
	return err
}

// Read returns the classified pieces in the store at path, from from up to
// until (the zero time for no end), of the machines which names: "mine" (the
// signed-in user's own), "all" (every machine pulled), or one machine's name
// or id. Like a sorter's own records, it leaves out pieces lost (dead).
func Read(path, which string, from, until time.Time) (*records.Records, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	cond, args := machines(which)
	args = append(args, float64(from.Unix()), float64(records.End(until)))
	rows, err := db.Query(`
		select m.id, m.name, p.part_id, coalesce(p.part_name, ''), coalesce(p.color_id, ''), coalesce(p.color_name, ''),
		       coalesce(p.confidence, 0), p.seen_at
		from pieces p join machines m on m.id = p.machine_id
		where `+cond+` and p.seen_at >= ? and p.seen_at < ?
		  and p.classification_status = 'classified' and p.dead = 0 and p.part_id is not null
		order by p.seen_at`, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer rows.Close()
	r := &records.Records{CopiedAt: st.ModTime().Unix()}
	machine := map[string]int32{}
	for rows.Next() {
		var p records.Piece
		var id, name, color string
		var seen float64
		if err := rows.Scan(&id, &name, &p.BrickLinkPart, &p.PartName, &color, &p.ColorName, &p.Confidence, &seen); err != nil {
			return nil, err
		}
		i, ok := machine[id]
		if !ok {
			i = int32(len(r.Machines))
			machine[id] = i
			r.Machines = append(r.Machines, name)
		}
		p.Machine = i
		p.BrickLinkColor, p.ColorKnown = records.Color(color)
		p.SeenAt = int64(seen)
		r.Add(p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(r.Pieces) == 0 {
		return nil, fmt.Errorf("%s: no classified pieces of %q", path, which)
	}
	return r, nil
}

// machines is the condition on machines m picking the ones which names.
func machines(which string) (string, []any) {
	switch which {
	case "mine":
		return "m.mine = 1", nil
	case "all":
		return "1 = 1", nil
	}
	return "(lower(m.name) = lower(?) or m.id = ?)", []any{which, which}
}

// Sorting is how much one machine sorted, and for how long.
type Sorting struct {
	Machine string
	// Every piece it reported, and the ones it classified.
	Pieces, Classified int
	// The time it spent sorting: the gaps between one piece and the next
	// of a minute or less, as Hive counts a machine's active time.
	Seconds float64
}

// activeGap is the longest gap between pieces still counted as sorting.
const activeGap = 60.0

// SortingTime reads how much each machine which names (as Read takes it)
// sorted, and for how long.
func SortingTime(path, which string) ([]Sorting, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	cond, args := machines(which)
	rows, err := db.Query(`
		select m.id, m.name, p.seen_at, p.classification_status = 'classified' and p.dead = 0 and p.part_id is not null
		from pieces p join machines m on m.id = p.machine_id
		where `+cond+` and p.seen_at is not null
		order by m.id, p.seen_at`, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer rows.Close()
	var out []Sorting
	var last float64
	var at string
	for rows.Next() {
		var id, name string
		var seen float64
		var classified bool
		if err := rows.Scan(&id, &name, &seen, &classified); err != nil {
			return nil, err
		}
		if len(out) == 0 || id != at {
			out = append(out, Sorting{Machine: name})
			at, last = id, seen
		}
		m := &out[len(out)-1]
		if gap := seen - last; gap > 0 && gap <= activeGap {
			m.Seconds += gap
		}
		last = seen
		m.Pieces++
		if classified {
			m.Classified++
		}
	}
	return out, rows.Err()
}
