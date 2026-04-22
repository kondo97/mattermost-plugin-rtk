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

// apiResponse is the common wrapper for all RTK API responses.
type apiResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

// createMeetingRequest is the request body for creating a meeting.
type createMeetingRequest struct {
	Title                string `json:"title"`
	RecordOnStart        bool   `json:"record_on_start"`
	LiveStreamOnStart    bool   `json:"live_stream_on_start"`
	WaitingRoomEnabled   bool   `json:"waiting_room_enabled"`
	TranscriptionEnabled bool   `json:"transcription_enabled"`
	RaiseHandEnabled     bool   `json:"raise_hand_enabled"`
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
func (c *client) CreateMeeting(opts CreateMeetingOptions) (*Meeting, error) {
	url := fmt.Sprintf("%s/meetings", c.baseURL)
	body, err := json.Marshal(createMeetingRequest{
		Title:                "",
		WaitingRoomEnabled:   opts.WaitingRoomEnabled,
		TranscriptionEnabled: opts.TranscriptionEnabled,
		RaiseHandEnabled:     opts.RaiseHandEnabled,
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
	return &Meeting{ID: result.Data.ID}, nil
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
	return &Token{Token: result.Data.Token}, nil
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

// meetingParticipantItem is a single entry in the list-participants response.
type meetingParticipantItem struct {
	CustomParticipantID string `json:"custom_participant_id"`
}

// meetingParticipantsData wraps the participants array returned by the API.
type meetingParticipantsData struct {
	Participants []meetingParticipantItem `json:"participants"`
}

// GetMeetingParticipants returns the custom participant IDs currently connected to a meeting.
func (c *client) GetMeetingParticipants(meetingID string) ([]string, error) {
	url := fmt.Sprintf("%s/meetings/%s/active-participants", c.baseURL, meetingID)
	resp, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetMeetingParticipants request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrMeetingNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetMeetingParticipants: unexpected status %d", resp.StatusCode)
	}

	var result apiResponse[meetingParticipantsData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode GetMeetingParticipants response")
	}

	ids := make([]string, 0, len(result.Data.Participants))
	for _, p := range result.Data.Participants {
		if p.CustomParticipantID != "" {
			ids = append(ids, p.CustomParticipantID)
		}
	}
	return ids, nil
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
