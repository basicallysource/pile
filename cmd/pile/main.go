// pile serves a collection of sorted LEGO pieces, kept as lots: every part
// and color in each, which sets and custom models the pieces make up, how
// long one sorter fed bulk like each lot takes to collect a set, and which
// bins hold what.
package main

import (
	"flag"
	"log"
	"net/http"
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
	data := flag.String("data", "data", "folder holding rebrickable/ (the catalog CSVs), parts.db (a sorter Hive's parts database), custom-models.json, lots.json, machines/<name>/local_state.sqlite (each sorter's records), storage.json (the bins pieces are kept in) and stars.json (the starred sets); the last two are kept there when they change")
	addr := flag.String("addr", ":8790", "where to listen")
	flag.Parse()

	start := time.Now()
	cat, err := catalog.Load(filepath.Join(*data, "rebrickable"), filepath.Join(*data, "parts.db"), filepath.Join(*data, "custom-models.json"))
	if err != nil {
		log.Fatal(err)
	}
	ls, err := lots.Read(*data)
	if err != nil {
		log.Fatal(err)
	}
	st, err := storage.Read(*data)
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
	s, err := server.New(cat, ls, st, sr, sizes, *data)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d lots ready in %s", len(ls), time.Since(start).Round(time.Millisecond))
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, server.Handler(s, pile.App())))
}
