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

func (s *SqlChatEphemeralSettingStore) UpdateChatEphemeralSetting(setting *storage.StoredChatEphemeralSetting) error {
	if setting.IsEphemeral {
		return s.UpsertOne(setting)
	} else {
		return s.DeleteChatEphemeralSetting(setting.ID)
	}
}

func (s *SqlChatEphemeralSettingStore) DeleteChatEphemeralSetting(id types.JID) error {
	return s.DeleteById(id.String())
}
