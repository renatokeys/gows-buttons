package server

import (
	"context"
	"encoding/json"
	"fmt"
	__ "github.com/devlikeapro/gows/proto"
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/types"
	"time"
)

func parseTimeS(s uint64) *time.Time {
	seconds := int64(s)
	value := time.Unix(seconds, 0)
	return &value
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
	msg, err := cli.Storage.MessageStorage.GetMessage(id)
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
	pagination := toPagination(req.Pagination)
	filters, err := parseMessageFilters(req.Filters)
	if err != nil {
		return nil, err
	}
	messages, err := cli.Storage.MessageStorage.GetAllMessages(*filters, pagination)
	if err != nil {
		return nil, err
	}
	response, err := toJsonList(messages)
	if err != nil {
		return nil, fmt.Errorf("error marshaling messages: %w", err)
	}
	return response, nil
}

func parseMessageFilters(reqFilters *__.MessageFilters) (*storage.MessageFilter, error) {
	filters := storage.MessageFilter{}
	if reqFilters.Jid != nil {
		jid, err := types.ParseJID(reqFilters.Jid.Value)
		if err != nil {
			return nil, fmt.Errorf("error parsing jid %v: %w", reqFilters.Jid.Value, err)
		}
		filters.Jid = &jid
	}
	if reqFilters.TimestampGte != nil {
		filters.TimestampGte = parseTimeS(reqFilters.TimestampGte.Value)
	}
	if reqFilters.TimestampLte != nil {
		filters.TimestampLte = parseTimeS(reqFilters.TimestampLte.Value)
	}
	if reqFilters.FromMe != nil {
		filters.FromMe = &reqFilters.FromMe.Value
	}
	return &filters, nil
}

func (s *Server) GetContactById(ctx context.Context, req *__.EntityByIdRequest) (*__.Json, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	user, err := types.ParseJID(req.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing jid %v: %w", req.Id, err)
	}

	contact, err := cli.Storage.ContactStorage.GetContact(user)
	if err != nil {
		return nil, fmt.Errorf("error getting contact %v: %w", user, err)
	}
	response, err := toJson(contact)
	if err != nil {
		return nil, fmt.Errorf("error marshaling contact %v: %w", user, err)
	}
	return response, nil
}

func (s *Server) GetContacts(ctx context.Context, req *__.GetContactsRequest) (*__.JsonList, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	pagination := toPagination(req.Pagination)
	sort := toStorageSort(req.SortBy)
	contacts, err := cli.Storage.ContactStorage.GetAllContacts(sort, pagination)
	if err != nil {
		return nil, err
	}
	response, err := toJsonList(contacts)
	if err != nil {
		return nil, fmt.Errorf("error marshaling contacts: %w", err)
	}
	return response, nil
}

func toPagination(pagination *__.Pagination) storage.Pagination {
	return storage.Pagination{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}
}

func toStorageSort(sortBy *__.SortBy) storage.Sort {
	var order storage.SortOrder
	switch sortBy.Order {
	case __.SortBy_ASC:
		order = storage.SortAsc
	case __.SortBy_DESC:
		order = storage.SortDesc
	}

	sort := storage.Sort{
		Field: sortBy.Field,
		Order: order,
	}
	return sort
}
