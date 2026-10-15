package hive

import (
	"testing"
	"time"

	"github.com/basicallysource/pile/internal/records"
)

func TestPieceColor(t *testing.T) {
	at := func(unix int64) *string {
		s := time.Unix(unix, 0).UTC().Format(time.RFC3339)
		return &s
	}
	id := func(s string) *string { return &s }
	owner := func(color string, when int64) colorLabel {
		return colorLabel{Color: id(color), ByOwner: true, At: at(when)}
	}
	other := func(color string, when int64) colorLabel {
		return colorLabel{Color: id(color), At: at(when)}
	}
	model := func(color, name string, made int64) colorPrediction {
		return colorPrediction{Source: "rerun", Model: name, Color: color, ColorName: id("Color " + color), MadeAt: at(made)}
	}
	cantTell := colorLabel{CantTell: true, ByOwner: true, At: at(900)}
	noColor := colorLabel{ByOwner: true, At: at(900)}
	predictions := []colorPrediction{model("85", "colormodel-v2", 300), model("86", "colormodel-v1", 100)}

	for _, c := range []struct {
		name        string
		corrected   string
		correctedAt float64
		labels      []colorLabel
		predictions []colorPrediction
		id, by      string
	}{
		{"the operator's correction", "5", 200, nil, predictions, "5", records.CorrectedColor},
		{"the owner's label newer than the correction", "5", 200, []colorLabel{owner("1", 100), owner("3", 300)}, predictions, "3", records.CorrectedColor},
		{"the correction newer than the owner's labels", "5", 400, []colorLabel{owner("3", 300)}, predictions, "5", records.CorrectedColor},
		{"the owner's verdict over a newer label by someone else", "", 0, []colorLabel{owner("3", 300), other("4", 500)}, predictions, "3", records.CorrectedColor},
		{"the correction over a newer label by someone else", "5", 200, []colorLabel{other("4", 500)}, predictions, "5", records.CorrectedColor},
		{"someone else's newest label, with no verdict of the owner's", "", 0, []colorLabel{other("4", 500), other("2", 600), cantTell, noColor}, predictions, "2", records.CorrectedColor},
		{"can't tell and no color are no verdict", "", 0, []colorLabel{cantTell, noColor}, predictions, "85", "colormodel-v2"},
		{"the newest model answer by when it was made", "", 0, nil, predictions, "85", "colormodel-v2"},
		{"the machine's own", "", 0, nil, nil, "11", ""},
	} {
		got, _, by := pieceColor("11", "Black", c.corrected, c.correctedAt, c.labels, c.predictions)
		if got != c.id || by != c.by {
			t.Errorf("%s: %s by %q, want %s by %q", c.name, got, by, c.id, c.by)
		}
	}
}
