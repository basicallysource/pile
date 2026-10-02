// Package collect works out how long one sorter, fed bulk LEGO at random,
// takes to come across every piece of a set: in exact colors, in near colors
// (colors that were renamed or swapped over the years count as one), or in
// any color.
//
// The model: the sorter draws pieces one at a time from a well-mixed stream
// whose mix is what the sorters' records show (a stream, from one or more
// lots). A set needs n_k of each key k (a part in a color, a part in a near
// color, or a part); its share of the stream is p_k. Counting pieces as a
// Poisson process (which, at the thousands of pieces involved, is the
// discrete draw), the counts of the keys are independent, so the chance the
// set is complete after t pieces is the product over its keys of
// P(Poisson(p_k t) >= n_k), a coupon collector with quotas; and the share of
// the set found by then is the sum over keys of E[min(Poisson(p_k t), n_k)]
// over its pieces. Times are read off both by bisection.
//
// The shares come from the records, smoothed toward LEGO's own use of each
// key (empirical Bayes): a key's count c_k in N pieces is Poisson(N p_k), and
// p_k is Gamma with mean mu_k = e^a q_k^b (L_k/24)^c and shape
// alpha_k = e^g0 s_k^g1. q_k is the key's share of all the pieces in every
// set LEGO sold (each set once); L_k its part's length in millimeters when
// over 24 (long parts reach a sorter less often than LEGO's use of them says:
// people pick them out, feeders balk); s_k how many sets have the key (a key
// in many sets follows LEGO's use of it more closely than one in a few, whose
// share in bulk hangs on how many of those few were made). a, b, c, g0 and g1
// are fitted to the records by maximum likelihood over every key in the
// catalog (a negative binomial). A key's share is its posterior mean
// (c_k + alpha_k) / (N + alpha_k/mu_k): its own count when it was seen often,
// LEGO's use of it, scaled to the stream, when it was never seen.
//
// Uncertainty, three ways. The chance spread of one run (the 10th and 90th
// percentile of the time to every piece). Which bulk the sorters happened to
// be fed: the stream's machine-days resampled with replacement, the shares
// worked out again each time (the fit held). And the keys never seen, whose
// shares come from the catalog alone and on which the time to every piece
// hangs: that time again with them ten times rarer, and a floor from the
// data alone, the time to every key the sorters did see.
//
// What a sorter cannot be expected to find is left out of a set: printed
// parts and stickers, minifigures and their parts (minidoll parts too),
// specialty parts (a part in a category that is not a building piece, such as
// string, electronics or Duplo, or a mold used in fewer than MinSets sets),
// and parts too big to go through a sorter (MaxWidth, MaxLength).
package collect

import (
	"cmp"
	"database/sql"
	"fmt"
	"math"
	"slices"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/match"
	"github.com/basicallysource/pile/internal/records"
)

// Mode is how strictly a piece must match a set's line.
type Mode int

const (
	Exact Mode = iota
	// Colors LEGO renamed or swapped count as one: NearColors.
	Near
	// The part in any color.
	Any
)

func (m Mode) String() string { return [...]string{"exact colors", "near colors", "any color"}[m] }

// NearColors are the groups of Rebrickable colors near mode counts as one:
// the grays and browns LEGO changed in 2003 and 2004, and the silver that
// replaced Pearl Light Gray.
var NearColors = [][]int32{
	{71, 7},    // Light Bluish Gray, Light Gray
	{72, 8},    // Dark Bluish Gray, Dark Gray
	{70, 6},    // Reddish Brown, Brown
	{179, 135}, // Flat Silver, Pearl Light Gray
}

var nearOf = func() map[int32]int32 {
	m := map[int32]int32{}
	for _, g := range NearColors {
		for _, c := range g {
			m[c] = g[0]
		}
	}
	return m
}()

// Key is a part in a color as a mode sees it.
type Key struct{ Part, Color int32 }

// anyColor stands for every color in any mode.
const anyColor = -1

func (m Mode) key(part, color int32) Key {
	switch m {
	case Near:
		if g, ok := nearOf[color]; ok {
			color = g
		}
	case Any:
		color = anyColor
	}
	return Key{part, color}
}

// MinSets is how many sets a mold must be in not to be a specialty part.
const MinSets = 10

