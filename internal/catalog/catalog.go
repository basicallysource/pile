// Package catalog loads Rebrickable's catalog (parts, colors, sets and every
// set's inventory) from its CSV downloads, and the BrickLink-to-Rebrickable
// part and color mapping from a sorter Hive's parts database.
package catalog

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Color struct {
	ID          int32
	Name        string
	RGB         string
	Transparent bool
	BrickLinkID int32
}

type Part struct {
	Num      string
	Name     string
	Category string
	Counting Counting
}

// Counting says whether a part counts toward a set's completeness.
type Counting int

const (
	Counted Counting = iota
	// Printed parts and stickers: a sorter cannot be expected to find them.
	Printed
	// Minifigure parts (heads, torsos, legs, hair, hats, the figures' tools):
	// a set is complete without its figures.
	FigurePart
)

type Line struct {
	Part     string
	Color    int32
	Quantity int32
	ImageURL string
}

type Minifigure struct {
	Num      string
	Name     string
	Quantity int32
	ImageURL string
}

type Set struct {
	Num      string
	Name     string
	Year     int32
	Theme    string
	ImageURL string
	// The theme at the top of Theme's tree ("Star Wars" for "Ultimate
	// Collector Series").
	ThemeGroup string
	// Made under license (Star Wars, Marvel, Minecraft...).
	Licensed bool
	// In a theme of loose bricks: buckets, brick boxes, service packs.
	BulkTheme bool
	// A free custom model (MOC) rather than a set LEGO sold.
	Custom bool
	// A custom model's designer and its page on Rebrickable.
	Designer    string
	URL         string
	Lines       []Line
	Minifigures []Minifigure
}

type Catalog struct {
	Colors map[int32]*Color
	Parts  map[string]*Part
	Sets   map[string]*Set
	// Canonical maps a part to the one part standing for its mold variants and
	// alternates, which look alike to a camera.
	Canonical map[string]string
	// Image of a part in a color, from any set's inventory.
	Images map[PartColor]string
	// BrickLink ids to Rebrickable ones.
	BrickLinkParts  map[string]string
	BrickLinkColors map[int32]int32
	DownloadedAt    int64
}

type PartColor struct {
	Part  string
	Color int32
}

// Load reads the Rebrickable CSVs in dir, the BrickLink mapping in
// hivePartsDB, and the custom models in customModels (skipped when absent).
func Load(dir, hivePartsDB, customModels string) (*Catalog, error) {
	c := &Catalog{
		Colors:    map[int32]*Color{},
		Parts:     map[string]*Part{},
		Sets:      map[string]*Set{},
		Canonical: map[string]string{},
		Images:    map[PartColor]string{},
	}
	if st, err := os.Stat(filepath.Join(dir, "inventory_parts.csv.gz")); err == nil {
		c.DownloadedAt = st.ModTime().Unix()
	}
	steps := []func(string) error{c.loadColors, c.loadParts, c.loadRelationships, c.loadSets}
	for _, step := range steps {
		if err := step(dir); err != nil {
			return nil, err
		}
	}
	if err := c.loadBrickLink(hivePartsDB); err != nil {
		return nil, fmt.Errorf("bricklink mapping from %s: %w", hivePartsDB, err)
	}
	if err := c.loadCustom(customModels); err != nil {
		return nil, fmt.Errorf("custom models from %s: %w", customModels, err)
	}
	return c, nil
}

// Canon is the part standing for p's mold variants and alternates.
func (c *Catalog) Canon(p string) string {
	if q, ok := c.Canonical[p]; ok {
		return q
	}
	return p
}

