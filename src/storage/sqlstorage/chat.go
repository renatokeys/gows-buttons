package sqlstorage

import (
	"encoding/json"
	sq "github.com/Masterminds/squirrel"
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/types"
)

type SqlChatStorage struct {
	*EntityRepository[storage.StoredChat]
}

type ChatMapper struct {
}

func (f *ChatMapper) ToFields(entity *storage.StoredChat) map[string]interface{} {
	return map[string]interface{}{
		"jid":                    entity.Jid,
		"name":                   entity.Name,
		"conversation_timestamp": entity.ConversationTimestamp,
	}
}
func (f *ChatMapper) Marshal(entity *storage.StoredChat) ([]byte, error) {
	return json.Marshal(entity)
}

func (f *ChatMapper) Unmarshal(data []byte, entity *storage.StoredChat) error {
	return json.Unmarshal(data, entity)
}

var _ storage.ChatStorage = (*SqlChatStorage)(nil)

func (gc *GContainer) NewChatStorage() *SqlChatStorage {
	repo := NewEntityRepository[storage.StoredChat](
		gc.db,
		ChatTable,
		&ChatMapper{},
	)
	return &SqlChatStorage{
		repo,
	}
}

func (s SqlChatStorage) GetChat(jid types.JID) (*storage.StoredChat, error) {
	condition := sq.Eq{"jid": jid.String()}
	return s.GetBy([]sq.Sqlizer{condition})
}

func (s SqlChatStorage) GetChats(sortBy storage.Sort, pagination storage.Pagination) ([]*storage.StoredChat, error) {
	sort := []storage.Sort{
		sortBy,
	}
	return s.FilterBy([]sq.Sqlizer{}, sort, pagination)
}

func (s SqlChatStorage) UpsertChat(chat *storage.StoredChat) error {
	return s.UpsertOne(chat)
}

func (s SqlChatStorage) DeleteChat(jid types.JID) error {
	condition := sq.Eq{"jid": jid.String()}
	return s.DeleteBy([]sq.Sqlizer{condition})
}
