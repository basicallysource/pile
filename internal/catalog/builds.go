package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/basicallysource/pile/internal/ldraw"
)

// BuildPicturesURL is where pile serves the pictures of builds, each at its
// build's number (BuildPicture), from the folder BuildPictures names.
const BuildPicturesURL = "/pictures/builds/"

// BuildPictures is the folder in a data folder that holds the pictures of
// builds, drawn by cmd/render-builds.
func BuildPictures(data string) string { return filepath.Join(data, "set-models", "pictures") }

// BuildPicture is where in BuildPictures a build's picture is.
func BuildPicture(num string) string { return filepath.FromSlash(num) + ".png" }

// A 3D model of a set, as set-models/index.json lists it (written by
// scripts/builds/gather.py).
type setModel struct {
	Set    string `json:"set"`
	File   string `json:"file"`
	Author string `json:"author"`
	Notes  string `json:"notes"`
	Page   string `json:"page"`
}

const (
	// A build needs this many counted pieces (plain parts and accessories):
	// fewer is a minifigure, a flower, a loose tool.
	minBuildPieces = 10
	// One its model gives no name needs this many: smaller is a stand, or
	// a detail that only rests on the rest.
	minUnnamedPieces = 40
)

// loadBuilds reads the 3D models of sets in dir and adds the builds within
// each set: the separate things its main model places (a building, a
// vehicle), each matched like a set. A missing folder adds none.
func (c *Catalog) loadBuilds(dir string, lib *ldraw.Library) error {
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var models []setModel
	if err := json.Unmarshal(b, &models); err != nil {
		return err
	}
	// Each set's model is read on its own, one per core; only the set
	// itself is written meanwhile.
	work := make(chan setModel)
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sm := range work {
				m, err := ldraw.ReadFile(filepath.Join(dir, sm.File))
				if err != nil {
					log.Printf("builds: %s: %v; skipped", sm.File, err)
					continue
				}
				set := c.Sets[sm.Set]
				set.Source = &Source{Author: sm.Author, URL: sm.Page, Notes: sm.Notes}
				c.addBuilds(set, m, lib)
			}
		}()
	}
	for _, sm := range models {
		if set := c.Sets[sm.Set]; set != nil && set.Sold() {
			work <- sm
		}
	}
	close(work)
	wg.Wait()

	var sets, builds, drawn int
	pictures := filepath.Join(dir, "pictures")
	for _, sm := range models {
		set := c.Sets[sm.Set]
		if set == nil || len(set.Builds) == 0 {
			continue
		}
		sets++
		for _, b := range set.Builds {
			c.Sets[b.Num] = b
			builds++
			// Its own picture once drawn; its set's until then.
			if _, err := os.Stat(filepath.Join(pictures, BuildPicture(b.Num))); err == nil {
				b.ImageURL = BuildPicturesURL + b.Num + ".png"
				drawn++
			}
		}
	}
	log.Printf("builds: %d in %d of %d set models, %d drawn", builds, sets, len(models), drawn)
	return nil
}

