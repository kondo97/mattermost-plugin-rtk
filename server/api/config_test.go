package api

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
	h, _ := newTestAPI(t, nil, nil)
	h.configFn = func() ConfigStatus { return ConfigStatus{Enabled: true, OrgID: "org1"} }

	w := serveWithUser(t, h, http.MethodGet, "/api/v1/config/status", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["enabled"])
	assert.Nil(t, resp["feature_flags"], "feature_flags must not be present")
}

func TestHandleConfigStatus_Disabled(t *testing.T) {
	h, _ := newTestAPI(t, nil, nil)
	h.configFn = func() ConfigStatus { return ConfigStatus{} }

	w := serveWithUser(t, h, http.MethodGet, "/api/v1/config/status", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["enabled"])
	assert.Nil(t, resp["feature_flags"], "feature_flags must not be present")
}

func TestHandleAdminConfigStatus_Admin(t *testing.T) {
	h, mmAPI := newTestAPI(t, nil, nil)
	h.configFn = func() ConfigStatus {
		return ConfigStatus{Enabled: true, OrgID: "org1"}
	}
	mmAPI.On("HasPermissionTo", "admin1", model.PermissionManageSystem).Return(true)

	w := serveWithUser(t, h, http.MethodGet, "/api/v1/config/admin-status", "admin1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["enabled"])
	assert.Equal(t, "org1", resp["cloudflare_org_id"])
	assert.Nil(t, resp["cloudflare_api_key"], "API key must never be returned")
	assert.Nil(t, resp["feature_flags"], "feature_flags must not be present")
}

func TestHandleAdminConfigStatus_Forbidden(t *testing.T) {
	h, mmAPI := newTestAPI(t, nil, nil)
	h.configFn = func() ConfigStatus { return ConfigStatus{} }
	mmAPI.On("HasPermissionTo", "user1", model.PermissionManageSystem).Return(false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/admin-status", nil)
	req.Header.Set("Mattermost-User-ID", "user1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

