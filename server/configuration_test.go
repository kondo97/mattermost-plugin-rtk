package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- GetEffectiveOrgID ---

func TestGetEffectiveOrgID_EnvSet(t *testing.T) {
	t.Setenv("RTK_ORG_ID", "env-org-id")
	cfg := &configuration{CloudflareOrgID: "config-org-id"}
	assert.Equal(t, "env-org-id", cfg.GetEffectiveOrgID())
}

func TestGetEffectiveOrgID_EnvEmpty(t *testing.T) {
	t.Setenv("RTK_ORG_ID", "")
	cfg := &configuration{CloudflareOrgID: "config-org-id"}
	assert.Equal(t, "", cfg.GetEffectiveOrgID())
}

func TestGetEffectiveOrgID_NoEnv(t *testing.T) {
	cfg := &configuration{CloudflareOrgID: "config-org-id"}
	assert.Equal(t, "config-org-id", cfg.GetEffectiveOrgID())
}

// --- GetEffectiveAPIKey ---

func TestGetEffectiveAPIKey_EnvSet(t *testing.T) {
	t.Setenv("RTK_API_KEY", "env-api-key")
	cfg := &configuration{CloudflareAPIKey: "config-api-key"}
	assert.Equal(t, "env-api-key", cfg.GetEffectiveAPIKey())
}

func TestGetEffectiveAPIKey_EnvEmpty(t *testing.T) {
	t.Setenv("RTK_API_KEY", "")
	cfg := &configuration{CloudflareAPIKey: "config-api-key"}
	assert.Equal(t, "", cfg.GetEffectiveAPIKey())
}

func TestGetEffectiveAPIKey_NoEnv(t *testing.T) {
	cfg := &configuration{CloudflareAPIKey: "config-api-key"}
	assert.Equal(t, "config-api-key", cfg.GetEffectiveAPIKey())
}
