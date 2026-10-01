// import-custom-models turns custom models gathered from Rebrickable (a JSON
// object of model number to its listing and its parts as Rebrickable's parts
// CSV export) into the data folder's custom-models.json, which pile loads.
// The gathering is by hand, signed in to Rebrickable: docs/custom-models.md.
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

type gathered struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Href     string `json:"href"`
	Img      string `json:"img"`
	Designer string `json:"designer"`
	CSV      string `json:"csv"`
}

type line struct {
	Part     string `json:"part"`
	Color    int32  `json:"color"`
	Quantity int32  `json:"quantity"`
}

type model struct {
	Num      string `json:"num"`
	Name     string `json:"name"`
	Designer string `json:"designer"`
	URL      string `json:"url"`
	Image    string `json:"image"`
	Lines    []line `json:"lines"`
}

func main() {
	out := flag.String("out", "data/custom-models.json", "the custom models file to write")
	flag.Parse()
	all := map[string]model{}
	for _, path := range flag.Args() {
		b, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		var in map[string]gathered
		if err := json.Unmarshal(b, &in); err != nil {
			log.Fatalf("%s: %v", path, err)
		}
		for _, g := range in {
			lines, err := parts(g.CSV)
			if err != nil {
				log.Printf("%s: %v; skipped", g.ID, err)
				continue
			}
			// Rebrickable's result card adds an "Alt" badge to alternate builds.
			name := strings.TrimSuffix(g.Name, " Alt")
			all[g.ID] = model{Num: g.ID, Name: name, Designer: g.Designer, URL: g.Href, Image: g.Img, Lines: lines}
		}
	}
	models := make([]model, 0, len(all))
	for _, m := range all {
		models = append(models, m)
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Num < models[j].Num })
	b, err := json.Marshal(models)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("%d custom models to %s", len(models), *out)
}

// parts reads Rebrickable's parts CSV export (Part,Color,Quantity,Is Spare),
// leaving out spares.
func parts(text string) ([]line, error) {
	r := csv.NewReader(strings.NewReader(text))
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var out []line
	for i, row := range rows {
		if i == 0 || len(row) < 4 || row[3] == "True" {
			continue
		}
		color, err := strconv.Atoi(row[1])
		if err != nil {
			return nil, err
		}
		q, err := strconv.Atoi(row[2])
		if err != nil {
			return nil, err
		}
		out = append(out, line{Part: row[0], Color: int32(color), Quantity: int32(q)})
	}
	return out, nil
}
