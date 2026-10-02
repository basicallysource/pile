package collect

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"

	"github.com/basicallysource/pile/internal/match"
)

// Need is what a set asks of a stream in one mode: each key and how many.
type Need struct {
	Keys []Key
	N    []float64
	// Its counted pieces, and the ones left out of them by reason.
	Pieces  float64
	Left    map[string]int32
	Figures int32
}

// Needs is a set's counted pieces as a mode sees them, nil when the catalog
// has no such set.
func (c *Catalog) Needs(num string, m Mode) *Need {
	e := c.Ix.Entry(num)
	if e == nil {
		return nil
	}
	nd := &Need{Left: map[string]int32{}, Figures: e.Minifigures}
	if e.Printed > 0 {
		nd.Left["printed or sticker"] = e.Printed
	}
	if e.FigureParts > 0 {
		nd.Left["minifigure part"] = e.FigureParts
	}
	at := map[Key]int{}
	for _, l := range e.Lines {
		if why := c.Why(l.Part); why != "" {
			nd.Left[why] += l.Quantity
			continue
		}
		k := m.key(l.Part, l.Color)
		i, ok := at[k]
		if !ok {
			i = len(nd.Keys)
			at[k] = i
			nd.Keys = append(nd.Keys, k)
			nd.N = append(nd.N, 0)
		}
		nd.N[i] += float64(l.Quantity)
		nd.Pieces += float64(l.Quantity)
	}
	return nd
}

// completeLog is log of the chance every key has its count after t pieces.
func completeLog(nd *Need, p []float64, t float64) float64 {
	var s float64
	for i, n := range nd.N {
		lp, _ := atLeast(n, p[i]*t)
		s += lp
	}
	return s
}

// found is the expected share of the set's pieces found after t pieces.
func found(nd *Need, p []float64, t float64) float64 {
	var s float64
	for i, n := range nd.N {
		s += expectedMin(n, p[i]*t)
	}
	return s / nd.Pieces
}

// expectedMin is E[min(X, n)] for X ~ Poisson(mu): n less the sum over j
// below n of (n - j) P(X = j), the probabilities built up in logs so a large
// mu does not underflow them.
func expectedMin(n, mu float64) float64 {
	if mu <= 0 {
		return 0
	}
	if mu > n+12*math.Sqrt(n)+40 {
		return n
	}
	lmu := math.Log(mu)
	lp := -mu
	var s float64
	for j := 0.0; j < n; j++ {
		if j > 0 {
			lp += lmu - math.Log(j)
		}
		if j > mu && lp < -40 {
			break
		}
		s += (n - j) * math.Exp(lp)
	}
	return n - s
}

// solve finds t where f (increasing in t) reaches target, by bisection in
// log t, between 1 and 1e15 pieces.
func solve(f func(t float64) float64, target float64) float64 {
	lo, hi := 0.0, math.Log(1e15)
	if f(math.Exp(hi)) < target {
		return math.Inf(1)
	}
	for i := 0; i < 50; i++ {
		mid := (lo + hi) / 2
		if f(math.Exp(mid)) < target {
			lo = mid
		} else {
			hi = mid
		}
	}
	return math.Exp(hi)
}

// share is when the expected share found reaches target.
func share(nd *Need, p []float64, target float64) float64 {
	return solve(func(t float64) float64 { return found(nd, p, t) }, target)
}

// complete is when every piece is found: the median and the 10th and 90th
// percentile.
func complete(nd *Need, p []float64) (median, fast, slow float64) {
	whole := func(t float64) float64 { return completeLog(nd, p, t) }
	return solve(whole, math.Log(0.5)), solve(whole, math.Log(0.1)), solve(whole, math.Log(0.9))
}

// Times is how many pieces a sorter goes through before it has found a share
// of a set, or all of it.
type Times struct {
	// Expected share found: half, nine in ten, 99 in 100.
	Half, Most, Nearly float64
	// Every piece: the median, and the chance range (one run in ten is
	// faster than Fast, one in ten slower than Slow).
	All, Fast, Slow float64
}

// Rare is one of a set's keys and how long it alone takes.
type Rare struct {
	Key  Key
	Need float64
	Seen float64
	// Its estimated share of the stream, and the pieces before its count is
	// expected (Need over share).
	Share, Wait float64
}

// Result is one set in one mode against one model.
type Result struct {
	Set  *match.Entry
	Mode Mode
	Need *Need
	// Distinct keys, and the ones the stream has never had a piece of, and
	// their pieces.
	Keys, Unseen int
	UnseenPieces float64
	At           Times
	// The median pieces to every key the stream has had a piece of: a floor
	// on At.All from the data alone.
	SeenAll float64
	// The median pieces to every piece were the keys never seen ten times
	// rarer than estimated: how much At.All leans on them.
	RarerAll float64
	// From which bulk the sorters happened to get: the 5th and 95th
	// percentile of Most and All over resamplings of the stream's days.
	MostLow, MostHigh, AllLow, AllHigh float64
	// The keys that hold it up the longest, longest first.
	Slowest []Rare
}

