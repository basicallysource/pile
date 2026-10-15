// Package server serves the collection over the PileService contract, and the app.
package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sync/atomic"

	"github.com/basicallysource/pile/gen/pile/v1/pilev1connect"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/collect"
	"github.com/basicallysource/pile/internal/hive"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/match"
	"github.com/basicallysource/pile/internal/stars"
	"github.com/basicallysource/pile/internal/storage"
)

// Shared is what every Service over one data folder shares, and keeps when
// one is built again with fresh lots: the catalog, the parts' sizes, the
// storage and the stars. A bin or a star changed through one Service is
// changed for them all.
type Shared struct {
	Cat *catalog.Catalog
	// Each part's size (collect.LoadSizes).
	Sizes   map[string]collect.Size
	Storage *storage.Kept
	Stars   *stars.Stars
	// The data folder: its hive.sqlite, and the builds' pictures.
	DataDir string
}

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
	dataDir    string
	storage    *storage.Kept
	// What the storage's bins hold, for the storage it was last worked out
	// for (stock).
	stocked atomic.Pointer[stock]
	stars   *stars.Stars
	// The API, answered by this Service.
	api http.Handler
}

// New readies every lot and each lot's stack, puts their pieces in the
// storage's bins and starts working out the lots' collect times.
func New(sh *Shared, ls []*lots.Lot) (*Service, error) {
	s := &Service{cat: sh.Cat, index: match.NewIndex(sh.Cat), lots: map[string]*lot{}, dataDir: sh.DataDir, storage: sh.Storage, stars: sh.Stars}
	for _, l := range ls {
		t := newLot(l, sh.Cat, s.index)
		s.order = append(s.order, t)
		s.lots[l.ID] = t
	}
	s.stacks = newStacks(s.index, s.order, s.lots)
	if err := s.storage.Now().Check(func(id string) bool { return s.lots[id] != nil }); err != nil {
		return nil, fmt.Errorf("%s: %w", storage.File, err)
	}
	s.stock()
	s.collecting = newCollecting(sh.Cat, collect.NewCatalog(sh.Cat, s.index, sh.Sizes), filepath.Join(sh.DataDir, hive.FleetFile), s.order)
	_, s.api = pilev1connect.NewPileServiceHandler(s)
	return s, nil
}

// Close stops the Service's background work (its collect times), once it is
// no longer served. A request it is answering still finishes.
func (s *Service) Close() { s.collecting.close() }
