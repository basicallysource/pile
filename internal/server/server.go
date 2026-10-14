// Package server serves the collection over the PileService contract, and the app.
package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/basicallysource/pile/gen/pile/v1/pilev1connect"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/collect"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/match"
	"github.com/basicallysource/pile/internal/stars"
	"github.com/basicallysource/pile/internal/storage"
)

// Service holds the catalog, its index, every lot, the stacks of lots views
// look at, the lots' collect times as they are worked out, what the
// storage's bins hold, and the stars.
type Service struct {
	cat        *catalog.Catalog
	index      *match.Index
	order      []*lot
	lots       map[string]*lot
	stacks     *stacks
	collecting *collecting
	// The data folder, where a new storage is kept.
	dataDir string
	stock   atomic.Pointer[stock]
	storing sync.Mutex
	stars   *stars.Stars
}

// New readies every lot and each lot's stack, puts their pieces in the
// storage's bins and starts working out the lots' collect times; dataDir is
// the data folder (its hive.sqlite, and where a new storage is kept), sizes
// each part's size (collect.LoadSizes).
func New(cat *catalog.Catalog, ls []*lots.Lot, st *storage.Storage, sr *stars.Stars, sizes map[string]collect.Size, dataDir string) (*Service, error) {
	s := &Service{cat: cat, index: match.NewIndex(cat), lots: map[string]*lot{}, dataDir: dataDir, stars: sr}
	for _, l := range ls {
		t := newLot(l, cat, s.index)
		s.order = append(s.order, t)
		s.lots[l.ID] = t
	}
	s.stacks = newStacks(s.index, s.order, s.lots)
	if err := st.Check(func(id string) bool { return s.lots[id] != nil }); err != nil {
		return nil, fmt.Errorf("%s: %w", storage.File, err)
	}
	s.stock.Store(newStock(st, s.lots, cat))
	s.collecting = newCollecting(cat, collect.NewCatalog(cat, s.index, sizes), filepath.Join(dataDir, "hive.sqlite"), s.order)
	return s, nil
}

// Handler serves the API, the pictures of builds drawn into the data folder
// (catalog.BuildPictures), and the app's static build for every other path
// (index.html for the app's own routes).
func Handler(s *Service, app fs.FS) http.Handler {
	mux := http.NewServeMux()
	path, h := pilev1connect.NewPileServiceHandler(s)
	mux.Handle(path, h)
	mux.Handle(catalog.BuildPicturesURL, http.StripPrefix(catalog.BuildPicturesURL, http.FileServer(http.Dir(catalog.BuildPictures(s.dataDir)))))
	files := http.FileServerFS(app)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" {
			if _, err := fs.Stat(app, name); err != nil {
				r.URL.Path = "/"
			}
		}
		files.ServeHTTP(w, r)
	})
	return mux
}
