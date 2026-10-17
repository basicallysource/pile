package server

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
)

const (
	// What a search sends when the request says no number.
	setsFound  = 20
	partsFound = 30
)

// kindOrder puts sets LEGO sold first, then builds within them, then custom
// models.
var kindOrder = map[catalog.Kind]int{catalog.SoldSet: 0, catalog.Build: 1, catalog.CustomModel: 2}

// searchedSet is a set, custom model or build as FindSets searches it: the
// words of its name and number, those of a build's set or a custom model's
// designer and collection, and the first word of its number with a digit
// in it ("1234" of 1234-1, and of its builds).
type searchedSet struct {
	set       *catalog.Set
	need      int32
	own, more []string
	number    string
}

// searchedSets is every set, custom model and build with counted lines.
func (s *Service) searchedSets() []searchedSet {
	out := make([]searchedSet, 0, len(s.index.Entries))
	for _, e := range s.index.Entries {
		set := e.Set
		ss := searchedSet{set: set, need: e.Need, own: words(set.Name + " " + set.Num)}
		switch {
		case set.Of != nil:
			ss.more = words(set.Of.Name + " " + set.Of.Num)
		case set.Kind == catalog.CustomModel:
			ss.more = words(set.Designer + " " + set.Collection)
		}
		for _, w := range words(set.Num) {
			if strings.ContainsAny(w, "0123456789") {
				ss.number = w
				break
			}
		}
		out = append(out, ss)
	}
	return out
}

// FindSets is the sets, custom models and builds within sets (each with
// counted lines, so a sort-out queue can take it) whose name or number, or
// whose set's name or number for a build, the query's words start: its
// number first, then whole words, sets before builds before custom models,
// the biggest first.
func (s *Service) FindSets(_ context.Context, req *connect.Request[pilev1.FindSetsRequest]) (*connect.Response[pilev1.FindSetsResponse], error) {
	query := words(req.Msg.Query)
	out := &pilev1.FindSetsResponse{}
	if len(query) == 0 {
		return connect.NewResponse(out), nil
	}
	type hit struct {
		rank int
		*searchedSet
	}
	var hits []hit
	all := s.setWords()
	for i := range all {
		ss := &all[i]
		ok, whole := matches(query, ss.own, ss.more)
		if !ok {
			continue
		}
		rank := 2
		switch {
		case ss.number != "" && slices.Contains(query, ss.number):
			rank = 0
		case whole:
			rank = 1
		}
		hits = append(hits, hit{rank, ss})
	}
	slices.SortFunc(hits, func(a, b hit) int {
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(kindOrder[a.set.Kind], kindOrder[b.set.Kind]), cmp.Compare(b.need, a.need), cmp.Compare(a.set.Num, b.set.Num))
	})
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = setsFound
	}
	for _, h := range hits[:min(len(hits), limit)] {
		f := &pilev1.FoundSet{SetNum: h.set.Num, Name: h.set.Name, Kind: kinds[h.set.Kind], Year: h.set.Year, Theme: h.set.Theme,
			ImageUrl: h.set.ImageURL, Designer: h.set.Designer, Collection: h.set.Collection, Pieces: h.need}
		if h.set.Of != nil {
			f.OfSetNum, f.OfName = h.set.Of.Num, h.set.Of.Name
		}
		out.Sets = append(out.Sets, f)
	}
	return connect.NewResponse(out), nil
}

// searchedPart is a catalog part as FindParts searches it.
type searchedPart struct {
	part  *catalog.Part
	words []string
}

// searchedParts is every catalog part, ready to search, by number.
func (s *Service) searchedParts() []searchedPart {
	out := make([]searchedPart, 0, len(s.cat.Parts))
	for _, p := range s.cat.Parts {
		out = append(out, searchedPart{p, words(p.Name + " " + p.Num)})
	}
	slices.SortFunc(out, func(a, b searchedPart) int { return cmp.Compare(a.part.Num, b.part.Num) })
	return out
}

// FindParts is the catalog's parts whose name or number the query's words
// start, with how many pieces of each the lots have in the colors asked
// for: its number first, then whole words, then the most pieces first.
func (s *Service) FindParts(_ context.Context, req *connect.Request[pilev1.FindPartsRequest]) (*connect.Response[pilev1.FindPartsResponse], error) {
	st, err := s.stacks.get(req.Msg.Lots)
	if err != nil {
		return nil, err
	}
	query := words(req.Msg.Query)
	out := &pilev1.FindPartsResponse{}
	if len(query) == 0 {
		return connect.NewResponse(out), nil
	}
	lots := st.piles.Sum
	type hit struct {
		rank   int
		part   *catalog.Part
		pieces int32
	}
	var hits []hit
	for _, sp := range s.partWords() {
		ok, whole := matches(query, sp.words)
		if !ok {
			continue
		}
		h := hit{rank: 2, part: sp.part}
		switch {
		case strings.ToLower(sp.part.Num) == strings.Join(query, ""):
			h.rank = 0
		case whole:
			h.rank = 1
		}
		if id, ok := s.index.PartID(sp.part.Num); ok {
			if len(req.Msg.Colors) == 0 {
				h.pieces = lots.HasPart(id)
			}
			for _, c := range req.Msg.Colors {
				h.pieces += lots.Has(id, c)
			}
		}
		hits = append(hits, h)
	}
	slices.SortFunc(hits, func(a, b hit) int {
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(b.pieces, a.pieces), cmp.Compare(a.part.Num, b.part.Num))
	})
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = partsFound
	}
	for _, h := range hits[:min(len(hits), limit)] {
		out.Parts = append(out.Parts, &pilev1.PartCount{PartNum: h.part.Num, Name: h.part.Name, Category: h.part.Category, Count: h.pieces, ImageUrl: s.partImage(h.part.Num, req.Msg.Colors)})
	}
	return connect.NewResponse(out), nil
}

// partImage is a picture of a part: in the first of the colors that has
// one, else in any color.
func (s *Service) partImage(num string, colors []int32) string {
	for _, c := range colors {
		if img := s.cat.Images[catalog.PartColor{Part: num, Color: c}]; img != "" {
			return img
		}
	}
	return s.cat.PartImages[num]
}
