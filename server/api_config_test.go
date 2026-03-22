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
