package store

import "context"

// Store defines the interface for plugin store operations.
type Store interface {
	// GetCallByChannel returns the active call for a channel, or nil if none exists.
	GetCallByChannel(channelID string) (*CallSession, error)
	// GetCallByID returns the call with the given ID, or nil if not found.
	GetCallByID(callID string) (*CallSession, error)
	// GetCallByMeetingID returns the call with the given RTK meeting ID, or nil if not found.
	GetCallByMeetingID(meetingID string) (*CallSession, error)
	// SaveCall persists a call session (creates or updates).
	// Note: participants are stored in rtk_call_participants and are NOT written by SaveCall;
	// use CreateCallSession on initial creation and AddCallParticipant/RemoveCallParticipant
	// for participant mutations.
	SaveCall(session *CallSession) error
	// CreateCallSession atomically inserts a new call session row together with its initial
	// participants (taken from session.Participants) in a single transaction. Use this for
	// the initial creation of a call so the row and its creator participant are visible
	// to other nodes simultaneously.
	CreateCallSession(session *CallSession) error
	// AddCallParticipant inserts userID into the participants for callID if and only if
	// the call is still active (endat = 0). Returns the updated participants list,
	// active=false (with empty list) if the call has already ended, and added=true
	// if this invocation actually inserted a new row (false on ON CONFLICT no-op).
	// Concurrent invocations from multiple nodes are serialized via SELECT FOR UPDATE on
	// the call row, preventing lost updates and ghost participants on ended calls.
	// Callers can use `added` to decide whether a compensating RemoveCallParticipant
	// is appropriate when subsequent steps (e.g. RTK token generation) fail.
	AddCallParticipant(callID, userID string) (participants []string, active bool, added bool, err error)
	// RemoveCallParticipant deletes userID from the participants for callID. If this leaves
	// the call with zero participants, it atomically marks the call as ended.
	// Returns the updated participants list, endedNow=true only when this invocation's
	// transaction transitioned the call from active to ended, and the call's endAt
	// timestamp (0 when the call is still active). When the call was already ended
	// before this call ran, endAt is the prior timestamp and endedNow is false; this
	// distinction lets callers avoid duplicate end-side-effect emission.
	RemoveCallParticipant(callID, userID string) (participants []string, endedNow bool, endAt int64, err error)
	// EndCall marks a call as ended with the given timestamp.
	EndCall(callID string, endAt int64) error
	// UpdateCallSessionID sets the RTK session ID for a call.
	// Called from the webhook handler when meeting.participantJoined is received.
	UpdateCallSessionID(callID, sessionID string) error

	// GetChannelMeeting returns the stored row id, RTK meeting ID, and the app config ID for a channel.
	// Returns empty strings if none exists.
	GetChannelMeeting(channelID string) (id string, meetingID string, appConfigID string, err error)
	// SaveChannelMeeting persists the RTK meeting ID and app config ID for a channel.
	// Returns the row id (preserved on conflict).
	SaveChannelMeeting(channelID, meetingID string, appConfigID string) (string, error)

	// GetActiveAppConfigID returns the ID of the rtk_app_config row whose status='active', or empty string.
	GetActiveAppConfigID() (string, error)

	// StoreAppConfig records or reactivates the app configuration for the given app_id.
	// Atomically flips any existing active row to inactive, then either reactivates an
	// existing row with the same app_id (preserving its id) or inserts a new row.
	// Returns the id of the now-active row.
	StoreAppConfig(accountID, appID string) (string, error)
	// GetAppID retrieves the active RTK app ID, or empty string if not set.
	GetAppID() (string, error)

	// StoreWebhookConfig records a new webhook configuration entry (append-only history).
	StoreWebhookConfig(appConfigID string, webhookID string) error
	// ClearWebhookConfig appends a cleared-state entry to the webhook history.
	ClearWebhookConfig(appConfigID string) error
	// GetWebhookConfig retrieves the most recent webhook ID.
	// Returns empty string if no configuration has been stored yet.
	GetWebhookConfig() (webhookID string, err error)

	// WithAppLock acquires a cluster-wide advisory lock keyed by the given string and
	// runs fn while holding it. The lock is released when fn returns. Used to
	// serialize EnsureApp across HA Mattermost nodes so that ListApps→CreateApp→StoreAppConfig
	// is not racy.
	WithAppLock(ctx context.Context, key string, fn func() error) error

	// GetAllCallsChannels returns every row in rtk_calls_channels.
	// Used by GET /api/v1/channels to enumerate explicitly-registered channels.
	GetAllCallsChannels() ([]*CallsChannel, error)
	// GetCallsChannel returns the rtk_calls_channels row for a channel, or nil if none exists.
	GetCallsChannel(channelID string) (*CallsChannel, error)
	// UpsertCallsChannel inserts or updates a rtk_calls_channels row.
	UpsertCallsChannel(channel *CallsChannel) error
	// GetAllActiveCalls returns every call session with endat = 0.
	GetAllActiveCalls() ([]*CallSession, error)
}
