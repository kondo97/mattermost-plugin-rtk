package sqlstore

import (
	"database/sql"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

func (s *Store) StoreAppConfig(accountID, appID string) (string, error) {
	id := model.NewId()
	_, err := s.db.Exec(
		`INSERT INTO rtk_app_config (id, account_id, app_id, createat) VALUES ($1, $2, $3, $4)`,
		id, accountID, appID, time.Now().UnixMilli(),
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to store app config")
	}
	return id, nil
}

func (s *Store) GetAppID() (string, error) {
	var appID string
	err := s.db.QueryRow(
		`SELECT app_id FROM rtk_app_config ORDER BY createat DESC LIMIT 1`,
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

func (s *Store) GetLatestAppConfigID() (string, error) {
	var id string
	err := s.db.QueryRow(
		`SELECT id FROM rtk_app_config ORDER BY createat DESC LIMIT 1`,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, errors.Wrap(err, "failed to get latest app config ID")
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
