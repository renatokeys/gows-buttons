package gows

import (
	"runtime/debug"

	"github.com/avast/retry-go"
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type StorageEventHandler struct {
	gows    *GoWS
	log     waLog.Logger
	storage *storage.Storage
}

func (st *StorageEventHandler) GetMessageForRetry(requester, to types.JID, id types.MessageID) *waE2E.Message {
	msg, err := st.storage.Messages.GetMessage(id)
	if err != nil {
		st.log.Errorf("Error getting message for retry - requester %v, to %v, id %v: %v", requester, to, id, err)
		return nil
	}
	return msg.Message.RawMessage
}

func isRealMessage(event *events.Message) bool {
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

func (st *StorageEventHandler) handleEvent(event interface{}) {
	// Handle all panic and log error + stack
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			st.log.Errorf("Panic happened when handling event: %v. Stack: %s. Event: %v", err, stack, event)
		}
	}()

	switch event.(type) {
	case *events.Message:
		msg := event.(*events.Message)
		var status storage.Status
		if msg.Info.IsFromMe {
			status = storage.StatusServerAck
		} else {
			status = storage.StatusDeliveryAck
		}
		st.handleSaveMessage(msg, &status)
		st.handleMessageEvent(msg)
	case *events.Receipt:
		st.handleReceipt(event.(*events.Receipt))
	case *events.HistorySync:
		st.handleHistorySync(event.(*events.HistorySync))
	// Groups
	case *events.JoinedGroup:
		st.handleMeJoinedGroup(event.(*events.JoinedGroup))
	case *events.GroupInfo:
		left := st.handleMeLeftGroup(event.(*events.GroupInfo))
		if left {
			return
		}
		st.handleGroupInfo(event.(*events.GroupInfo))
	case *events.DeleteChat:
		st.handleDeleteChat(event.(*events.DeleteChat))
	}
}

func (st *StorageEventHandler) handleSaveMessage(event *events.Message, status *storage.Status) {
	messageToStore := &storage.StoredMessage{
		Message: event,
		Status:  status,
		IsReal:  isRealMessage(event),
	}

	err := st.storage.Messages.UpsertOneMessage(messageToStore)
	if err != nil {
		st.log.Errorf("Error storing message %v(%v): %v", event.Info.Chat, event.Info.ID, err)
	}
}

func (st *StorageEventHandler) handleMessageEvent(event *events.Message) {
	// Revoked message
	isRevoked := event.Message.ProtocolMessage != nil && *event.Message.ProtocolMessage.Type == waE2E.ProtocolMessage_REVOKE
	if isRevoked {
		err := st.storage.Messages.DeleteMessage(*event.Message.ProtocolMessage.Key.ID)
		if err != nil {
			st.log.Errorf("Error deleting message %v: %v", *event.Message.ProtocolMessage.Key.ID, err)
		}
		return
	}

	// Chat ephemeral settings - changed
	isProtocolMessage := event.Message != nil && event.Message.ProtocolMessage != nil
	if isProtocolMessage {
		setting := ExtractEphemeralSettingsFromProtocolMessage(event.Info, event.Message.ProtocolMessage)
		if setting != nil {
			err := st.storage.ChatEphemeralSetting.UpdateChatEphemeralSetting(setting)
			if err != nil {
				st.log.Errorf("Error updating chat ephemeral setting %v: %v", setting.ID, err)
			}
			st.log.Debugf("Changed chat ephemeral setting %v (enabled: %v)", setting.ID, setting.IsEphemeral)
			return
		}
	}

	// Chat ephemeral settings - from message
	setting := ExtractEphemeralSettingsFromMsg(event)
	if setting != nil {
		err := st.storage.ChatEphemeralSetting.UpdateChatEphemeralSetting(setting)
		if err != nil {
			st.log.Errorf("Error updating chat ephemeral setting %v: %v", setting.ID, err)
		}
		st.log.Debugf("Initial chat ephemeral setting %v (enabled: %v)", setting.ID, setting.IsEphemeral)
		// Do not return - we still need to handle the message
		// return
	}
}

