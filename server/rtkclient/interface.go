package rtkclient

import "errors"

// ErrMeetingNotFound is returned when the requested RTK meeting does not exist (HTTP 404).
var ErrMeetingNotFound = errors.New("meeting not found")

// ErrWebhookNotFound is returned by GetWebhook when the requested webhook does not exist (HTTP 404).
var ErrWebhookNotFound = errors.New("webhook not found")

// ErrWebhookConflict is returned by RegisterWebhook when a webhook with the same URL already
// exists on the RTK side (HTTP 409). Callers can use errors.Is to detect this case and
// recover by deleting the conflicting webhook before retrying.
var ErrWebhookConflict = errors.New("webhook already exists")

// CreateMeetingOptions holds optional server-side settings for a new RTK meeting.
type CreateMeetingOptions struct {
}

// WebhookInfo holds summary information about a registered RTK webhook.
type WebhookInfo struct {
	ID  string
	URL string
}

// RTKClient defines the interface for interacting with the Cloudflare RealtimeKit API.
type RTKClient interface {
	// CreateMeeting creates a new RTK meeting and returns the meeting.
	CreateMeeting(opts CreateMeetingOptions) (*Meeting, error)
	// GenerateToken adds a participant to a meeting and returns an auth token.
	GenerateToken(meetingID, userID, displayName, preset string) (*Token, error)
	// EndMeeting terminates an RTK meeting.
	EndMeeting(meetingID string) error
	// RegisterWebhook registers a webhook endpoint with RTK for the given events.
	// Returns the webhook ID and signing secret on success.
	// Returns ErrWebhookConflict (HTTP 409) if a webhook with the same URL already exists.
	RegisterWebhook(url string, events []string) (id, secret string, err error)
	// DeleteWebhook removes a previously registered RTK webhook by ID.
	DeleteWebhook(webhookID string) error
	// GetWebhook returns the webhook with the given ID.
	// Returns ErrWebhookNotFound (HTTP 404) if no such webhook exists.
	GetWebhook(id string) (*WebhookInfo, error)
	// ListWebhooks returns all webhooks registered for this organisation.
	ListWebhooks() ([]WebhookInfo, error)
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
