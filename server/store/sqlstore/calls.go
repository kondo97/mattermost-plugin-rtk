package sqlstore

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/kondo97/mattermost-plugin-rtk/server/store"
)

// GetCallByChannel returns the active call (endat = 0) for a channel, or nil.
func (s *Store) GetCallByChannel(channelID string) (*store.CallSession, error) {
	row := s.db.QueryRow(
		`SELECT id, channel_id, creator_id, meeting_id, participants, createat, updateat, endat, post_id, rtk_channel_meeting_id, session_id
		 FROM rtk_call_sessions
		 WHERE channel_id = $1 AND endat = 0`,
		channelID,
	)
	return s.scanSession(row)
}

// GetCallByID returns the call with the given ID (active or ended), or nil if not found.
func (s *Store) GetCallByID(callID string) (*store.CallSession, error) {
	row := s.db.QueryRow(
		`SELECT id, channel_id, creator_id, meeting_id, participants, createat, updateat, endat, post_id, rtk_channel_meeting_id, session_id
		 FROM rtk_call_sessions
		 WHERE id = $1`,
		callID,
	)
	return s.scanSession(row)
}

// GetCallByMeetingID returns the active call matching the given RTK meeting ID, or nil if not found.
func (s *Store) GetCallByMeetingID(meetingID string) (*store.CallSession, error) {
	row := s.db.QueryRow(
		`SELECT id, channel_id, creator_id, meeting_id, participants, createat, updateat, endat, post_id, rtk_channel_meeting_id, session_id
		 FROM rtk_call_sessions
		 WHERE meeting_id = $1 AND endat = 0`,
		meetingID,
	)
	return s.scanSession(row)
}

func (s *Store) scanSession(row *sql.Row) (*store.CallSession, error) {
	var session store.CallSession
	var participantsJSON string
	err := row.Scan(
		&session.ID,
		&session.ChannelID,
		&session.CreatorID,
		&session.MeetingID,
		&participantsJSON,
		&session.CreateAt,
		&session.UpdateAt,
		&session.EndAt,
		&session.PostID,
		&session.ChannelMeetingID,
		&session.SessionID,
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
func (s *Store) SaveCall(session *store.CallSession) error {
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

	_, err = s.db.Exec(
		`INSERT INTO rtk_call_sessions
			(id, channel_id, creator_id, meeting_id, participants, createat, updateat, endat, post_id, rtk_channel_meeting_id, session_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				channel_id         = EXCLUDED.channel_id,
				creator_id         = EXCLUDED.creator_id,
				meeting_id         = EXCLUDED.meeting_id,
				participants       = EXCLUDED.participants,
				updateat           = EXCLUDED.updateat,
				endat              = EXCLUDED.endat,
				post_id            = EXCLUDED.post_id,
				rtk_channel_meeting_id = EXCLUDED.rtk_channel_meeting_id,
				session_id         = EXCLUDED.session_id`,
		session.ID,
		session.ChannelID,
		session.CreatorID,
		session.MeetingID,
		string(participantsJSON),
		session.CreateAt,
		session.UpdateAt,
		session.EndAt,
		session.PostID,
		session.ChannelMeetingID,
		session.SessionID,
	)
	return errors.Wrap(err, "failed to save call session")
}

// UpdateCallParticipants updates the participants list for the given call ID.
func (s *Store) UpdateCallParticipants(callID string, participants []string) error {
	participantsJSON, err := json.Marshal(participants)
	if err != nil {
		return errors.Wrap(err, "failed to marshal participants")
	}
	result, err := s.db.Exec(
		`UPDATE rtk_call_sessions SET participants = $1, updateat = $2 WHERE id = $3`,
		string(participantsJSON), time.Now().UnixMilli(), callID,
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

// EndCall marks a call as ended by setting endat and updateat.
func (s *Store) EndCall(callID string, endAt int64) error {
	result, err := s.db.Exec(
		`UPDATE rtk_call_sessions SET endat = $1, updateat = $1 WHERE id = $2`,
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

// UpdateCallSessionID sets the RTK session ID for the given call.
func (s *Store) UpdateCallSessionID(callID, sessionID string) error {
	result, err := s.db.Exec(
		`UPDATE rtk_call_sessions SET session_id = $1, updateat = $2 WHERE id = $3`,
		sessionID, time.Now().UnixMilli(), callID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update call session ID")
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

// GetChannelMeeting returns the row id, RTK meeting ID, and app config ID for a channel.
// Returns empty strings if none exists.
func (s *Store) GetChannelMeeting(channelID string) (id string, meetingID string, appConfigID string, err error) {
	err = s.db.QueryRow(
		`SELECT id, meeting_id, app_config_id FROM rtk_channel_meetings WHERE channel_id = $1`,
		channelID,
	).Scan(&id, &meetingID, &appConfigID)
	if err == sql.ErrNoRows {
		return "", "", "", nil
	}
	if err != nil {
		return "", "", "", errors.Wrap(err, "failed to get channel meeting")
	}
	return id, meetingID, appConfigID, nil
}

// SaveChannelMeeting persists the RTK meeting ID and app config ID for a channel (upsert).
// On conflict, the existing row id is preserved. Returns the row id.
func (s *Store) SaveChannelMeeting(channelID, meetingID string, appConfigID string) (string, error) {
	if meetingID == "" {
		return "", errors.New("meetingID must not be empty")
	}
	now := time.Now().UnixMilli()
	var id string
	err := s.db.QueryRow(
		`INSERT INTO rtk_channel_meetings (id, channel_id, meeting_id, app_config_id, createat, updateat)
		 VALUES ($1, $2, $3, $4, $5, $5)
		 ON CONFLICT (channel_id) DO UPDATE
		   SET meeting_id    = EXCLUDED.meeting_id,
		       app_config_id = EXCLUDED.app_config_id,
		       updateat      = EXCLUDED.updateat
		 RETURNING id`,
		model.NewId(), channelID, meetingID, appConfigID, now,
	).Scan(&id)
	if err != nil {
		return "", errors.Wrap(err, "failed to save channel meeting")
	}
	return id, nil
}
