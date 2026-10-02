// export-pieces writes a lot's pieces as CSV for analysis outside pile:
// counted by machine, part and color, in the sorter's BrickLink ids and in
// Rebrickable's (the part standing for its mold variants too), with the
// part's category and how many sets LEGO sold have it.
//
//	export-pieces -data DIR -lot everything > everything.csv
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/collect"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/match"
)

func main() {
	data := flag.String("data", "data", "pile's data folder")
	id := flag.String("lot", "", "the lot, by id")
	flag.Parse()

	cat, err := catalog.Load(filepath.Join(*data, "rebrickable"), filepath.Join(*data, "parts.db"), filepath.Join(*data, "custom-models.json"))
	if err != nil {
		log.Fatal(err)
	}
	all, err := lots.Read(*data)
	if err != nil {
		log.Fatal(err)
	}
	var lot *lots.Lot
	for _, l := range all {
		if l.ID == *id {
			lot = l
		}
	}
	if lot == nil {
		log.Fatalf("no lot %s in lots.json", *id)
	}
	ix := match.NewIndex(cat)
	c := collect.NewCatalog(cat, ix, nil)

	type key struct {
		machine int32
		part    string
		color   int32
		known   bool
	}
	type tally struct {
		n          int
		confidence float64
		partName   string
		colorName  string
	}
	counts := map[key]*tally{}
	for _, p := range lot.Records.Pieces {
		k := key{p.Machine, p.BrickLinkPart, p.BrickLinkColor, p.ColorKnown}
		t := counts[k]
		if t == nil {
			t = &tally{partName: p.PartName, colorName: p.ColorName}
			counts[k] = t
		}
		t.n++
		t.confidence += p.Confidence
	}
	keys := make([]key, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return counts[keys[i]].n > counts[keys[j]].n })

	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"machine", "bl_part", "bl_color", "part_name", "color_name", "rb_part", "rb_canonical", "rb_color", "rb_color_name", "rb_part_name", "category", "sets_with_part", "count", "mean_confidence"})
	for _, k := range keys {
		t := counts[k]
		blColor := ""
		if k.known {
			blColor = fmt.Sprint(k.color)
		}
		var rbPart, canon, rbColor, rbColorName, rbName, category, sets string
		if p := cat.Parts[cat.BrickLinkParts[k.part]]; p != nil {
			rbPart, rbName, category = p.Num, p.Name, p.Category
			id := ix.Part(p.Num)
			canon = ix.PartNum(id)
			sets = fmt.Sprint(c.SetsWith(id))
		}
		if col, ok := cat.BrickLinkColors[k.color]; ok && k.known {
			rbColor, rbColorName = fmt.Sprint(col), cat.Colors[col].Name
		}
		w.Write([]string{lot.Records.Machines[k.machine], k.part, blColor, t.partName, t.colorName, rbPart, canon, rbColor, rbColorName, rbName, category, sets,
			fmt.Sprint(t.n), fmt.Sprintf("%.3f", t.confidence/float64(t.n))})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}