// each calls fn with every row of a gzipped CSV, by column name.
func each(dir, name string, fn func(row func(string) string) error) error {
	f, err := os.Open(filepath.Join(dir, name+".csv.gz"))
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	r := csv.NewReader(gz)
	r.ReuseRecord = true
	head, err := r.Read()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	col := map[string]int{}
	for i, h := range head {
		col[h] = i
	}
	var rec []string
	get := func(k string) string {
		i, ok := col[k]
		if !ok {
			panic(fmt.Sprintf("%s.csv has no column %q", name, k))
		}
		return rec[i]
	}
	for {
		rec, err = r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := fn(get); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
}

func num(s string) int32 {
	n, _ := strconv.Atoi(s)
	return int32(n)
}

func (c *Catalog) loadColors(dir string) error {
	return each(dir, "colors", func(row func(string) string) error {
		id := num(row("id"))
		c.Colors[id] = &Color{ID: id, Name: row("name"), RGB: row("rgb"), Transparent: row("is_trans") == "True" || row("is_trans") == "t"}
		return nil
	})
}

func (c *Catalog) loadParts(dir string) error {
	cats := map[string]string{}
	if err := each(dir, "part_categories", func(row func(string) string) error {
		cats[row("id")] = row("name")
		return nil
	}); err != nil {
		return err
	}
	return each(dir, "parts", func(row func(string) string) error {
		cat := cats[row("part_cat_id")]
		p := &Part{Num: row("part_num"), Name: row("name"), Category: cat}
		switch {
		case cat == "Stickers":
			p.Counting = Printed
		case strings.HasPrefix(cat, "Minifig"):
			p.Counting = FigurePart
		}
		c.Parts[p.Num] = p
		return nil
	})
}

// loadRelationships marks printed parts and joins mold variants and
// alternates into one canonical part each (union-find).
func (c *Catalog) loadRelationships(dir string) error {
	parent := map[string]string{}
	var find func(string) string
	find = func(p string) string {
		q, ok := parent[p]
		if !ok || q == p {
			return p
		}
		r := find(q)
		parent[p] = r
		return r
	}
	err := each(dir, "part_relationships", func(row func(string) string) error {
		child, par := row("child_part_num"), row("parent_part_num")
		switch row("rel_type") {
		case "P":
			if p := c.Parts[child]; p != nil && p.Counting == Counted {
				p.Counting = Printed
			}
		case "M", "A":
			a, b := find(child), find(par)
			if a != b {
				parent[a] = b
			}
		}
		return nil
	})
	for p := range parent {
		c.Canonical[p] = find(p)
	}
	return err
}

func (c *Catalog) loadSets(dir string) error {
	type theme struct{ name, parent string }
	themes := map[string]theme{}
	if err := each(dir, "themes", func(row func(string) string) error {
		themes[row("id")] = theme{row("name"), row("parent_id")}
		return nil
	}); err != nil {
		return err
	}
	if err := each(dir, "sets", func(row func(string) string) error {
		n := row("set_num")
		s := &Set{Num: n, Name: row("name"), Year: num(row("year")), Theme: themes[row("theme_id")].name, ImageURL: row("img_url"), URL: "https://rebrickable.com/sets/" + n + "/"}
		for id := row("theme_id"); id != ""; id = themes[id].parent {
			t := themes[id]
			s.ThemeGroup = t.name
			s.BulkTheme = s.BulkTheme || bulkThemes[t.name]
			// A licensed theme can sit under one of LEGO's own (Ghostbusters
			// under Icons).
			s.Licensed = s.Licensed || licensedThemes[t.name]
		}
		c.Sets[n] = s
		return nil
	}); err != nil {
		return err
	}
	// A set's first inventory version is the one sold first; later versions
	// are corrections or re-releases.
	type inv struct {
		id      string
		version int32
	}
	first := map[string]inv{}
	invSet := map[string]string{}
	if err := each(dir, "inventories", func(row func(string) string) error {
		s, v := row("set_num"), num(row("version"))
		if cur, ok := first[s]; !ok || v < cur.version {
			first[s] = inv{row("id"), v}
		}
		return nil
	}); err != nil {
		return err
	}
	for s, i := range first {
		invSet[i.id] = s
	}
	if err := each(dir, "inventory_parts", func(row func(string) string) error {
		img := row("img_url")
		pc := PartColor{row("part_num"), num(row("color_id"))}
		if _, ok := c.Images[pc]; !ok && img != "" {
			c.Images[pc] = img
		}
		set := c.Sets[invSet[row("inventory_id")]]
		if set == nil || row("is_spare") == "True" {
			return nil
		}
		set.Lines = append(set.Lines, Line{Part: pc.Part, Color: pc.Color, Quantity: num(row("quantity")), ImageURL: img})
		return nil
	}); err != nil {
		return err
	}
	figs := map[string]Minifigure{}
	if err := each(dir, "minifigs", func(row func(string) string) error {
		figs[row("fig_num")] = Minifigure{Num: row("fig_num"), Name: row("name"), ImageURL: row("img_url")}
		return nil
	}); err != nil {
		return err
	}
	return each(dir, "inventory_minifigs", func(row func(string) string) error {
		set := c.Sets[invSet[row("inventory_id")]]
		if set == nil {
			return nil
		}
		f := figs[row("fig_num")]
		f.Num = row("fig_num")
		f.Quantity = num(row("quantity"))
		set.Minifigures = append(set.Minifigures, f)
		return nil
	})
}
