package storage

import (
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
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
	Messages             MessageStorage
	Contacts             ContactStorage
	Chats                ChatStorage
	Groups               GroupStorage
	ChatEphemeralSetting ChatEphemeralSettingStorage
}

type MessageFilter struct {
	Jid          *types.JID
	TimestampGte *time.Time
	TimestampLte *time.Time
	FromMe       *bool
}

type MessageStorage interface {
	UpsertOneMessage(msg *StoredMessage) error
	GetLastMessagesInChats(sortBy Sort, pagination Pagination) ([]*StoredMessage, error)
	GetAllMessages(filters MessageFilter, pagination Pagination) ([]*StoredMessage, error)
	GetChatMessages(jid types.JID, filters MessageFilter, pagination Pagination) ([]*StoredMessage, error)
	GetMessage(id types.MessageID) (*StoredMessage, error)
	DeleteChatMessages(jid types.JID) error
	DeleteMessage(id types.MessageID) error
}

type GroupStorage interface {
	FetchGroups() error
	UpdateGroup(update *events.GroupInfo) error
	UpsertOneGroup(group *types.GroupInfo) error
	GetAllGroups(sort Sort, pagination Pagination) ([]*types.GroupInfo, error)
	GetGroup(jid types.JID) (*types.GroupInfo, error)
	DeleteGroup(jid types.JID) error
	DeleteGroups() error
}

type ContactStorage interface {
	GetContact(user types.JID) (*StoredContact, error)
	GetAllContacts(sortBy Sort, pagination Pagination) ([]*StoredContact, error)
}

type ChatStorage interface {
	GetChats(sortBy Sort, pagination Pagination) ([]*StoredChat, error)
}

type ChatEphemeralSettingStorage interface {
	GetChatEphemeralSetting(id types.JID) (*StoredChatEphemeralSetting, error)
	UpsertChatEphemeralSetting(setting *StoredChatEphemeralSetting) error
	DeleteChatEphemeralSetting(id types.JID) error
}