// The biggest a part may be and still reach a sorter, in millimeters: its
// width (the second-longest side of its box) and its length (the longest).
// Read off the records: sorters have classified next to nothing wider than 7
// studs or longer than 17, against what LEGO's use of those parts predicts.
const (
	MaxWidth  = 56.0
	MaxLength = 136.0
)

// Size is a part's length and width in millimeters, the two longest sides of
// its box.
type Size struct{ Length, Width float64 }

// specialtyCategories are Rebrickable part categories of things that are not
// building pieces a sorter sorts, or not of the system bricks sorters see.
var specialtyCategories = map[string]bool{
	"Duplo, Quatro and Primo":                      true,
	"Electronics":                                  true,
	"String, Bands and Reels":                      true,
	"Non-Buildable Figures (Duplo, Fabuland, etc)": true,
	"Large Buildable Figures":                      true,
	"Belville, Scala and Fabuland":                 true,
	"Modulex":                                      true,
	"Znap":                                         true,
	"Clikits":                                      true,
	"HO Scale":                                     true,
	"Non-System Parts":                             true,
	"Pen & Watch":                                  true,
	"Magnets and Holders":                          true,
	"Stickers":                                     true,
}

// Catalog is what every stream and set is measured against: the catalog, its
// index, and which parts are left out.
type Catalog struct {
	Cat *catalog.Catalog
	Ix  *match.Index
	// Sets LEGO sold with each part (any color) among their counted lines.
	setsWith map[int32]int
	// Each part's size, where known.
	size map[int32]Size
}

// NewCatalog readies the catalog for collect-time; sizes is each part's size
// by Rebrickable part number (LoadSizes).
func NewCatalog(cat *catalog.Catalog, ix *match.Index, sizes map[string]Size) *Catalog {
	c := &Catalog{Cat: cat, Ix: ix, setsWith: map[int32]int{}, size: map[int32]Size{}}
	for num, sz := range sizes {
		if _, ok := cat.Parts[num]; ok {
			c.size[ix.Part(num)] = sz
		}
	}
	for _, e := range ix.Entries {
		if e.Hidden || e.Set.Custom {
			continue
		}
		seen := map[int32]bool{}
		for _, l := range e.Lines {
			if !seen[l.Part] {
				seen[l.Part] = true
				c.setsWith[l.Part]++
			}
		}
	}
	return c
}

// Why says why a part is left out of every set, or "" when it counts.
func (c *Catalog) Why(part int32) string {
	p := c.Cat.Parts[c.Ix.PartNum(part)]
	switch {
	case p != nil && strings.HasPrefix(p.Category, "Minidoll"):
		return "minidoll part"
	case p != nil && specialtyCategories[p.Category]:
		return "not a building piece (" + p.Category + ")"
	case c.setsWith[part] < MinSets:
		return "specialty mold"
	case c.size[part].Width > MaxWidth || c.size[part].Length > MaxLength:
		return "too big for a sorter"
	}
	return ""
}

// SetsWith is how many sets LEGO sold have the part (any color).
func (c *Catalog) SetsWith(part int32) int { return c.setsWith[part] }

// Size is the part's size, zero when unknown.
func (c *Catalog) Size(part int32) Size { return c.size[part] }

// LoadSizes reads each part's size from a sorter Hive's parts database (its
// part_geometry, boxes from LDraw's models).
func LoadSizes(partsDB string) (map[string]Size, error) {
	db, err := sql.Open("sqlite", "file:"+partsDB+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`select part_num, bbox_x_mm, bbox_y_mm, bbox_z_mm from part_geometry where bbox_x_mm > 0`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", partsDB, err)
	}
	defer rows.Close()
	out := map[string]Size{}
	for rows.Next() {
		var num string
		d := make([]float64, 3)
		if err := rows.Scan(&num, &d[0], &d[1], &d[2]); err != nil {
			return nil, err
		}
		slices.Sort(d)
		out[num] = Size{Length: d[2], Width: d[1]}
	}
	return out, rows.Err()
}

// Stream is the mix of pieces some sorters classified.
type Stream struct {
	// Classified pieces, and the ones of a part the catalog knows.
	Pieces, Known int
	// Pieces by part (index id) and Rebrickable color; anyColor when the
	// classifier gave no color or the catalog has none for it.
	Counts map[Key]int
	// The same for each day of each machine, the bulk it was fed that day:
	// what the uncertainty resamples.
	Days []Day
}

