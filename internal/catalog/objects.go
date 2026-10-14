package catalog

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/basicallysource/pile/internal/ldraw"
)

// object is one thing a 3D model builds (a building, a vehicle): parts
// that hold together, apart from every other part, found from where the
// parts are and not from how the model's author grouped them, which is
// often by building step. Copies of one object are one object.
type object struct {
	// The name of the author's submodel with the most of its parts in it,
	// when that says more than a step number; else empty.
	name   string
	copies int32
	lines  map[PartColor]int32 // one copy
	// One copy's parts where the model puts them, to draw it.
	parts []ldraw.Placed
}

const (
	// Two parts hold together when the body of one goes this far into the
	// other's (a pin in a hole, a bar in a clip, an axle in a wheel), or a
	// stud of one is inside the other's body. Faces that only touch, and
	// a part resting on a smooth surface, do not.
	holdDepth = 1.5
	// Where studs are looked up: a cube this wide.
	studCell = 20.0
	// Where bodies are compared: a cube this wide.
	bodyCell = 40.0
)

// A stud is in a body when it is this far inside it on every axis.
const studInside = 0.5

type modelPart struct {
	placed ldraw.Placed
	lines  []PartColor
	body   ldraw.Box
}

type modelStud struct {
	at   [3]float64
	part int
}

// objects splits the model's main model into the objects it builds.
func (c *Catalog) objects(m *ldraw.Model, setName string, lib *ldraw.Library, lost func(string)) []*object {
	// Every part, and every submodel placed (with the parts inside it).
	var parts []modelPart
	var studs []modelStud
	type placement struct {
		title string
		parts []int
	}
	var places []*placement
	open := map[string]bool{}
	var walk func(name string, color int32, at ldraw.Transform, chain []*placement)
	walk = func(name string, color int32, at ldraw.Transform, chain []*placement) {
		if open[name] {
			return
		}
		open[name] = true
		defer delete(open, name)
		for _, r := range m.Files[name] {
			col := r.Color
			if col == ldraw.Inherit || col == 24 {
				col = color
			}
			t := r.Place.Then(at)
			if m.Has(r.Name) {
				p := &placement{title: m.Titles[r.Name]}
				places = append(places, p)
				walk(r.Name, col, t, append(chain[:len(chain):len(chain)], p))
				continue
			}
			i := len(parts)
			mp := modelPart{placed: ldraw.Placed{Name: r.Name, Color: col, At: t}}
			c.ldrawPiece(r.Name, col, lib, func(pc PartColor) { mp.lines = append(mp.lines, pc) }, lost)
			if sh, ok := lib.Shape(r.Name); ok {
				mp.body = sh.Body.Moved(t)
				for _, s := range sh.Studs {
					studs = append(studs, modelStud{t.Apply(s), i})
				}
			} else {
				mp.body = ldraw.Empty.Add(t.Pos)
			}
			parts = append(parts, mp)
			for _, p := range chain {
				p.parts = append(p.parts, i)
			}
		}
	}
	walk(m.Main(), ldraw.Inherit, ldraw.Identity, nil)

	uf := newUnion(len(parts))
	holdByBodies(parts, uf)
	holdByStuds(parts, studs, uf)

	// Each group of parts held together, in the model's order.
	var groups [][]int
	at := map[int]int{}
	for i := range parts {
		r := uf.find(i)
		g, ok := at[r]
		if !ok {
			g = len(groups)
			at[r] = g
			groups = append(groups, nil)
		}
		groups[g] = append(groups[g], i)
	}

	// Name each by the submodel that covers the most of it, of those mostly
	// in it (a sign or a loose piece of a building can stand apart) and
	// covering at least a quarter of it.
	names := make([]string, len(groups))
	best := make([]int, len(groups))
	for _, p := range places {
		if len(p.parts) == 0 || !telling(p.title) {
			continue
		}
		in := map[int]int{}
		g, most := 0, 0
		for _, i := range p.parts {
			h := at[uf.find(i)]
			in[h]++
			if in[h] > most {
				g, most = h, in[h]
			}
		}
		if 2*most >= len(p.parts) && 4*most >= len(groups[g]) && most > best[g] {
			best[g], names[g] = most, cleanName(p.title)
		}
	}
	// Pieces, not parts placed: a model can draw stickers and decoration
	// as loose geometry.
	all := 0
	for _, p := range parts {
		all += len(p.lines)
	}
	for g, ps := range groups {
		n := 0
		for _, i := range ps {
			n += len(parts[i].lines)
		}
		// Most of the model is the set itself, unless one submodel is most
		// of it (not a floor of a building).
		if 2*n > all && 2*best[g] < len(ps) {
			names[g] = setName
		}
	}

	var out []*object
	same := map[string]*object{}
	for g, ps := range groups {
		lines := map[PartColor]int32{}
		for _, i := range ps {
			for _, pc := range parts[i].lines {
				lines[pc]++
			}
		}
		key := linesKey(lines)
		if o, ok := same[key]; ok {
			o.copies++
			if o.name == "" {
				o.name = names[g]
			}
			continue
		}
		o := &object{name: names[g], copies: 1, lines: lines}
		for _, i := range ps {
			o.parts = append(o.parts, parts[i].placed)
		}
		same[key] = o
		out = append(out, o)
	}
	return out
}

