package ldraw

import (
	"bufio"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Placed is a part where a model puts it, in the color it ends up in.
type Placed struct {
	Name  string
	Color int32
	At    Transform
}

// Drawer draws parts from the library as a still picture: from above and to
// the front left, as building instructions show a model, shaded, with the
// library's edge lines and LDraw's colors (LDConfig.ldr).
type Drawer struct {
	lib    *Library
	colors map[int32]paint
	mu     sync.Mutex
	shapes map[string]*drawing
}

type paint struct {
	face, edge color.NRGBA
}

// drawing is one library file's own triangles and lines, and what it places.
type drawing struct {
	tris  []face
	lines []face // two points each
	refs  []Ref
}

type face struct {
	p     [3][3]float64
	color int32
}

// NewDrawer draws from lib, in the colors of its LDConfig.ldr.
func NewDrawer(lib *Library) *Drawer {
	d := &Drawer{lib: lib, colors: map[int32]paint{}, shapes: map[string]*drawing{}}
	if f, err := os.Open(filepath.Join(lib.dir, "LDConfig.ldr")); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			fs := strings.Fields(sc.Text())
			if len(fs) < 3 || fs[1] != "!COLOUR" {
				continue
			}
			var code int64 = -1
			p := paint{}
			alpha := uint8(255)
			for i := 2; i+1 < len(fs); i++ {
				switch fs[i] {
				case "CODE":
					code, _ = strconv.ParseInt(fs[i+1], 10, 32)
				case "VALUE":
					p.face = hexColor(fs[i+1])
				case "EDGE":
					p.edge = hexColor(fs[i+1])
				case "ALPHA":
					a, _ := strconv.Atoi(fs[i+1])
					alpha = uint8(a)
				}
			}
			p.face.A, p.edge.A = alpha, 255
			if code >= 0 {
				d.colors[int32(code)] = p
			}
		}
	}
	return d
}

func hexColor(s string) color.NRGBA {
	v, _ := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

// paintOf is the color code c's face and edge colors; LDraw's direct colors
// (0x2RRGGBB) are their own.
func (d *Drawer) paintOf(c int32) paint {
	if p, ok := d.colors[c]; ok {
		return p
	}
	if c >= 0x2000000 && c < 0x3000000 {
		f := color.NRGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255}
		return paint{face: f, edge: color.NRGBA{f.R / 2, f.G / 2, f.B / 2, 255}}
	}
	return paint{face: color.NRGBA{160, 165, 169, 255}, edge: color.NRGBA{80, 80, 80, 255}}
}

func (d *Drawer) shape(name string) *drawing {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s, ok := d.shapes[name]; ok {
		return s
	}
	s := &drawing{}
	d.shapes[name] = s
	d.lib.mu.Lock()
	f := d.lib.file(name)
	d.lib.mu.Unlock()
	if f.path == "" {
		return s
	}
	s.refs = f.model.Files[f.model.Main()]
	r, err := os.Open(f.path)
	if err != nil {
		return s
	}
	defer r.Close()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 {
			continue
		}
		n := map[string]int{"2": 2, "3": 3, "4": 4}[fs[0]]
		if n == 0 || len(fs) < 2+3*n {
			continue
		}
		c, err := strconv.ParseInt(fs[1], 0, 64)
		if err != nil {
			continue
		}
		var v [4][3]float64
		ok := true
		for i := 0; i < n && ok; i++ {
			for j := 0; j < 3; j++ {
				x, err := strconv.ParseFloat(fs[2+3*i+j], 64)
				ok = ok && err == nil
				v[i][j] = x
			}
		}
		if !ok {
			continue
		}
		switch n {
		case 2:
			s.lines = append(s.lines, face{[3][3]float64{v[0], v[1]}, int32(c)})
		case 3:
			s.tris = append(s.tris, face{[3][3]float64{v[0], v[1], v[2]}, int32(c)})
		case 4:
			s.tris = append(s.tris, face{[3][3]float64{v[0], v[1], v[2]}, int32(c)}, face{[3][3]float64{v[0], v[2], v[3]}, int32(c)})
		}
	}
	return s
}

