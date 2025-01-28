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

func (st *GOWSStorage) handleEvent(event interface{}) {
	switch event.(type) {
	case *events.Message:
		msg := event.(*events.Message)
		st.handleMessageEvent(msg)
		st.log.Debugf("Stored message %v(%v)", msg.Info.Chat, msg.Info.ID)
	case *events.Receipt:
		st.handleReceipt(event.(*events.Receipt))
	case *events.HistorySync:
		st.handleHistorySync(event.(*events.HistorySync))
	}
}

func (st *GOWSStorage) handleMessageEvent(event *events.Message) {
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

	err := st.storage.MessageStorage.UpsertOneMessage(&storage.StoredMessage{
		Status:  status,
		Message: event,
	})
	if err != nil {
		st.log.Errorf("Error storing message %v(%v): %v", event.Info.Chat, event.Info.ID, err)
	}
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
		msg, err := st.storage.MessageStorage.GetMessage(id)
		if err != nil {
			st.log.Errorf("Error getting message %v(%v): %v", event.Chat, id, err)
			continue
		}
		if msg.Status >= status {
			continue
		}
		msg.Status = status
		err = st.storage.MessageStorage.UpsertOneMessage(msg)
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
