// Package lots reads the collection's lots, set out in the data folder's
// lots.json: each a named stretch of sorted pieces (one box, one sort, or
// everything some sorters ever sorted), from one sorter's own records or from
// the records pulled from a Hive.
package lots

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/basicallysource/pile/internal/hive"
	"github.com/basicallysource/pile/internal/records"
)

// Lot is one entry of lots.json. Its pieces come from exactly one of Machine
// and Hive.
type Lot struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// A sorter whose own records are copied into machines/<machine>/local_state.sqlite.
	Machine string `json:"machine,omitempty"`
	// Or machines pulled from a Hive into hive.sqlite (cmd/pull-hive): "mine"
	// (the signed-in user's own), "all" (every machine pulled), or one
	// machine's name.
	Hive string `json:"hive,omitempty"`
	// The days it was sorted, YYYY-MM-DD in local time; Until is the first
	// day after it. Either may be empty for no bound.
	From  string `json:"from,omitempty"`
	Until string `json:"until,omitempty"`
	// Left off the collection page: it opens only from the lot menu.
	Unlisted bool `json:"unlisted,omitempty"`

	Records *records.Records `json:"-"`
}

// Read loads lots.json in dir and each lot's pieces.
func Read(dir string) ([]*Lot, error) {
	b, err := os.ReadFile(filepath.Join(dir, "lots.json"))
	if err != nil {
		return nil, err
	}
	var lots []*Lot
	if err := json.Unmarshal(b, &lots); err != nil {
		return nil, fmt.Errorf("lots.json: %w", err)
	}
	for _, l := range lots {
		from, err := day(l.From)
		if err != nil {
			return nil, fmt.Errorf("lot %s: from: %w", l.ID, err)
		}
		until, err := day(l.Until)
		if err != nil {
			return nil, fmt.Errorf("lot %s: until: %w", l.ID, err)
		}
		switch {
		case (l.Machine == "") == (l.Hive == ""):
			return nil, fmt.Errorf("lot %s: give one of machine and hive", l.ID)
		case l.Machine != "":
			l.Records, err = records.ReadSorter(filepath.Join(dir, "machines", l.Machine, "local_state.sqlite"), l.Machine, from, until)
		default:
			l.Records, err = hive.Read(filepath.Join(dir, "hive.sqlite"), l.Hive, from, until)
		}
		if err != nil {
			return nil, fmt.Errorf("lot %s: %w", l.ID, err)
		}
	}
	return lots, nil
}

func day(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.ParseInLocation("2006-01-02", s, time.Local)
}
