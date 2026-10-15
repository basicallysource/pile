package hive

// The records pull: the key owner's machines that lots name, each read whole
// through Hive's records API into records.sqlite (machinestore.go).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

// recordsMachine is one of the key owner's machines, as
// GET /api/records/machines lists it.
type recordsMachine struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// record is what pile keeps of one piece, as
// GET /api/records/machines/{id}/pieces gives it.
type record struct {
	UUID       string   `json:"piece_uuid"`
	LocalID    int64    `json:"local_id"`
	SeenAt     *string  `json:"seen_at"`
	Dead       bool     `json:"dead"`
	Status     *string  `json:"classification_status"`
	Part       *string  `json:"part_id"`
	PartName   *string  `json:"part_name"`
	Color      *string  `json:"color_id"`
	ColorName  *string  `json:"color_name"`
	Confidence *float64 `json:"confidence"`
	Category   *string  `json:"category_id"`
	// False when a person said the part is wrong.
	PartCorrect *bool `json:"part_correct"`
	// The machine's operator's color correction, and when it was made.
	Corrected   *string           `json:"color_corrected_id"`
	CorrectedAt *string           `json:"correction_updated_at"`
	Labels      []colorLabel      `json:"color_labels"`
	Predictions []colorPrediction `json:"color_predictions"`
}

// colorLabel is a person's color label on Hive.
type colorLabel struct {
	Color    *string `json:"color_id"`
	CantTell bool    `json:"cant_tell"`
	// Given by the machine's owner.
	ByOwner bool    `json:"by_owner"`
	At      *string `json:"at"`
}

// colorPrediction is the top color of one run of a color model over the
// piece's pictures after the sort.
type colorPrediction struct {
	Source    string  `json:"source"`
	Model     string  `json:"model_name"`
	Color     string  `json:"color_id"`
	ColorName *string `json:"color_name"`
	MadeAt    *string `json:"made_at"`
}

// sortingRuns is a machine's lots and its runs, by start, as
// GET /api/records/machines/{id}/sorting-runs gives them.
type sortingRuns struct {
	Lots []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"lots"`
	Runs []struct {
		ID        string  `json:"id"`
		LotID     *string `json:"lot_id"`
		AddsToLot bool    `json:"adds_to_lot"`
		StartedAt string  `json:"started_at"`
		// When the next run started; none for the newest.
		EndsAt *string `json:"ends_at"`
	} `json:"runs"`
}

// The most pieces a records page is asked for.
const recordsPage = 500

// Pulled is what a records pull read of one machine.
type Pulled struct {
	// The machine as it was asked for, and its name on Hive.
	Asked, Name string
	Pieces      int
	// What was read differs from what the pull before read.
	Changed bool
}

// PullMachines reads each machine asked for (by its Hive name, in any case,
// or its id; one of the key owner's) whole through c: every piece, a page at a
// time with a pause between pages, and its lots and sorting runs. It writes
// each into the records store at path in one transaction, replacing what the
// store had of it, unless nothing changed since the last pull. On an error it
// returns what it pulled before it.
func PullMachines(ctx context.Context, c *KeyClient, path string, asked []string) ([]Pulled, error) {
	var list struct {
		Machines []recordsMachine `json:"machines"`
	}
	if err := c.Get(ctx, "/api/records/machines", &list); err != nil {
		return nil, fmt.Errorf("list the key's machines: %w", err)
	}
	s, err := openRecords(path)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	var out []Pulled
	byID := map[string]Pulled{}
	for _, a := range asked {
		i := slices.IndexFunc(list.Machines, func(m recordsMachine) bool { return strings.EqualFold(m.Name, a) || m.ID == a })
		if i < 0 {
			return out, fmt.Errorf("no machine %q among the key owner's", a)
		}
		m := list.Machines[i]
		p, ok := byID[m.ID]
		if !ok {
			if p, err = pullRecords(ctx, c, s, m); err != nil {
				return out, fmt.Errorf("machine %s: %w", m.Name, err)
			}
			byID[m.ID] = p
		}
		p.Asked = a
		out = append(out, p)
	}
	return out, nil
}

// pullRecords reads one machine and keeps it in the store.
func pullRecords(ctx context.Context, c *KeyClient, s *recordsStore, m recordsMachine) (Pulled, error) {
	at := "/api/records/machines/" + url.PathEscape(m.ID)
	var pieces []record
	var cursor *int64
	for {
		q := url.Values{"limit": {fmt.Sprint(recordsPage)}}
		if cursor != nil {
			q.Set("cursor", fmt.Sprint(*cursor))
		}
		var page struct {
			Items      []record `json:"items"`
			NextCursor *int64   `json:"next_cursor"`
		}
		if err := c.Get(ctx, at+"/pieces?"+q.Encode(), &page); err != nil {
			return Pulled{}, err
		}
		pieces = append(pieces, page.Items...)
		if page.NextCursor == nil {
			break
		}
		if cursor != nil && *page.NextCursor >= *cursor {
			return Pulled{}, fmt.Errorf("the pieces' cursor went from %d to %d, not down", *cursor, *page.NextCursor)
		}
		cursor = page.NextCursor
		select {
		case <-ctx.Done():
			return Pulled{}, ctx.Err()
		case <-time.After(pause):
		}
	}
	var runs sortingRuns
	if err := c.Get(ctx, at+"/sorting-runs", &runs); err != nil {
		return Pulled{}, err
	}
	b, err := json.Marshal(struct {
		Name   string
		Pieces []record
		Runs   sortingRuns
	}{m.Name, pieces, runs})
	if err != nil {
		return Pulled{}, err
	}
	sum := sha256.Sum256(b)
	changed, err := s.put(m, pieces, runs, hex.EncodeToString(sum[:]))
	return Pulled{Name: m.Name, Pieces: len(pieces), Changed: changed}, err
}
