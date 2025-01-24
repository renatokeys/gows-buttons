package server

import (
	"context"
	"encoding/json"
	"fmt"
	__ "github.com/devlikeapro/gows/proto"
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/types"
)

func setOptionalValue[T any](src *T, dest **T) {
	if src != nil {
		*dest = src
	}
}

func toJson(data interface{}) (*__.Json, error) {
	d, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &__.Json{Data: string(d)}, nil
}

func toJsonList[T any](data []T) (*__.JsonList, error) {
	list := make([]*__.Json, 0, len(data))
	for _, d := range data {
		j, err := toJson(d)
		if err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	return &__.JsonList{Elements: list}, nil
}

func (s *Server) GetMessageById(ctx context.Context, req *__.EntityByIdRequest) (*__.Json, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	id := req.Id
	msg, err := cli.Storage.MessageStore.GetMessage(id)
	if err != nil {
		return nil, fmt.Errorf("error getting message %v: %w", id, err)
	}
	response, err := toJson(msg)
	if err != nil {
		return nil, fmt.Errorf("error marshaling message %v: %w", id, err)
	}
	return response, nil
}

func (s *Server) GetMessages(ctx context.Context, req *__.GetMessagesRequest) (*__.JsonList, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	pagination := storage.Pagination{
		Limit:  req.Pagination.Limit,
		Offset: req.Pagination.Offset,
	}

	// Filters
	filters := storage.MessageFilter{}
	if req.Filters.Jid != nil {
		jid, err := types.ParseJID(req.Filters.Jid.Value)
		if err != nil {
			return nil, fmt.Errorf("error parsing jid %v: %w", req.Filters.Jid.Value, err)
		}
		filters.Jid = &jid
	}
	if req.Filters.TimestampGte != nil {
		filters.TimestampGte = &req.Filters.TimestampGte.Value
	}
	if req.Filters.TimestampLte != nil {
		filters.TimestampLte = &req.Filters.TimestampLte.Value
	}
	if req.Filters.FromMe != nil {
		filters.FromMe = &req.Filters.FromMe.Value
	}

	messages, err := cli.Storage.MessageStore.GetAllMessages(filters, pagination)
	if err != nil {
		return nil, err
	}
	response, err := toJsonList(messages)
	if err != nil {
		return nil, fmt.Errorf("error marshaling messages: %w", err)
	}
	return response, nil
}
