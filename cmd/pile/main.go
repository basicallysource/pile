// pile serves a collection of sorted LEGO pieces, kept as lots: every part
// and color in each, and which sets and custom models the pieces make up.
package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"time"

	pile "github.com/basicallysource/pile"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/server"
)

func main() {
	data := flag.String("data", "data", "folder holding rebrickable/ (the catalog CSVs), parts.db (a sorter Hive's parts database), custom-models.json, lots.json, and machines/<name>/local_state.sqlite (each sorter's records)")
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
	s := server.New(cat, ls)
	log.Printf("%d lots ready in %s", len(ls), time.Since(start).Round(time.Millisecond))
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, server.Handler(s, pile.App())))
}
