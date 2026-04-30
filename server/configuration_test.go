package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- GetEffectiveAccountID ---

func TestGetEffectiveAccountID_EnvSet(t *testing.T) {
	t.Setenv("RTK_ACCOUNT_ID", "env-account-id")
	cfg := &configuration{CloudflareAccountID: "config-account-id"}
	assert.Equal(t, "env-account-id", cfg.GetEffectiveAccountID())
}

func TestGetEffectiveAccountID_EnvEmpty(t *testing.T) {
	t.Setenv("RTK_ACCOUNT_ID", "")
	cfg := &configuration{CloudflareAccountID: "config-account-id"}
	assert.Equal(t, "", cfg.GetEffectiveAccountID())
}

func TestGetEffectiveAccountID_NoEnv(t *testing.T) {
	cfg := &configuration{CloudflareAccountID: "config-account-id"}
	assert.Equal(t, "config-account-id", cfg.GetEffectiveAccountID())
}

// --- GetEffectiveAPIToken ---

func TestGetEffectiveAPIToken_EnvSet(t *testing.T) {
	t.Setenv("RTK_API_TOKEN", "env-api-token")
	cfg := &configuration{CloudflareAPIToken: "config-api-token"}
	assert.Equal(t, "env-api-token", cfg.GetEffectiveAPIToken())
}

func TestGetEffectiveAPIToken_EnvEmpty(t *testing.T) {
	t.Setenv("RTK_API_TOKEN", "")
	cfg := &configuration{CloudflareAPIToken: "config-api-token"}
	assert.Equal(t, "", cfg.GetEffectiveAPIToken())
}

func TestGetEffectiveAPIToken_NoEnv(t *testing.T) {
	cfg := &configuration{CloudflareAPIToken: "config-api-token"}
	assert.Equal(t, "config-api-token", cfg.GetEffectiveAPIToken())
}

// --- AppIDFromEnv / GetEffectiveAppID ---

func TestGetEffectiveAppID_EnvSet(t *testing.T) {
	t.Setenv("RTK_APP_ID", "env-app-id")
	cfg := &configuration{}
	assert.Equal(t, "env-app-id", cfg.GetEffectiveAppID())
	assert.True(t, cfg.AppIDFromEnv())
}

func TestGetEffectiveAppID_EnvEmpty(t *testing.T) {
	t.Setenv("RTK_APP_ID", "")
	cfg := &configuration{}
	assert.Equal(t, "", cfg.GetEffectiveAppID())
	// LookupEnv considers empty-but-set as set.
	assert.True(t, cfg.AppIDFromEnv())
}

func TestGetEffectiveAppID_NoEnv(t *testing.T) {
	cfg := &configuration{}
	assert.Equal(t, "", cfg.GetEffectiveAppID())
	assert.False(t, cfg.AppIDFromEnv())
}
