package sqlstore

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/kondo97/mattermost-plugin-rtk/server/store"
)

// callSessionSelectColumns is the SELECT projection used by all Get* methods.
// Participants are loaded via a correlated subquery against rtk_call_participants
// so the in-memory CallSession remains the same shape after normalization.
const callSessionSelectColumns = `
    s.id, s.channel_id, s.creator_id, s.meeting_id,
    s.createat, s.updateat, s.endat, s.post_id, s.rtk_channel_meeting_id, s.session_id,
    COALESCE(
        (SELECT array_agg(p.user_id ORDER BY p.joined_at)
         FROM rtk_call_participants p
         WHERE p.rtk_call_sessions_id = s.id),
        ARRAY[]::text[]
    ) AS participants`

// GetCallByChannel returns the active call (endat = 0) for a channel, or nil.
func (s *Store) GetCallByChannel(channelID string) (*store.CallSession, error) {
	row := s.db.QueryRow(
		`SELECT `+callSessionSelectColumns+`
		 FROM rtk_call_sessions s
		 WHERE s.channel_id = $1 AND s.endat = 0`,
		channelID,
	)
	return scanSessionRow(row)
}

// GetCallByID returns the call with the given ID (active or ended), or nil if not found.
func (s *Store) GetCallByID(callID string) (*store.CallSession, error) {
	row := s.db.QueryRow(
		`SELECT `+callSessionSelectColumns+`
		 FROM rtk_call_sessions s
		 WHERE s.id = $1`,
		callID,
	)
	return scanSessionRow(row)
}

// GetCallByMeetingID returns the active call matching the given RTK meeting ID, or nil if not found.
func (s *Store) GetCallByMeetingID(meetingID string) (*store.CallSession, error) {
	row := s.db.QueryRow(
		`SELECT `+callSessionSelectColumns+`
		 FROM rtk_call_sessions s
		 WHERE s.meeting_id = $1 AND s.endat = 0`,
		meetingID,
	)
	return scanSessionRow(row)
}

// scanSessionRow scans a row produced by callSessionSelectColumns.
func scanSessionRow(row *sql.Row) (*store.CallSession, error) {
	var session store.CallSession
	var participants pq.StringArray
	err := row.Scan(
		&session.ID,
		&session.ChannelID,
		&session.CreatorID,
		&session.MeetingID,
		&session.CreateAt,
		&session.UpdateAt,
		&session.EndAt,
		&session.PostID,
		&session.ChannelMeetingID,
		&session.SessionID,
		&participants,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan session")
	}
	session.Participants = []string(participants)
	if session.Participants == nil {
		session.Participants = []string{}
	}
	return &session, nil
}

// SaveCall persists a call session (insert or full update on conflict).
// Participants are NOT written by this method — use CreateCallSession on initial
// creation and Add/RemoveCallParticipant for participant mutations.
func (s *Store) SaveCall(session *store.CallSession) error {
	if err := validateSession(session); err != nil {
		return err
	}
	_, err := s.db.Exec(saveCallSQL, saveCallArgs(session)...)
	return errors.Wrap(err, "failed to save call session")
}

