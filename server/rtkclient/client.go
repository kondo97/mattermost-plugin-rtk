package rtkclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const defaultBaseURL = "https://api.realtime.cloudflare.com/v2"

type client struct {
	httpClient *http.Client
	baseURL    string
	orgID      string
	apiKey     string
}

// NewClient creates a new RTKClient with the given Cloudflare credentials.
func NewClient(orgID, apiKey string) RTKClient {
	return &client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    defaultBaseURL,
		orgID:      orgID,
		apiKey:     apiKey,
	}
}

// createMeetingRequest is the request body for creating a meeting.
type createMeetingRequest struct {
	Title string `json:"title"`
}

// createMeetingResponse is the response body from creating a meeting.
type createMeetingResponse struct {
	ID string `json:"id"`
}

// addParticipantRequest is the request body for adding a participant.
type addParticipantRequest struct {
	PresetName          string `json:"preset_name"`
	CustomParticipantID string `json:"custom_participant_id"`
}

// addParticipantResponse is the response body from adding a participant.
type addParticipantResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// CreateMeeting creates a new RTK meeting and returns the meeting ID.
func (c *client) CreateMeeting() (*Meeting, error) {
	url := fmt.Sprintf("%s/apps/%s/meetings", c.baseURL, c.orgID)
	body, err := json.Marshal(createMeetingRequest{Title: ""})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateMeeting request")
	}

	resp, err := c.doRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "CreateMeeting request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("CreateMeeting: unexpected status %d", resp.StatusCode)
	}

	var result createMeetingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode CreateMeeting response")
	}
	return &Meeting{ID: result.ID}, nil
}

// GenerateToken adds a participant to the meeting and returns an auth token.
func (c *client) GenerateToken(meetingID, userID, preset string) (*Token, error) {
	url := fmt.Sprintf("%s/apps/%s/meetings/%s/participants", c.baseURL, c.orgID, meetingID)
	body, err := json.Marshal(addParticipantRequest{
		PresetName:          preset,
		CustomParticipantID: userID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal GenerateToken request")
	}

	resp, err := c.doRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "GenerateToken request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GenerateToken: unexpected status %d", resp.StatusCode)
	}

	var result addParticipantResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode GenerateToken response")
	}
	return &Token{Token: result.Token}, nil
}

// EndMeeting terminates an RTK meeting (best effort — caller should not abort on error).
func (c *client) EndMeeting(meetingID string) error {
	url := fmt.Sprintf("%s/apps/%s/meetings/%s", c.baseURL, c.orgID, meetingID)
	resp, err := c.doRequest(http.MethodDelete, url, nil)
	if err != nil {
		return errors.Wrap(err, "EndMeeting request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("EndMeeting: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// registerWebhookRequest is the request body for registering a webhook.
type registerWebhookRequest struct {
	TargetURL string   `json:"targetUrl"`
	Events    []string `json:"events"`
}

// registerWebhookResponse is the response body from registering a webhook.
type registerWebhookResponse struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

// RegisterWebhook registers a webhook endpoint with RTK for the given events.
func (c *client) RegisterWebhook(url string, events []string) (id, secret string, err error) {
	reqURL := fmt.Sprintf("%s/apps/%s/webhooks", c.baseURL, c.orgID)
	body, err := json.Marshal(registerWebhookRequest{TargetURL: url, Events: events})
	if err != nil {
		return "", "", errors.Wrap(err, "failed to marshal RegisterWebhook request")
	}

	resp, err := c.doRequest(http.MethodPost, reqURL, body)
	if err != nil {
		return "", "", errors.Wrap(err, "RegisterWebhook request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("RegisterWebhook: unexpected status %d", resp.StatusCode)
	}

	var result registerWebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", errors.Wrap(err, "failed to decode RegisterWebhook response")
	}
	return result.ID, result.Secret, nil
}

// DeleteWebhook removes a previously registered RTK webhook by ID.
func (c *client) DeleteWebhook(webhookID string) error {
	url := fmt.Sprintf("%s/apps/%s/webhooks/%s", c.baseURL, c.orgID, webhookID)
	resp, err := c.doRequest(http.MethodDelete, url, nil)
	if err != nil {
		return errors.Wrap(err, "DeleteWebhook request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("DeleteWebhook: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// doRequest executes an authenticated HTTP request.
func (c *client) doRequest(method, url string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	req.SetBasicAuth(c.orgID, c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}
