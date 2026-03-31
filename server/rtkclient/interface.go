package rtkclient

import "errors"

// ErrMeetingNotFound is returned when the requested RTK meeting does not exist (HTTP 404).
var ErrMeetingNotFound = errors.New("meeting not found")

// RTKClient defines the interface for interacting with the Cloudflare RealtimeKit API.
type RTKClient interface {
	// CreateMeeting creates a new RTK meeting and returns the meeting.
	CreateMeeting() (*Meeting, error)
	// GenerateToken adds a participant to a meeting and returns an auth token.
	GenerateToken(meetingID, userID, displayName, preset string) (*Token, error)
	// EndMeeting terminates an RTK meeting.
	EndMeeting(meetingID string) error
	// RegisterWebhook registers a webhook endpoint with RTK for the given events.
	// Returns the webhook ID and signing secret on success.
	RegisterWebhook(url string, events []string) (id, secret string, err error)
	// DeleteWebhook removes a previously registered RTK webhook by ID.
	DeleteWebhook(webhookID string) error
	// GetMeetingParticipants returns the custom participant IDs currently connected to a meeting.
	GetMeetingParticipants(meetingID string) ([]string, error)
}

// Meeting represents an RTK meeting returned by the Cloudflare API.
type Meeting struct {
	ID string
}

// Token represents an RTK participant auth token returned by the Cloudflare API.
type Token struct {
	Token string
}
