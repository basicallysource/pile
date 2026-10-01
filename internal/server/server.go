// Package server serves the pile over the PileService contract, and the app.
package server

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/gen/pile/v1/pilev1connect"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/match"
)

type Service struct {
	pile *Pile
}

// Handler serves the API, and the app's static build for every other path
// (index.html for the app's own routes).
func Handler(p *Pile, app fs.FS) http.Handler {
	mux := http.NewServeMux()
	path, h := pilev1connect.NewPileServiceHandler(&Service{pile: p})
	mux.Handle(path, h)
	files := http.FileServerFS(app)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" {
			if _, err := fs.Stat(app, name); err != nil {
				r.URL.Path = "/"
			}
		}
		files.ServeHTTP(w, r)
	})
	return mux
}

func (s *Service) GetOverview(_ context.Context, _ *connect.Request[pilev1.GetOverviewRequest]) (*connect.Response[pilev1.GetOverviewResponse], error) {
	p := s.pile
	return connect.NewResponse(&pilev1.GetOverviewResponse{
		Pieces:          p.Pieces,
		Lots:            int32(len(p.Lots)),
		UnmatchedPieces: p.UnmatchedPieces(),
		FirstSeenUnix:   p.Records.FirstSeen,
		LastSeenUnix:    p.Records.LastSeen,
		SnapshotUnix:    p.Records.CopiedAt,
		CatalogUnix:     p.Catalog.DownloadedAt,
		MachineName:     p.Machine,
		SetsRanked:      int32(p.Result.Ranked),
		ExplainedPieces: p.Result.Explains,
		CompleteSets:    int32(p.Result.Complete),
	}), nil
}

func (s *Service) ListLots(_ context.Context, _ *connect.Request[pilev1.ListLotsRequest]) (*connect.Response[pilev1.ListLotsResponse], error) {
	p := s.pile
	out := &pilev1.ListLotsResponse{}
	for _, l := range p.Lots {
		out.Lots = append(out.Lots, &pilev1.Lot{
			PartNum:        l.Part.Num,
			BricklinkId:    l.BrickLinkPart,
			Name:           l.Part.Name,
			Color:          color(l.Color),
			Count:          l.Count,
			ImageUrl:       p.Catalog.Images[catalog.PartColor{Part: l.Part.Num, Color: l.Color.ID}],
			Category:       l.Part.Category,
			MeanConfidence: float32(l.confidence / float64(l.Count)),
		})
	}
	for _, u := range p.Unmatched {
		out.Unmatched = append(out.Unmatched, &pilev1.UnmatchedLot{BricklinkId: u.BrickLinkPart, Name: u.Name, ColorName: u.ColorName, Count: u.Count})
	}
	return connect.NewResponse(out), nil
}

func (s *Service) ListSets(_ context.Context, req *connect.Request[pilev1.ListSetsRequest]) (*connect.Response[pilev1.ListSetsResponse], error) {
	r := s.pile.Result
	list := r.Explained
	if req.Msg.Ranking == pilev1.Ranking_RANKING_ALONE {
		list = r.Alone
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 || limit > len(list) {
		limit = len(list)
	}
	out := &pilev1.ListSetsResponse{}
	for _, m := range list[:limit] {
		out.Sets = append(out.Sets, setMatch(m))
	}
	return connect.NewResponse(out), nil
}

func (s *Service) GetSet(_ context.Context, req *connect.Request[pilev1.GetSetRequest]) (*connect.Response[pilev1.GetSetResponse], error) {
	p := s.pile
	set := p.Catalog.Sets[req.Msg.SetNum]
	if set == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no set "+req.Msg.SetNum))
	}
	r := p.Result
	alone := r.AloneMatch(set.Num)
	if alone == nil {
		alone = &match.Match{Set: set}
	}
	out := &pilev1.GetSetResponse{Alone: setMatch(alone)}
	picked := r.Picked(set.Num)
	if picked != nil {
		out.Explained = setMatch(picked)
	}
	// Lines that share a part (or its variants) and color share the pile's
	// pieces, first come first served.
	inPile := map[catalog.PartColor]int32{}
	claimed := map[catalog.PartColor]int32{}
	for _, l := range set.Lines {
		pc := catalog.PartColor{Part: l.Part, Color: l.Color}
		canon := catalog.PartColor{Part: p.Catalog.Canon(l.Part), Color: l.Color}
		if _, ok := inPile[canon]; !ok {
			inPile[canon] = r.InPile(pc)
			claimed[canon] = picked.Claimed(r, pc)
		}
		part := p.Catalog.Parts[l.Part]
		line := &pilev1.SetLine{PartNum: l.Part, Color: color(p.Catalog.Colors[l.Color]), Need: l.Quantity, ImageUrl: l.ImageURL}
		if part != nil {
			line.Name, line.Printed = part.Name, part.Printed
		}
		if !line.Printed {
			line.InPile = min(inPile[canon], l.Quantity)
			inPile[canon] -= line.InPile
			line.Explained = min(claimed[canon], l.Quantity)
			claimed[canon] -= line.Explained
		}
		out.Lines = append(out.Lines, line)
	}
	for _, f := range set.Minifigures {
		out.Minifigures = append(out.Minifigures, &pilev1.Minifigure{FigNum: f.Num, Name: f.Name, Quantity: f.Quantity, ImageUrl: f.ImageURL})
	}
	return connect.NewResponse(out), nil
}

func setMatch(m *match.Match) *pilev1.SetMatch {
	return &pilev1.SetMatch{
		SetNum:               m.Set.Num,
		Name:                 m.Set.Name,
		Year:                 m.Set.Year,
		Theme:                m.Set.Theme,
		ImageUrl:             m.Set.ImageURL,
		Have:                 m.Have,
		Need:                 m.Need,
		WeightedCompleteness: m.Weighted,
		Evidence:             m.Evidence,
		PrintedParts:         m.Printed,
		Minifigures:          m.Minifigures,
		SameContents:         m.SameContents,
		Pick:                 int32(m.Pick),
	}
}

func color(c *catalog.Color) *pilev1.Color {
	if c == nil {
		return nil
	}
	return &pilev1.Color{Id: c.ID, Name: c.Name, Rgb: c.RGB, Transparent: c.Transparent, BricklinkId: c.BrickLinkID}
}
