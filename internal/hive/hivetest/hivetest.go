// Package hivetest is a fake Hive serving the records API, for tests: the
// key owner's machines, their pieces newest first a page at a time, and
// their lots and sorting runs.
package hivetest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Key is the API key the fake Hive takes.
const Key = "test-key-records-read"

// Hive is a fake Hive. Change its machines with Set while it serves.
type Hive struct {
	*httptest.Server
	// The most pieces a page holds, whatever the request asks.
	Page int

	mu       sync.Mutex
	machines []*Machine
	pages    int
}

// Machine is one machine as the records API gives it.
type Machine struct {
	ID, Name string
	Pieces   []Piece
	Lots     []Lot
	Runs     []Run
}

// Piece is one piece's record, by its JSON fields.
type Piece map[string]any

// Lot and Run are a machine's lot and sorting run.
type Lot struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Run struct {
	ID        string  `json:"id"`
	LotID     *string `json:"lot_id"`
	AddsToLot bool    `json:"adds_to_lot"`
	StartedAt string  `json:"started_at"`
	EndsAt    *string `json:"ends_at"`
}

// At is a unix time as Hive gives times.
func At(unix int64) string { return time.Unix(unix, 0).UTC().Format(time.RFC3339Nano) }

// Classified is a classified piece of a part in a color (BrickLink ids),
// seen at a unix time; its local id is the time.
func Classified(seen int64, part, color string) Piece {
	return Piece{
		"piece_uuid": "p" + strconv.FormatInt(seen, 10), "local_id": seen, "seen_at": At(seen),
		"dead": false, "classification_status": "classified", "part_id": part, "part_name": "Part " + part,
		"color_id": color, "color_name": "Color " + color, "confidence": 0.9, "category_id": "bricks",
		"part_correct": nil, "color_corrected_id": nil, "correction_updated_at": nil,
		"color_labels": []any{}, "color_predictions": []any{},
	}
}

// Begun is a run of a lot that started at a unix time, adding to the lot or
// another pass over pieces it counted.
type Begun struct {
	Lot  string
	Adds bool
	At   int64
}

// Runs is a machine's runs, in order of start: each ends when the next
// starts.
func Runs(runs ...Begun) []Run {
	var out []Run
	for i, b := range runs {
		lot := b.Lot
		out = append(out, Run{ID: strconv.Itoa(i + 1), LotID: &lot, AddsToLot: b.Adds, StartedAt: At(b.At)})
		if i > 0 {
			end := out[i].StartedAt
			out[i-1].EndsAt = &end
		}
	}
	return out
}

// New starts a fake Hive serving machines; it stops when the test ends.
func New(t *testing.T, machines ...*Machine) *Hive {
	h := &Hive{Page: 500, machines: machines}
	h.Server = httptest.NewServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.Close)
	return h
}

// Set replaces the machines it serves.
func (h *Hive) Set(machines ...*Machine) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.machines = machines
}

// Pages is how many pages of pieces it served.
func (h *Hive) Pages() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pages
}

func (h *Hive) serve(w http.ResponseWriter, r *http.Request) {
	// As the proxy in front of a real Hive does with a client's default
	// user agent.
	if strings.HasPrefix(r.UserAgent(), "Go-http-client") {
		http.Error(w, "error code: 1010", http.StatusForbidden)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+Key {
		http.Error(w, `{"detail":"Not authenticated"}`, http.StatusUnauthorized)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if r.URL.Path == "/api/records/machines" {
		var list []map[string]any
		for _, m := range h.machines {
			list = append(list, map[string]any{"id": m.ID, "name": m.Name, "last_seen_at": nil, "piece_count": len(m.Pieces)})
		}
		reply(w, map[string]any{"machines": list})
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, "/api/records/machines/")
	id, what, _ := strings.Cut(rest, "/")
	i := slices.IndexFunc(h.machines, func(m *Machine) bool { return m.ID == id })
	if !ok || i < 0 {
		http.NotFound(w, r)
		return
	}
	m := h.machines[i]
	head := map[string]any{"id": m.ID, "name": m.Name}
	switch what {
	case "pieces":
		reply(w, h.page(m, r, head))
	case "sorting-runs":
		reply(w, map[string]any{"machine": head, "lots": nonNil(m.Lots), "runs": nonNil(m.Runs)})
	default:
		http.NotFound(w, r)
	}
}

// page is the machine's pieces newest first (by local id) below the cursor.
func (h *Hive) page(m *Machine, r *http.Request, head map[string]any) map[string]any {
	h.pages++
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	limit = min(max(limit, 1), 1000, h.Page)
	cursor, err := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)
	if err != nil {
		cursor = 1 << 62
	}
	ps := slices.Clone(m.Pieces)
	slices.SortFunc(ps, func(a, b Piece) int { return int(local(b) - local(a)) })
	ps = slices.DeleteFunc(ps, func(p Piece) bool { return local(p) >= cursor })
	out := map[string]any{"machine": head, "items": nonNil(ps[:min(limit, len(ps))]), "next_cursor": nil}
	if len(ps) > limit {
		out["next_cursor"] = local(ps[limit-1])
	}
	return out
}

func local(p Piece) int64 { return p["local_id"].(int64) }

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
