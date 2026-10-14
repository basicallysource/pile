// Package ldraw reads LDraw models (.ldr, and .mpd files of several
// submodels) and the LDraw parts library, enough to count a model's parts
// (which part, in which LDraw color, how many) and to tell which of them
// touch: every part's place, and its bounding box from the library.
//
// A model is a list of references, each a part or a submodel placed in a
// color at a position and rotation. Color 16 is the color the reference
// above it gave (24 is for edge lines, which a part never uses). Distances
// are LDraw units: a stud is 20 wide, a brick 24 tall, a plate 8; -y is up.
package ldraw

import (
	"bufio"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// Inherit is the color a reference takes from the one placing it.
const Inherit = 16

// Ref is one part or submodel placed in a color, where Place puts it.
type Ref struct {
	Color int32
	Name  string
	Place Transform
}

// Transform places a file in its parent's frame: a point p of it lands at
// Rot·p + Pos.
type Transform struct {
	Pos [3]float64
	Rot [9]float64 // row-major
}

// Identity places a file where it is.
var Identity = Transform{Rot: [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}}

// Then is t applied inside parent: a point of t's file lands where parent
// then puts it.
func (t Transform) Then(parent Transform) Transform {
	var out Transform
	r, p := parent.Rot, t.Rot
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			out.Rot[3*i+j] = r[3*i]*p[j] + r[3*i+1]*p[3+j] + r[3*i+2]*p[6+j]
		}
		out.Pos[i] = r[3*i]*t.Pos[0] + r[3*i+1]*t.Pos[1] + r[3*i+2]*t.Pos[2] + parent.Pos[i]
	}
	return out
}

// Apply is where p lands.
func (t Transform) Apply(p [3]float64) [3]float64 {
	var out [3]float64
	for i := 0; i < 3; i++ {
		out[i] = t.Rot[3*i]*p[0] + t.Rot[3*i+1]*p[1] + t.Rot[3*i+2]*p[2] + t.Pos[i]
	}
	return out
}

// Box is an axis-aligned box; an empty one has Min above Max.
type Box struct{ Min, Max [3]float64 }

// Empty is the box with nothing in it.
var Empty = Box{Min: [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}, Max: [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}}

func (b Box) IsEmpty() bool { return b.Min[0] > b.Max[0] }

// Add is b grown to hold p.
func (b Box) Add(p [3]float64) Box {
	for i := 0; i < 3; i++ {
		b.Min[i] = min(b.Min[i], p[i])
		b.Max[i] = max(b.Max[i], p[i])
	}
	return b
}

// Union is the box holding both.
func (b Box) Union(o Box) Box {
	if o.IsEmpty() {
		return b
	}
	return b.Add(o.Min).Add(o.Max)
}

// Moved is the box holding b once t places it.
func (b Box) Moved(t Transform) Box {
	if b.IsEmpty() {
		return b
	}
	out := Empty
	for c := 0; c < 8; c++ {
		p := [3]float64{b.Min[0], b.Min[1], b.Min[2]}
		for i := 0; i < 3; i++ {
			if c&(1<<i) != 0 {
				p[i] = b.Max[i]
			}
		}
		out = out.Add(t.Apply(p))
	}
	return out
}

// Overlap is how deep two boxes go into each other: the least, over the
// three axes, of their overlap. Below zero they are apart.
func (b Box) Overlap(o Box) float64 {
	d := math.Inf(1)
	for i := 0; i < 3; i++ {
		d = min(d, min(b.Max[i], o.Max[i])-max(b.Min[i], o.Min[i]))
	}
	return d
}

// Model is an LDraw file: its submodels in the order they appear, each the
// references it places. The first is the main model; a plain .ldr is one
// submodel.
type Model struct {
	Order []string
	Files map[string][]Ref
	// Each submodel's name as its file wrote it.
	Titles map[string]string
	// The box holding each submodel's own lines, triangles and quads (a
	// part's, in the library), not what it places.
	Shapes map[string]Box
}

// Name is how LDraw names a file: lowercased, forward slashes, no ".dat".
func Name(s string) string {
	s = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), `\`, "/"))
	return strings.TrimSuffix(s, ".dat")
}

// Main is the name of the main model.
func (m *Model) Main() string { return m.Order[0] }

// Has is whether name is a submodel of m rather than a part.
func (m *Model) Has(name string) bool {
	_, ok := m.Files[name]
	return ok
}

// ReadFile reads the model at path.
func ReadFile(path string) (*Model, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Read(f)
}

// Read reads a model. Lines it neither counts nor measures by (comments,
// conditional lines, embedded data) are skipped.
func Read(r io.Reader) (*Model, error) {
	m := &Model{Files: map[string][]Ref{}, Titles: map[string]string{}, Shapes: map[string]Box{}}
	cur := ""
	inData := false
	start := func(title string) {
		cur = Name(title)
		if _, ok := m.Files[cur]; !ok {
			m.Files[cur] = nil
			m.Titles[cur] = strings.TrimSpace(title)
			m.Shapes[cur] = Empty
			m.Order = append(m.Order, cur)
		}
	}
	nums := func(f []string) ([]float64, bool) {
		out := make([]float64, len(f))
		for i, s := range f {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, false
			}
			out[i] = v
		}
		return out, true
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			continue
		}
		if f[0] == "0" {
			switch {
			case len(f) >= 3 && f[1] == "FILE":
				inData = false
				start(strings.Join(f[2:], " "))
			case len(f) >= 2 && (f[1] == "!DATA" || f[1] == "NOFILE"):
				inData = f[1] == "!DATA"
			}
			continue
		}
		if inData {
			continue
		}
		if cur == "" && len(m.Order) == 0 {
			start("")
		}
		switch f[0] {
		case "1":
			if len(f) < 15 {
				continue
			}
			c, err := strconv.ParseInt(f[1], 0, 64)
			v, ok := nums(f[2:14])
			if err != nil || !ok {
				continue
			}
			t := Transform{Pos: [3]float64{v[0], v[1], v[2]}}
			copy(t.Rot[:], v[3:12])
			m.Files[cur] = append(m.Files[cur], Ref{Color: int32(c), Name: Name(strings.Join(f[14:], " ")), Place: t})
		case "2", "3", "4":
			n := map[string]int{"2": 2, "3": 3, "4": 4}[f[0]]
			if len(f) < 2+3*n {
				continue
			}
			v, ok := nums(f[2 : 2+3*n])
			if !ok {
				continue
			}
			b := m.Shapes[cur]
			for i := 0; i < n; i++ {
				b = b.Add([3]float64{v[3*i], v[3*i+1], v[3*i+2]})
			}
			m.Shapes[cur] = b
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(m.Order) == 0 {
		start("")
	}
	return m, nil
}

// Walk calls part with every part placed by submodel name, through its
// submodels, in the color it ends up in and where it ends up; color and at
// are where name itself is placed.
func (m *Model) Walk(name string, color int32, at Transform, part func(name string, color int32, at Transform)) {
	m.walk(name, color, at, part, map[string]bool{})
}

func (m *Model) walk(name string, color int32, at Transform, part func(string, int32, Transform), open map[string]bool) {
	if open[name] {
		return // a submodel that places itself
	}
	open[name] = true
	defer delete(open, name)
	for _, r := range m.Files[name] {
		c := r.Color
		if c == Inherit || c == 24 {
			c = color
		}
		t := r.Place.Then(at)
		if m.Has(r.Name) {
			m.walk(r.Name, c, t, part, open)
		} else {
			part(r.Name, c, t)
		}
	}
}
