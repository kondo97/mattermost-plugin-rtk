package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// boolPtr is a test helper that returns a pointer to the given bool value.
func boolPtr(b bool) *bool { return &b }

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

// --- Feature Flag Tests ---
// Each flag is tested for: env "true", env "false", nil (default ON), explicit &false.

func testFeatureFlag(t *testing.T, envVar string, getFn func(*configuration) bool, field **bool) {
	t.Helper()

	t.Run("env_true", func(t *testing.T) {
		t.Setenv(envVar, "true")
		cfg := &configuration{}
		assert.True(t, getFn(cfg))
	})

	t.Run("env_true_uppercase", func(t *testing.T) {
		t.Setenv(envVar, "TRUE")
		cfg := &configuration{}
		assert.True(t, getFn(cfg))
	})

	t.Run("env_false", func(t *testing.T) {
		t.Setenv(envVar, "false")
		cfg := &configuration{}
		assert.False(t, getFn(cfg))
	})

	t.Run("env_other_value_treated_as_false", func(t *testing.T) {
		t.Setenv(envVar, "1")
		cfg := &configuration{}
		assert.False(t, getFn(cfg))
	})

	t.Run("nil_defaults_to_true", func(t *testing.T) {
		cfg := &configuration{}
		assert.True(t, getFn(cfg))
	})

	t.Run("explicit_false", func(t *testing.T) {
		cfg := &configuration{}
		*field = boolPtr(false)
		assert.False(t, getFn(cfg))
		*field = nil // reset
	})

	t.Run("explicit_true", func(t *testing.T) {
		cfg := &configuration{}
		*field = boolPtr(true)
		assert.True(t, getFn(cfg))
		*field = nil // reset
	})
}

func TestIsScreenShareEnabled(t *testing.T) {
	cfg := &configuration{}
	testFeatureFlag(t, "RTK_SCREEN_SHARE_ENABLED", func(c *configuration) bool { return cfg.IsScreenShareEnabled() }, &cfg.ScreenShareEnabled)
}

func TestIsVideoEnabled(t *testing.T) {
	cfg := &configuration{}
	testFeatureFlag(t, "RTK_VIDEO_ENABLED", func(c *configuration) bool { return cfg.IsVideoEnabled() }, &cfg.VideoEnabled)
}

func TestIsParticipantsEnabled(t *testing.T) {
	cfg := &configuration{}
	testFeatureFlag(t, "RTK_PARTICIPANTS_ENABLED", func(c *configuration) bool { return cfg.IsParticipantsEnabled() }, &cfg.ParticipantsEnabled)
}
