// pile serves what a sorter sorted: every part and color, and which LEGO
// sets the pieces make up.
package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"time"

	pile "github.com/basicallysource/pile"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/machine"
	"github.com/basicallysource/pile/internal/server"
)

func main() {
	data := flag.String("data", "data", "folder holding rebrickable/ (the catalog CSVs), parts.db (a sorter Hive's parts database) and machine/local_state.sqlite (the sorter's records)")
	since := flag.String("since", "", "only pieces classified on or after this day (YYYY-MM-DD, local time)")
	name := flag.String("machine", "", "the sorter's name, as shown")
	addr := flag.String("addr", ":8790", "where to listen")
	flag.Parse()

	var from time.Time
	if *since != "" {
		t, err := time.ParseInLocation("2006-01-02", *since, time.Local)
		if err != nil {
			log.Fatalf("-since: %v", err)
		}
		from = t
	}
	start := time.Now()
	cat, err := catalog.Load(filepath.Join(*data, "rebrickable"), filepath.Join(*data, "parts.db"))
	if err != nil {
		log.Fatal(err)
	}
	rec, err := machine.Read(filepath.Join(*data, "machine", "local_state.sqlite"), from)
	if err != nil {
		log.Fatal(err)
	}
	p := server.Build(cat, rec, *name)
	log.Printf("%d pieces in %d lots, %d sets picked, ready in %s", p.Pieces, len(p.Lots), len(p.Result.Explained), time.Since(start).Round(time.Millisecond))
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, server.Handler(p, pile.App())))
}
