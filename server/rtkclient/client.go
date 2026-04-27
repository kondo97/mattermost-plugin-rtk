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

type client struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string
}

// NewClient creates a new RTKClient with the given Cloudflare credentials.
func NewClient(accountID, appID, apiToken string) RTKClient {
	return &client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/realtime/kit/%s", accountID, appID),
		apiToken:   apiToken,
	}
}

// apiResponse is the common wrapper for all RTK API responses.
// The Cloudflare RealtimeKit API consistently uses "data" as the result key.
type apiResponse[T any] struct {
	Success bool `json:"success"`
	Result  T    `json:"data"`
}

// createMeetingRequest is the request body for creating a meeting.
type createMeetingRequest struct {
	Title string `json:"title"`
}

// createMeetingData is the data field in the create meeting response.
type createMeetingData struct {
	ID string `json:"id"`
}

// addParticipantRequest is the request body for adding a participant.
type addParticipantRequest struct {
	Name                string `json:"name"`
	PresetName          string `json:"preset_name"`
	CustomParticipantID string `json:"custom_participant_id"`
}

// addParticipantData is the data field in the add participant response.
type addParticipantData struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// CreateMeeting creates a new RTK meeting and returns the meeting ID.
func (c *client) CreateMeeting() (*Meeting, error) {
	url := fmt.Sprintf("%s/meetings", c.baseURL)
	body, err := json.Marshal(createMeetingRequest{
		Title: "",
	})
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

	var result apiResponse[createMeetingData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode CreateMeeting response")
	}
	if !result.Success {
		return nil, fmt.Errorf("CreateMeeting: API returned success=false")
	}
	if result.Result.ID == "" {
		return nil, fmt.Errorf("CreateMeeting: API returned empty meeting ID")
	}
	return &Meeting{ID: result.Result.ID}, nil
}

// GenerateToken adds a participant to the meeting and returns an auth token.
func (c *client) GenerateToken(meetingID, userID, displayName, preset string) (*Token, error) {
	return c.generateTokenOnce(meetingID, userID, displayName, preset)
}

// generateTokenOnce makes a single add-participant API call.
func (c *client) generateTokenOnce(meetingID, userID, displayName, preset string) (*Token, error) {
	url := fmt.Sprintf("%s/meetings/%s/participants", c.baseURL, meetingID)
	body, err := json.Marshal(addParticipantRequest{
		Name:                displayName,
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

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrMeetingNotFound
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GenerateToken: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result apiResponse[addParticipantData]
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Wrap(err, "failed to decode GenerateToken response")
	}
	if !result.Success {
		return nil, fmt.Errorf("GenerateToken: API returned success=false: %s", string(respBody))
	}
	return &Token{Token: result.Result.Token}, nil
}

// getWebhookData is the data field in the get-webhook response.
type getWebhookData struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// GetWebhook returns the webhook with the given ID.
// Returns ErrWebhookNotFound if the webhook does not exist (HTTP 404).
func (c *client) GetWebhook(id string) (*WebhookInfo, error) {
	url := fmt.Sprintf("%s/webhooks/%s", c.baseURL, id)
	resp, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetWebhook request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrWebhookNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetWebhook: unexpected status %d", resp.StatusCode)
	}

	var result apiResponse[getWebhookData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode GetWebhook response")
	}
	return &WebhookInfo{ID: result.Result.ID, URL: result.Result.URL}, nil
}

// registerWebhookRequest is the request body for registering a webhook.
type registerWebhookRequest struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// registerWebhookData is the data field in the register webhook response.
type registerWebhookData struct {
	ID string `json:"id"`
}

// registerWebhookResponse wraps the register-webhook result.
// The RTK register-webhook endpoint returns {"success":true,"data":{"id":"...",...}}.
type registerWebhookResponse struct {
	Success bool                `json:"success"`
	Data    registerWebhookData `json:"data"`
}

// RegisterWebhook registers a webhook endpoint with RTK for the given events.
func (c *client) RegisterWebhook(webhookURL string, events []string) (string, error) {
	reqURL := fmt.Sprintf("%s/webhooks", c.baseURL)
	body, err := json.Marshal(registerWebhookRequest{
		Name:   "mattermost-plugin-rtk",
		URL:    webhookURL,
		Events: events,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal RegisterWebhook request")
	}

	resp, err := c.doRequest(http.MethodPost, reqURL, body)
	if err != nil {
		return "", errors.Wrap(err, "RegisterWebhook request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusConflict {
		return "", ErrWebhookConflict
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("RegisterWebhook: unexpected status %d", resp.StatusCode)
	}

	var result registerWebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", errors.Wrap(err, "failed to decode RegisterWebhook response")
	}
	if !result.Success {
		return "", fmt.Errorf("RegisterWebhook: API returned success=false")
	}
	return result.Data.ID, nil
}

// listWebhookItem is a single entry in the list-webhooks response.
type listWebhookItem struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// listWebhooksResponse wraps the array of webhooks.
// The RTK list-webhooks endpoint returns {"success":true,"data":[...]}.
type listWebhooksResponse struct {
	Success bool              `json:"success"`
	Data    []listWebhookItem `json:"data"`
}

// ListWebhooks returns all webhooks registered for this organisation.
func (c *client) ListWebhooks() ([]WebhookInfo, error) {
	url := fmt.Sprintf("%s/webhooks", c.baseURL)
	resp, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "ListWebhooks request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ListWebhooks: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read ListWebhooks response body")
	}

	var result listWebhooksResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrapf(err, "failed to decode ListWebhooks response: %s", string(body))
	}

	out := make([]WebhookInfo, 0, len(result.Data))
	for _, w := range result.Data {
		out = append(out, WebhookInfo{ID: w.ID, URL: w.URL})
	}
	return out, nil
}

// DeleteWebhook removes a previously registered RTK webhook by ID.
func (c *client) DeleteWebhook(webhookID string) error {
	url := fmt.Sprintf("%s/webhooks/%s", c.baseURL, webhookID)
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

// getMeetingData is the data field in the Get Meeting by ID response.
// Only the ID is consumed; remaining fields are ignored intentionally.
type getMeetingData struct {
	ID string `json:"id"`
}

// GetMeeting fetches a meeting by ID to verify it still exists in Cloudflare.
// Returns ErrMeetingNotFound on HTTP 404. All other non-2xx responses or
// transport errors are surfaced as wrapped errors and must be treated as
// transient by callers (i.e. do not infer "meeting deleted" from them).
//
// Endpoint: GET /meetings/{meetingID}
// Cloudflare docs: MeetingGetMeetingByIDResponse model.
func (c *client) GetMeeting(meetingID string) (*Meeting, error) {
	url := fmt.Sprintf("%s/meetings/%s", c.baseURL, meetingID)
	resp, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetMeeting request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrMeetingNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetMeeting: unexpected status %d", resp.StatusCode)
	}

	var result apiResponse[getMeetingData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode GetMeeting response")
	}
	if result.Result.ID == "" {
		return nil, fmt.Errorf("GetMeeting: API returned empty meeting ID")
	}
	return &Meeting{ID: result.Result.ID}, nil
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

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}
