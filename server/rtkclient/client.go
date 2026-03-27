package rtkclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// apiResponse is the common wrapper for all RTK API responses.
type apiResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

// createMeetingRequest is the request body for creating a meeting.
type createMeetingRequest struct {
	Title              string `json:"title"`
	RecordOnStart      bool   `json:"record_on_start"`
	LiveStreamOnStart  bool   `json:"live_stream_on_start"`
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

	var result apiResponse[createMeetingData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode CreateMeeting response")
	}
	return &Meeting{ID: result.Data.ID}, nil
}

// GenerateToken adds a participant to the meeting and returns an auth token.
// If the preset does not exist, it is auto-created and the request is retried.
func (c *client) GenerateToken(meetingID, userID, displayName, preset string) (*Token, error) {
	token, respBody, err := c.generateTokenOnce(meetingID, userID, displayName, preset)
	if err != nil && isPresetNotFound(respBody, err) {
		if createErr := c.EnsurePreset(preset); createErr != nil {
			return nil, fmt.Errorf("failed to create preset %q: %w", preset, createErr)
		}
		token, _, err = c.generateTokenOnce(meetingID, userID, displayName, preset)
	}
	return token, err
}

// generateTokenOnce makes a single add-participant API call.
func (c *client) generateTokenOnce(meetingID, userID, displayName, preset string) (*Token, []byte, error) {
	url := fmt.Sprintf("%s/meetings/%s/participants", c.baseURL, meetingID)
	body, err := json.Marshal(addParticipantRequest{
		Name:                displayName,
		PresetName:          preset,
		CustomParticipantID: userID,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal GenerateToken request")
	}

	resp, err := c.doRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "GenerateToken request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, respBody, fmt.Errorf("GenerateToken: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result apiResponse[addParticipantData]
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, respBody, errors.Wrap(err, "failed to decode GenerateToken response")
	}
	return &Token{Token: result.Data.Token}, respBody, nil
}

// isPresetNotFound checks whether the error or response body indicates a missing preset.
func isPresetNotFound(respBody []byte, err error) bool {
	if err == nil {
		return false
	}
	s := err.Error() + string(respBody)
	return strings.Contains(s, "No preset found") || strings.Contains(s, "preset")
}

// EnsurePreset creates a preset with full media permissions if it does not already exist.
func (c *client) EnsurePreset(presetName string) error {
	preset := map[string]interface{}{
		"name": presetName,
		"config": map[string]interface{}{
			"view_type":             "GROUP_CALL",
			"max_screenshare_count": 1,
			"max_video_streams": map[string]interface{}{
				"desktop": 9,
				"mobile":  9,
			},
			"media": map[string]interface{}{
				"screenshare": map[string]interface{}{
					"frame_rate": 15,
					"quality":    "hd",
				},
				"video": map[string]interface{}{
					"frame_rate": 30,
					"quality":    "hd",
				},
			},
		},
		"permissions": map[string]interface{}{
			"media": map[string]interface{}{
				"audio":       map[string]interface{}{"can_produce": "ALLOWED"},
				"video":       map[string]interface{}{"can_produce": "ALLOWED"},
				"screenshare": map[string]interface{}{"can_produce": "ALLOWED"},
			},
			"stage_enabled":                      false,
			"can_record":                         true,
			"can_spotlight":                      true,
			"kick_participant":                   true,
			"disable_participant_audio":          true,
			"disable_participant_video":          true,
			"disable_participant_screensharing":  true,
			"pin_participant":                    true,
			"accept_waiting_requests":            true,
			"can_change_participant_permissions": true,
			"show_participant_list":              true,
			"can_edit_display_name":              true,
			"hidden_participant":                 false,
			"is_recorder":                        false,
			"waiting_room_type":                  "SKIP",
			"chat": map[string]interface{}{
				"public":  map[string]interface{}{"can_send": true, "text": true, "files": true},
				"private": map[string]interface{}{"can_send": true, "can_receive": true, "text": true, "files": true},
			},
			"polls": map[string]interface{}{
				"can_create": true,
				"can_vote":   true,
				"can_view":   true,
			},
		},
	}

	body, err := json.Marshal(preset)
	if err != nil {
		return errors.Wrap(err, "failed to marshal preset request")
	}

	reqURL := fmt.Sprintf("%s/presets", c.baseURL)
	resp, err := c.doRequest(http.MethodPost, reqURL, body)
	if err != nil {
		return errors.Wrap(err, "EnsurePreset request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	// Conflict means preset already exists — that's fine
	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("EnsurePreset: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// EndMeeting terminates an RTK meeting (best effort — caller should not abort on error).
func (c *client) EndMeeting(meetingID string) error {
	url := fmt.Sprintf("%s/meetings/%s", c.baseURL, meetingID)
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
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// registerWebhookData is the data field in the register webhook response.
type registerWebhookData struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

// RegisterWebhook registers a webhook endpoint with RTK for the given events.
func (c *client) RegisterWebhook(webhookURL string, events []string) (id, secret string, err error) {
	reqURL := fmt.Sprintf("%s/webhooks", c.baseURL)
	body, err := json.Marshal(registerWebhookRequest{
		Name:   "mattermost-plugin-rtk",
		URL:    webhookURL,
		Events: events,
	})
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

	var result apiResponse[registerWebhookData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", errors.Wrap(err, "failed to decode RegisterWebhook response")
	}
	return result.Data.ID, result.Data.Secret, nil
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
