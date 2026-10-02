// collect-time works out how long one sorter, fed bulk at random in the mix
// its records show, takes to come across every piece of each set given: in
// exact colors, in near colors, and in any color (package collect has the
// model). Each lot named is a stream of its own. It prints a table per lot
// and can write everything as JSON.
//
//	collect-time -data DIR -lots mine,everything -sets 75192-1,6212-1
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/collect"
	"github.com/basicallysource/pile/internal/hive"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/match"
)

func main() {
	data := flag.String("data", "data", "pile's data folder")
	lotIDs := flag.String("lots", "", "the lots to use as streams, by id, comma separated")
	setList := flag.String("sets", "", "the sets, by number (75192-1), comma separated; empty for the standard ones (internal/collect/sets.txt)")
	rate := flag.Float64("rate", 0, "classified pieces an hour of sorting; 0 for each lot's own, from its Hive records")
	reps := flag.Int("resamples", 200, "resamplings of each stream's machine-days, for the uncertainty")
	simulate := flag.Int("simulate", 0, "also run the sorter piece by piece this many times per set, to check the closed form")
	out := flag.String("json", "", "write every result here as JSON")
	flag.Parse()

	start := time.Now()
	cat, err := catalog.Load(filepath.Join(*data, "rebrickable"), filepath.Join(*data, "parts.db"), filepath.Join(*data, "custom-models.json"))
	if err != nil {
		log.Fatal(err)
	}
	all, err := lots.Read(*data)
	if err != nil {
		log.Fatal(err)
	}
	sizes, err := collect.LoadSizes(filepath.Join(*data, "parts.db"))
	if err != nil {
		log.Fatal(err)
	}
	c := collect.NewCatalog(cat, match.NewIndex(cat), sizes)
	log.Printf("catalog and lots in %s", time.Since(start).Round(time.Millisecond))

	sets := split(*setList)
	if len(sets) == 0 {
		sets = collect.Standard()
	}
	var report []lotReport
	for _, id := range split(*lotIDs) {
		l := find(all, id)
		r := lotReport{ID: l.ID, Name: l.Name, Machines: l.Records.Machines}
		s := c.NewStream(l.Records)
		r.Pieces, r.Known = s.Pieces, s.Known
		r.Rate = *rate
		if l.Hive != "" {
			st, err := hive.SortingTime(filepath.Join(*data, "hive.sqlite"), l.Hive)
			if err != nil {
				log.Fatal(err)
			}
			for _, m := range st {
				r.Sorting = append(r.Sorting, sorting{m.Machine, m.Pieces, m.Classified, m.Seconds / 3600})
				r.SortingHours += m.Seconds / 3600
				r.Classified += m.Classified
			}
			if r.Rate == 0 && r.SortingHours > 0 {
				r.Rate = float64(r.Classified) / r.SortingHours
			}
		}
		if r.Rate == 0 {
			log.Fatalf("lot %s: no sorting time to take a rate from; pass -rate", l.ID)
		}
		exact := c.Fit(s)
		for _, mode := range []collect.Mode{collect.Exact, collect.Near, collect.Any} {
			md := exact.In(mode)
			r.Models = append(r.Models, model{mode.String(), md.A, md.B, md.C, md.G0, md.G1, md.Keys, md.SeenKeys, md.SeenPieces})
			// The sets are independent: work them out side by side.
			rows := make([]row, len(sets))
			var wg sync.WaitGroup
			work := make(chan int)
			for range runtime.NumCPU() {
				wg.Go(func() {
					for i := range work {
						res, err := c.Time(md, sets[i], *reps, 1)
						if err != nil {
							log.Fatal(err)
						}
						rows[i] = result(c, res, l.ID, r.Rate)
						if *simulate > 0 {
							rows[i].Simulated = c.Simulate(md, sets[i], *simulate, 1, 3e8)
						}
					}
				})
			}
			for i := range sets {
				work <- i
			}
			close(work)
			wg.Wait()
			r.Results = append(r.Results, rows...)
			log.Printf("%s, %s: %d sets in %s", l.ID, mode, len(sets), time.Since(start).Round(time.Second))
		}
		report = append(report, r)
		print(r)
	}
	if *out != "" {
		b, err := json.MarshalIndent(report, "", " ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			log.Fatal(err)
		}
	}
}

type lotReport struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Machines []string `json:"machines"`
	// Classified pieces, and the ones of a part the catalog knows.
	Pieces int `json:"pieces"`
	Known  int `json:"known"`
	// Classified pieces an hour of sorting, and where it came from.
	Rate         float64   `json:"rate_per_hour"`
	SortingHours float64   `json:"sorting_hours"`
	Classified   int       `json:"classified_in_sorting"`
	Sorting      []sorting `json:"sorting"`
	Models       []model   `json:"models"`
	Results      []row     `json:"results"`
}

type sorting struct {
	Machine    string  `json:"machine"`
	Pieces     int     `json:"pieces"`
	Classified int     `json:"classified"`
	Hours      float64 `json:"hours"`
}

type model struct {
	Mode       string  `json:"mode"`
	A          float64 `json:"a"`
	B          float64 `json:"b"`
	C          float64 `json:"c"`
	G0         float64 `json:"g0"`
	G1         float64 `json:"g1"`
	Keys       int     `json:"catalog_keys"`
	SeenKeys   int     `json:"seen_keys"`
	SeenPieces float64 `json:"seen_pieces"`
}

