package match

import "sort"

// Match is a set or custom model against a pile.
type Match struct {
	*Entry
	// Pieces found, each line capped at what it needs, and of those the ones
	// in another color (any-color matching only).
	Have, WrongColor int32
	// Have weighted by rarity, over Need weighted the same way.
	Weighted float64
	// The rarity-weighted pieces found: the evidence it is there.
	Evidence float64
	// Its place in Explain's order, from 1; 0 when not picked.
	Pick int
}

// Share is the fraction of its counted pieces found.
func (m *Match) Share() float64 { return float64(m.Have) / float64(max(m.Need, 1)) }

// Complete is every counted piece found.
func (m *Match) Complete() bool { return m.Have >= m.Need }

// Rarity is how rare its pieces are on average: high for a set a common
// pile would not complete by chance.
func (m *Match) Rarity() float64 { return m.Evidence / float64(max(m.Need, 1)) }

func (m *Match) rank() float64 { return m.Evidence * m.Weighted }

// Score matches one entry against the pile.
func (ix *Index) Score(e *Entry, p *Pile, anyColor bool) (*Match, Found) {
	f := p.find(e, anyColor)
	m := &Match{Entry: e}
	var wNeed float64
	for i, l := range e.Lines {
		h := f.Exact[i] + f.Other[i]
		w := ix.weight(l, anyColor)
		m.Have += h
		m.WrongColor += f.Other[i]
		wNeed += float64(l.Quantity) * w
		m.Evidence += float64(h) * w
	}
	if wNeed > 0 {
		m.Weighted = m.Evidence / wNeed
	}
	return m, f
}

// ScoreAll matches every shown entry with a piece in the pile, best
// evidence first.
func (ix *Index) ScoreAll(p *Pile, anyColor bool) []*Match {
	var out []*Match
	for _, e := range ix.Entries {
		if e.Hidden {
			continue
		}
		if m, _ := ix.Score(e, p, anyColor); m.Have > 0 {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rank() > out[j].rank() })
	return out
}
