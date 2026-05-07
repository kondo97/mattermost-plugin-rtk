package sqlstore

import (
	"database/sql"
	"encoding/json"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/kondo97/mattermost-plugin-rtk/server/store"
)

// GetAllCallsChannels returns every row in rtk_calls_channels.
func (s *Store) GetAllCallsChannels() ([]*store.CallsChannel, error) {
	rows, err := s.db.Query(
		`SELECT channel_id, enabled, props FROM rtk_calls_channels`,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query rtk_calls_channels")
	}
	defer func() { _ = rows.Close() }()

	var out []*store.CallsChannel
	for rows.Next() {
		ch, err := scanCallsChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rtk_calls_channels rows error")
	}
	return out, nil
}

// GetCallsChannel returns the rtk_calls_channels row for a channel, or nil.
func (s *Store) GetCallsChannel(channelID string) (*store.CallsChannel, error) {
	row := s.db.QueryRow(
		`SELECT channel_id, enabled, props FROM rtk_calls_channels WHERE channel_id = $1`,
		channelID,
	)
	ch, err := scanCallsChannel(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return ch, err
}

// UpsertCallsChannel inserts or updates a rtk_calls_channels row.
func (s *Store) UpsertCallsChannel(channel *store.CallsChannel) error {
	if channel == nil {
		return errors.New("channel must not be nil")
	}
	if channel.ChannelID == "" {
		return errors.New("channel.ChannelID must not be empty")
	}
	props := channel.Props
	if len(props) == 0 {
		props = []byte("{}")
	} else if !json.Valid(props) {
		return errors.New("channel.Props is not valid JSON")
	}
	_, err := s.db.Exec(
		`INSERT INTO rtk_calls_channels (channel_id, enabled, props)
		 VALUES ($1, $2, $3::jsonb)
		 ON CONFLICT (channel_id) DO UPDATE
		   SET enabled = EXCLUDED.enabled,
		       props   = EXCLUDED.props`,
		channel.ChannelID, channel.Enabled, string(props),
	)
	return errors.Wrap(err, "failed to upsert rtk_calls_channels")
}

// GetAllActiveCalls returns every call session with endat = 0.
func (s *Store) GetAllActiveCalls() ([]*store.CallSession, error) {
	rows, err := s.db.Query(
		`SELECT ` + callSessionSelectColumns + `
		 FROM rtk_call_sessions s
		 WHERE s.endat = 0`,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query active calls")
	}
	defer func() { _ = rows.Close() }()

	var out []*store.CallSession
	for rows.Next() {
		var session store.CallSession
		var participants pq.StringArray
		if err := rows.Scan(
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
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan active call")
		}
		session.Participants = []string(participants)
		if session.Participants == nil {
			session.Participants = []string{}
		}
		out = append(out, &session)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "active calls rows error")
	}
	return out, nil
}

// rowScanner abstracts *sql.Row and *sql.Rows for shared scan helpers.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanCallsChannel(r rowScanner) (*store.CallsChannel, error) {
	var ch store.CallsChannel
	var props []byte
	if err := r.Scan(&ch.ChannelID, &ch.Enabled, &props); err != nil {
		return nil, err
	}
	ch.Props = props
	return &ch, nil
}

// MaybeImportCallsChannels copies rows from the Calls plugin's `calls_channels`
// table into `rtk_calls_channels` exactly once.
//
// Behavior:
//   - No-op if `calls_channels` does not exist (Calls plugin not installed).
//   - No-op if `rtk_calls_channels` is non-empty (already imported, or RTK has
//     started writing its own rows).
//   - Otherwise, INSERT … SELECT all rows from `calls_channels`. ON CONFLICT
//     DO NOTHING guarantees idempotency even if two cluster nodes race.
//
// Returns the number of rows copied. Errors are returned to the caller; the
// caller decides whether to fail activation or merely log.
func (s *Store) MaybeImportCallsChannels() (int64, error) {
	var exists bool
	if err := s.db.QueryRow(
		`SELECT EXISTS (
		   SELECT 1 FROM information_schema.tables
		   WHERE table_name = 'calls_channels'
		 )`,
	).Scan(&exists); err != nil {
		return 0, errors.Wrap(err, "failed to probe for calls_channels table")
	}
	if !exists {
		return 0, nil
	}

	var rtkCount int64
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM rtk_calls_channels`,
	).Scan(&rtkCount); err != nil {
		return 0, errors.Wrap(err, "failed to count rtk_calls_channels rows")
	}
	if rtkCount > 0 {
		return 0, nil
	}

	res, err := s.db.Exec(
		`INSERT INTO rtk_calls_channels (channel_id, enabled, props)
		 SELECT channelid,
		        COALESCE(enabled, TRUE),
		        CASE WHEN props IS NULL OR props = 'null'::jsonb THEN '{}'::jsonb ELSE props END
		 FROM calls_channels
		 ON CONFLICT (channel_id) DO NOTHING`,
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to import calls_channels")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to read RowsAffected")
	}
	return n, nil
}


