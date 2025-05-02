package storage

import (
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type Status int

const (
	StatusError       Status = 0
	StatusPending     Status = 1
	StatusServerAck   Status = 2
	StatusDeliveryAck Status = 3
	StatusRead        Status = 4
	StatusPlayed      Status = 5
)

// StoredMessage contains a message and some additional data.
type StoredMessage struct {
	*events.Message
	IsReal bool
	Status *Status
}

type StoredContact struct {
	Jid      types.JID
	Name     string
	PushName string
}

type StoredChat struct {
	Jid                   types.JID
	Name                  string
	ConversationTimestamp time.Time
}

type EphemeralSetting struct {
	Initiator     *waE2E.DisappearingMode_Initiator
	Trigger       *waE2E.DisappearingMode_Trigger
	InitiatedByMe *bool
	Timestamp     *int64
	Expiration    uint32
}

type StoredChatEphemeralSetting struct {
	ID          types.JID
	IsEphemeral bool
	Setting     *EphemeralSetting
}

func NotEphemeral(jid types.JID) *StoredChatEphemeralSetting {
	return &StoredChatEphemeralSetting{
		ID:          jid,
		IsEphemeral: false,
	}
}

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

type MessageFilter struct {
	Jid          *types.JID
	TimestampGte *time.Time
	TimestampLte *time.Time
	FromMe       *bool
	Status       *Status
}