// The view: turned about the vertical axis, then tipped toward the viewer.
var view = func() Transform {
	turn, tip := -40*math.Pi/180, 28*math.Pi/180
	cy, sy, cx, sx := math.Cos(turn), math.Sin(turn), math.Cos(tip), math.Sin(tip)
	y := Transform{Rot: [9]float64{cy, 0, sy, 0, 1, 0, -sy, 0, cy}}
	x := Transform{Rot: [9]float64{1, 0, 0, 0, cx, -sx, 0, sx, cx}}
	return y.Then(x)
}()

// Draw draws parts in a square picture size wide, the parts filling it.
// It draws as it walks the parts, so no build is held whole as triangles:
// first it frames the picture by the parts' bodies, then draws every face
// and edge straight onto the canvas, keeping only the see-through faces to
// draw last.
func (d *Drawer) Draw(parts []Placed, size int) image.Image {
	const ss = 2 // drawn this many times larger, then shrunk, for smooth edges
	w := size * ss
	img := newCanvas(w)
	box := Empty
	for _, p := range parts {
		if sh, ok := d.lib.Shape(p.Name); ok {
			box = box.Union(sh.Body.Moved(p.At.Then(view)))
		}
	}
	if box.IsEmpty() {
		return img.shrink(ss)
	}
	// Studs stand above the bodies; leave room for them.
	scale := 0.9 * float64(w) / max(box.Max[0]-box.Min[0], box.Max[1]-box.Min[1], 1)
	cx, cy := (box.Min[0]+box.Max[0])/2, (box.Min[1]+box.Max[1])/2
	screen := func(p [3]float64) [3]float64 {
		return [3]float64{(p[0]-cx)*scale + float64(w)/2, (p[1]-cy)*scale + float64(w)/2, p[2]}
	}
	light := [3]float64{-0.35, -0.85, -0.4}
	ln := math.Sqrt(light[0]*light[0] + light[1]*light[1] + light[2]*light[2])
	for i := range light {
		light[i] /= ln
	}
	edge := max(1, float64(ss)*float64(size)/400)
	var clear []face
	var walk func(name string, c int32, at Transform, depth int)
	walk = func(name string, c int32, at Transform, depth int) {
		if depth > 16 {
			return
		}
		s := d.shape(name)
		resolve := func(fc int32) int32 {
			switch fc {
			case Inherit:
				return c
			case 24:
				return -c - 1 // the edge color of c
			}
			return fc
		}
		for _, t := range s.tris {
			f := face{[3][3]float64{at.Apply(t.p[0]), at.Apply(t.p[1]), at.Apply(t.p[2])}, resolve(t.color)}
			col := d.paintFor(f.color)
			if col.A < 255 {
				clear = append(clear, f)
				continue
			}
			img.triangle(screen(f.p[0]), screen(f.p[1]), screen(f.p[2]), shade(f.p, col, light), true)
		}
		for _, l := range s.lines {
			img.line(screen(at.Apply(l.p[0])), screen(at.Apply(l.p[1])), d.paintFor(resolve(l.color)), edge)
		}
		for _, r := range s.refs {
			rc := r.Color
			if rc == Inherit {
				rc = c
			}
			walk(r.Name, rc, r.Place.Then(at), depth+1)
		}
	}
	for _, p := range parts {
		walk(p.Name, p.Color, p.At.Then(view), 0)
	}
	for _, t := range clear {
		img.triangle(screen(t.p[0]), screen(t.p[1]), screen(t.p[2]), shade(t.p, d.paintFor(t.color), light), false)
	}
	return img.shrink(ss)
}

// paintFor is a resolved code's color: its face, or (below zero) the edge
// color of code -c-1.
func (d *Drawer) paintFor(c int32) color.NRGBA {
	if c < 0 {
		return d.paintOf(-c - 1).edge
	}
	return d.paintOf(c).face
}