// Day is one machine's pieces of one day.
type Day struct {
	Pieces int
	Counts map[Key]int
}

// NewStream counts records' pieces in the catalog's terms.
func (c *Catalog) NewStream(rs ...*records.Records) *Stream {
	s := &Stream{Counts: map[Key]int{}}
	for ri, r := range rs {
		days := map[[3]int64]*Day{}
		for _, pc := range r.Pieces {
			at := [3]int64{int64(ri), int64(pc.Machine), pc.SeenAt / 86400}
			d := days[at]
			if d == nil {
				d = &Day{Counts: map[Key]int{}}
				days[at] = d
			}
			s.Pieces++
			d.Pieces++
			part := c.Cat.Parts[c.Cat.BrickLinkParts[pc.BrickLinkPart]]
			if part == nil {
				continue
			}
			s.Known++
			color := int32(anyColor)
			if col, ok := c.Cat.BrickLinkColors[pc.BrickLinkColor]; ok && pc.ColorKnown {
				color = col
			}
			k := Key{c.Ix.Part(part.Num), color}
			s.Counts[k]++
			d.Counts[k]++
		}
		keys := make([][3]int64, 0, len(days))
		for k := range days {
			keys = append(keys, k)
		}
		slices.SortFunc(keys, func(a, b [3]int64) int {
			return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1]), cmp.Compare(a[2], b[2]))
		})
		for _, k := range keys {
			s.Days = append(s.Days, *days[k])
		}
	}
	return s
}

// counts is the stream's counts as mode sees them. In exact and near modes a
// piece of unknown color matches no key.
func (s *Stream) counts(m Mode) map[Key]float64 {
	out := map[Key]float64{}
	for k, n := range s.Counts {
		if k.Color == anyColor && m != Any {
			continue
		}
		out[m.key(k.Part, k.Color)] += float64(n)
	}
	return out
}

// prior is every counted, not left out key's share of the pieces of every
// set LEGO sold, each set once, and how many sets have it.
func (c *Catalog) prior(m Mode) (map[Key]float64, map[Key]float64) {
	q := map[Key]float64{}
	sets := map[Key]float64{}
	var total float64
	for _, e := range c.Ix.Entries {
		if e.Hidden || e.Set.Custom {
			continue
		}
		in := map[Key]bool{}
		for _, l := range e.Lines {
			if c.Why(l.Part) != "" {
				continue
			}
			k := m.key(l.Part, l.Color)
			q[k] += float64(l.Quantity)
			total += float64(l.Quantity)
			if !in[k] {
				in[k] = true
				sets[k]++
			}
		}
	}
	for k := range q {
		q[k] /= total
	}
	return q, sets
}

// Model is a stream's shares, smoothed toward the catalog, in one mode.
type Model struct {
	Mode   Mode
	N      float64
	stream *Stream
	counts map[Key]float64
	prior  map[Key]float64
	sets   map[Key]float64
	// mu_k = e^A q_k^B (L_k/24)^C, and the Gamma's shape alpha_k = e^G0
	// s_k^G1 (how closely the stream follows LEGO's own use of a key: smaller
	// is looser).
	A, B, C, G0, G1 float64
	long            map[Key]float64
	// Keys of the catalog the stream has a piece of, and its pieces among
	// them.
	SeenKeys, Keys int
	SeenPieces     float64
}

