package server

import (
	"context"
	"errors"
	"go.mau.fi/whatsmeow/types"

	__ "github.com/devlikeapro/gows/proto"
)

func (s *Server) GetAllLids(ctx context.Context, req *__.GetLidsRequest) (*__.JsonList, error) {
	_, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	return nil, errors.New("not implemented")
}

func (s *Server) GetLidsCount(ctx context.Context, req *__.Session) (*__.OptionalUInt64, error) {
	_, err := s.Sm.Get(req.GetId())
	if err != nil {
		return nil, err
	}
	return nil, errors.New("not implemented")
}

func (s *Server) FindPNByLid(ctx context.Context, req *__.EntityByIdRequest) (*__.OptionalString, error) {
	gows, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	pn, err := types.ParseJID(req.GetId())
	if err != nil {
		return nil, err
	}

	cli := gows.Client
	lid, err := cli.Store.LIDs.GetLIDForPN(ctx, pn)
	if err != nil {
		return nil, err
	}

	return &__.OptionalString{Value: lid.String()}, nil
}

func (s *Server) FindLIDByPhoneNumber(ctx context.Context, req *__.EntityByIdRequest) (*__.OptionalString, error) {
	gows, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	lid, err := types.ParseJID(req.GetId())
	if err != nil {
		return nil, err
	}

	cli := gows.Client
	pn, err := cli.Store.LIDs.GetLIDForPN(ctx, lid)
	if err != nil {
		return nil, err
	}

	return &__.OptionalString{Value: pn.String()}, nil
}
