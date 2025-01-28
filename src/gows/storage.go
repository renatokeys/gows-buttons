package gows

import (
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type GOWSStorage struct {
	gows    *GoWS
	log     waLog.Logger
	storage *storage.Storage
}

func shouldStoreMessage(event *events.Message) bool {
	if event.Message == nil {
		return false
	}
	msg := event.Message
	switch {
	case msg.Conversation != nil:
		return true
	case msg.ExtendedTextMessage != nil:
		return true
	case msg.ImageMessage != nil:
		return true
	case msg.ContactMessage != nil:
		return true
	case msg.LocationMessage != nil:
		return true
	case msg.VideoMessage != nil:
		return true
	case msg.AudioMessage != nil:
		return true
	case msg.DocumentMessage != nil:
		return true
	case msg.StickerMessage != nil:
		return true
	case msg.ContactsArrayMessage != nil:
		return true
	case msg.TemplateMessage != nil:
		return true
	case msg.ListMessage != nil:
		return true
	case msg.RichResponseMessage != nil:
		return true
	case msg.PollCreationMessage != nil:
		return true
	case msg.PollCreationMessageV2 != nil:
		return true
	case msg.PollCreationMessageV3 != nil:
		return true
	case msg.PollCreationMessageV4 != nil:
		return true
	case msg.PollCreationMessageV5 != nil:
		return true
	default:
		return false
	}
}

func (st *GOWSStorage) handleEvent(event interface{}) {
	switch event.(type) {
	case *events.Message:
		msg := event.(*events.Message)
		saved := st.handleMessageEvent(msg)
		if saved {
			st.log.Debugf("Stored message %v(%v)", msg.Info.Chat, msg.Info.ID)
		}
	case *events.Receipt:
		st.handleReceipt(event.(*events.Receipt))
	case *events.HistorySync:
		st.handleHistorySync(event.(*events.HistorySync))
	}
}

func (st *GOWSStorage) handleMessageEvent(event *events.Message) bool {
	if !shouldStoreMessage(event) {
		return false
	}
	var status storage.Status
	if event.SourceWebMsg != nil && event.SourceWebMsg.Status != nil {
		status = storage.Status(*event.SourceWebMsg.Status)
	} else {
		if event.Info.IsFromMe {
			status = storage.StatusServerAck
		} else {
			status = storage.StatusDeliveryAck
		}
	}

	err := st.storage.Messages.UpsertOneMessage(&storage.StoredMessage{
		Status:  status,
		Message: event,
	})
	if err != nil {
		st.log.Errorf("Error storing message %v(%v): %v", event.Info.Chat, event.Info.ID, err)
	}
	return true
}

func (st *GOWSStorage) handleReceipt(event *events.Receipt) {
	var status storage.Status
	switch event.Type {
	case types.ReceiptTypeDelivered:
		status = storage.StatusDeliveryAck
	case types.ReceiptTypeRead:
		status = storage.StatusRead
	case types.ReceiptTypePlayed:
		status = storage.StatusPlayed
	}
	for _, id := range event.MessageIDs {
		msg, err := st.storage.Messages.GetMessage(id)
		if err != nil {
			st.log.Errorf("Error getting message %v(%v): %v", event.Chat, id, err)
			continue
		}
		if msg.Status >= status {
			continue
		}
		msg.Status = status
		err = st.storage.Messages.UpsertOneMessage(msg)
		if err != nil {
			st.log.Errorf("Error updating status for message %v(%v): %v", event.Chat, id, err)
			continue
		}
	}
	st.log.Debugf("Updated status for %v messages in %v to %v", len(event.MessageIDs), event.Chat, status)
}

func (st *GOWSStorage) handleHistorySync(event *events.HistorySync) {
	for _, conv := range event.Data.Conversations {
		chatJID, err := types.ParseJID(conv.GetId())
		if err != nil {
			st.log.Errorf("Error parsing JID: %v", err)
			continue
		}
		go st.saveHistoryForOneChat(conv, chatJID)
	}
	st.log.Debugf("Saved history for %v chats", len(event.Data.Conversations))
}

func (st *GOWSStorage) saveHistoryForOneChat(conv *waHistorySync.Conversation, chatJID types.JID) {
	for _, historyMsg := range conv.GetMessages() {
		evt, err := st.gows.ParseWebMessage(chatJID, historyMsg.GetMessage())
		if err != nil {
			st.log.Errorf("Error parsing message: %v", err)
			continue
		}
		st.handleMessageEvent(evt)
	}
	st.log.Debugf("Saved %v messages in %v", len(conv.GetMessages()), chatJID)
}
