package sqlstore

import (
	"time"

	"github.com/pkg/errors"
)

type migration struct {
	version int
	sql     string
}

// migrations lists every schema change in ascending version order.
// Each entry must be idempotent (CREATE … IF NOT EXISTS, etc.) because a
// failed run might leave the migrations table behind without recording the
// version.
var migrations = []migration{
	{
		version: 1,
		sql: `CREATE TABLE IF NOT EXISTS rtk_call_sessions (
			id                VARCHAR(36)  NOT NULL,
			channel_id        VARCHAR(26)  NOT NULL,
			creator_id        VARCHAR(26)  NOT NULL,
			meeting_id        VARCHAR(64)  NOT NULL DEFAULT '',
			participants      TEXT         NOT NULL,
			start_at          BIGINT       NOT NULL DEFAULT 0,
			end_at            BIGINT       NOT NULL DEFAULT 0,
			post_id           VARCHAR(26)  NOT NULL DEFAULT '',
			cleanup_fail_count INT         NOT NULL DEFAULT 0,
			PRIMARY KEY (id)
		)`,
	},
	{
		version: 2,
		sql: `CREATE TABLE IF NOT EXISTS rtk_config (
			config_key   VARCHAR(64) NOT NULL,
			config_value TEXT        NOT NULL DEFAULT '',
			PRIMARY KEY (config_key)
		)`,
	},
	{
		version: 3,
		sql:     `CREATE INDEX IF NOT EXISTS idx_rtk_call_channel ON rtk_call_sessions (channel_id)`,
	},
	{
		version: 4,
		sql:     `CREATE INDEX IF NOT EXISTS idx_rtk_call_meeting ON rtk_call_sessions (meeting_id)`,
	},
}

// RunMigrations ensures the schema is up to date. It creates the migrations
// tracking table if needed and applies any pending migrations in order.
func (s *Store) RunMigrations() error {
	if err := s.ensureMigrationsTable(); err != nil {
		return errors.Wrap(err, "failed to ensure migrations table")
	}

	current, err := s.currentVersion()
	if err != nil {
		return errors.Wrap(err, "failed to get current migration version")
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if _, err := s.db.Exec(m.sql); err != nil {
			return errors.Wrapf(err, "failed to apply migration version %d", m.version)
		}
		if err := s.recordMigration(m.version); err != nil {
			return errors.Wrapf(err, "failed to record migration version %d", m.version)
		}
	}
	return nil
}

func (s *Store) ensureMigrationsTable() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS rtk_schema_migrations (
		version    INT    NOT NULL,
		applied_at BIGINT NOT NULL,
		PRIMARY KEY (version)
	)`)
	return errors.Wrap(err, "failed to create migrations table")
}

func (s *Store) currentVersion() (int, error) {
	var v int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM rtk_schema_migrations`).Scan(&v)
	if err != nil {
		return 0, errors.Wrap(err, "failed to query migration version")
	}
	return v, nil
}

func (s *Store) recordMigration(version int) error {
	_, err := s.db.Exec(
		`INSERT INTO rtk_schema_migrations (version, applied_at) VALUES ($1, $2)`,
		version,
		time.Now().UnixMilli(),
	)
	return errors.Wrap(err, "failed to record migration")
}
