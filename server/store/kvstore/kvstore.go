package kvstore

// KVStore defines the interface for plugin store operations.
type KVStore interface {
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

	// GetActiveCallIDs returns the list of currently active call IDs.
	GetActiveCallIDs() ([]string, error)
	// AddActiveCallID adds a call ID to the active calls index.
	AddActiveCallID(callID string) error
	// RemoveActiveCallID removes a call ID from the active calls index.
	RemoveActiveCallID(callID string) error

	// StoreWebhookID persists the registered RTK webhook ID.
	StoreWebhookID(id string) error
	// GetWebhookID retrieves the registered RTK webhook ID, or empty string if not set.
	GetWebhookID() (string, error)
	// StoreWebhookSecret persists the RTK webhook signing secret.
	StoreWebhookSecret(secret string) error
	// GetWebhookSecret retrieves the RTK webhook signing secret.
	GetWebhookSecret() (string, error)
}
