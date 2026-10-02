package collect

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestAtLeast(t *testing.T) {
	// P(X >= n) and P(X < n) for X ~ Poisson(mu), from scipy.stats.poisson.
	cases := []struct{ n, mu, p, q float64 }{
		{1, 1, 0.6321205588285577, 0.36787944117144245},
		{5, 3, 0.18473675547622787, 0.8152632445237722},
		{100, 100, 0.5132987982791487, 0.48670120172085135},
		{10, 30, 0.9999928782491372, 7.121750862815593e-06},
		{50, 5, 2.181059214078525e-32, 1},
		{3, 2, 0.32332358381693654, 0.6766764161830634},
	}
	for _, c := range cases {
		lp, q := atLeast(c.n, c.mu)
		if p := math.Exp(lp); math.Abs(p/c.p-1) > 1e-9 {
			t.Errorf("P(X >= %v), mu %v: %v, want %v", c.n, c.mu, p, c.p)
		}
		if math.Abs(q-c.q) > 1e-9*c.q+1e-15 {
			t.Errorf("P(X < %v), mu %v: %v, want %v", c.n, c.mu, q, c.q)
		}
	}
}

func TestExpectedMin(t *testing.T) {
	// E[min(X, n)] is the sum over j from 1 to n of P(X >= j).
	for _, c := range []struct{ n, mu float64 }{{3, 2}, {1, 0.01}, {40, 35}, {200, 260}, {500, 480}} {
		var want float64
		for j := 1.0; j <= c.n; j++ {
			lp, _ := atLeast(j, c.mu)
			want += math.Exp(lp)
		}
		if got := expectedMin(c.n, c.mu); math.Abs(got-want) > 1e-9*want {
			t.Errorf("E[min(X, %v)], mu %v: %v, want %v", c.n, c.mu, got, want)
		}
	}
	if got, want := expectedMin(3, 2), 3-9*math.Exp(-2); math.Abs(got-want) > 1e-12 {
		t.Errorf("E[min(X, 3)], mu 2: %v, want %v", got, want)
	}
}

// The closed form against the draw itself: a small set of three keys in a
// stream where they are 1%, 0.2% and 0.05% of the pieces.
func TestTimesMatchDrawing(t *testing.T) {
	nd := &Need{Keys: make([]Key, 3), N: []float64{4, 2, 1}, Pieces: 7}
	p := []float64{0.01, 0.002, 0.0005}
	all, fast, _ := complete(nd, p)
	r := rand.New(rand.NewPCG(3, 4))
	var runs []float64
	for run := 0; run < 4000; run++ {
		have := make([]float64, 3)
		var drawn float64
		for have[0] < 4 || have[1] < 2 || have[2] < 1 {
			drawn++
			u := r.Float64()
			switch {
			case u < p[0]:
				have[0]++
			case u < p[0]+p[1]:
				have[1]++
			case u < p[0]+p[1]+p[2]:
				have[2]++
			}
		}
		runs = append(runs, drawn)
	}
	if med := percentile(runs, 0.5); math.Abs(med/all-1) > 0.05 {
		t.Errorf("median pieces to the whole set: drawn %v, closed form %v", med, all)
	}
	if f := percentile(runs, 0.1); math.Abs(f/fast-1) > 0.07 {
		t.Errorf("one run in ten: drawn %v, closed form %v", f, fast)
	}
}