// holdByBodies joins parts whose bodies go into each other. Each body is
// filed under the cells of a coarse grid it covers, so only parts that
// share a cell are compared.
func holdByBodies(parts []modelPart, uf *union) {
	type key [3]int64
	cell := func(v float64) int64 { return int64(math.Floor(v / bodyCell)) }
	grid := map[key][]int{}
	for i, p := range parts {
		b := p.body
		if b.IsEmpty() {
			continue
		}
		for x := cell(b.Min[0]); x <= cell(b.Max[0]); x++ {
			for y := cell(b.Min[1]); y <= cell(b.Max[1]); y++ {
				for z := cell(b.Min[2]); z <= cell(b.Max[2]); z++ {
					k := key{x, y, z}
					for _, j := range grid[k] {
						if uf.find(i) != uf.find(j) && parts[j].body.Overlap(b) > holdDepth {
							uf.join(i, j)
						}
					}
					grid[k] = append(grid[k], i)
				}
			}
		}
	}
}

// holdByStuds joins each part to the parts whose studs are inside its body.
func holdByStuds(parts []modelPart, studs []modelStud, uf *union) {
	cell := func(v float64) int64 { return int64(math.Floor(v / studCell)) }
	type key [3]int64
	grid := map[key][]int{}
	for i, s := range studs {
		k := key{cell(s.at[0]), cell(s.at[1]), cell(s.at[2])}
		grid[k] = append(grid[k], i)
	}
	for i, p := range parts {
		b := p.body
		if b.IsEmpty() {
			continue
		}
		for x := cell(b.Min[0]); x <= cell(b.Max[0]); x++ {
			for y := cell(b.Min[1]); y <= cell(b.Max[1]); y++ {
				for z := cell(b.Min[2]); z <= cell(b.Max[2]); z++ {
					for _, si := range grid[key{x, y, z}] {
						s := studs[si]
						if s.part != i && uf.find(s.part) != uf.find(i) && inside(s.at, b) {
							uf.join(s.part, i)
						}
					}
				}
			}
		}
	}
}

func inside(p [3]float64, b ldraw.Box) bool {
	for i := 0; i < 3; i++ {
		if p[i] < b.Min[i]+studInside || p[i] > b.Max[i]-studInside {
			return false
		}
	}
	return true
}

var (
	// A set's number before a submodel's name: "6000 - ", "6000-1_".
	setPrefix = regexp.MustCompile(`^\s*\d+(-\d+)?\s*[-_ ]+\s*`)
	modelExt  = regexp.MustCompile(`(?i)\.(ldr|mpd|dat)$`)
	// Names that say where in the building it is, not what it is: "1",
	// "Step 3", "step84-16", "137-210", "sub 2", "main".
	stepName = regexp.MustCompile(`(?i)(^(sub|subfile|submodel|part|section|stage|bag|book|page|model|assembly|assy)?[\s\-_#.]*[0-9]+[a-z]?$|^(main|main model|model|untitled)$|step[\s\-_]*[0-9])`)
	// The submodels LDraw models use to bend a flexible part (a hose, a
	// string), which are one part, not a thing.
	bentPart = regexp.MustCompile(`(?i)(flex|hose|bend|bent|cable|string|rubber|band|chain|tube)`)
	letter   = regexp.MustCompile(`[A-Za-z]`)
)

// cleanName is a submodel's name as an object's: without the set's number
// or the file's extension, and of a name in parts ("Ground Floor - Pram"),
// the last.
func cleanName(title string) string {
	n := modelExt.ReplaceAllString(strings.TrimSpace(title), "")
	n = setPrefix.ReplaceAllString(n, "")
	n = strings.TrimSpace(strings.ReplaceAll(n, "_", " "))
	if i := strings.LastIndex(n, " - "); i >= 0 {
		if last := strings.TrimSpace(n[i+3:]); len(letter.FindAllString(last, -1)) >= 3 {
			return last
		}
	}
	return n
}

// telling is whether a submodel's name says what it is: some letters, not
// a step's number or a bent part.
func telling(title string) bool {
	n := cleanName(title)
	return len(letter.FindAllString(n, -1)) >= 3 && !stepName.MatchString(n) && !bentPart.MatchString(n)
}

func linesKey(lines map[PartColor]int32) string {
	var b strings.Builder
	for _, k := range sortedKeys(lines) {
		b.WriteString(k.Part)
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(int(k.Color)))
		b.WriteByte('x')
		b.WriteString(strconv.Itoa(int(lines[k])))
		b.WriteByte(' ')
	}
	return b.String()
}

// union is a union-find over n items.
type union struct{ parent []int }

func newUnion(n int) *union {
	u := &union{parent: make([]int, n)}
	for i := range u.parent {
		u.parent[i] = i
	}
	return u
}

func (u *union) find(i int) int {
	for u.parent[i] != i {
		u.parent[i] = u.parent[u.parent[i]]
		i = u.parent[i]
	}
	return i
}

func (u *union) join(a, b int) { u.parent[u.find(a)] = u.find(b) }
