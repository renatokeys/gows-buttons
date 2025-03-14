package gows

import (
	"github.com/devlikeapro/gows/storage"
	meowstorage "github.com/devlikeapro/gows/storage/meow"
	"github.com/devlikeapro/gows/storage/sqlstorage"
	"github.com/devlikeapro/gows/storage/views"
)

func BuildStorage(container *sqlstorage.GContainer, gows *GoWS) *storage.Storage {
	st := &storage.Storage{}
	st.Messages = container.NewMessageStorage()
	st.Groups = container.NewGroupStorage()
	st.ChatEphemeralSetting = container.NewChatEphemeralSettingStorage()
	st.Contacts = meowstorage.NewContactStorage(gows.Store)
	st.Groups = NewGroupCacheStorage(gows, st.Groups, st.ChatEphemeralSetting)
	st.Chats = views.NewChatView(st.Messages, st.Contacts, st.Groups)
	return st
}
