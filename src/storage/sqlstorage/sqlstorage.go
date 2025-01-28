package sqlstorage

import (
	"github.com/devlikeapro/gows/storage"
)

func (gc *GContainer) NewStorage() *storage.Storage {
	return &storage.Storage{
		MessageStorage: gc.NewMessageStorage(),
		ChatStorage:    gc.NewChatStorage(),
	}
}
