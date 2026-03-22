package rtkclient

// RTKClient defines the interface for interacting with the Cloudflare RealtimeKit API.
type RTKClient interface {
	// CreateMeeting creates a new RTK meeting with the given preset and returns the meeting.
	CreateMeeting(preset string) (*Meeting, error)
	// GenerateToken adds a participant to a meeting and returns an auth token.
	GenerateToken(meetingID, userID, preset string) (*Token, error)
	// EndMeeting terminates an RTK meeting.
	EndMeeting(meetingID string) error
}

// Meeting represents an RTK meeting returned by the Cloudflare API.
type Meeting struct {
	ID string
}

// Token represents an RTK participant auth token returned by the Cloudflare API.
type Token struct {
	Token string
}
