// Package records holds the pieces sorters classified, read either from a
// copy of one sorter's own records (its local_state.sqlite) or from the
// records pulled from a Hive (package hive).
package records

import "slices"

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
	// The category its sorter's profile sent it to (the profile's id for
	// it, such as "bricks_red"): which of the sorter's bins it went into.
	// Empty when the records do not say (records pulled from a Hive).
	Category string
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

// Span is a stretch of time in unix seconds, from From up to Until (0 for no
// end).
type Span struct{ From, Until int64 }

func (s Span) holds(t int64) bool { return t >= s.From && (s.Until == 0 || t < s.Until) }

// Drop takes out the pieces one sorter (by name) saw within s.
func (r *Records) Drop(machine string, s Span) {
	r.filter(func(p Piece) bool { return r.Machines[p.Machine] != machine || !s.holds(p.SeenAt) })
}

// Keep keeps only the pieces seen within one of spans.
func (r *Records) Keep(spans []Span) {
	r.filter(func(p Piece) bool {
		return slices.ContainsFunc(spans, func(s Span) bool { return s.holds(p.SeenAt) })
	})
}

// filter keeps the pieces keep says to, and the first and last seen times of
// those.
func (r *Records) filter(keep func(Piece) bool) {
	pieces := r.Pieces
	r.Pieces, r.FirstSeen, r.LastSeen = pieces[:0], 0, 0
	for _, p := range pieces {
		if keep(p) {
			r.Add(p)
		}
	}
}

// Add appends a piece, keeping the first and last seen times.
func (r *Records) Add(p Piece) {
	if r.FirstSeen == 0 || p.SeenAt < r.FirstSeen {
		r.FirstSeen = p.SeenAt
	}
	r.LastSeen = max(r.LastSeen, p.SeenAt)
	r.Pieces = append(r.Pieces, p)
}
