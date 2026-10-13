// Package stars is the sets and custom models starred to come back to, kept
// in the data folder's stars.json and changed one at a time while pile runs.
package stars

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// File is the stars' file in the data folder.
const File = "stars.json"

// Star is one starred set or custom model, and when.
type Star struct {
	Set string `json:"set"`
	At  int64  `json:"at"`
}

// Stars is every star, most recent first, and the folder it is kept in.
type Stars struct {
	dir string
	mu  sync.Mutex
	all []Star
}

// Read loads dir's stars.json; with none there, nothing is starred.
func Read(dir string) (*Stars, error) {
	s := &Stars{dir: dir}
	b, err := os.ReadFile(filepath.Join(dir, File))
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.all); err != nil {
		return nil, fmt.Errorf("%s: %w", File, err)
	}
	return s, nil
}

// Sets is the starred set numbers, most recent first.
func (s *Stars) Sets() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.all))
	for i, st := range s.all {
		out[i] = st.Set
	}
	return out
}

// Has is which sets are starred, at the moment it is asked.
func (s *Stars) Has() map[string]bool {
	out := map[string]bool{}
	for _, n := range s.Sets() {
		out[n] = true
	}
	return out
}

// Set stars or unstars one, keeping the file, whole or not at all.
func (s *Stars) Set(num string, starred bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all := slices.DeleteFunc(slices.Clone(s.all), func(st Star) bool { return st.Set == num })
	if starred {
		all = append([]Star{{Set: num, At: time.Now().Unix()}}, all...)
	}
	b, err := json.MarshalIndent(all, "", "\t")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, File+".new")
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, File)); err != nil {
		return err
	}
	s.all = all
	return nil
}
