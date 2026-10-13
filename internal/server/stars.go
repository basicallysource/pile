package server

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
)

func (s *Service) ListStars(_ context.Context, _ *connect.Request[pilev1.ListStarsRequest]) (*connect.Response[pilev1.ListStarsResponse], error) {
	return connect.NewResponse(&pilev1.ListStarsResponse{SetNums: s.stars.Sets()}), nil
}

// SetStar stars or unstars a set or custom model the catalog has, and keeps
// it in the data folder before it answers.
func (s *Service) SetStar(_ context.Context, req *connect.Request[pilev1.SetStarRequest]) (*connect.Response[pilev1.SetStarResponse], error) {
	if s.cat.Sets[req.Msg.SetNum] == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no set "+req.Msg.SetNum))
	}
	if err := s.stars.Set(req.Msg.SetNum, req.Msg.Starred); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pilev1.SetStarResponse{SetNums: s.stars.Sets()}), nil
}
