package storage

import (
	"go.mau.fi/whatsmeow/types"
)

type Pagination struct {
	Offset uint64
	Limit  uint64
}

type MessageFilter struct {
	Jid          *types.JID
	TimestampGte *uint64
	TimestampLte *uint64
	FromMe       *bool
}

type MessageStore interface {
	UpsertOneMessage(msg *StoredMessage) error
	GetAllMessages(filters MessageFilter, pagination Pagination) ([]*StoredMessage, error)
	GetChatMessages(jid types.JID, filters MessageFilter, pagination Pagination) ([]*StoredMessage, error)
	GetMessage(id types.MessageID) (*StoredMessage, error)
	DeleteChatMessages(jid types.JID) error
	DeleteMessage(id types.MessageID) error
}

type Storage struct {
	MessageStore MessageStore
}