// shade lights a face from above, to the front left, both sides alike.
func shade(p [3][3]float64, c color.NRGBA, light [3]float64) color.NRGBA {
	a := [3]float64{p[1][0] - p[0][0], p[1][1] - p[0][1], p[1][2] - p[0][2]}
	b := [3]float64{p[2][0] - p[0][0], p[2][1] - p[0][1], p[2][2] - p[0][2]}
	n := [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
	l := math.Sqrt(n[0]*n[0] + n[1]*n[1] + n[2]*n[2])
	k := 0.6
	if l > 0 {
		k = 0.55 + 0.45*math.Abs((n[0]*light[0]+n[1]*light[1]+n[2]*light[2])/l)
	}
	f := func(v uint8) uint8 { return uint8(min(255, float64(v)*k)) }
	return color.NRGBA{f(c.R), f(c.G), f(c.B), c.A}
}

// canvas is a picture being drawn, with a depth for every pixel (nearer is
// smaller).
type canvas struct {
	w     int
	pix   []color.NRGBA
	depth []float64
}

func newCanvas(w int) *canvas {
	c := &canvas{w: w, pix: make([]color.NRGBA, w*w), depth: make([]float64, w*w)}
	for i := range c.depth {
		c.depth[i] = math.Inf(1)
	}
	return c
}

func (c *canvas) put(i int, col color.NRGBA, z float64, solid bool) {
	if z > c.depth[i] {
		return
	}
	if solid {
		c.pix[i], c.depth[i] = col, z
		return
	}
	// See-through: over what is behind it, without hiding what comes later.
	a := float64(col.A) / 255
	under := c.pix[i]
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*a + float64(y)*(1-a)) }
	if under.A == 0 {
		c.pix[i] = col
		return
	}
	c.pix[i] = color.NRGBA{mix(col.R, under.R), mix(col.G, under.G), mix(col.B, under.B), max(under.A, col.A)}
}

func (c *canvas) triangle(a, b, d [3]float64, col color.NRGBA, solid bool) {
	minX := max(0, int(math.Floor(min(a[0], b[0], d[0]))))
	maxX := min(c.w-1, int(math.Ceil(max(a[0], b[0], d[0]))))
	minY := max(0, int(math.Floor(min(a[1], b[1], d[1]))))
	maxY := min(c.w-1, int(math.Ceil(max(a[1], b[1], d[1]))))
	area := (b[0]-a[0])*(d[1]-a[1]) - (b[1]-a[1])*(d[0]-a[0])
	if area == 0 {
		return
	}
	for y := minY; y <= maxY; y++ {
		py := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			w0 := ((b[0]-px)*(d[1]-py) - (b[1]-py)*(d[0]-px)) / area
			w1 := ((d[0]-px)*(a[1]-py) - (d[1]-py)*(a[0]-px)) / area
			w2 := 1 - w0 - w1
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			c.put(y*c.w+x, col, w0*a[2]+w1*b[2]+w2*d[2], solid)
		}
	}
}

// line draws an edge a little in front of its faces, so they do not hide
// it whether they are drawn before it or after.
func (c *canvas) line(a, b [3]float64, col color.NRGBA, width float64) {
	dx, dy := b[0]-a[0], b[1]-a[1]
	n := int(math.Max(math.Abs(dx), math.Abs(dy))) + 1
	r := int(width / 2)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y, z := a[0]+dx*t, a[1]+dy*t, a[2]+(b[2]-a[2])*t-0.6
		for oy := -r; oy <= r; oy++ {
			for ox := -r; ox <= r; ox++ {
				px, py := int(x)+ox, int(y)+oy
				if px >= 0 && py >= 0 && px < c.w && py < c.w {
					if i := py*c.w + px; z <= c.depth[i] {
						c.pix[i], c.depth[i] = col, z
					}
				}
			}
		}
	}
}

// shrink averages each k×k block into one pixel.
func (c *canvas) shrink(k int) image.Image {
	w := c.w / k
	out := image.NewNRGBA(image.Rect(0, 0, w, w))
	for y := 0; y < w; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, a float64
			for j := 0; j < k; j++ {
				for i := 0; i < k; i++ {
					p := c.pix[(y*k+j)*c.w+x*k+i]
					pa := float64(p.A)
					r, g, b, a = r+float64(p.R)*pa, g+float64(p.G)*pa, b+float64(p.B)*pa, a+pa
				}
			}
			if a > 0 {
				out.SetNRGBA(x, y, color.NRGBA{uint8(r / a), uint8(g / a), uint8(b / a), uint8(a / float64(k*k))})
			}
		}
	}
	return out
}
