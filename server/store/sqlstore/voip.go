// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"

	"github.com/pkg/errors"
)

// StoreVoIPToken stores (or replaces) a VoIP device token for a user.
func (s *Store) StoreVoIPToken(userID, token string) error {
	var q string
	if s.isPostgres() {
		q = `INSERT INTO RTK_VoIPTokens (UserId, Token) VALUES (?, ?)
			 ON CONFLICT (UserId) DO UPDATE SET Token = EXCLUDED.Token`
	} else {
		q = `INSERT INTO RTK_VoIPTokens (UserId, Token) VALUES (?, ?)
			 ON DUPLICATE KEY UPDATE Token = VALUES(Token)`
	}
	if _, err := s.exec(q, userID, token); err != nil {
		return errors.Wrap(err, "StoreVoIPToken")
	}
	return nil
}

// GetVoIPToken retrieves the VoIP device token for a user, or "" if not set.
func (s *Store) GetVoIPToken(userID string) (string, error) {
	var token string
	err := s.queryRow(`SELECT Token FROM RTK_VoIPTokens WHERE UserId = ?`, userID).Scan(&token)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return token, errors.Wrap(err, "GetVoIPToken")
}
