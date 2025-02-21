package sqlstorage

import (
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/avast/retry-go"
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/types"
)

type SqlMessageStore struct {
	*EntityRepository[storage.StoredMessage]
}

var _ storage.MessageStorage = (*SqlMessageStore)(nil)

func (gc *GContainer) NewMessageStorage() *SqlMessageStore {
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

	sort := []storage.Sort{
		{
			Field: "timestamp",
			Order: storage.SortDesc,
		},
	}
	return s.FilterBy(conditions, sort, pagination)
}

func (s SqlMessageStore) GetChatMessages(jid types.JID, filters storage.MessageFilter, pagination storage.Pagination) ([]*storage.StoredMessage, error) {
	filters.Jid = &jid
	return s.GetAllMessages(filters, pagination)
}

func (s SqlMessageStore) GetMessage(id types.MessageID) (msg *storage.StoredMessage, err error) {
	err = retry.Do(
		func() error {
			msg, err = s.GetById(id)
			if err != nil {
				return err
			}
			return nil
		},
		retry.Attempts(6),
	)
	return msg, err
}

func (s SqlMessageStore) DeleteChatMessages(jid types.JID) error {
	return s.DeleteBy([]sq.Sqlizer{sq.Eq{"jid": jid}})
}

func (s SqlMessageStore) DeleteMessage(id types.MessageID) error {
	return s.DeleteById(id)
}

// getLastMessagesPostgresSubquery generates the subquery for PostgreSQL to fetch the ID of the last message per chat.
func (s SqlMessageStore) getLastMessagesPostgresSubquery() *sq.SelectBuilder {
	query := sq.Select("DISTINCT ON (jid) id").
		From(s.table.Name).
		OrderBy("jid, timestamp DESC")
	return &query
}

// getLastMessagesSQLiteSubquery generates the subquery for SQLite3 to fetch the ID of the last message per chat.
func (s SqlMessageStore) getLastMessagesSQLiteSubquery() *sq.SelectBuilder {
	query := sq.Select("id").
		FromSelect(
			sq.Select("id", "jid", "timestamp", "ROW_NUMBER() OVER (PARTITION BY jid ORDER BY timestamp DESC) as rn").
				From(s.table.Name),
			"sub").
		Where("rn = 1")
	return &query
}

// getLastMessageSubquery selects the appropriate subquery based on the database type.
func (s SqlMessageStore) getLastMessageSubquery() (*sq.SelectBuilder, error) {
	switch s.db.DriverName() {
	case "postgres":
		return s.getLastMessagesPostgresSubquery(), nil
	case "sqlite3":
		return s.getLastMessagesSQLiteSubquery(), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", s.db.DriverName())
	}
}

// GetLastMessagesInChats retrieves the last messages in chats based on sorting and pagination.
func (s SqlMessageStore) GetLastMessagesInChats(sortBy storage.Sort, pagination storage.Pagination) ([]*storage.StoredMessage, error) {
	// Generate the subquery to get the ID of the last message per chat
	subQuery, err := s.getLastMessageSubquery()
	if err != nil {
		return nil, err
	}
	subQueryText, _, err := (*subQuery).ToSql()
	if err != nil {
		return nil, err
	}

	// Main query to get the full details of the last messages
	sql := sq.Select("data").
		From(s.table.Name).
		Where("id IN (" + subQueryText + ")")
	return s.Retrieve(sql, pagination, []storage.Sort{sortBy})
}
