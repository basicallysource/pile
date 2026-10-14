// Package catalog loads what pieces are matched against: Rebrickable's
// catalog (parts, colors, sets and every set's inventory) from its CSV
// downloads, which Rebrickable part and color each BrickLink and LDraw id is
// from a sorter Hive's parts database, and the models beside the sets LEGO
// sold: free custom models, the builds within sets (from 3D models of the
// sets), and designers' collections.
package catalog

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/basicallysource/pile/internal/ldraw"
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
	Class    Class
}

// Class is the kind of part, for what counts toward a set's completeness
// (match.Counting says which do).
type Class int

const (
	Plain Class = iota
	// Printed parts and stickers: a sorter rarely sees one as its own part.
	Printed
	// Minifigure body parts: heads, torsos and arms, legs, hair and hats,
	// what is worn at the hip or neck.
	FigurePart
	// What Rebrickable files under minifig accessories and their shields,
	// weapons and tools: taps, cups, life rings, wrenches. Often part of the
	// build.
	Accessory
)

// The minifig categories of each class, by Rebrickable's names.
var figureCategories = map[string]Class{
	"Minifigs":                          FigurePart,
	"Minifig Heads":                     FigurePart,
	"Minifig Upper Body":                FigurePart,
	"Minifig Lower Body":                FigurePart,
	"Minifig Headwear":                  FigurePart,
	"Minifig Headwear Accessories":      FigurePart,
	"Minifig Hipwear":                   FigurePart,
	"Minifig Neckwear":                  FigurePart,
	"Minifig Accessories":               Accessory,
	"Minifig Shields, Weapons, & Tools": Accessory,
}

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

// Kind is what a Set is: a set LEGO sold, or one of the models matched
// beside them.
type Kind int

const (
	// SoldSet is a set LEGO sold. Only these are evidence of what a pile came
	// from, and only these weigh how rare a part is.
	SoldSet Kind = iota
	// CustomModel is a free custom model (MOC), from Rebrickable or a
	// designer's collection.
	CustomModel
	// Build is one build within a set LEGO sold (a building, a vehicle), from
	// a 3D model of the set.
	Build
)

type Set struct {
	Num      string
	Name     string
	Year     int32
	Theme    string
	ImageURL string
	// The theme at the top of Theme's tree ("Star Wars" for "Ultimate
	// Collector Series"). A collection's models have the collection's name.
	ThemeGroup string
	// Made under license (Star Wars, Marvel, Minecraft...).
	Licensed bool
	// In a theme of loose bricks: buckets, brick boxes, service packs.
	BulkTheme bool
	Kind      Kind
	// A custom model's designer, and its page (on Rebrickable, or the
	// designer's site). A sold set's page is on Rebrickable.
	Designer string
	URL      string
	// The designer's collection a custom model is in; empty for one from
	// Rebrickable.
	Collection string
	// A build's set, and how many of it the set has.
	Of     *Set
	Copies int32
	// A sold set's builds, in its model's order.
	Builds []*Set
	// One copy of a build's parts where its set's model puts them, to draw
	// it.
	Parts []ldraw.Placed
	// The file its lines were read from, when not Rebrickable's: a sold
	// set's and its builds' 3D model, a collection model's digital file.
	Source *Source
	// For one copy, a build's.
	Lines       []Line
	Minifigures []Minifigure
}

// Sold is whether s is a set LEGO sold.
func (s *Set) Sold() bool { return s.Kind == SoldSet }

// Source is a file a model's lines were read from, and who made it.
type Source struct {
	// Who made the file (a 3D model of a set, or a digital copy of a
	// designer's model), and its page.
	Author string
	URL    string
	// What the file says it lacks or gets wrong, in its maker's words.
	Notes string
	// Of a sold set's 3D model: the pieces of the set its builds hold, and
	// the model's parts no catalog part stands for, which no build has.
	Placed   int32
	Unplaced []string
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
	// Parts by their number but for a closing letter, the plainest first.
	stems map[string][]string
	// BrickLink and LDraw ids to Rebrickable ones.
	BrickLinkParts  map[string]string
	BrickLinkColors map[int32]int32
	LDrawParts      map[string]string
	LDrawColors     map[int32]int32
	DownloadedAt    int64
}

type PartColor struct {
	Part  string
	Color int32
}

// Load reads the catalog from a data folder: the Rebrickable CSVs in
// rebrickable/, the id mapping in parts.db, and when they are there, the
// custom models in custom-models.json, the 3D models of sets in set-models/
// (docs/builds.md), the designers' collections in collections/
// (docs/collections.md), and the LDraw parts library in ldraw/, which
// reading the models leans on.
func Load(data string) (*Catalog, error) {
	c := &Catalog{
		Colors:    map[int32]*Color{},
		Parts:     map[string]*Part{},
		Sets:      map[string]*Set{},
		Canonical: map[string]string{},
		Images:    map[PartColor]string{},
		stems:     map[string][]string{},
	}
	dir := filepath.Join(data, "rebrickable")
	if st, err := os.Stat(filepath.Join(dir, "inventory_parts.csv.gz")); err == nil {
		c.DownloadedAt = st.ModTime().Unix()
	}
	steps := []func(string) error{c.loadColors, c.loadParts, c.loadRelationships, c.loadSets}
	for _, step := range steps {
		if err := step(dir); err != nil {
			return nil, err
		}
	}
	for _, ps := range c.stems {
		sort.Strings(ps)
	}
	if err := c.loadIDs(filepath.Join(data, "parts.db")); err != nil {
		return nil, fmt.Errorf("id mapping from parts.db: %w", err)
	}
	if err := c.loadCustom(filepath.Join(data, "custom-models.json")); err != nil {
		return nil, fmt.Errorf("custom models: %w", err)
	}
	lib := ldraw.OpenLibrary(existing(filepath.Join(data, "ldraw")))
	if err := c.loadBuilds(filepath.Join(data, "set-models"), lib); err != nil {
		return nil, fmt.Errorf("builds: %w", err)
	}
	if err := c.loadCollections(filepath.Join(data, "collections"), lib); err != nil {
		return nil, fmt.Errorf("collections: %w", err)
	}
	return c, nil
}

// existing is dir when it is there, else "".
func existing(dir string) string {
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return dir
	}
	return ""
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
			p.Class = Printed
		case strings.HasPrefix(cat, "Minifig"):
			p.Class = FigurePart
			if c, ok := figureCategories[cat]; ok {
				p.Class = c
			}
		}
		c.Parts[p.Num] = p
		stem := moldLetters.ReplaceAllString(p.Num, "")
		c.stems[stem] = append(c.stems[stem], p.Num)
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
			if p := c.Parts[child]; p != nil && p.Class == Plain {
				p.Class = Printed
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
