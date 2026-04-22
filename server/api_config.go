package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
)

// configFeatureFlags builds the feature flags map from the current configuration.
// Credentials are never included in this map (SEC-03).
func configFeatureFlags(cfg *configuration) map[string]bool {
	return map[string]bool{
		"screenShare":  cfg.IsScreenShareEnabled(),
		"video":        cfg.IsVideoEnabled(),
		"participants": cfg.IsParticipantsEnabled(),
	}
}

// handleConfigStatus handles GET /api/v1/config/status.
func (p *Plugin) handleConfigStatus(w http.ResponseWriter, r *http.Request) {
	cfg := p.getConfiguration()
	enabled := cfg.GetEffectiveOrgID() != "" && cfg.GetEffectiveAPIKey() != ""

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"enabled":       enabled,
		"feature_flags": configFeatureFlags(cfg),
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
		"org_id_via_env":    cfg.OrgIDFromEnv(),
		"api_key_via_env":   cfg.APIKeyFromEnv(),
		"cloudflare_org_id": cfg.CloudflareOrgID,
		"feature_flags":     configFeatureFlags(cfg),
	})
}
