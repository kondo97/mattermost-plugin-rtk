package sqlstore

import (
	"context"
	"database/sql"
	"hash/fnv"
	"time"

	"github.com/lib/pq"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

// pgUniqueViolation is the SQLSTATE code for unique_violation in PostgreSQL.
const pgUniqueViolation = "23505"

// StoreAppConfig records or reactivates the app configuration for the given app_id.
// It runs in a single transaction:
//  1. Demote any other currently-active rows to inactive.
//  2. Upsert the row for the given app_id with status='active' (preserving the id
//     of an existing row so downstream references like rtk_channel_meetings.app_config_id
//     stay valid across active/inactive cycles).
//
// On a partial unique index violation (a concurrent transaction won the race for
// status='active'), the active row is re-read; if it points to the same app_id,
// its id is returned, otherwise the error is propagated.
func (s *Store) StoreAppConfig(accountID, appID string) (string, error) {
	now := time.Now().UnixMilli()
	id, err := s.storeAppConfigOnce(accountID, appID, now)
	if err == nil {
		return id, nil
	}
	if !isUniqueViolation(err) {
		return "", err
	}
	// Concurrent transaction beat us. Re-check the current active row.
	activeAppID, activeID, readErr := s.getActiveAppConfig()
	if readErr != nil {
		return "", errors.Wrap(readErr, "failed to read active app config after unique violation")
	}
	if activeAppID == appID {
		return activeID, nil
	}
	return "", errors.Wrapf(err, "another active app config exists for a different app_id (got %q, want %q)", activeAppID, appID)
}

func (s *Store) storeAppConfigOnce(accountID, appID string, now int64) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", errors.Wrap(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	if _, execErr := tx.Exec(
		`UPDATE rtk_app_config SET status = 'inactive', updateat = $1 WHERE status = 'active' AND app_id <> $2`,
		now, appID,
	); execErr != nil {
		return "", errors.Wrap(execErr, "failed to demote previous active app config")
	}

	var id string
	err = tx.QueryRow(
		`INSERT INTO rtk_app_config (id, account_id, app_id, status, createat, updateat)
		 VALUES ($1, $2, $3, 'active', $4, $4)
		 ON CONFLICT (app_id) DO UPDATE
		   SET status = 'active', updateat = EXCLUDED.updateat, account_id = EXCLUDED.account_id
		 RETURNING id`,
		model.NewId(), accountID, appID, now,
	).Scan(&id)
	if err != nil {
		return "", errors.Wrap(err, "failed to upsert app config")
	}

	if err := tx.Commit(); err != nil {
		return "", errors.Wrap(err, "failed to commit app config transaction")
	}
	return id, nil
}

func (s *Store) getActiveAppConfig() (appID string, id string, err error) {
	err = s.db.QueryRow(
		`SELECT app_id, id FROM rtk_app_config WHERE status = 'active' LIMIT 1`,
	).Scan(&appID, &id)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return appID, id, err
}

func (s *Store) GetAppID() (string, error) {
	var appID string
	err := s.db.QueryRow(
		`SELECT app_id FROM rtk_app_config WHERE status = 'active' LIMIT 1`,
	).Scan(&appID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return appID, errors.Wrap(err, "failed to get app ID")
}

func (s *Store) StoreWebhookConfig(appConfigID string, webhookID string) error {
	id := model.NewId()
	_, err := s.db.Exec(
		`INSERT INTO rtk_webhook_config (id, app_config_id, webhook_id, createat) VALUES ($1, $2, $3, $4)`,
		id, appConfigID, webhookID, time.Now().UnixMilli(),
	)
	return errors.Wrap(err, "failed to store webhook config")
}

func (s *Store) ClearWebhookConfig(appConfigID string) error {
	id := model.NewId()
	_, err := s.db.Exec(
		`INSERT INTO rtk_webhook_config (id, app_config_id, webhook_id, createat) VALUES ($1, $2, '', $3)`,
		id, appConfigID, time.Now().UnixMilli(),
	)
	return errors.Wrap(err, "failed to clear webhook config")
}

// GetActiveAppConfigID returns the id of the rtk_app_config row currently marked active.
func (s *Store) GetActiveAppConfigID() (string, error) {
	var id string
	err := s.db.QueryRow(
		`SELECT id FROM rtk_app_config WHERE status = 'active' LIMIT 1`,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, errors.Wrap(err, "failed to get active app config ID")
}

func (s *Store) GetWebhookConfig() (webhookID string, err error) {
	err = s.db.QueryRow(
		`SELECT webhook_id FROM rtk_webhook_config ORDER BY createat DESC LIMIT 1`,
	).Scan(&webhookID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return webhookID, errors.Wrap(err, "failed to get webhook config")
}

// WithAppLock acquires a cluster-wide PostgreSQL advisory lock keyed by a hash of
// `key` and runs `fn` while holding it. The lock is released when fn returns.
//
// pg_advisory_lock is connection-scoped, so a dedicated connection is checked out
// from the pool for the duration of the call. If the connection is forcibly closed
// before unlock, PostgreSQL releases the lock automatically.
func (s *Store) WithAppLock(ctx context.Context, key string, fn func() error) error {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	// All 64 bits of the FNV hash are intentionally preserved as the advisory-lock key.
	// Negative int64 values are valid arguments to pg_advisory_lock(bigint).
	lockKey := int64(h.Sum64()) //nolint:gosec // G115: intentional bit-preserving uint64→int64 cast

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to acquire connection for advisory lock")
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return errors.Wrap(err, "failed to acquire advisory lock")
	}
	defer func() {
		// Best-effort unlock with a fresh background context so cancellation of ctx
		// doesn't prevent us from releasing the lock. If the connection is dead the
		// lock is released automatically when the connection closes.
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey)
	}()

	return fn()
}

// isUniqueViolation reports whether err is a PostgreSQL unique_violation error.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == pgUniqueViolation
	}
	return false
}
