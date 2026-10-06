// Package records holds the pieces sorters classified, read either from a
// copy of one sorter's own records (its local_state.sqlite) or from the
// records pulled from a Hive (package hive).
package records

// Piece is one physical piece a sorter saw and classified, in BrickLink's ids.
type Piece struct {
	BrickLinkPart  string
	PartName       string
	BrickLinkColor int32
	ColorName      string
	// False when the color is unknown (the classifier's "any color").
	ColorKnown bool
	Confidence float64
	SeenAt     int64
	// Which of Records.Machines sorted it.
	Machine int32
}

// Records is the classified pieces of one or more sorters over a stretch of
// time.
type Records struct {
	Pieces []Piece
	// The sorters the pieces came from, by name (two may share one).
	Machines []string
	// When the records were copied or pulled.
	CopiedAt  int64
	FirstSeen int64
	LastSeen  int64
}

// Drop takes out the pieces one sorter (by name) saw from from up to until
// (unix seconds; until 0 for no end), keeping the first and last seen times.
func (r *Records) Drop(machine string, from, until int64) {
	kept := r.Pieces[:0]
	r.FirstSeen, r.LastSeen = 0, 0
	for _, p := range r.Pieces {
		if r.Machines[p.Machine] == machine && p.SeenAt >= from && (until == 0 || p.SeenAt < until) {
			continue
		}
		if r.FirstSeen == 0 || p.SeenAt < r.FirstSeen {
			r.FirstSeen = p.SeenAt
		}
		r.LastSeen = max(r.LastSeen, p.SeenAt)
		kept = append(kept, p)
	}
	r.Pieces = kept
}

// Add appends a piece, keeping the first and last seen times.
func (r *Records) Add(p Piece) {
	if r.FirstSeen == 0 || p.SeenAt < r.FirstSeen {
		r.FirstSeen = p.SeenAt
	}
	r.LastSeen = max(r.LastSeen, p.SeenAt)
	r.Pieces = append(r.Pieces, p)
}
