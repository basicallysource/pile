// pile serves a collection of sorted LEGO pieces, kept as lots: every part
// and color in each, which sets, builds within sets and custom models the
// pieces make up, how long one sorter fed bulk like each lot takes to
// collect a set, and which bins hold what.
//
// The lots that read one of the owner's machines on a Hive (hiveMachine in
// lots.json) are read again through Hive's records API at start and every
// -refresh, with the API key in PILE_HIVE_KEY (refresh.go).
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	pile "github.com/basicallysource/pile"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/collect"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/server"
	"github.com/basicallysource/pile/internal/stars"
	"github.com/basicallysource/pile/internal/storage"
)

func main() {
	data := flag.String("data", "data", "folder holding rebrickable/ (the catalog CSVs), parts.db (a sorter Hive's parts database), custom-models.json, set-models/ (3D models of sets, for their builds), ldraw/ (the LDraw parts library), collections/ (designers' models), lots.json, records.sqlite (the lots' machines, read through a Hive's records API), hive.sqlite (records pulled by pull-hive), storage.json (the bins pieces are kept in) and stars.json (the starred sets); records.sqlite and the last two are kept there when they change")
	addr := flag.String("addr", ":8790", "where to listen")
	base := flag.String("hive", "https://hive.basically.website", "the Hive whose records API the lots' machines (hiveMachine) are read from, with the API key in "+keyVar)
	every := flag.Duration("refresh", 10*time.Minute, "how often to read the lots' machines from the Hive again")
	flag.Parse()

	start := time.Now()
	ctx := context.Background()
	parsed, err := lots.Parse(*data)
	if err != nil {
		log.Fatal(err)
	}
	r, err := newRefresher(*data, *base, os.Getenv(keyVar), lots.HiveMachines(parsed))
	if err != nil {
		log.Fatal(err)
	}
	if r != nil {
		if err := r.first(ctx); err != nil {
			log.Fatal(err)
		}
	}
	cat, err := catalog.Load(*data)
	if err != nil {
		log.Fatal(err)
	}
	ls, err := lots.Read(*data)
	if err != nil {
		log.Fatal(err)
	}
	kept, err := storage.Read(*data)
	if err != nil {
		log.Fatal(err)
	}
	sr, err := stars.Read(*data)
	if err != nil {
		log.Fatal(err)
	}
	sizes, err := collect.LoadSizes(filepath.Join(*data, "parts.db"))
	if err != nil {
		log.Fatal(err)
	}
	sh := &server.Shared{Cat: cat, Sizes: sizes, Storage: kept, Stars: sr, DataDir: *data}
	s, err := server.New(sh, ls)
	if err != nil {
		log.Fatal(err)
	}
	live := server.NewLive(s)
	log.Printf("%d lots ready in %s", len(ls), time.Since(start).Round(time.Millisecond))
	if r != nil {
		r.serve(sh, live, ls)
		go r.run(ctx, *every)
	}
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, server.Handler(live, pile.App())))
}