type rare struct {
	Part      string  `json:"part"`
	Name      string  `json:"name"`
	Color     string  `json:"color"`
	LengthMM  float64 `json:"length_mm"`
	WidthMM   float64 `json:"width_mm"`
	Need      float64 `json:"need"`
	Seen      float64 `json:"seen"`
	Share     float64 `json:"share"`
	WaitHours float64 `json:"wait_hours"`
	SetsWith  int     `json:"sets_with_part"`
}

type row struct {
	Lot     string           `json:"lot"`
	Set     string           `json:"set"`
	Name    string           `json:"name"`
	Year    int32            `json:"year"`
	Mode    string           `json:"mode"`
	Pieces  float64          `json:"pieces_counted"`
	Left    map[string]int32 `json:"left_out"`
	Figures int32            `json:"minifigures"`
	Keys    int              `json:"keys"`
	Unseen  int              `json:"unseen_keys"`
	UnseenP float64          `json:"unseen_pieces"`
	// Hours of sorting at the lot's rate.
	Half, Most, Nearly float64
	All, Fast, Slow    float64
	SeenAll, RarerAll  float64
	MostLow, MostHigh  float64
	AllLow, AllHigh    float64
	// The same in pieces sorted.
	AllPieces  float64   `json:"all_pieces"`
	MostPieces float64   `json:"most_pieces"`
	Slowest    []rare    `json:"slowest"`
	Simulated  []float64 `json:"simulated_pieces,omitempty"`
}

func result(c *collect.Catalog, r *collect.Result, lot string, rate float64) row {
	h := func(pieces float64) float64 { return finite(pieces / rate) }
	out := row{
		Lot: lot, Set: r.Set.Set.Num, Name: r.Set.Set.Name, Year: r.Set.Set.Year, Mode: r.Mode.String(),
		Pieces: r.Need.Pieces, Left: r.Need.Left, Figures: r.Need.Figures,
		Keys: r.Keys, Unseen: r.Unseen, UnseenP: r.UnseenPieces,
		Half: h(r.At.Half), Most: h(r.At.Most), Nearly: h(r.At.Nearly),
		All: h(r.At.All), Fast: h(r.At.Fast), Slow: h(r.At.Slow),
		SeenAll: h(r.SeenAll), RarerAll: h(r.RarerAll),
		MostLow: h(r.MostLow), MostHigh: h(r.MostHigh), AllLow: h(r.AllLow), AllHigh: h(r.AllHigh),
		AllPieces: finite(r.At.All), MostPieces: finite(r.At.Most),
	}
	for _, s := range r.Slowest {
		num := c.Ix.PartNum(s.Key.Part)
		x := rare{Part: num, Need: s.Need, Seen: s.Seen, Share: s.Share, WaitHours: h(s.Wait), SetsWith: c.SetsWith(s.Key.Part), LengthMM: c.Size(s.Key.Part).Length, WidthMM: c.Size(s.Key.Part).Width, Color: "any"}
		if p := c.Cat.Parts[num]; p != nil {
			x.Name = p.Name
		}
		if col := c.Cat.Colors[s.Key.Color]; col != nil && r.Mode != collect.Any {
			x.Color = col.Name
		}
		out.Slowest = append(out.Slowest, x)
	}
	return out
}

func print(r lotReport) {
	fmt.Printf("\n## %s (%s)\n\n", r.Name, r.ID)
	fmt.Printf("%d classified pieces (%d of parts in the catalog) from %d machines; %.0f classified pieces an hour of sorting (%d in %.0f hours)\n\n",
		r.Pieces, r.Known, len(r.Machines), r.Rate, r.Classified, r.SortingHours)
	for _, m := range r.Models {
		fmt.Printf("- %s: a=%.2f b=%.2f c=%.2f g0=%.2f g1=%.2f; %d of %d catalog keys seen\n", m.Mode, m.A, m.B, m.C, m.G0, m.G1, m.SeenKeys, m.Keys)
	}
	fmt.Println("\nHours of sorting. To half, 90% and 99% of the pieces (expected), and to every piece (median); for 90% and every piece, the 5th to 95th percentile over resamplings of the machine-days; for every piece, one run in ten faster or slower, the time were the never-seen keys ten times rarer, and the time to every key the sorters have seen.")
	fmt.Println()
	fmt.Println("| set | mode | pieces | unseen keys | 50% | 90% | 90% range | 99% | all | all range | all, runs | never-seen 10x rarer | all seen |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, x := range r.Results {
		fmt.Printf("| %s %s (%d) | %s | %.0f | %d of %d | %s | %s | %s to %s | %s | %s | %s to %s | %s to %s | %s | %s |\n",
			x.Set, x.Name, x.Year, x.Mode, x.Pieces, x.Unseen, x.Keys,
			hours(x.Half), hours(x.Most), hours(x.MostLow), hours(x.MostHigh), hours(x.Nearly),
			hours(x.All), hours(x.AllLow), hours(x.AllHigh), hours(x.Fast), hours(x.Slow), hours(x.RarerAll), hours(x.SeenAll))
	}
}

// finite is x, or -1 for never: JSON has no infinity.
func finite(x float64) float64 {
	if math.IsInf(x, 1) {
		return -1
	}
	return x
}

// hours says a number of hours readably.
func hours(h float64) string {
	switch {
	case h < 0:
		return "never"
	case h < 1:
		return fmt.Sprintf("%.0f min", h*60)
	case h < 100:
		return fmt.Sprintf("%.1f h", h)
	case h < 24*365:
		return fmt.Sprintf("%.0f h", h)
	}
	return fmt.Sprintf("%.1f y", h/24/365)
}

func split(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

func find(all []*lots.Lot, id string) *lots.Lot {
	for _, l := range all {
		if l.ID == id {
			return l
		}
	}
	log.Fatalf("no lot %s in lots.json", id)
	return nil
}
