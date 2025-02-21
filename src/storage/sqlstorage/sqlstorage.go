package sqlstorage

import (
	"github.com/devlikeapro/gows/storage"
)

func (gc *GContainer) NewStorage() *storage.Storage {
	return &storage.Storage{
		Messages: gc.NewMessageStorage(),
		Groups:   gc.NewGroupStorage(),
	}
}