func (st *StorageEventHandler) handleReceipt(event *events.Receipt) {
	var status storage.Status
	switch event.Type {
	case types.ReceiptTypeDelivered:
		status = storage.StatusDeliveryAck
	case types.ReceiptTypeRead:
		status = storage.StatusRead
	case types.ReceiptTypePlayed:
		status = storage.StatusPlayed
	default:
		st.log.Debugf("Unknown receipt type: %v", event.Type)
		return
	}
	for _, id := range event.MessageIDs {
		st.log.Debugf("Updating status for message %v(%v) to %v (receipt type: '%v')", event.Chat, id, status, event.Type.GoString())
		msg, err := st.storage.Messages.GetMessage(id)
		if err != nil {
			st.log.Errorf("Error getting message %v(%v): %v", event.Chat, id, err)
			continue
		}
		if msg.Status != nil && *msg.Status >= status {
			continue
		}
		msg.Status = &status
		err = st.storage.Messages.UpsertOneMessage(msg)
		if err != nil {
			st.log.Errorf("Error updating status for message %v(%v): %v", event.Chat, id, err)
			continue
		}
		st.log.Debugf("Updated status for message %v(%v) to %v", event.Chat, id, status)
	}
}

func (st *StorageEventHandler) handleHistorySync(event *events.HistorySync) {
	for _, conv := range event.Data.Conversations {
		jid, err := types.ParseJID(conv.GetId())
		if err != nil {
			st.log.Errorf("Error parsing JID: %v", err)
			continue
		}
		go st.saveHistoryForOneChat(conv, jid)
	}
	st.log.Debugf("Saved history for %v chats", len(event.Data.Conversations))
}

func (st *StorageEventHandler) saveHistoryForOneChat(conv *waHistorySync.Conversation, chatJID types.JID) {
	historyMessages := conv.GetMessages()
	for _, historyMsg := range historyMessages {
		message := historyMsg.GetMessage()
		msg, err := st.gows.ParseWebMessage(chatJID, message)
		if err != nil {
			st.log.Errorf("Error parsing message: %v", err)
			continue
		}

		var status storage.Status
		if msg.SourceWebMsg != nil && msg.SourceWebMsg.Status != nil {
			status = storage.Status(*msg.SourceWebMsg.Status)
		}

		st.handleSaveMessage(msg, &status)
	}
	st.log.Debugf("Saved %v messages in %v", len(conv.GetMessages()), chatJID)

	setting := ExtractEphemeralSettingsFromConversation(conv, chatJID)
	if setting != nil {
		err := st.storage.ChatEphemeralSetting.UpdateChatEphemeralSetting(setting)
		if err != nil {
			st.log.Errorf("Error updating chat ephemeral setting %v: %v", setting.ID, err)
		}
		st.log.Debugf("Initial chat ephemeral setting %v (enabled: %v)", setting.ID, setting.IsEphemeral)
	}
}

func (st *StorageEventHandler) handleMeJoinedGroup(group *events.JoinedGroup) {
	err := st.storage.Groups.UpsertOneGroup(&group.GroupInfo)
	if err != nil {
		st.log.Errorf("Error storing group %v: %v", group.JID, err)
	}
	st.log.Debugf("I joined group %v", group.JID)
}

func (st *StorageEventHandler) handleMeLeftGroup(info *events.GroupInfo) bool {
	jid := st.gows.Store.ID
	for _, leave := range info.Leave {
		if leave == jid.ToNonAD() {
			st.log.Debugf("I left group %v", info.JID)
			err := st.storage.Groups.DeleteGroup(info.JID)
			if err != nil {
				st.log.Errorf("Error deleting group %v: %v", info.JID, err)
			}
			return true
		}
	}
	return false
}

func (st *StorageEventHandler) handleGroupInfo(info *events.GroupInfo) {
	err := retry.Do(func() error {
		return st.storage.Groups.UpdateGroup(info)
	})
	if err != nil {
		st.log.Errorf("Error updating group %v: %v", info.JID, err)
	}
	return
}

func (st *StorageEventHandler) handleDeleteChat(event *events.DeleteChat) {
	err := st.storage.Messages.DeleteChatMessages(event.JID, event.Timestamp)
	if err != nil {
		st.log.Errorf("Error deleting chat messages %v: %v", event.JID, err)
	}
	err = st.storage.ChatEphemeralSetting.DeleteChatEphemeralSetting(event.JID, event.Timestamp)
	if err != nil {
		st.log.Errorf("Error deleting chat ephemeral setting %v: %v", event.JID, err)
	}
	st.log.Debugf("Deleted chat %v", event.JID)
}
