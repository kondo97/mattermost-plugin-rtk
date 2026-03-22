package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
)

// handleConfigStatus handles GET /api/v1/config/status.
func (p *Plugin) handleConfigStatus(w http.ResponseWriter, r *http.Request) {
	cfg := p.getConfiguration()
	enabled := cfg.GetEffectiveOrgID() != "" && cfg.GetEffectiveAPIKey() != ""

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"enabled": enabled,
	})
}

// handleAdminConfigStatus handles GET /api/v1/config/admin-status.
func (p *Plugin) handleAdminConfigStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if !p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
		writeError(w, http.StatusForbidden, "system admin permission required")
		return
	}

	cfg := p.getConfiguration()
	enabled := cfg.GetEffectiveOrgID() != "" && cfg.GetEffectiveAPIKey() != ""

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"enabled":           enabled,
		"cloudflare_org_id": cfg.CloudflareOrgID,
	})
}
