// Package lots reads the collection's lots: each a named stretch of one
// sorter's records (one box, one sort), set out in the data folder's
// lots.json.
package lots

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/basicallysource/pile/internal/machine"
)

// Lot is one entry of lots.json.
type Lot struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// The sorter whose records hold it: machines/<machine>/local_state.sqlite.
	Machine string `json:"machine"`
	// The days it was sorted, YYYY-MM-DD in local time; Until is the first
	// day after it, empty for no end.
	From  string `json:"from"`
	Until string `json:"until"`

	Records *machine.Records `json:"-"`
}

// Read loads lots.json in dir and each lot's pieces from its sorter's records.
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
		l.Records, err = machine.Read(filepath.Join(dir, "machines", l.Machine, "local_state.sqlite"), from, until)
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
