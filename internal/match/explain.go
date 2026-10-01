package match

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
func (ix *Index) Explain(p *Pile, anyColor bool) []*Entry {
	left := p.Clone()
	var pool []*Entry
	for _, e := range ix.Entries {
		if e.Hidden || e.Set.Custom {
			continue
		}
		if m, _ := ix.Score(e, left, anyColor); m.Evidence >= minPickEvidence && m.Weighted >= minPickWeighted {
			pool = append(pool, e)
		}
	}
	var order []*Entry
	for {
		var best *Match
		at := -1
		for i, e := range pool {
			if e == nil {
				continue
			}
			m, _ := ix.Score(e, left, anyColor)
			if m.Evidence < minPickEvidence || m.Weighted < minPickWeighted {
				continue
			}
			if best == nil || m.rank() > best.rank() {
				best, at = m, i
			}
		}
		if best == nil {
			return order
		}
		pool[at] = nil
		left.Take(best.Entry, anyColor)
		order = append(order, best.Entry)
	}
}
