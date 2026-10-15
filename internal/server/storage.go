package server

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/storage"
)

func (s *Service) GetStorage(_ context.Context, _ *connect.Request[pilev1.GetStorageRequest]) (*connect.Response[pilev1.GetStorageResponse], error) {
	return connect.NewResponse(&pilev1.GetStorageResponse{Bins: s.stock().binMessages()}), nil
}

// SetStorage checks the new storage, keeps it in the data folder, and only
// then serves it, with the pieces in its bins.
func (s *Service) SetStorage(_ context.Context, req *connect.Request[pilev1.SetStorageRequest]) (*connect.Response[pilev1.SetStorageResponse], error) {
	st := &storage.Storage{Bins: make([]storage.Bin, 0, len(req.Msg.Bins))}
	for _, b := range req.Msg.Bins {
		st.Bins = append(st.Bins, storage.Bin{
			Address:    strings.TrimSpace(b.Address),
			Name:       strings.TrimSpace(b.Name),
			Lot:        b.LotId,
			Categories: b.Categories,
		})
	}
	if err := st.Check(func(id string) bool { return s.lots[id] != nil }); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.storage.Replace(st); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pilev1.SetStorageResponse{Bins: s.stock().binMessages()}), nil
}

func (s *Service) FindPieces(_ context.Context, req *connect.Request[pilev1.FindPiecesRequest]) (*connect.Response[pilev1.FindPiecesResponse], error) {
	lotName := func(id string) string {
		if t := s.lots[id]; t != nil {
			return t.Name
		}
		return id
	}
	return connect.NewResponse(s.stock().find(req.Msg, lotName)), nil
}