// addBuilds splits a set's 3D model into its builds (objects.go), reads
// each one's lines, fits them to the set's inventory, and gives them to
// the set; it writes nothing else, so sets can be read side by side.
func (c *Catalog) addBuilds(set *Set, m *ldraw.Model, lib *ldraw.Library) {
	var unplaced []string
	lost := func(what string) { unplaced = append(unplaced, what) }
	objs := c.objects(m, set.Name, lib, lost)
	total := map[PartColor]int32{}
	for _, o := range objs {
		for pc, n := range o.lines {
			total[pc] += n * o.copies
		}
	}

	// Fit the model to the set: a piece the model has more of than the set
	// becomes one the set has more of than the model. First the same part
	// or one alike (a variant of its mold, or one number but for a letter)
	// in the same color; then one alike in any color, since a model can
	// draw an old set in today's colors (light gray as light bluish gray).
	inv := map[PartColor]int32{}
	for _, l := range set.Lines {
		inv[PartColor{l.Part, l.Color}] += l.Quantity
	}
	invKeys := sortedKeys(inv)
	fit := func(sameColor bool) {
		for _, k := range sortedKeys(total) {
			excess := total[k] - inv[k]
			if excess <= 0 {
				continue
			}
			for _, q := range invKeys {
				if excess <= 0 {
					break
				}
				if q == k || (sameColor && q.Color != k.Color) || !c.alike(k.Part, q.Part) {
					continue
				}
				t := min(excess, inv[q]-total[q])
				// Move t pieces, whole copies of an object at a time.
				for _, o := range objs {
					n := min(o.lines[k], t/o.copies)
					if n <= 0 {
						continue
					}
					o.lines[k] -= n
					o.lines[q] += n
					total[k] -= n * o.copies
					total[q] += n * o.copies
					t -= n * o.copies
					excess -= n * o.copies
				}
			}
		}
	}
	fit(true)
	fit(false)
	keys := sortedKeys(total)

	src := set.Source
	for _, k := range keys {
		if _, ok := c.Parts[k.Part]; !ok && total[k] > 0 {
			unplaced = append(unplaced, k.Part)
		}
	}
	for k, n := range inv {
		src.Placed += min(n, total[k])
	}
	src.Unplaced = dedupe(unplaced)

	slugs := map[string]int{}
	var builds []*Set
	for _, o := range objs {
		s := &Set{
			Name: o.name, Year: set.Year, Theme: set.Theme, ThemeGroup: set.ThemeGroup,
			ImageURL: set.ImageURL, Licensed: set.Licensed, BulkTheme: set.BulkTheme,
			Kind: Build, Of: set, Copies: o.copies, Source: src, Parts: o.parts,
		}
		var counted, figure int32
		for _, k := range sortedKeys(o.lines) {
			n := o.lines[k]
			p := c.Parts[k.Part]
			if p == nil || n <= 0 {
				continue
			}
			s.Lines = append(s.Lines, Line{Part: k.Part, Color: k.Color, Quantity: n, ImageURL: c.Images[k]})
			switch p.Class {
			case Plain, Accessory:
				counted += n
			case FigurePart:
				figure += n
			}
		}
		if counted < minBuildPieces || figure >= counted || (s.Name == "" && counted < minUnnamedPieces) {
			continue
		}
		builds = append(builds, s)
	}
	// One build is the whole set.
	if len(builds) < 2 {
		return
	}
	for i, s := range builds {
		if s.Name == "" {
			s.Name = fmt.Sprintf("Build %d", i+1)
		}
		slug := slugOf(s.Name)
		if slug == "" {
			slug = fmt.Sprintf("build-%d", i+1)
		}
		slugs[slug]++
		if slugs[slug] > 1 {
			slug += fmt.Sprintf("-%d", slugs[slug])
		}
		s.Num = set.Num + "/" + slug
	}
	set.Builds = builds
}

// ldrawPiece reads one part a model places, in an LDraw color, as the
// catalog's: add gets each piece it is (a part made of parts is each of
// them; a piece of geometry is none), and lost what has no catalog part or
// color. A part only LDraw numbers is kept under its LDraw number, which
// fitting the model to its set may match.
func (c *Catalog) ldrawPiece(name string, color int32, lib *ldraw.Library, add func(PartColor), lost func(string)) {
	if p, ok := c.LDrawPart(name); ok {
		if col, ok := c.LDrawColor(color); ok {
			add(PartColor{p, col})
		} else {
			lost(fmt.Sprintf("%s in LDraw color %d", name, color))
		}
		return
	}
	switch kind, refs := lib.Look(name); kind {
	case ldraw.Geometry:
	case ldraw.Assembly:
		for _, r := range refs {
			rc := r.Color
			if rc == ldraw.Inherit {
				rc = color
			}
			c.ldrawPiece(r.Name, rc, lib, add, lost)
		}
	default:
		if col, ok := c.LDrawColor(color); ok {
			add(PartColor{name, col})
		} else {
			lost(fmt.Sprintf("%s in LDraw color %d", name, color))
		}
	}
}

var moldLetters = regexp.MustCompile(`[a-z]+$`)

// alike is whether two parts are one part to a set's inventory: variants
// of one mold, or one number but for a letter.
func (c *Catalog) alike(a, b string) bool {
	return c.Canon(a) == c.Canon(b) || moldLetters.ReplaceAllString(a, "") == moldLetters.ReplaceAllString(b, "")
}

var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugOf(name string) string {
	return strings.Trim(notSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

func sortedKeys(m map[PartColor]int32) []PartColor {
	ks := make([]PartColor, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool {
		if ks[i].Part != ks[j].Part {
			return ks[i].Part < ks[j].Part
		}
		return ks[i].Color < ks[j].Color
	})
	return ks
}

func dedupe(ss []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
