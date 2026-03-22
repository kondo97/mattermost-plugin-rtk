package kvstore

// KVStore defines the interface for plugin KV store operations.
type KVStore interface {
	// GetTemplateData retrieves template data for a user.
	GetTemplateData(userID string) (string, error)

	// GetCallByChannel returns the active call for a channel, or nil if none exists.
	GetCallByChannel(channelID string) (*CallSession, error)
	// GetCallByID returns the call with the given ID, or nil if not found.
	GetCallByID(callID string) (*CallSession, error)
	// GetAllActiveCalls returns all currently active calls (EndAt == 0).
	GetAllActiveCalls() ([]*CallSession, error)
	// SaveCall persists a call session (creates or updates).
	SaveCall(session *CallSession) error
	// UpdateCallParticipants updates the participants list for a call.
	UpdateCallParticipants(callID string, participants []string) error
	// EndCall marks a call as ended with the given timestamp.
	EndCall(callID string, endAt int64) error

	// SetHeartbeat records a heartbeat timestamp for a participant in a call.
	SetHeartbeat(callID, userID string, ts int64) error
	// GetHeartbeat returns the last heartbeat timestamp for a participant, or 0 if not found.
	GetHeartbeat(callID, userID string) (int64, error)

	// StoreVoIPToken stores a VoIP device token for a user.
	StoreVoIPToken(userID, token string) error
	// GetVoIPToken retrieves the VoIP device token for a user.
	GetVoIPToken(userID string) (string, error)
}
