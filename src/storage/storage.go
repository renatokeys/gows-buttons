package storage

import (
	"go.mau.fi/whatsmeow/types"
	"time"
)

type SortOrder string

const (
	SortAsc  SortOrder = "ASC"
	SortDesc SortOrder = "DESC"
)

type Sort struct {
	Field string
	Order SortOrder
}

type Pagination struct {
	Offset uint64
	Limit  uint64
}

type Storage struct {
	MessageStorage MessageStorage
	ContactStorage ContactStorage
}

type MessageFilter struct {
	Jid          *types.JID
	TimestampGte *time.Time
	TimestampLte *time.Time
	FromMe       *bool
}

type MessageStorage interface {
	UpsertOneMessage(msg *StoredMessage) error
	GetAllMessages(filters MessageFilter, pagination Pagination) ([]*StoredMessage, error)
	GetChatMessages(jid types.JID, filters MessageFilter, pagination Pagination) ([]*StoredMessage, error)
	GetMessage(id types.MessageID) (*StoredMessage, error)
	DeleteChatMessages(jid types.JID) error
	DeleteMessage(id types.MessageID) error
}

type ContactStorage interface {
	GetContact(user types.JID) (*StoredContact, error)
	GetAllContacts(sortBy Sort, pagination Pagination) ([]*StoredContact, error)
}
