package sqlstore

import (
	"database/sql"
	"encoding/json"

	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
	"github.com/pkg/errors"
)

// GetCallByChannel returns the active call (end_at = 0) for a channel, or nil.
func (s *Store) GetCallByChannel(channelID string) (*kvstore.CallSession, error) {
	p1 := s.placeholder(1)
	row := s.db.QueryRow(
		`SELECT id, channel_id, creator_id, meeting_id, participants, start_at, end_at, post_id, cleanup_fail_count
		 FROM rtk_call_sessions
		 WHERE channel_id = `+p1+` AND end_at = 0`,
		channelID,
	)
	return s.scanSession(row)
}

// GetCallByID returns the call with the given ID (active or ended), or nil if not found.
func (s *Store) GetCallByID(callID string) (*kvstore.CallSession, error) {
	p1 := s.placeholder(1)
	row := s.db.QueryRow(
		`SELECT id, channel_id, creator_id, meeting_id, participants, start_at, end_at, post_id, cleanup_fail_count
		 FROM rtk_call_sessions
		 WHERE id = `+p1,
		callID,
	)
	return s.scanSession(row)
}

// GetCallByMeetingID returns the call matching the given RTK meeting ID, or nil if not found.
func (s *Store) GetCallByMeetingID(meetingID string) (*kvstore.CallSession, error) {
	p1 := s.placeholder(1)
	row := s.db.QueryRow(
		`SELECT id, channel_id, creator_id, meeting_id, participants, start_at, end_at, post_id, cleanup_fail_count
		 FROM rtk_call_sessions
		 WHERE meeting_id = `+p1,
		meetingID,
	)
	return s.scanSession(row)
}

func (s *Store) scanSession(row *sql.Row) (*kvstore.CallSession, error) {
	var session kvstore.CallSession
	var participantsJSON string
	err := row.Scan(
		&session.ID,
		&session.ChannelID,
		&session.CreatorID,
		&session.MeetingID,
		&participantsJSON,
		&session.StartAt,
		&session.EndAt,
		&session.PostID,
		&session.CleanupFailCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan session")
	}
	if err := json.Unmarshal([]byte(participantsJSON), &session.Participants); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal participants")
	}
	if session.Participants == nil {
		session.Participants = []string{}
	}
	return &session, nil
}

// SaveCall persists a call session (insert or full update on conflict).
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

	participantsJSON, err := json.Marshal(session.Participants)
	if err != nil {
		return errors.Wrap(err, "failed to marshal participants")
	}

	var query string
	if s.isPostgres() {
		query = `INSERT INTO rtk_call_sessions
			(id, channel_id, creator_id, meeting_id, participants, start_at, end_at, post_id, cleanup_fail_count)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET
				channel_id         = EXCLUDED.channel_id,
				creator_id         = EXCLUDED.creator_id,
				meeting_id         = EXCLUDED.meeting_id,
				participants       = EXCLUDED.participants,
				start_at           = EXCLUDED.start_at,
				end_at             = EXCLUDED.end_at,
				post_id            = EXCLUDED.post_id,
				cleanup_fail_count = EXCLUDED.cleanup_fail_count`
	} else {
		query = `INSERT INTO rtk_call_sessions
			(id, channel_id, creator_id, meeting_id, participants, start_at, end_at, post_id, cleanup_fail_count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				channel_id         = VALUES(channel_id),
				creator_id         = VALUES(creator_id),
				meeting_id         = VALUES(meeting_id),
				participants       = VALUES(participants),
				start_at           = VALUES(start_at),
				end_at             = VALUES(end_at),
				post_id            = VALUES(post_id),
				cleanup_fail_count = VALUES(cleanup_fail_count)`
	}

	_, err = s.db.Exec(query,
		session.ID,
		session.ChannelID,
		session.CreatorID,
		session.MeetingID,
		string(participantsJSON),
		session.StartAt,
		session.EndAt,
		session.PostID,
		session.CleanupFailCount,
	)
	return errors.Wrap(err, "failed to save call session")
}

// UpdateCallParticipants updates the participants list for the given call ID.
func (s *Store) UpdateCallParticipants(callID string, participants []string) error {
	participantsJSON, err := json.Marshal(participants)
	if err != nil {
		return errors.Wrap(err, "failed to marshal participants")
	}
	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	result, err := s.db.Exec(
		`UPDATE rtk_call_sessions SET participants = `+p1+` WHERE id = `+p2,
		string(participantsJSON), callID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update call participants")
	}
	n, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	if n == 0 {
		return errors.New("call not found")
	}
	return nil
}

// EndCall marks a call as ended by setting end_at.
func (s *Store) EndCall(callID string, endAt int64) error {
	p1 := s.placeholder(1)
	p2 := s.placeholder(2)
	result, err := s.db.Exec(
		`UPDATE rtk_call_sessions SET end_at = `+p1+` WHERE id = `+p2,
		endAt, callID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to end call")
	}
	n, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	if n == 0 {
		return errors.New("call not found")
	}
	return nil
}

// GetActiveCallIDs returns the IDs of all calls where end_at = 0.
func (s *Store) GetActiveCallIDs() ([]string, error) {
	rows, err := s.db.Query(`SELECT id FROM rtk_call_sessions WHERE end_at = 0`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query active calls")
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, "failed to scan call ID")
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate active call IDs")
	}
	if ids == nil {
		return []string{}, nil
	}
	return ids, nil
}

// AddActiveCallID is a no-op: active calls are derived from end_at = 0 in the DB.
func (s *Store) AddActiveCallID(_ string) error { return nil }

// RemoveActiveCallID is a no-op: active calls are derived from end_at = 0 in the DB.
func (s *Store) RemoveActiveCallID(_ string) error { return nil }
