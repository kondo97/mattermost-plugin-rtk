package sqlstore

import (
	"database/sql"

	"github.com/pkg/errors"
)

const (
	keyWebhookID     = "webhook_id"
	keyWebhookSecret = "webhook_secret"
)

func (s *Store) configGet(key string) (string, error) {
	p1 := s.placeholder(1)
	var value string
	err := s.db.QueryRow(
		`SELECT config_value FROM rtk_config WHERE config_key = `+p1,
		key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", errors.Wrap(err, "failed to get config value")
	}
	return value, nil
}

func (s *Store) configSet(key, value string) error {
	var query string
	if s.isPostgres() {
		query = `INSERT INTO rtk_config (config_key, config_value) VALUES ($1, $2)
			ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value`
	} else {
		query = `INSERT INTO rtk_config (config_key, config_value) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE config_value = VALUES(config_value)`
	}
	_, err := s.db.Exec(query, key, value)
	return errors.Wrap(err, "failed to set config value")
}

func (s *Store) StoreWebhookID(id string) error         { return s.configSet(keyWebhookID, id) }
func (s *Store) GetWebhookID() (string, error)          { return s.configGet(keyWebhookID) }
func (s *Store) StoreWebhookSecret(secret string) error { return s.configSet(keyWebhookSecret, secret) }
func (s *Store) GetWebhookSecret() (string, error)      { return s.configGet(keyWebhookSecret) }
