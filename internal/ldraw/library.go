package ldraw

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Library is the LDraw parts library unpacked in a folder (parts/ and p/,
// as in LDraw's complete.zip). It says what a model's reference is when no
// catalog knows it (a part made of other parts, such as a wheel with its
// tire, or an alias of another part; a primitive or subpart, a piece of a
// part's geometry and never a part of its own; or nothing it has), and a
// part's shape: its body and its studs, which is how parts hold together.
type Library struct {
	dir   string
	mu    sync.Mutex
	files map[string]*libFile
}

type libFile struct {
	path  string // "" when the library does not have it
	model *Model
	kind  Kind
	parts []Ref // an assembly's parts
	shape *Shape
}

// Shape is a part's body (the box holding its geometry but its studs) and
// where its studs are (each one's middle), in the part's own frame. A
// stud of one part inside another's body is what holds them together.
type Shape struct {
	Body  Box
	Studs [][3]float64
}

// Kind is what the library has under a name.
type Kind int

const (
	// Unknown: not in the library (or no library).
	Unknown Kind = iota
	// Part: a part, drawn from primitives and subparts.
	Part
	// Assembly: a part made of other parts (LDraw's shortcuts and aliases).
	Assembly
	// Geometry: a primitive or subpart, never a part of its own.
	Geometry
)

// OpenLibrary opens the library in dir; an empty dir gives one that knows
// nothing.
func OpenLibrary(dir string) *Library {
	return &Library{dir: dir, files: map[string]*libFile{}}
}

// Look says what name is, and for an assembly the parts it is made of.
func (l *Library) Look(name string) (Kind, []Ref) {
	if l == nil || l.dir == "" {
		return Unknown, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f := l.file(name)
	return f.kind, f.parts
}

// Shape is a part's shape; false when the library does not have it.
func (l *Library) Shape(name string) (Shape, bool) {
	if l == nil || l.dir == "" {
		return Shape{Body: Empty}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	s := l.shape(name, map[string]bool{})
	return s, !s.Body.IsEmpty()
}

// The primitives of a stud on top of a part (plain, hollow, printed and
// their variants); stud3 and stud4 are the tubes under one.
var studPrimitive = regexp.MustCompile(`^stud(p\d+|\d*[a-z]?)$`)

func isStud(name string) bool {
	b := path.Base(name)
	return studPrimitive.MatchString(b) && !strings.HasPrefix(b, "stud3") && !strings.HasPrefix(b, "stud4")
}

func (l *Library) shape(name string, open map[string]bool) Shape {
	f := l.file(name)
	if f.shape != nil {
		return *f.shape
	}
	if f.model == nil || open[name] {
		return Shape{Body: Empty}
	}
	open[name] = true
	defer delete(open, name)
	m := f.model
	s := Shape{Body: m.Shapes[m.Main()]}
	for _, r := range m.Files[m.Main()] {
		if isStud(r.Name) {
			// A stud stands 4 tall above its origin.
			s.Studs = append(s.Studs, r.Place.Apply([3]float64{0, -2, 0}))
			continue
		}
		sub := l.shape(r.Name, open)
		s.Body = s.Body.Union(sub.Body.Moved(r.Place))
		for _, p := range sub.Studs {
			s.Studs = append(s.Studs, r.Place.Apply(p))
		}
	}
	f.shape = &s
	return s
}

// file reads name from the library once: where it is, and what it is.
func (l *Library) file(name string) *libFile {
	if f, ok := l.files[name]; ok {
		return f
	}
	f := &libFile{kind: Unknown}
	// Mark it while it is read, so a file that places itself ends.
	l.files[name] = f
	geometry := strings.HasPrefix(name, "s/")
	for _, c := range []struct {
		dir      string
		geometry bool
	}{{"parts", geometry}, {"p", true}} {
		path := filepath.Join(l.dir, c.dir, filepath.FromSlash(name)+".dat")
		r, err := os.Open(path)
		if err != nil {
			continue
		}
		m, err := Read(r)
		r.Close()
		if err != nil {
			continue
		}
		f.path, f.model = path, m
		if c.geometry {
			f.kind = Geometry
			return f
		}
		// An assembly places other parts; a part places only primitives and
		// subparts.
		for _, ref := range m.Files[m.Main()] {
			if k := l.file(ref.Name).kind; k == Part || k == Assembly {
				f.parts = append(f.parts, ref)
			}
		}
		f.kind = Part
		if len(f.parts) > 0 {
			f.kind = Assembly
		}
		return f
	}
	return f
}
