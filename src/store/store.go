package store

import (
	"go.mau.fi/whatsmeow/types"
)

type MessageStore interface {
	UpsertManyMessages(msg []StoredMessage) error
	UpsertOneMessage(msg StoredMessage) error
	UpdateStatus(id types.MessageID, status Status) error
	GetChatMessages(jid types.JID) ([]StoredMessage, error)
	GetMessage(id types.MessageID) (StoredMessage, error)
	DeleteChatMessages(jid types.JID) error
	DeleteMessage(id types.MessageID) error
}
