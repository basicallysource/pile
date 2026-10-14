// check-models says how well the models beside the sets were read: for each
// sold set with a 3D model, the share of its pieces its builds hold, its
// builds, and the model's parts no catalog part stands for; for each
// designer's collection, every model's pieces and what was left out. pile
// loads the same; this only reports.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strings"

	"github.com/basicallysource/pile/internal/catalog"
)

func main() {
	data := flag.String("data", "data", "pile's data folder")
	only := flag.String("set", "", "report only this set or model number")
	profile := flag.String("cpuprofile", "", "write a CPU profile of loading here")
	flag.Parse()
	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}
	cat, err := catalog.Load(*data)
	pprof.StopCPUProfile()
	if err != nil {
		log.Fatal(err)
	}
	var nums []string
	for n, s := range cat.Sets {
		if s.Source != nil && s.Kind != catalog.Build && (*only == "" || n == *only) {
			nums = append(nums, n)
		}
	}
	sort.Strings(nums)
	var pieces, placed int32
	for _, n := range nums {
		s := cat.Sets[n]
		var have int32
		for _, l := range s.Lines {
			have += l.Quantity
		}
		src := s.Source
		if s.Sold() {
			pieces += have
			placed += src.Placed
			fmt.Printf("%s %s: %d of %d pieces in %d builds\n", s.Num, s.Name, src.Placed, have, len(s.Builds))
			for _, b := range s.Builds {
				var q int32
				for _, l := range b.Lines {
					q += l.Quantity
				}
				fmt.Printf("    %-40s x%d  %d pieces\n", b.Name, b.Copies, q)
			}
		} else {
			fmt.Printf("%s %s (%s): %d pieces\n", s.Num, s.Name, s.Collection, have)
		}
		if len(src.Unplaced) > 0 {
			fmt.Printf("    left out: %s\n", strings.Join(src.Unplaced, ", "))
		}
	}
	if pieces > 0 {
		fmt.Printf("sold sets: %d of %d pieces (%.1f%%) in builds\n", placed, pieces, 100*float64(placed)/float64(pieces))
	}
}
