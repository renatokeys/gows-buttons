package sqlstorage

import (
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/types"
)

func (gc *GContainer) NewChatEphemeralSettingStorage() *SqlChatEphemeralSettingStore {
	repo := NewEntityRepository[storage.StoredChatEphemeralSetting](
		gc.db,
		ChatEphemeralSettingsTable,
		&ChatEphemeralSettingMapper{},
	)
	return &SqlChatEphemeralSettingStore{
		repo,
	}
}

type SqlChatEphemeralSettingStore struct {
	*EntityRepository[storage.StoredChatEphemeralSetting]
}

var _ storage.ChatEphemeralSettingStorage = (*SqlChatEphemeralSettingStore)(nil)

func (s *SqlChatEphemeralSettingStore) GetChatEphemeralSetting(id types.JID) (*storage.StoredChatEphemeralSetting, error) {
	return s.GetById(id.String())
}

func (s *SqlChatEphemeralSettingStore) UpsertChatEphemeralSetting(setting *storage.StoredChatEphemeralSetting) error {
	return s.UpsertOne(setting)
}

func (s *SqlChatEphemeralSettingStore) DeleteChatEphemeralSetting(id types.JID) error {
	return s.DeleteById(id.String())
}