// CreateCallSession atomically inserts the call session row together with the
// initial participants from session.Participants. Used by CreateCall so other
// nodes never observe an active call without its creator participant row.
func (s *Store) CreateCallSession(session *store.CallSession) error {
	if err := validateSession(session); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	if _, execErr := tx.Exec(saveCallSQL, saveCallArgs(session)...); execErr != nil {
		return errors.Wrap(execErr, "failed to insert call session")
	}

	for i, userID := range session.Participants {
		// joined_at preserves the input order using createat as the base.
		joinedAt := session.CreateAt + int64(i)
		if _, execErr := tx.Exec(
			`INSERT INTO rtk_call_participants (id, rtk_call_sessions_id, user_id, joined_at)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (rtk_call_sessions_id, user_id) DO NOTHING`,
			model.NewId(), session.ID, userID, joinedAt,
		); execErr != nil {
			return errors.Wrap(execErr, "failed to insert initial participant")
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return errors.Wrap(commitErr, "failed to commit create call session")
	}
	return nil
}

func validateSession(session *store.CallSession) error {
	if session == nil {
		return errors.New("session must not be nil")
	}
	if session.ID == "" {
		return errors.New("session.ID must not be empty")
	}
	if session.ChannelID == "" {
		return errors.New("session.ChannelID must not be empty")
	}
	return nil
}

const saveCallSQL = `INSERT INTO rtk_call_sessions
		(id, channel_id, creator_id, meeting_id, createat, updateat, endat, post_id, rtk_channel_meeting_id, session_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			channel_id             = EXCLUDED.channel_id,
			creator_id             = EXCLUDED.creator_id,
			meeting_id             = EXCLUDED.meeting_id,
			updateat               = EXCLUDED.updateat,
			endat                  = EXCLUDED.endat,
			post_id                = EXCLUDED.post_id,
			rtk_channel_meeting_id = EXCLUDED.rtk_channel_meeting_id,
			session_id             = EXCLUDED.session_id`

func saveCallArgs(session *store.CallSession) []any {
	return []any{
		session.ID,
		session.ChannelID,
		session.CreatorID,
		session.MeetingID,
		session.CreateAt,
		session.UpdateAt,
		session.EndAt,
		session.PostID,
		session.ChannelMeetingID,
		session.SessionID,
	}
}

// AddCallParticipant inserts userID into rtk_call_participants for callID, but
// only if the call is still active. The (SELECT … FOR UPDATE) on the call row
// serializes against EndCall and concurrent participant mutations on other
// cluster nodes, preventing ghost participants on calls that have just ended.
func (s *Store) AddCallParticipant(callID, userID string) ([]string, bool, bool, error) {
	if callID == "" || userID == "" {
		return nil, false, false, errors.New("callID and userID must not be empty")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, false, false, errors.Wrap(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	var endAt int64
	err = tx.QueryRow(
		`SELECT endat FROM rtk_call_sessions WHERE id = $1 FOR UPDATE`,
		callID,
	).Scan(&endAt)
	if err == sql.ErrNoRows {
		return []string{}, false, false, nil
	}
	if err != nil {
		return nil, false, false, errors.Wrap(err, "failed to lock call row")
	}
	if endAt != 0 {
		return []string{}, false, false, nil
	}

	res, execErr := tx.Exec(
		`INSERT INTO rtk_call_participants (id, rtk_call_sessions_id, user_id, joined_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (rtk_call_sessions_id, user_id) DO NOTHING`,
		model.NewId(), callID, userID, time.Now().UnixMilli(),
	)
	if execErr != nil {
		return nil, false, false, errors.Wrap(execErr, "failed to insert participant")
	}
	rows, raErr := res.RowsAffected()
	if raErr != nil {
		return nil, false, false, errors.Wrap(raErr, "failed to read RowsAffected")
	}
	added := rows == 1

	if _, execErr := tx.Exec(
		`UPDATE rtk_call_sessions SET updateat = $1 WHERE id = $2`,
		time.Now().UnixMilli(), callID,
	); execErr != nil {
		return nil, false, false, errors.Wrap(execErr, "failed to bump updateat")
	}

	participants, err := selectParticipants(tx, callID)
	if err != nil {
		return nil, false, false, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, false, false, errors.Wrap(commitErr, "failed to commit add participant")
	}
	return participants, true, added, nil
}

// RemoveCallParticipant deletes userID from the participants for callID. If this
// leaves the call empty (BR-13) the call is atomically marked as ended in the
// same transaction so concurrent JoinCall on other nodes cannot insert a ghost
// participant. endedNow distinguishes the case where THIS invocation transitioned
// the call to ended (caller should emit end-side-effects) from the case where
// the call was already ended before the call ran (caller should NOT re-emit).
func (s *Store) RemoveCallParticipant(callID, userID string) ([]string, bool, int64, error) {
	if callID == "" || userID == "" {
		return nil, false, 0, errors.New("callID and userID must not be empty")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, false, 0, errors.Wrap(err, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback() }()

	var endAt int64
	err = tx.QueryRow(
		`SELECT endat FROM rtk_call_sessions WHERE id = $1 FOR UPDATE`,
		callID,
	).Scan(&endAt)
	if err == sql.ErrNoRows {
		return []string{}, false, 0, nil
	}
	if err != nil {
		return nil, false, 0, errors.Wrap(err, "failed to lock call row")
	}
	if endAt != 0 {
		// Already ended before we ran: report current participants (likely empty)
		// and the existing endAt, but endedNow=false so callers do not double-emit.
		participants, perr := selectParticipants(tx, callID)
		if perr != nil {
			return nil, false, 0, perr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, false, 0, errors.Wrap(commitErr, "failed to commit remove participant")
		}
		return participants, false, endAt, nil
	}

	if _, execErr := tx.Exec(
		`DELETE FROM rtk_call_participants WHERE rtk_call_sessions_id = $1 AND user_id = $2`,
		callID, userID,
	); execErr != nil {
		return nil, false, 0, errors.Wrap(execErr, "failed to delete participant")
	}

	participants, err := selectParticipants(tx, callID)
	if err != nil {
		return nil, false, 0, err
	}

	now := time.Now().UnixMilli()
	endedNow := false
	if len(participants) == 0 {
		if _, execErr := tx.Exec(
			`UPDATE rtk_call_sessions SET endat = $1, updateat = $1 WHERE id = $2 AND endat = 0`,
			now, callID,
		); execErr != nil {
			return nil, false, 0, errors.Wrap(execErr, "failed to auto-end call")
		}
		endAt = now
		endedNow = true
	} else {
		if _, execErr := tx.Exec(
			`UPDATE rtk_call_sessions SET updateat = $1 WHERE id = $2`,
			now, callID,
		); execErr != nil {
			return nil, false, 0, errors.Wrap(execErr, "failed to bump updateat")
		}
		endAt = 0
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, false, 0, errors.Wrap(commitErr, "failed to commit remove participant")
	}
	return participants, endedNow, endAt, nil
}

// selectParticipants returns the ordered participants for a call, run inside the
// caller's transaction so it sees its own pending insert/delete.
func selectParticipants(tx *sql.Tx, callID string) ([]string, error) {
	var arr pq.StringArray
	err := tx.QueryRow(
		`SELECT COALESCE(array_agg(user_id ORDER BY joined_at), ARRAY[]::text[])
		 FROM rtk_call_participants WHERE rtk_call_sessions_id = $1`,
		callID,
	).Scan(&arr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select participants")
	}
	out := []string(arr)
	if out == nil {
		out = []string{}
	}
	return out, nil
}

// EndCall marks a call as ended by setting endat and updateat. Idempotent: if
// the call is already ended (or does not exist), it is a no-op and no error is
// returned. This pairs with RemoveCallParticipant's atomic auto-end so an
// explicit EndCall racing with the last LeaveCall does not produce a spurious
// "call not found" error.
func (s *Store) EndCall(callID string, endAt int64) error {
	_, err := s.db.Exec(
		`UPDATE rtk_call_sessions SET endat = $1, updateat = $1 WHERE id = $2 AND endat = 0`,
		endAt, callID,
	)
	return errors.Wrap(err, "failed to end call")
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
