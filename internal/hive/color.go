package hive

import "github.com/basicallysource/pile/internal/records"

// pieceColor is the color pile counts a piece read through the records API
// in: a BrickLink color id, its name ("" when nothing read names it), and who
// gave it (records.Piece.ColorBy).
//
// A person's verdict comes first: the newer of the machine operator's
// correction and the machine owner's newest label on Hive; with neither, the
// newest label anyone else gave. A label that says the color can't be told,
// or gives none, is no verdict. Without a verdict it is the newest answer of
// a color model run over the piece's pictures after the sort (Hive's own
// model among them), and without one of those the machine's own, from while
// it sorted.
func pieceColor(own, ownName, corrected string, correctedAt float64, labels []colorLabel, predictions []colorPrediction) (id, name, by string) {
	verdict, at, found := corrected, correctedAt, corrected != ""
	newest := func(byOwner bool) {
		for _, l := range labels {
			if l.ByOwner != byOwner || l.CantTell || l.Color == nil || *l.Color == "" {
				continue
			}
			if t, _ := seconds(l.At); !found || t >= at {
				verdict, at, found = *l.Color, t, true
			}
		}
	}
	newest(true)
	if !found {
		newest(false)
	}
	if found {
		return verdict, colorName(verdict, own, ownName, predictions), records.CorrectedColor
	}
	var made float64
	best := -1
	for i, p := range predictions {
		// Oldest first, so of two made at once the later is the newer.
		if t, _ := seconds(p.MadeAt); best < 0 || t >= made {
			best, made = i, t
		}
	}
	if best >= 0 {
		p := predictions[best]
		by := p.Model
		if by == "" {
			by = p.Source
		}
		return p.Color, colorName(p.Color, own, ownName, predictions), by
	}
	return own, ownName, ""
}

// colorName is the name of a color id, as the machine or a color model gave
// it.
func colorName(id, own, ownName string, predictions []colorPrediction) string {
	if id == own && ownName != "" {
		return ownName
	}
	for i := len(predictions) - 1; i >= 0; i-- {
		if p := predictions[i]; p.Color == id && p.ColorName != nil {
			return *p.ColorName
		}
	}
	return ""
}
