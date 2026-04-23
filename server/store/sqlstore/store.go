package sqlstore

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Store implements kvstore.KVStore using a SQL database.
type Store struct {
	db         *sql.DB
	driverName string
}

// NewStore creates a new SQL-backed store. Call RunMigrations before use.
func NewStore(db *sql.DB, driverName string) (*Store, error) {
	if db == nil {
		return nil, errors.New("db must not be nil")
	}
	return &Store{db: db, driverName: driverName}, nil
}

// isPostgres reports whether the backing database is PostgreSQL.
func (s *Store) isPostgres() bool {
	return strings.HasPrefix(s.driverName, "postgres")
}

// placeholder returns the SQL positional placeholder for the i-th parameter
// (1-indexed). PostgreSQL uses $1, $2, … while MySQL uses ?.
func (s *Store) placeholder(i int) string {
	if s.isPostgres() {
		return fmt.Sprintf("$%d", i)
	}
	return "?"
}

// GetTemplateData is kept for interface compatibility; not used in production.
func (s *Store) GetTemplateData(_ string) (string, error) {
	return "", nil
}