// Time works out one set against a model, resampling the stream's days reps
// times for the uncertainty.
func (c *Catalog) Time(md *Model, num string, reps int, seed uint64) (*Result, error) {
	nd := c.Needs(num, md.Mode)
	if nd == nil {
		return nil, fmt.Errorf("no set %s with counted pieces", num)
	}
	r := &Result{Set: c.Ix.Entry(num), Mode: md.Mode, Need: nd, Keys: len(nd.Keys)}
	p := make([]float64, len(nd.Keys))
	rarer := make([]float64, len(nd.Keys))
	seen := &Need{}
	var sp []float64
	for i, k := range nd.Keys {
		p[i] = md.Share(k)
		rarer[i] = p[i]
		if n := md.Seen(k); n > 0 {
			seen.Keys = append(seen.Keys, k)
			seen.N = append(seen.N, nd.N[i])
			seen.Pieces += nd.N[i]
			sp = append(sp, p[i])
		} else {
			r.Unseen++
			r.UnseenPieces += nd.N[i]
			rarer[i] /= 10
		}
		r.Slowest = append(r.Slowest, Rare{Key: k, Need: nd.N[i], Seen: md.Seen(k), Share: p[i], Wait: nd.N[i] / p[i]})
	}
	sort.Slice(r.Slowest, func(i, j int) bool { return r.Slowest[i].Wait > r.Slowest[j].Wait })
	r.Slowest = r.Slowest[:min(len(r.Slowest), 8)]

	r.At.Half, r.At.Most, r.At.Nearly = share(nd, p, 0.5), share(nd, p, 0.9), share(nd, p, 0.99)
	r.At.All, r.At.Fast, r.At.Slow = complete(nd, p)
	r.SeenAll, _, _ = complete(seen, sp)
	r.RarerAll, _, _ = complete(nd, rarer)

	var most, all []float64
	for _, pb := range md.resample(nd.Keys, reps, seed) {
		most = append(most, share(nd, pb, 0.9))
		all = append(all, solve(func(t float64) float64 { return completeLog(nd, pb, t) }, math.Log(0.5)))
	}
	r.MostLow, r.MostHigh = percentile(most, 0.05), percentile(most, 0.95)
	r.AllLow, r.AllHigh = percentile(all, 0.05), percentile(all, 0.95)
	return r, nil
}

// resample draws the stream's days with replacement, reps times, and gives
// the keys' shares in each draw (the model's fit held).
func (md *Model) resample(keys []Key, reps int, seed uint64) [][]float64 {
	at := map[Key]int{}
	for i, k := range keys {
		at[k] = i
	}
	days := md.stream.Days
	counts := make([][]float64, len(days))
	for d, day := range days {
		counts[d] = make([]float64, len(keys))
		for k, n := range day.Counts {
			if k.Color == anyColor && md.Mode != Any {
				continue
			}
			if i, ok := at[md.Mode.key(k.Part, k.Color)]; ok {
				counts[d][i] += float64(n)
			}
		}
	}
	rng := rand.New(rand.NewPCG(seed, uint64(len(keys))))
	var out [][]float64
	for rep := 0; rep < reps; rep++ {
		c := make([]float64, len(keys))
		var n float64
		for range days {
			d := rng.IntN(len(days))
			n += float64(days[d].Pieces)
			for i, x := range counts[d] {
				c[i] += x
			}
		}
		p := make([]float64, len(keys))
		for i, k := range keys {
			p[i] = md.shareFrom(k, c[i], n)
		}
		out = append(out, p)
	}
	return out
}

func percentile(xs []float64, q float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s[min(len(s)-1, int(q*float64(len(s))))]
}

// Simulate runs the sorter piece by piece, runs times, with the model's
// shares, and returns the pieces each run took to find the whole set, in
// order. Only the set's own pieces are drawn one by one: the pieces of other
// keys between them come in a lump (negative binomial, by its normal
// approximation, as the counts are large). It checks the closed form; runs
// expected to need more than maxHits of the set's own pieces are skipped
// (nil).
func (c *Catalog) Simulate(md *Model, num string, runs int, seed uint64, maxHits float64) []float64 {
	nd := c.Needs(num, md.Mode)
	p := make([]float64, len(nd.Keys))
	var ps float64
	for i, k := range nd.Keys {
		p[i] = md.Share(k)
		ps += p[i]
	}
	if all, _, _ := complete(nd, p); all*ps > maxHits {
		return nil
	}
	cum := make([]float64, len(p))
	var acc float64
	for i := range p {
		acc += p[i] / ps
		cum[i] = acc
	}
	rng := rand.New(rand.NewPCG(seed, 7))
	var out []float64
	have := make([]float64, len(p))
	for run := 0; run < runs; run++ {
		clear(have)
		short := len(p)
		var hits float64
		for short > 0 {
			u := rng.Float64() * acc
			i := sort.SearchFloat64s(cum, u)
			if i >= len(p) {
				i = len(p) - 1
			}
			hits++
			have[i]++
			if have[i] == nd.N[i] {
				short--
			}
		}
		mean := hits * (1 - ps) / ps
		sd := math.Sqrt(hits*(1-ps)) / ps
		out = append(out, hits+math.Max(0, mean+sd*rng.NormFloat64()))
	}
	sort.Float64s(out)
	return out
}
