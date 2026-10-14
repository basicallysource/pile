package catalog

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/basicallysource/pile/internal/ldraw"
)

// A designer's collection, as collections/<id>/collection.json describes
// it; each model's file is beside it (docs/collections.md).
type collection struct {
	Name     string `json:"name"`
	Designer string `json:"designer"`
	URL      string `json:"url"`
	Models   []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Year int32  `json:"year"`
		// Its page on the designer's site, and its picture.
		URL   string `json:"url"`
		Image string `json:"image"`
		// The file its parts are read from, who made it and where it is from.
		File   string `json:"file"`
		Author string `json:"author"`
		Source string `json:"source"`
		Notes  string `json:"notes"`
	} `json:"models"`
}

// loadCollections adds the models of every collection in dir to the sets,
// as custom models in that collection. A missing folder adds none.
func (c *Catalog) loadCollections(dir string, lib *ldraw.Library) error {
	ents, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), "collection.json")
		b, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		var col collection
		if err := json.Unmarshal(b, &col); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for _, m := range col.Models {
			lines, src, err := c.readModelFile(filepath.Join(dir, e.Name(), m.File), lib)
			if err != nil {
				log.Printf("collections: %s %s: %v; skipped", e.Name(), m.File, err)
				continue
			}
			src.Author, src.URL, src.Notes = m.Author, m.Source, m.Notes
			s := &Set{
				Num: e.Name() + "/" + m.ID, Name: m.Name, Year: m.Year, Theme: col.Name, ThemeGroup: col.Name,
				ImageURL: m.Image, Kind: CustomModel, Designer: col.Designer, URL: m.URL, Collection: col.Name, Source: src,
			}
			for _, k := range sortedKeys(lines) {
				s.Lines = append(s.Lines, Line{Part: k.Part, Color: k.Color, Quantity: lines[k], ImageURL: c.Images[k]})
			}
			c.Sets[s.Num] = s
		}
		log.Printf("collections: %s, %d models", col.Name, len(col.Models))
	}
	return nil
}

// readModelFile reads a model's parts from a parts list (a CSV exported from
// BrickLink Studio or Rebrickable, BrickLink's XML) or an LDraw model, as
// the catalog's parts and colors; what it cannot place is in the Source.
func (c *Catalog) readModelFile(path string, lib *ldraw.Library) (map[PartColor]int32, *Source, error) {
	lines := map[PartColor]int32{}
	src := &Source{}
	var lost []string
	add := func(pc PartColor, n int32) {
		p, ok := c.LikePart(pc.Part)
		if !ok {
			lost = append(lost, pc.Part)
			return
		}
		lines[PartColor{p, pc.Color}] += n
		src.Placed += n
	}
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ldr", ".mpd":
		var m *ldraw.Model
		if m, err = ldraw.ReadFile(path); err == nil {
			m.Walk(m.Main(), ldraw.Inherit, ldraw.Identity, func(name string, color int32, _ ldraw.Transform) {
				c.ldrawPiece(name, color, lib, func(pc PartColor) { add(pc, 1) }, func(s string) { lost = append(lost, s) })
			})
		}
	case ".csv":
		err = c.readPartsCSV(path, add, func(s string) { lost = append(lost, s) })
	case ".xml":
		err = c.readBrickLinkXML(path, add, func(s string) { lost = append(lost, s) })
	default:
		err = fmt.Errorf("no reader for %s files", filepath.Ext(path))
	}
	src.Unplaced = dedupe(lost)
	return lines, src, err
}

// readPartsCSV reads a parts list exported as CSV: BrickLink Studio's
// (BLItemNo, BLColorId, LdrawId, LDrawColorId, Qty) or Rebrickable's (Part,
// Color, Quantity, Is Spare).
func (c *Catalog) readPartsCSV(path string, add func(PartColor, int32), lost func(string)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil {
		return err
	}
	col := map[string]int{}
	for i, h := range head {
		col[strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))] = i
	}
	get := func(rec []string, k string) string {
		if i, ok := col[k]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	n32 := func(s string) int32 { n, _ := strconv.Atoi(s); return int32(n) }
	_, studio := col["BLItemNo"]
	_, rebrickable := col["Part"]
	if !studio && !rebrickable {
		return fmt.Errorf("not a parts list CSV I know: %v", head)
	}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if rebrickable {
			if strings.EqualFold(get(rec, "Is Spare"), "true") {
				continue
			}
			add(PartColor{get(rec, "Part"), n32(get(rec, "Color"))}, n32(get(rec, "Quantity")))
			continue
		}
		qty := n32(get(rec, "Qty"))
		bl, ld := get(rec, "BLItemNo"), ldraw.Name(get(rec, "LdrawId"))
		if qty <= 0 || (bl == "" && ld == "") {
			continue // the totals at its end
		}
		if pc, ok := c.FromBrickLink(bl, n32(get(rec, "BLColorId")), true); ok {
			add(pc, qty)
			continue
		}
		colr, ok := c.BrickLinkColors[n32(get(rec, "BLColorId"))]
		if !ok {
			colr, ok = c.LDrawColor(n32(get(rec, "LDrawColorId")))
		}
		p := bl
		if p == "" {
			p = ld
		}
		if !ok || p == "" {
			lost(fmt.Sprintf("%s (BrickLink %s, color %s)", get(rec, "PartName"), bl, get(rec, "ColorName")))
			continue
		}
		add(PartColor{p, colr}, qty)
	}
}

// readBrickLinkXML reads a BrickLink wanted list or inventory XML.
func (c *Catalog) readBrickLinkXML(path string, add func(PartColor, int32), lost func(string)) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var inv struct {
		Items []struct {
			Type   string `xml:"ITEMTYPE"`
			ID     string `xml:"ITEMID"`
			Color  int32  `xml:"COLOR"`
			MinQty int32  `xml:"MINQTY"`
			Qty    int32  `xml:"QTY"`
		} `xml:"ITEM"`
	}
	if err := xml.Unmarshal(b, &inv); err != nil {
		return err
	}
	for _, it := range inv.Items {
		if it.Type != "" && it.Type != "P" {
			continue
		}
		n := max(it.MinQty, it.Qty)
		if pc, ok := c.FromBrickLink(it.ID, it.Color, true); ok {
			add(pc, n)
			continue
		}
		col, ok := c.BrickLinkColors[it.Color]
		if !ok {
			lost(fmt.Sprintf("BrickLink %s in color %d", it.ID, it.Color))
			continue
		}
		add(PartColor{it.ID, col}, n)
	}
	return nil
}
