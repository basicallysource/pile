package server

import (
	"strings"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/match"
)

func (s *Service) lotMessage(t *lot) *pilev1.Lot {
	return &pilev1.Lot{
		Id:           t.ID,
		Name:         t.Name,
		Description:  t.Description,
		SnapshotUnix: t.Records.CopiedAt,
		Unlisted:     t.Unlisted,
		Totals:       s.stacks.one[t.ID].totals,
	}
}

// machineName names the sorters pieces came from, when they are few enough
// to name.
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
	out := &pilev1.SetMatch{
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
		Kind:                 kinds[s.Kind],
		Designer:             s.Designer,
		Url:                  s.URL,
		WrongColor:           m.WrongColor,
		ThemeGroup:           s.ThemeGroup,
		Licensed:             s.Licensed,
		Bulk:                 m.Bulk,
		DistinctParts:        m.DistinctParts,
		Rarity:               m.Rarity,
		MostlyBasic:          m.MostlyBasic,
		FewParts:             m.FewParts,
		NothingUnusual:       m.NothingUnusual,
		Accessories:          m.Accessories,
		Collection:           s.Collection,
		Copies:               s.Copies,
	}
	if s.Of != nil {
		out.OfSetNum, out.OfName = s.Of.Num, s.Of.Name
	}
	if src := s.Source; src != nil {
		out.SourceAuthor, out.SourceUrl, out.SourceNotes = src.Author, src.URL, src.Notes
	}
	return out
}

var kinds = map[catalog.Kind]pilev1.Kind{
	catalog.SoldSet:     pilev1.Kind_KIND_SET,
	catalog.CustomModel: pilev1.Kind_KIND_CUSTOM,
	catalog.Build:       pilev1.Kind_KIND_BUILD,
}

var partClass = map[catalog.Class]pilev1.PartClass{
	catalog.Plain:      pilev1.PartClass_PART_CLASS_PLAIN,
	catalog.Printed:    pilev1.PartClass_PART_CLASS_PRINTED,
	catalog.FigurePart: pilev1.PartClass_PART_CLASS_FIGURE,
	catalog.Accessory:  pilev1.PartClass_PART_CLASS_ACCESSORY,
}

func color(c *catalog.Color) *pilev1.Color {
	if c == nil {
		return nil
	}
	return &pilev1.Color{Id: c.ID, Name: c.Name, Rgb: c.RGB, Transparent: c.Transparent, BricklinkId: c.BrickLinkID}
}
