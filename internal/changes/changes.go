// Package changes is how sets will be built where they differ from their
// inventories: a line's colors (or a whole color's), other parts that make
// one of a line's part, and whether a line counts toward the set's
// completeness. They are kept in the data folder's changes.json, written
// whole (or not at all) each time one changes while pile runs, so building
// instructions made later can follow them (docs/changes.md).
package changes

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// File is the changes' file in the data folder.
const File = "changes.json"

// Change is one change to a set: for one line (Part in Color, as the set
// has them) or, for its colors only, every line in Color (Part empty).
// Exactly one of Colors, Swap and Counted is set.
type Change struct {
	Part  string `json:"part,omitempty"`
	Color int32  `json:"color"`
	// The colors it takes instead of its own.
	Colors *Colors `json:"colors,omitempty"`
	// Other parts that make one of it once its own part runs out, tried in
	// turn.
	Swap []StandIn `json:"swap,omitempty"`
	// Counted toward the set's completeness, or left out, whatever the view
	// counts.
	Counted *bool `json:"counted,omitempty"`
}

// Colors is which colors a line takes: one other color, several to fill it
// from in any mix (the first counts as its color), or any color of its
// part. Neither is its own color, to undo its color's change for one part.
type Colors struct {
	Use []int32 `json:"use,omitempty"`
	Any bool    `json:"any,omitempty"`
}

// StandIn makes one of a part: so many of each of Parts, or (Fits) any
// combination of smaller plates, tiles or bricks of its kind and height
// that covers its footprint.
type StandIn struct {
	Parts []Piece `json:"parts,omitempty"`
	Fits  bool    `json:"fits,omitempty"`
}

// Piece is so many of a part (Rebrickable's number).
type Piece struct {
	Part     string `json:"part"`
	Quantity int32  `json:"quantity"`
}

// Kind is which of its three things a change changes.
type Kind int

const (
	Recolor Kind = iota
	Swap
	Count
)

// Kind is which thing c changes.
func (c Change) Kind() Kind {
	switch {
	case c.Swap != nil:
		return Swap
	case c.Counted != nil:
		return Count
	}
	return Recolor
}

// Valid says what is wrong with c, if anything: it changes exactly one
// thing, a swap or a count is for one part, and every stand-in is something.
func (c Change) Valid() error {
	n := 0
	for _, set := range []bool{c.Colors != nil, c.Swap != nil, c.Counted != nil} {
		if set {
			n++
		}
	}
	switch {
	case n != 1:
		return errors.New("a change changes one thing: its colors, its parts or whether it counts")
	case c.Part == "" && c.Kind() != Recolor:
		return errors.New("only colors change for every line in a color")
	case c.Colors != nil && c.Colors.Any && len(c.Colors.Use) > 0:
		return errors.New("any color, or some colors, not both")
	case c.Swap != nil && len(c.Swap) == 0:
		return errors.New("a swap has a stand-in")
	}
	for _, s := range c.Swap {
		if s.Fits == (len(s.Parts) > 0) {
			return errors.New("a stand-in is some parts, or any that fit")
		}
		for _, p := range s.Parts {
			if p.Part == "" || p.Quantity < 1 || p.Quantity > maxQuantity {
				return fmt.Errorf("a stand-in's part needs a number and 1 to %d of it", maxQuantity)
			}
		}
	}
	return nil
}

// The most of one part a stand-in takes.
const maxQuantity = 99

// same is whether c and d are for the same line and change the same thing.
func (c Change) same(d Change) bool {
	return c.Part == d.Part && c.Color == d.Color && c.Kind() == d.Kind()
}

// Kept is every set's changes, and the folder they are kept in.
type Kept struct {
	dir string
	mu  sync.Mutex
	all map[string][]Change
}

// Read loads dir's changes.json; with none there, no set is changed.
func Read(dir string) (*Kept, error) {
	k := &Kept{dir: dir, all: map[string][]Change{}}
	b, err := os.ReadFile(filepath.Join(dir, File))
	if errors.Is(err, os.ErrNotExist) {
		return k, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &k.all); err != nil {
		return nil, fmt.Errorf("%s: %w", File, err)
	}
	return k, nil
}

// All is every set's changes, at the moment it is asked. The lists are
// never changed in place, so they are safe to read while others change.
func (k *Kept) All() map[string][]Change {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := make(map[string][]Change, len(k.all))
	for set, cs := range k.all {
		out[set] = cs
	}
	return out
}

// Of is one set's changes, in the order they were made.
func (k *Kept) Of(set string) []Change {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.all[set]
}

// Make makes a change to a set, replacing its change of the same line and
// kind (in that one's place), and keeps the file.
func (k *Kept) Make(set string, c Change) error {
	if err := c.Valid(); err != nil {
		return err
	}
	return k.change(set, func(cs []Change) []Change {
		if i := slices.IndexFunc(cs, c.same); i >= 0 {
			cs[i] = c
			return cs
		}
		return append(cs, c)
	})
}

// Undo takes back a set's change of the same line and kind as c, and keeps
// the file.
func (k *Kept) Undo(set string, c Change) error {
	return k.change(set, func(cs []Change) []Change { return slices.DeleteFunc(cs, c.same) })
}

// change applies fn to a copy of a set's changes and writes them all, whole
// or not at all; only once written does it hold them.
func (k *Kept) change(set string, fn func([]Change) []Change) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	cs := fn(slices.Clone(k.all[set]))
	all := make(map[string][]Change, len(k.all)+1)
	for s, c := range k.all {
		all[s] = c
	}
	if len(cs) == 0 {
		delete(all, set)
	} else {
		all[set] = cs
	}
	b, err := json.MarshalIndent(all, "", "\t")
	if err != nil {
		return err
	}
	tmp := filepath.Join(k.dir, File+".new")
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(k.dir, File)); err != nil {
		return err
	}
	k.all = all
	return nil
}
