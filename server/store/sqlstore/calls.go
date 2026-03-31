// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
	"github.com/pkg/errors"
)

// GetCallByChannel returns the active call for a channel, or nil if none is active.
func (s *Store) GetCallByChannel(channelID string) (*kvstore.CallSession, error) {
	row := s.queryRow(
		`SELECT Id, ChannelId, CreatorId, MeetingId, Participants, StartAt, EndAt, PostId
		   FROM RTK_Calls
		  WHERE ChannelId = ? AND EndAt = 0
		  LIMIT 1`,
		channelID,
	)
	session, err := scanCallSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return session, errors.Wrap(err, "GetCallByChannel")
}

// GetCallByID returns the call with the given ID, or nil if not found.
func (s *Store) GetCallByID(callID string) (*kvstore.CallSession, error) {
	row := s.queryRow(
		`SELECT Id, ChannelId, CreatorId, MeetingId, Participants, StartAt, EndAt, PostId
		   FROM RTK_Calls
		  WHERE Id = ?`,
		callID,
	)
	session, err := scanCallSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return session, errors.Wrap(err, "GetCallByID")
}

// GetCallByMeetingID returns the call with the given RTK meeting ID, or nil if not found.
func (s *Store) GetCallByMeetingID(meetingID string) (*kvstore.CallSession, error) {
	row := s.queryRow(
		`SELECT Id, ChannelId, CreatorId, MeetingId, Participants, StartAt, EndAt, PostId
		   FROM RTK_Calls
		  WHERE MeetingId = ?
		  LIMIT 1`,
		meetingID,
	)
	session, err := scanCallSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return session, errors.Wrap(err, "GetCallByMeetingID")
}

// SaveCall inserts or updates a call session.
func (s *Store) SaveCall(session *kvstore.CallSession) error {
	if session == nil {
		return errors.New("session must not be nil")
	}
	if session.ID == "" {
		return errors.New("session.ID must not be empty")
	}
	if session.ChannelID == "" {
		return errors.New("session.ChannelID must not be empty")
	}

	participants, err := json.Marshal(session.Participants)
	if err != nil {
		return errors.Wrap(err, "SaveCall: marshal participants")
	}

	var q string
	if s.isPostgres() {
		q = `INSERT INTO RTK_Calls (Id, ChannelId, CreatorId, MeetingId, Participants, StartAt, EndAt, PostId)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (Id) DO UPDATE SET
			   ChannelId    = EXCLUDED.ChannelId,
			   CreatorId    = EXCLUDED.CreatorId,
			   MeetingId    = EXCLUDED.MeetingId,
			   Participants = EXCLUDED.Participants,
			   StartAt      = EXCLUDED.StartAt,
			   EndAt        = EXCLUDED.EndAt,
			   PostId       = EXCLUDED.PostId`
	} else {
		q = `INSERT INTO RTK_Calls (Id, ChannelId, CreatorId, MeetingId, Participants, StartAt, EndAt, PostId)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
			   ChannelId    = VALUES(ChannelId),
			   CreatorId    = VALUES(CreatorId),
			   MeetingId    = VALUES(MeetingId),
			   Participants = VALUES(Participants),
			   StartAt      = VALUES(StartAt),
			   EndAt        = VALUES(EndAt),
			   PostId       = VALUES(PostId)`
	}

	if _, err := s.exec(q,
		session.ID, session.ChannelID, session.CreatorID, session.MeetingID,
		string(participants), session.StartAt, session.EndAt, session.PostID,
	); err != nil {
		return errors.Wrap(err, "SaveCall")
	}
	return nil
}

// UpdateCallParticipants updates the participants list for the given call.
func (s *Store) UpdateCallParticipants(callID string, participants []string) error {
	p, err := json.Marshal(participants)
	if err != nil {
		return errors.Wrap(err, "UpdateCallParticipants: marshal participants")
	}
	result, err := s.exec(
		`UPDATE RTK_Calls SET Participants = ? WHERE Id = ?`,
		string(p), callID,
	)
	if err != nil {
		return errors.Wrap(err, "UpdateCallParticipants")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("UpdateCallParticipants: call %q not found", callID)
	}
	return nil
}

// EndCall marks a call as ended with the given Unix timestamp (ms).
func (s *Store) EndCall(callID string, endAt int64) error {
	result, err := s.exec(
		`UPDATE RTK_Calls SET EndAt = ? WHERE Id = ?`,
		endAt, callID,
	)
	if err != nil {
		return errors.Wrap(err, "EndCall")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("EndCall: call %q not found", callID)
	}
	return nil
}

// GetActiveCallIDs returns IDs of all calls that have not yet ended (EndAt = 0).
func (s *Store) GetActiveCallIDs() ([]string, error) {
	rows, err := s.db.Query(s.db.Rebind(`SELECT Id FROM RTK_Calls WHERE EndAt = 0`))
	if err != nil {
		return nil, errors.Wrap(err, "GetActiveCallIDs")
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, "GetActiveCallIDs: scan")
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// AddActiveCallID is a no-op: a call is considered active while EndAt = 0 in RTK_Calls.
func (s *Store) AddActiveCallID(_ string) error { return nil }

// RemoveActiveCallID is a no-op: EndCall sets EndAt, which makes the call inactive.
func (s *Store) RemoveActiveCallID(_ string) error { return nil }

// scanCallSession reads a single RTK_Calls row.
func scanCallSession(row *sql.Row) (*kvstore.CallSession, error) {
	var (
		s            kvstore.CallSession
		participants string
	)
	err := row.Scan(
		&s.ID, &s.ChannelID, &s.CreatorID, &s.MeetingID,
		&participants, &s.StartAt, &s.EndAt, &s.PostID,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(participants), &s.Participants); err != nil {
		return nil, errors.Wrap(err, "scanCallSession: unmarshal participants")
	}
	return &s, nil
}
