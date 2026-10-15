// Package storage is where the collection's pieces are kept: bins, each at
// an address its owner labels it with ("3D-04"), holding one lot's pieces of
// the sorter categories it is given (the sorter's bin for that category,
// emptied into it), or every piece of a lot that no other bin of it holds. A
// category broken into finer ones by a later sort is given as those finer
// categories, each to its own bin. It is set out in the data folder's
// storage.json, and replaced whole while pile runs.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// File is the storage's file in the data folder.
const File = "storage.json"

// Bin is one place pieces are kept.
type Bin struct {
	// Where it is, as it is labelled: a shelf, a drawer, a box ("3D-04").
	Address string `json:"address"`
	// What it holds, in words ("Studs on side & brackets").
	Name string `json:"name,omitempty"`
	// The lot its pieces are from.
	Lot string `json:"lot"`
	// The sorter categories whose pieces it holds. None: the lot's pieces no
	// other bin of the lot holds.
	Categories []string `json:"categories,omitempty"`
}

// Storage is every bin.
type Storage struct {
	Bins []Bin `json:"bins"`
}

// Kept is the storage in use and the folder it is kept in, shared by
// everything serving it: replaced whole while pile runs.
type Kept struct {
	dir string
	mu  sync.Mutex
	now atomic.Pointer[Storage]
}

// Read loads dir's storage.json; with none there, the storage is empty.
func Read(dir string) (*Kept, error) {
	k := &Kept{dir: dir}
	b, err := os.ReadFile(filepath.Join(dir, File))
	if errors.Is(err, os.ErrNotExist) {
		k.now.Store(&Storage{})
		return k, nil
	}
	if err != nil {
		return nil, err
	}
	var s Storage
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", File, err)
	}
	k.now.Store(&s)
	return k, nil
}

// Now is the storage in use. It is never changed, only replaced.
func (k *Kept) Now() *Storage { return k.now.Load() }

// Replace keeps s in the folder, whole or not at all, and only then puts it
// in use.
func (k *Kept) Replace(s *Storage) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	b, err := json.MarshalIndent(s, "", "\t")
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
	k.now.Store(s)
	return nil
}

// Check says what is wrong with s, if anything: every bin has an address no
// other bin has and a lot that exists, and no lot's category (or the rest of
// a lot) is in two bins.
func (s *Storage) Check(lotExists func(id string) bool) error {
	addresses := map[string]bool{}
	held := map[[2]string]string{}
	for _, b := range s.Bins {
		a := strings.TrimSpace(b.Address)
		switch {
		case a == "":
			return errors.New("a bin has no address")
		case addresses[strings.ToLower(a)]:
			return fmt.Errorf("two bins are at %s", a)
		case !lotExists(b.Lot):
			return fmt.Errorf("bin %s: no lot %q", a, b.Lot)
		}
		addresses[strings.ToLower(a)] = true
		cats := b.Categories
		if len(cats) == 0 {
			cats = []string{""}
		}
		for _, c := range cats {
			k := [2]string{b.Lot, c}
			if other, ok := held[k]; ok {
				what := "the rest of lot " + b.Lot
				if c != "" {
					what = fmt.Sprintf("lot %s's %s", b.Lot, c)
				}
				return fmt.Errorf("%s is in both %s and %s", what, other, a)
			}
			held[k] = a
		}
	}
	return nil
}

// Holder finds the bin holding a lot's pieces of a category: the bin given
// that category, or else the lot's bin for the rest. ok is false when no bin
// holds them.
func (s *Storage) Holder() func(lot, category string) (bin int, ok bool) {
	byCat := map[[2]string]int{}
	for i, b := range s.Bins {
		if len(b.Categories) == 0 {
			byCat[[2]string{b.Lot, ""}] = i
		}
		for _, c := range b.Categories {
			byCat[[2]string{b.Lot, c}] = i
		}
	}
	return func(lot, category string) (int, bool) {
		if i, ok := byCat[[2]string{lot, category}]; ok && category != "" {
			return i, true
		}
		i, ok := byCat[[2]string{lot, ""}]
		return i, ok
	}
}
