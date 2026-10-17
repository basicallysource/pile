package match

import "sort"

// Match is a set or custom model against a pile, counting what a view
// counts.
type Match struct {
	*Entry
	// The pieces counted (this Need, not the entry's), those of them found,
	// each line capped at what it needs, and of those the ones in another
	// color (when the colors matched by take others).
	Need, Have, WrongColor int32
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

// Interest is how interesting it is to find: big, with rare pieces, and
// much of it there (completeness counts twice, so a huge set a quarter there
// does not lead).
func (m *Match) Interest() float64 { return m.Evidence * m.Weighted * m.Weighted }

// rank is the evidence a set is in the pile, which orders the likely sets.
func (m *Match) rank() float64 { return m.Evidence * m.Weighted }

// Score matches one entry against the pile, in the colors cs takes,
// counting the lines c counts (or its set's changes say); Found is for
// every line.
func (ix *Index) Score(e *Entry, p *Pile, cs Colors, c Counting) (*Match, Found) {
	f := p.find(e, cs)
	m := &Match{Entry: e}
	var wNeed float64
	for i, l := range e.Lines {
		if !e.Counts(i, c) {
			continue
		}
		m.Need += l.Quantity
		h := f.Exact[i] + f.Other[i] + f.made(i)
		w := ix.weight(l, cs, e.anyColor(i))
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

// ScoreAll matches every shown entry with a counted piece in the pile, most
// interesting first.
func (ix *Index) ScoreAll(p *Pile, cs Colors, c Counting) []*Match {
	var out []*Match
	for _, e := range ix.Entries {
		if e.Hidden {
			continue
		}
		if m, _ := ix.Score(e, p, cs, c); m.Have > 0 && m.Need > 0 {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Interest() > out[j].Interest() })
	return out
}
