// Package server serves the collection over the PileService contract, and the app.
package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/basicallysource/pile/gen/pile/v1/pilev1connect"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/collect"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/match"
)

// Service holds the catalog, its index, every lot ready to match, and the
// lots' collect times as they are worked out.
type Service struct {
	cat        *catalog.Catalog
	index      *match.Index
	order      []*lot
	lots       map[string]*lot
	collecting *collecting
}

// New readies every lot and starts working out their collect times; sizes is
// each part's size (collect.LoadSizes), hivePath the data folder's
// hive.sqlite.
func New(cat *catalog.Catalog, ls []*lots.Lot, sizes map[string]collect.Size, hivePath string) *Service {
	s := &Service{cat: cat, index: match.NewIndex(cat), lots: map[string]*lot{}}
	for _, l := range ls {
		t := newLot(l, cat, s.index)
		s.order = append(s.order, t)
		s.lots[l.ID] = t
	}
	s.collecting = newCollecting(cat, collect.NewCatalog(cat, s.index, sizes), hivePath, s.order)
	return s
}

// Handler serves the API, and the app's static build for every other path
// (index.html for the app's own routes).
func Handler(s *Service, app fs.FS) http.Handler {
	mux := http.NewServeMux()
	path, h := pilev1connect.NewPileServiceHandler(s)
	mux.Handle(path, h)
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
