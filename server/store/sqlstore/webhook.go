// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"

	"github.com/pkg/errors"
)

const (
	keyWebhookID     = "webhook:id"
	keyWebhookSecret = "webhook:secret"
)

// StoreWebhookID persists the registered RTK webhook ID.
func (s *Store) StoreWebhookID(id string) error {
	return s.upsertConfig(keyWebhookID, id)
}

// GetWebhookID retrieves the registered RTK webhook ID, or "" if not set.
func (s *Store) GetWebhookID() (string, error) {
	return s.getConfig(keyWebhookID)
}

// StoreWebhookSecret persists the RTK webhook signing secret.
func (s *Store) StoreWebhookSecret(secret string) error {
	return s.upsertConfig(keyWebhookSecret, secret)
}

// GetWebhookSecret retrieves the RTK webhook signing secret, or "" if not set.
func (s *Store) GetWebhookSecret() (string, error) {
	return s.getConfig(keyWebhookSecret)
}

func (s *Store) upsertConfig(key, value string) error {
	var q string
	if s.isPostgres() {
		q = `INSERT INTO RTK_WebhookConfig (Key, Value) VALUES (?, ?)
			 ON CONFLICT (Key) DO UPDATE SET Value = EXCLUDED.Value`
	} else {
		q = `INSERT INTO RTK_WebhookConfig (Key, Value) VALUES (?, ?)
			 ON DUPLICATE KEY UPDATE Value = VALUES(Value)`
	}
	if _, err := s.exec(q, key, value); err != nil {
		return errors.Wrapf(err, "upsertConfig(%q)", key)
	}
	return nil
}

func (s *Store) getConfig(key string) (string, error) {
	var value string
	err := s.queryRow(`SELECT Value FROM RTK_WebhookConfig WHERE Key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, errors.Wrapf(err, "getConfig(%q)", key)
}