// Fit fits the model of a stream in a mode.
func (c *Catalog) Fit(s *Stream, m Mode) *Model {
	md := &Model{Mode: m, N: float64(s.Pieces), stream: s, counts: s.counts(m), long: map[Key]float64{}}
	md.prior, md.sets = c.prior(m)
	md.Keys = len(md.prior)
	for k := range md.prior {
		md.long[k] = math.Log(max(c.size[k.Part].Length, 24) / 24)
	}
	type obs struct{ c, logq, logs, long float64 }
	var os []obs
	var inCatalog float64
	for k, q := range md.prior {
		n := md.counts[k]
		if n > 0 {
			md.SeenKeys++
			md.SeenPieces += n
		}
		inCatalog += n
		os = append(os, obs{n, math.Log(q), math.Log(md.sets[k]), md.long[k]})
	}
	N := md.N
	ll := func(x []float64) float64 {
		a, b, cl, g0, g1 := x[0], x[1], x[2], x[3], x[4]
		var sum float64
		for _, o := range os {
			alpha := math.Exp(g0 + g1*o.logs)
			lambda := N * math.Exp(a+b*o.logq+cl*o.long)
			la, _ := math.Lgamma(alpha)
			lc, _ := math.Lgamma(o.c + alpha)
			sum += lc - la + alpha*math.Log(alpha/(alpha+lambda))
			if o.c > 0 {
				sum += o.c * math.Log(lambda/(alpha+lambda))
			}
		}
		return -sum
	}
	best := nelderMead(ll, []float64{math.Log(inCatalog / N), 1, 0, -1, 0}, []float64{1, 0.3, 0.5, 1, 0.3})
	md.A, md.B, md.C, md.G0, md.G1 = best[0], best[1], best[2], best[3], best[4]
	return md
}

// Alpha is a key's Gamma shape: how closely its share follows LEGO's use of it.
func (md *Model) Alpha(k Key) float64 {
	return math.Exp(md.G0 + md.G1*math.Log(max(md.sets[k], 1)))
}

// mu is a key's expected share before its own count.
func (md *Model) mu(k Key) float64 {
	q, ok := md.prior[k]
	if !ok {
		return 0
	}
	return math.Exp(md.A + md.B*math.Log(q) + md.C*md.long[k])
}

// Share is a key's estimated share of the stream: its posterior mean.
func (md *Model) Share(k Key) float64 { return md.shareFrom(k, md.counts[k], md.N) }

// shareFrom is a key's posterior mean share given c of n pieces.
func (md *Model) shareFrom(k Key, c, n float64) float64 {
	mu := md.mu(k)
	if mu == 0 {
		// Not in any set's counted lines: the stream alone.
		return c / n
	}
	alpha := md.Alpha(k)
	return (c + alpha) / (n + alpha/mu)
}

// Seen is how many pieces of a key the stream has.
func (md *Model) Seen(k Key) float64 { return md.counts[k] }

// nelderMead minimizes f from x0 with initial steps (a plain downhill
// simplex: the fit has three parameters and a smooth likelihood).
func nelderMead(f func([]float64) float64, x0, step []float64) []float64 {
	n := len(x0)
	pts := make([][]float64, n+1)
	vals := make([]float64, n+1)
	for i := range pts {
		p := append([]float64(nil), x0...)
		if i > 0 {
			p[i-1] += step[i-1]
		}
		pts[i], vals[i] = p, f(p)
	}
	for iter := 0; iter < 2000; iter++ {
		// Order: best first.
		for i := 1; i <= n; i++ {
			for j := i; j > 0 && vals[j] < vals[j-1]; j-- {
				pts[j], pts[j-1] = pts[j-1], pts[j]
				vals[j], vals[j-1] = vals[j-1], vals[j]
			}
		}
		if math.Abs(vals[n]-vals[0]) < 1e-9*(1+math.Abs(vals[0])) {
			break
		}
		cen := make([]float64, n)
		for i := 0; i < n; i++ {
			for d := range cen {
				cen[d] += pts[i][d] / float64(n)
			}
		}
		at := func(t float64) []float64 {
			p := make([]float64, n)
			for d := range p {
				p[d] = cen[d] + t*(pts[n][d]-cen[d])
			}
			return p
		}
		r := at(-1)
		fr := f(r)
		switch {
		case fr < vals[0]:
			e := at(-2)
			if fe := f(e); fe < fr {
				pts[n], vals[n] = e, fe
			} else {
				pts[n], vals[n] = r, fr
			}
		case fr < vals[n-1]:
			pts[n], vals[n] = r, fr
		default:
			k := at(0.5)
			if fk := f(k); fk < vals[n] {
				pts[n], vals[n] = k, fk
				continue
			}
			for i := 1; i <= n; i++ {
				for d := range pts[i] {
					pts[i][d] = pts[0][d] + 0.5*(pts[i][d]-pts[0][d])
				}
				vals[i] = f(pts[i])
			}
		}
	}
	return pts[0]
}
