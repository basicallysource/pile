package match

import (
	"slices"
	"sort"
)

// What makes a set basic, the kind a big pile completes without saying much:
// a sculpture of plain bricks, a handful of different parts, or nothing a
// pile would not have anyway. Measured on The Brick Apple (90% plain, 25
// parts), the Monthly Mini Model Builds (9 to 16 parts, common pieces) and
// sets that are plainly models (Police Boat 42% plain, its five rarest
// pieces rarer than 80% of sets'), across Rebrickable's 2026 catalog.
const (
	// Mostly basic: at least this share of its pieces in the plain
	// categories.
	plainShare = 0.85
	// Few parts: this many different parts or fewer.
	fewParts = 10
	// Nothing unusual: its rarest pieces (this many) are on average as
	// common as those of the most common share of sets.
	rarestPieces  = 5
	commonestSets = 0.1
)

// The categories of plain bricks, by Rebrickable's names.
var plainCategories = map[string]bool{"Bricks": true, "Plates": true, "Tiles": true, "Bricks Sloped": true}

// judgePlain sets each entry's MostlyBasic, FewParts and NothingUnusual.
func (ix *Index) judgePlain() {
	rarest := make([]float64, len(ix.Entries))
	var sold []float64
	for i, e := range ix.Entries {
		var plain int32
		var ws []float64
		for _, l := range e.Lines {
			if !DefaultCounting.Counts(l.Class) {
				continue
			}
			if p := ix.cat.Parts[ix.nums[l.Part]]; p != nil && plainCategories[p.Category] {
				plain += l.Quantity
			}
			w := ix.weight(l, false)
			for range min(l.Quantity, rarestPieces) {
				ws = append(ws, w)
			}
		}
		e.MostlyBasic = float64(plain) >= plainShare*float64(max(e.Need, 1))
		e.FewParts = e.DistinctParts <= fewParts
		slices.SortFunc(ws, func(a, b float64) int {
			switch {
			case a > b:
				return -1
			case a < b:
				return 1
			}
			return 0
		})
		ws = ws[:min(len(ws), rarestPieces)]
		var sum float64
		for _, w := range ws {
			sum += w
		}
		rarest[i] = sum / float64(max(len(ws), 1))
		if !e.Hidden && e.Set.Sold() {
			sold = append(sold, rarest[i])
		}
	}
	sort.Float64s(sold)
	if len(sold) == 0 {
		return
	}
	floor := sold[int(commonestSets*float64(len(sold)))]
	for i, e := range ix.Entries {
		e.NothingUnusual = rarest[i] < floor
	}
}
