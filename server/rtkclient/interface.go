package rtkclient

import "errors"

// ErrMeetingNotFound is returned when the requested RTK meeting does not exist (HTTP 404).
var ErrMeetingNotFound = errors.New("meeting not found")

// App represents a Cloudflare RealtimeKit application.
type App struct {
	ID   string
	Name string
}

// ErrWebhookNotFound is returned by GetWebhook when the requested webhook does not exist (HTTP 404).
var ErrWebhookNotFound = errors.New("webhook not found")

// ErrWebhookConflict is returned by RegisterWebhook when a webhook with the same URL already
// exists on the RTK side (HTTP 409). Callers can use errors.Is to detect this case and
// recover by deleting the conflicting webhook before retrying.
var ErrWebhookConflict = errors.New("webhook already exists")

// WebhookInfo holds summary information about a registered RTK webhook.
type WebhookInfo struct {
	ID  string
	URL string
}

// RTKClient defines the interface for interacting with the Cloudflare RealtimeKit API.
type RTKClient interface {
	// CreateMeeting creates a new RTK meeting and returns the meeting.
	CreateMeeting() (*Meeting, error)
	// GenerateToken adds a participant to a meeting and returns an auth token.
	// callID identifies the call_session this participant is joining; it is embedded
	// into the RTK customParticipantId so webhook events can be correlated back to a
	// specific call (RTK Meetings are permanent and reusable across calls in the same
	// channel — without this binding, delayed webhooks from an old call could be
	// misattributed to a new call sharing the same meetingID).
	GenerateToken(meetingID, callID, userID, displayName, preset string) (*Token, error)
	// RegisterWebhook registers a webhook endpoint with RTK for the given events.
	// Returns the webhook ID on success.
	// Returns ErrWebhookConflict (HTTP 409) if a webhook with the same URL already exists.
	RegisterWebhook(url string, events []string) (id string, err error)
	// DeleteWebhook removes a previously registered RTK webhook by ID.
	DeleteWebhook(webhookID string) error
	// GetWebhook returns the webhook with the given ID.
	// Returns ErrWebhookNotFound (HTTP 404) if no such webhook exists.
	GetWebhook(id string) (*WebhookInfo, error)
	// ListWebhooks returns all webhooks registered for this organisation.
	ListWebhooks() ([]WebhookInfo, error)
	// GetMeeting verifies the existence of an RTK meeting via the Cloudflare
	// "Get Meeting by ID" endpoint (GET /meetings/{id}).
	// Returns the meeting on success, ErrMeetingNotFound (HTTP 404) when the
	// meeting has been deleted, or a wrapped error for transient failures.
	//
	// Note: this endpoint reflects only whether the Meeting resource itself
	// exists. It says nothing about whether anyone is currently connected
	// (sessions are a separate concept and may have ended while the Meeting
	// remains as a permanent reusable room — see app/calls.go).
	GetMeeting(meetingID string) (*Meeting, error)
}

// Meeting represents an RTK meeting returned by the Cloudflare API.
type Meeting struct {
	ID string
}

// Token represents an RTK participant auth token returned by the Cloudflare API.
type Token struct {
	Token string
}
