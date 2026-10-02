package match

import "container/heap"

const (
	// A pick needs this weighted completeness...
	minPickWeighted = 0.3
	// ...and this much evidence (the rarity-weighted pieces it claims, about
	// three uncommon pieces), so a pick is more than one common brick.
	minPickEvidence = 10
)

// Explain orders the sets LEGO sold that the pile most likely came from:
// it picks one at a time, best evidence first, taking each pick's pieces
// out of a copy of the pile before the next, so a bucket of basic bricks is
// picked once and the themed sets underneath it surface on their own. Which
// pick took which common piece is a guess, so only the order is kept.
//
// Taking pieces out only ever lowers a set's evidence, so a set's last score
// bounds every later one: the candidates wait in a heap by their last score,
// and only the one on top is scored again (lazy greedy). That picks exactly
// what scoring every candidate each round would, and keeps a pile of a
// million pieces, where thousands of sets qualify, fast.
func (ix *Index) Explain(p *Pile, anyColor bool) []*Entry {
	left := p.Clone()
	var h candidates
	for i, e := range ix.Entries {
		if e.Hidden || e.Set.Custom {
			continue
		}
		if m, _ := ix.Score(e, left, anyColor); m.qualifies() {
			h = append(h, candidate{e, m.rank(), i})
		}
	}
	heap.Init(&h)
	var order []*Entry
	for h.Len() > 0 {
		c := heap.Pop(&h).(candidate)
		m, _ := ix.Score(c.entry, left, anyColor)
		if !m.qualifies() {
			continue
		}
		if h.Len() > 0 && m.rank() < h[0].bound {
			c.bound = m.rank()
			heap.Push(&h, c)
			continue
		}
		left.Take(c.entry, anyColor)
		order = append(order, c.entry)
	}
	return order
}

func (m *Match) qualifies() bool {
	return m.Evidence >= minPickEvidence && m.Weighted >= minPickWeighted
}

type candidate struct {
	entry *Entry
	// Its rank when last scored: at least what it is now.
	bound float64
	// Its place in the index, which breaks ties.
	at int
}

type candidates []candidate

func (h candidates) Len() int { return len(h) }
func (h candidates) Less(i, j int) bool {
	if h[i].bound != h[j].bound {
		return h[i].bound > h[j].bound
	}
	return h[i].at < h[j].at
}
func (h candidates) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *candidates) Push(x any)   { *h = append(*h, x.(candidate)) }
func (h *candidates) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}
