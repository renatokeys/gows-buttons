package views

import (
	"github.com/devlikeapro/gows/storage"
)

type ChatView struct {
	Messages storage.MessageStorage
	Contacts storage.ContactStorage
}

var _ storage.ChatStorage = (*ChatView)(nil)

func NewChatView(message storage.MessageStorage, contacts storage.ContactStorage) *ChatView {
	return &ChatView{
		Messages: message,
		Contacts: contacts,
	}
}

func (s ChatView) GetChats(sortBy storage.Sort, pagination storage.Pagination) ([]*storage.StoredChat, error) {
	if sortBy.Field == "id" {
		sortBy.Field = "jid"
	}
	messages, err := s.Messages.GetLastMessagesInChats(sortBy, pagination)
	if err != nil {
		return nil, err
	}

	// ignore Name for now, only show Jid and ConversationTimestamp
	chats := make([]*storage.StoredChat, len(messages))
	for i, msg := range messages {
		var name string
		contact, err := s.Contacts.GetContact(msg.Info.Chat)
		switch {
		case err != nil:
			name = ""
		case contact == nil:
			name = ""
		case contact.Name != "":
			name = contact.Name
		case contact.PushName != "":
			name = contact.PushName
		}
		chat := &storage.StoredChat{
			Jid:                   msg.Info.Chat,
			ConversationTimestamp: msg.Info.Timestamp,
			Name:                  name,
		}
		chats[i] = chat
	}
	return chats, nil
}
