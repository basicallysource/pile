package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/changes"
	"github.com/basicallysource/pile/internal/hive/hivetest"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/server"
	"github.com/basicallysource/pile/internal/stars"
	"github.com/basicallysource/pile/internal/storage"
)

// A refresh that reads a change serves the lots made from it, through the
// same handler; one that reads none leaves the Service served as it is.
func TestRefreshServesNewLots(t *testing.T) {
	m := &hivetest.Machine{ID: "m-1", Name: "mysorter", Pieces: []hivetest.Piece{hivetest.Classified(100, "3001", "11")}}
	h := hivetest.New(t, m)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, lots.File), []byte(`[{"id": "box", "name": "Box", "description": "", "hiveMachine": "mysorter"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	parsed, err := lots.Parse(dir)
	if err != nil {
		t.Fatal(err)
	}
	r, err := newRefresher(dir, h.URL, hivetest.Key, lots.HiveMachines(parsed))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.first(ctx); err != nil {
		t.Fatal(err)
	}
	ls, err := lots.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	kept, err := storage.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	sr, err := stars.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	ch, err := changes.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	cat := &catalog.Catalog{
		Colors:          map[int32]*catalog.Color{0: {ID: 0, Name: "Black"}},
		Parts:           map[string]*catalog.Part{"3001": {Num: "3001", Name: "Brick 2 x 4", Category: "Bricks"}},
		Sets:            map[string]*catalog.Set{},
		Canonical:       map[string]string{},
		Images:          map[catalog.PartColor]string{},
		BrickLinkParts:  map[string]string{"3001": "3001"},
		BrickLinkColors: map[int32]int32{11: 0},
	}
	sh := &server.Shared{Cat: cat, Storage: kept, Stars: sr, Changes: ch, DataDir: dir}
	s, err := server.New(sh, ls)
	if err != nil {
		t.Fatal(err)
	}
	live := server.NewLive(s)
	defer func() { live.Service().Close() }()
	r.serve(sh, live, ls)
	web := httptest.NewServer(server.Handler(live, fstest.MapFS{"index.html": {Data: []byte("app")}}))
	defer web.Close()
	pieces := func() int {
		t.Helper()
		res, err := http.Post(web.URL+"/pile.v1.PileService/ListLots", "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out struct {
			Lots []struct {
				Totals struct {
					Pieces int `json:"pieces"`
				} `json:"totals"`
			} `json:"lots"`
		}
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil || len(out.Lots) != 1 {
			t.Fatalf("ListLots: %v, %d lots", err, len(out.Lots))
		}
		return out.Lots[0].Totals.Pieces
	}
	if n := pieces(); n != 1 {
		t.Fatalf("%d pieces at start, want 1", n)
	}

	if what, err := r.once(ctx); err != nil || what != "nothing changed" || live.Service() != s {
		t.Errorf("a refresh with nothing new: %q, %v, and a new Service %v", what, err, live.Service() != s)
	}

	m.Pieces = append(m.Pieces, hivetest.Classified(200, "3001", "11"))
	h.Set(m)
	if _, err := r.once(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pieces(); n != 2 {
		t.Errorf("%d pieces after a refresh that read a new one, want 2", n)
	}
}
