package server

import (
	"strings"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/match"
)

func (s *Service) lotMessage(t *lot) *pilev1.Lot {
	l := &pilev1.Lot{
		Id:              t.ID,
		Name:            t.Name,
		Description:     t.Description,
		MachineName:     machineName(t.Records.Machines),
		Pieces:          t.pieces,
		PartColors:      int32(len(t.parts)),
		UnmatchedPieces: t.unmatchedPieces(),
		FirstSeenUnix:   t.Records.FirstSeen,
		LastSeenUnix:    t.Records.LastSeen,
		SnapshotUnix:    t.Records.CopiedAt,
		Unlisted:        t.Unlisted,
	}
	for _, c := range t.colors {
		l.Colors = append(l.Colors, &pilev1.ColorShare{Color: color(c.color), Pieces: c.n})
	}
	return l
}

// machineName names the sorters a lot's pieces came from, when they are few
// enough to name.
func machineName(ms []string) string {
	switch len(ms) {
	case 1:
		return ms[0]
	case 2, 3:
		return strings.Join(ms[:len(ms)-1], ", ") + " and " + ms[len(ms)-1]
	}
	return ""
}

func setMatches(ms []*match.Match) []*pilev1.SetMatch {
	out := make([]*pilev1.SetMatch, len(ms))
	for i, m := range ms {
		out[i] = setMatch(m)
	}
	return out
}

func setMatch(m *match.Match) *pilev1.SetMatch {
	s := m.Set
	kind := pilev1.Kind_KIND_SET
	if s.Custom {
		kind = pilev1.Kind_KIND_CUSTOM
	}
	return &pilev1.SetMatch{
		SetNum:               s.Num,
		Name:                 s.Name,
		Year:                 s.Year,
		Theme:                s.Theme,
		ImageUrl:             s.ImageURL,
		Have:                 m.Have,
		Need:                 m.Need,
		WeightedCompleteness: m.Weighted,
		Evidence:             m.Evidence,
		PrintedParts:         m.Printed,
		Minifigures:          m.Minifigures,
		SameContents:         m.SameContents,
		Pick:                 int32(m.Pick),
		MinifigureParts:      m.FigureParts,
		MinifigureSet:        m.MinifigureSet,
		Kind:                 kind,
		Designer:             s.Designer,
		Url:                  s.URL,
		WrongColor:           m.WrongColor,
		ThemeGroup:           s.ThemeGroup,
		Licensed:             s.Licensed,
		Bulk:                 m.Bulk,
		DistinctParts:        m.DistinctParts,
		Rarity:               m.Rarity,
	}
}

func color(c *catalog.Color) *pilev1.Color {
	if c == nil {
		return nil
	}
	return &pilev1.Color{Id: c.ID, Name: c.Name, Rgb: c.RGB, Transparent: c.Transparent, BricklinkId: c.BrickLinkID}
}
