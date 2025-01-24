package sqlstorage

import "github.com/devlikeapro/gows/storage"

var _ storage.MessageStore = (*SqlMessageStore)(nil)

func (gc *GContainer) NewStorage() *storage.Storage {
	return &storage.Storage{
		MessageStore: gc.NewMessageStore(),
	}
}
