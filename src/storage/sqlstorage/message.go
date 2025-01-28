package sqlstorage

import (
	"encoding/json"
	sq "github.com/Masterminds/squirrel"
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/types"
)

type SqlMessageStore struct {
	*EntityRepository[storage.StoredMessage]
}

type MessageMapper struct {
}

func (f *MessageMapper) ToFields(entity *storage.StoredMessage) map[string]interface{} {
	return map[string]interface{}{
		"id":        entity.Info.ID,
		"jid":       entity.Info.Chat,
		"from_me":   entity.Info.IsFromMe,
		"timestamp": entity.Info.Timestamp,
	}
}
func (f *MessageMapper) Marshal(entity *storage.StoredMessage) ([]byte, error) {
	return json.Marshal(entity)
}

func (f *MessageMapper) Unmarshal(data []byte, entity *storage.StoredMessage) error {
	return json.Unmarshal(data, entity)
}

var _ storage.MessageStore = (*SqlMessageStore)(nil)

func (gc *GContainer) NewMessageStore() *SqlMessageStore {
	repo := NewEntityRepository[storage.StoredMessage](
		gc.db,
		MessageTable,
		&MessageMapper{},
	)
	return &SqlMessageStore{
		repo,
	}
}

func (s SqlMessageStore) UpsertOneMessage(msg *storage.StoredMessage) (err error) {
	return s.UpsertOne(msg)
}

func (s SqlMessageStore) GetAllMessages(filters storage.MessageFilter, pagination storage.Pagination) ([]*storage.StoredMessage, error) {
	conditions := make([]sq.Sqlizer, 0)
	if filters.Jid != nil {
		conditions = append(conditions, sq.Eq{"jid": filters.Jid})
	}
	if filters.TimestampGte != nil {
		conditions = append(conditions, sq.GtOrEq{"timestamp": filters.TimestampGte})
	}
	if filters.TimestampLte != nil {
		conditions = append(conditions, sq.LtOrEq{"timestamp": filters.TimestampLte})
	}
	if filters.FromMe != nil {
		conditions = append(conditions, sq.Eq{"from_me": filters.FromMe})
	}

	sort := []Sort{
		{
			Field: "timestamp",
			Order: SortDesc,
		},
	}
	return s.FilterBy(conditions, sort, pagination)
}

func (s SqlMessageStore) GetChatMessages(jid types.JID, filters storage.MessageFilter, pagination storage.Pagination) ([]*storage.StoredMessage, error) {
	filters.Jid = &jid
	return s.GetAllMessages(filters, pagination)
}

func (s SqlMessageStore) GetMessage(id types.MessageID) (*storage.StoredMessage, error) {
	return s.GetById(id)
}

func (s SqlMessageStore) DeleteChatMessages(jid types.JID) error {
	return s.DeleteBy([]sq.Sqlizer{sq.Eq{"jid": jid}})
}

func (s SqlMessageStore) DeleteMessage(id types.MessageID) error {
	return s.DeleteById(id)
}
