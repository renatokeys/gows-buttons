package sqlstorage

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type GContainer struct {
	*sqlstore.Container
	db      *sqlx.DB
	dialect string
}

func New(dialect, address string, log waLog.Logger) (*GContainer, error) {
	db, err := sqlx.Open(dialect, address)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if dialect == "sqlite3" || dialect == "sqlite" {
		err = configureSqlite3(db)
		if err != nil {
			defer db.Close()
			return nil, err
		}
	}

	container := sqlstore.NewWithDB(db.DB, dialect, log)
	err = container.Upgrade()
	if err != nil {
		defer container.Close()
		return nil, fmt.Errorf("failed to upgrade database: %w", err)
	}

	gcontainer := &GContainer{container, db, dialect}
	err = gcontainer.Migrate()
	if err != nil {
		defer container.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	return gcontainer, nil
}

func configureSqlite3(db *sqlx.DB) (err error) {
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return fmt.Errorf("failed to set WAL journal mode: %w", err)
	}
	_, err = db.Exec("PRAGMA synchronous=NORMAL;")
	if err != nil {
		return fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	// Maybe we'll need in the future for sqlite3
	// Error: database is locked
	// Usually _busy_timeout=10000 is enough, but sometimes it's not
	// https://github.com/mattn/go-sqlite3/issues/209
	// DO NOT add cache=shared, it's not safe
	// db.SetMaxOpenConns(1)

	return nil
}
