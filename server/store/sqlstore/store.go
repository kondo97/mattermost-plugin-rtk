package sqlstore

import (
	"database/sql"

	"github.com/pkg/errors"
)

// Store implements store.Store using a PostgreSQL database.
type Store struct {
	db *sql.DB
}

// NewStore creates a new SQL-backed store. Call RunMigrations before use.
func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("db must not be nil")
	}
	return &Store{db: db}, nil
}
