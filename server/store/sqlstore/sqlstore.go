// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// Store implements kvstore.KVStore using the plugin's underlying SQL database.
type Store struct {
	db         *sqlx.DB
	driverName string
}

// NewStore creates a new Store and ensures all required tables and indices exist.
func NewStore(client *pluginapi.Client) (*Store, error) {
	rawDB, err := client.Store.GetMasterDB()
	if err != nil {
		return nil, fmt.Errorf("sqlstore: get master db: %w", err)
	}

	driverName := client.Store.DriverName()
	s := &Store{
		db:         sqlx.NewDb(rawDB, driverName),
		driverName: driverName,
	}

	if err := s.createTablesIfNotExist(); err != nil {
		return nil, fmt.Errorf("sqlstore: create tables: %w", err)
	}

	return s, nil
}

func (s *Store) isPostgres() bool {
	return s.driverName == "postgres"
}

// placeholder returns the SQL placeholder for position n (1-indexed).
// PostgreSQL uses $1, $2, ...; MySQL uses ?.
func (s *Store) placeholder(n int) string {
	if s.isPostgres() {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

// queryRow executes a row query, re-binding placeholders for the current driver.
func (s *Store) queryRow(query string, args ...any) *sql.Row {
	return s.db.QueryRow(s.db.Rebind(query), args...)
}

// exec executes a statement, re-binding placeholders for the current driver.
func (s *Store) exec(query string, args ...any) (sql.Result, error) {
	return s.db.Exec(s.db.Rebind(query), args...)
}

func (s *Store) createTablesIfNotExist() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS RTK_Calls (
			Id          VARCHAR(26)  NOT NULL,
			ChannelId   VARCHAR(26)  NOT NULL,
			CreatorId   VARCHAR(26)  NOT NULL,
			MeetingId   VARCHAR(255) NOT NULL,
			Participants TEXT        NOT NULL,
			StartAt     BIGINT       NOT NULL,
			EndAt       BIGINT       NOT NULL,
			PostId      VARCHAR(26)  NOT NULL,
			PRIMARY KEY (Id)
		)`,
		`CREATE TABLE IF NOT EXISTS RTK_WebhookConfig (
			Key   VARCHAR(255) NOT NULL,
			Value TEXT         NOT NULL,
			PRIMARY KEY (Key)
		)`,
	}

	for _, q := range tables {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}

	// Indices are best-effort: errors (e.g. "already exists") are intentionally ignored.
	indices := []string{
		`CREATE INDEX IF NOT EXISTS idx_rtk_calls_channel ON RTK_Calls (ChannelId)`,
		`CREATE INDEX IF NOT EXISTS idx_rtk_calls_meeting ON RTK_Calls (MeetingId)`,
	}
	for _, q := range indices {
		s.db.Exec(q) //nolint:errcheck
	}

	return nil
}
