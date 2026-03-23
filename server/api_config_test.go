package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allFlagsTrue asserts that feature_flags is present and all 10 flags are true.
func allFlagsTrue(t *testing.T, resp map[string]any) {
	t.Helper()
	flags, ok := resp["feature_flags"].(map[string]any)
	require.True(t, ok, "feature_flags should be a map")
	for _, key := range []string{"recording", "screenShare", "polls", "transcription", "waitingRoom", "video", "chat", "plugins", "participants", "raiseHand"} {
		assert.Equal(t, true, flags[key], "feature flag %q should be true by default", key)
	}
}

func TestHandleConfigStatus_Enabled(t *testing.T) {
	p, _ := newTestPlugin(t, nil, nil)
	p.setConfiguration(&configuration{
		CloudflareOrgID:  "org1",
		CloudflareAPIKey: "key1",
	})
	p.router = p.initRouter()

	w := serveWithUser(t, p, http.MethodGet, "/api/v1/config/status", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["enabled"])
	allFlagsTrue(t, resp)
}

func TestHandleConfigStatus_Disabled(t *testing.T) {
	p, _ := newTestPlugin(t, nil, nil)
	p.setConfiguration(&configuration{})
	p.router = p.initRouter()

	w := serveWithUser(t, p, http.MethodGet, "/api/v1/config/status", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["enabled"])
	allFlagsTrue(t, resp) // flags default ON even when plugin disabled
}

func TestHandleConfigStatus_FeatureFlagDisabled(t *testing.T) {
	p, _ := newTestPlugin(t, nil, nil)
	p.setConfiguration(&configuration{
		CloudflareOrgID:  "org1",
		CloudflareAPIKey: "key1",
		RecordingEnabled: boolPtr(false),
	})
	p.router = p.initRouter()

	w := serveWithUser(t, p, http.MethodGet, "/api/v1/config/status", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	flags := resp["feature_flags"].(map[string]any)
	assert.Equal(t, false, flags["recording"])
	assert.Equal(t, true, flags["video"]) // other flags still ON
}

func TestHandleAdminConfigStatus_Admin(t *testing.T) {
	p, api := newTestPlugin(t, nil, nil)
	p.setConfiguration(&configuration{
		CloudflareOrgID:  "org1",
		CloudflareAPIKey: "key1",
	})
	p.router = p.initRouter()
	api.On("HasPermissionTo", "admin1", model.PermissionManageSystem).Return(true)

	w := serveWithUser(t, p, http.MethodGet, "/api/v1/config/admin-status", "admin1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["enabled"])
	assert.Equal(t, "org1", resp["cloudflare_org_id"])
	assert.Nil(t, resp["cloudflare_api_key"], "API key must never be returned")
	allFlagsTrue(t, resp)
}

func TestHandleAdminConfigStatus_Forbidden(t *testing.T) {
	p, api := newTestPlugin(t, nil, nil)
	p.setConfiguration(&configuration{})
	p.router = p.initRouter()
	api.On("HasPermissionTo", "user1", model.PermissionManageSystem).Return(false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/admin-status", nil)
	req.Header.Set("Mattermost-User-ID", "user1")
	w := httptest.NewRecorder()
	p.ServeHTTP(nil, w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
