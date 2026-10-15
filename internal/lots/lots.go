// Package lots reads the collection's lots, set out in the data folder's
// lots.json: each a named stretch of sorted pieces (one box, one sort, or
// everything some sorters ever sorted), from one of the owner's machines read
// through a Hive's records API or from the fleet pulled from a Hive.
package lots

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/basicallysource/pile/internal/hive"
	"github.com/basicallysource/pile/internal/records"
)

// File is the lots' file in the data folder.
const File = "lots.json"

// Lot is one entry of lots.json. Its pieces come from exactly one of
// HiveMachine and Hive.
type Lot struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// One of the API key owner's machines on a Hive, by its Hive name (in
	// any case) or id, read through the records API into records.sqlite.
	HiveMachine string `json:"hiveMachine,omitempty"`
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
	// With HiveMachine: one of that machine's own lots, by name, read with
	// it (its lots and sorting runs): the pieces of the runs that add to it.
	SorterLot string `json:"sorterLot,omitempty"`
	// Stretches of one sorter's pieces left out: pieces sorted again (a
	// re-sort of a box already counted) that would count twice.
	Exclude []Excluded `json:"exclude,omitempty"`

	Records *records.Records `json:"-"`

	// From, Until and Exclude, read.
	from, until time.Time
	drops       []drop
}

// Excluded is one sorter's pieces from From up to Until (RFC 3339 times;
// Until empty for no end), and why they are left out.
type Excluded struct {
	Machine string `json:"machine"`
	From    string `json:"from"`
	Until   string `json:"until,omitempty"`
	Why     string `json:"why"`
}

type drop struct {
	machine string
	span    records.Span
}

// Read loads lots.json in dir and each lot's pieces.
func Read(dir string) ([]*Lot, error) {
	ls, err := Parse(dir)
	if err != nil {
		return nil, err
	}
	for _, l := range ls {
		if err := l.read(dir); err != nil {
			return nil, err
		}
	}
	return ls, nil
}

// Parse reads lots.json in dir, each lot without its pieces.
func Parse(dir string) ([]*Lot, error) {
	b, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil {
		return nil, err
	}
	var ls []*Lot
	if err := json.Unmarshal(b, &ls); err != nil {
		return nil, fmt.Errorf("%s: %w", File, err)
	}
	for _, l := range ls {
		if err := l.parse(); err != nil {
			return nil, fmt.Errorf("lot %s: %w", l.ID, err)
		}
	}
	return ls, nil
}

func (l *Lot) parse() error {
	var err error
	if l.from, err = day(l.From); err != nil {
		return fmt.Errorf("from: %w", err)
	}
	if l.until, err = day(l.Until); err != nil {
		return fmt.Errorf("until: %w", err)
	}
	switch {
	case (l.HiveMachine == "") == (l.Hive == ""):
		return errors.New("give one of hiveMachine and hive")
	case l.SorterLot != "" && l.HiveMachine == "":
		return errors.New("sorterLot needs a hiveMachine")
	}
	for _, x := range l.Exclude {
		from, err := time.Parse(time.RFC3339, x.From)
		if err != nil {
			return fmt.Errorf("exclude from: %w", err)
		}
		var until int64
		if x.Until != "" {
			t, err := time.Parse(time.RFC3339, x.Until)
			if err != nil {
				return fmt.Errorf("exclude until: %w", err)
			}
			until = t.Unix()
		}
		l.drops = append(l.drops, drop{x.Machine, records.Span{From: from.Unix(), Until: until}})
	}
	return nil
}

// HiveMachines is the machines lots read through a Hive's records API, each
// once, as the lots name them.
func HiveMachines(ls []*Lot) []string {
	var out []string
	for _, l := range ls {
		if l.HiveMachine != "" && !slices.Contains(out, l.HiveMachine) {
			out = append(out, l.HiveMachine)
		}
	}
	return out
}

// Reread is ls with each lot that reads one of machines (as HiveMachine
// names it) read again from dir, as a new Lot; the other lots are ls's own.
func Reread(dir string, ls []*Lot, machines []string) ([]*Lot, error) {
	out := make([]*Lot, len(ls))
	for i, l := range ls {
		out[i] = l
		if l.HiveMachine == "" || !slices.Contains(machines, l.HiveMachine) {
			continue
		}
		fresh := *l
		if err := fresh.read(dir); err != nil {
			return nil, err
		}
		out[i] = &fresh
	}
	return out, nil
}

// read reads the lot's pieces: those within its days, of the runs adding to
// its SorterLot when it names one, less the stretches it excludes.
func (l *Lot) read(dir string) error {
	var err error
	if l.HiveMachine != "" {
		l.Records, err = readMachine(filepath.Join(dir, hive.RecordsFile), l)
	} else {
		l.Records, err = hive.ReadFleet(filepath.Join(dir, hive.FleetFile), l.Hive, l.from, l.until)
	}
	if err != nil {
		return fmt.Errorf("lot %s: %w", l.ID, err)
	}
	for _, d := range l.drops {
		l.Records.Drop(d.machine, d.span)
	}
	return nil
}

func readMachine(path string, l *Lot) (*records.Records, error) {
	r, err := hive.ReadMachine(path, l.HiveMachine, l.from, l.until)
	if err != nil || l.SorterLot == "" {
		return r, err
	}
	spans, err := hive.MachineLot(path, l.HiveMachine, l.SorterLot)
	if err != nil {
		return nil, err
	}
	r.Keep(spans)
	return r, nil
}

func day(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.ParseInLocation("2006-01-02", s, time.Local)
}
