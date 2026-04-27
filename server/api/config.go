package api

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
)

// handleConfigStatus handles GET /api/v1/config/status.
func (h *API) handleConfigStatus(w http.ResponseWriter, r *http.Request) {
	status := h.configFn()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"enabled": status.Enabled,
	})
}

// handleAdminConfigStatus handles GET /api/v1/config/admin-status.
func (h *API) handleAdminConfigStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if !h.app.HasPermissionTo(userID, model.PermissionManageSystem) {
		writeError(w, http.StatusForbidden, "system admin permission required")
		return
	}

	status := h.configFn()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"enabled":               status.Enabled,
		"account_id_via_env":    status.AccountIDViaEnv,
		"api_token_via_env":     status.APITokenViaEnv,
		"cloudflare_account_id": status.AccountID,
	})
}
