package sqlstore

import (
	"database/sql"
	"fmt"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type GContainer struct {
	*sqlstore.Container
	db      *sql.DB
	dialect string
}

func New(dialect, address string, log waLog.Logger) (*GContainer, error) {
	db, err := sql.Open(dialect, address)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	container := sqlstore.NewWithDB(db, dialect, log)
	err = container.Upgrade()
	if err != nil {
		defer container.Close()
		return nil, fmt.Errorf("failed to upgrade database: %w", err)
	}
	gcontainer := &GContainer{container, db, dialect}
	err = gcontainer.Migrate()
	if err != nil {
		defer gcontainer.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	return gcontainer, nil
}
