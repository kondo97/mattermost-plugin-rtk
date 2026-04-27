package store

// Store defines the interface for plugin store operations.
type Store interface {
	// GetCallByChannel returns the active call for a channel, or nil if none exists.
	GetCallByChannel(channelID string) (*CallSession, error)
	// GetCallByID returns the call with the given ID, or nil if not found.
	GetCallByID(callID string) (*CallSession, error)
	// GetCallByMeetingID returns the call with the given RTK meeting ID, or nil if not found.
	GetCallByMeetingID(meetingID string) (*CallSession, error)
	// SaveCall persists a call session (creates or updates). Also writes the meeting ID index.
	SaveCall(session *CallSession) error
	// UpdateCallParticipants updates the participants list for a call.
	UpdateCallParticipants(callID string, participants []string) error
	// EndCall marks a call as ended with the given timestamp.
	EndCall(callID string, endAt int64) error
	// UpdateCallSessionID sets the RTK session ID for a call.
	// Called from the webhook handler when meeting.participantJoined is received.
	UpdateCallSessionID(callID, sessionID string) error

	// GetChannelMeeting returns the stored RTK meeting ID and the app config ID for a channel.
	// Returns empty strings if none exists.
	GetChannelMeeting(channelID string) (meetingID string, appConfigID string, err error)
	// SaveChannelMeeting persists the RTK meeting ID and app config ID for a channel.
	SaveChannelMeeting(channelID, meetingID string, appConfigID string) error

	// GetLatestAppConfigID returns the ID of the most recent rtk_app_config entry, or empty string.
	GetLatestAppConfigID() (string, error)

	// StoreAppConfig records a new app configuration entry (append-only history).
	// Returns the ID of the newly inserted entry.
	StoreAppConfig(accountID, appID string) (string, error)
	// GetAppID retrieves the most recent RTK app ID, or empty string if not set.
	GetAppID() (string, error)

	// StoreWebhookConfig records a new webhook configuration entry (append-only history).
	StoreWebhookConfig(appConfigID string, webhookID string) error
	// ClearWebhookConfig appends a cleared-state entry to the webhook history.
	ClearWebhookConfig(appConfigID string) error
	// GetWebhookConfig retrieves the most recent webhook ID.
	// Returns empty string if no configuration has been stored yet.
	GetWebhookConfig() (webhookID string, err error)
}
