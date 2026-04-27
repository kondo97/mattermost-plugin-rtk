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

// AccountClient defines the interface for account-level Cloudflare RealtimeKit management
// operations that do not require an App ID (e.g. creating and listing apps).
type AccountClient interface {
	// CreateApp creates a new RealtimeKit app with the given name and returns it.
	CreateApp(name string) (*App, error)
	// ListApps returns all RealtimeKit apps for the account.
	ListApps() ([]App, error)
}

type accountClient struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string
}

// NewAccountClient creates a management-level RTK client that operates at the account scope.
// Use this client to create, retrieve, and list apps before an App ID is known.
func NewAccountClient(accountID, apiToken string) AccountClient {
	return &accountClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/realtime/kit", accountID),
		apiToken:   apiToken,
	}
}

// accountAPIResponse is the response wrapper for account-level RTK management endpoints.
// These endpoints use "data" as the result key, unlike the app-scoped endpoints which use "result".
type accountAPIResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

// createAppRequest is the request body for creating a RealtimeKit app.
type createAppRequest struct {
	Name string `json:"name"`
}

// appData is the app fields returned by the RTK API.
type appData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// createAppData is the data field in the create-app response.
// The API returns: {"success": true, "data": {"app": {"id": "...", "name": "..."}}}
type createAppData struct {
	App appData `json:"app"`
}

// CreateApp creates a new RealtimeKit app with the given name.
func (c *accountClient) CreateApp(name string) (*App, error) {
	body, err := json.Marshal(createAppRequest{Name: name})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal CreateApp request")
	}

	url := fmt.Sprintf("%s/apps", c.baseURL)
	resp, err := c.doRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "CreateApp request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("CreateApp: unexpected status %d", resp.StatusCode)
	}

	var result accountAPIResponse[createAppData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode CreateApp response")
	}
	if !result.Success {
		return nil, fmt.Errorf("CreateApp: API returned success=false")
	}
	return &App{ID: result.Data.App.ID, Name: result.Data.App.Name}, nil
}

// ListApps returns all RealtimeKit apps for the account.
func (c *accountClient) ListApps() ([]App, error) {
	url := fmt.Sprintf("%s/apps", c.baseURL)
	resp, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "ListApps request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ListApps: unexpected status %d", resp.StatusCode)
	}

	// The API returns: {"success": true, "data": [{"id": "...", "name": "..."}, ...]}
	var result accountAPIResponse[[]appData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to decode ListApps response")
	}

	apps := make([]App, 0, len(result.Data))
	for _, a := range result.Data {
		apps = append(apps, App{ID: a.ID, Name: a.Name})
	}
	return apps, nil
}

func (c *accountClient) doRequest(method, url string, body []byte) (*http.Response, error) {
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
