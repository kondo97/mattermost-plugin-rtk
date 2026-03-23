# Unit 5: Admin & Config — Code Generation Plan

## Unit Context

**Stories**: US-001 (Configure Cloudflare Credentials), US-002 (Toggle Feature Flags)
**Dependencies**: Unit 2 (api_config.go consumer of GetEffective*())
**Workspace root**: /Users/sei.kondo/git/mattermost-plugin-rtk

## Files to Modify / Create

| File | Action | Description |
|---|---|---|
| `server/configuration.go` | Modify | Add 10 `*bool` fields, update `GetEffective*()`, add 10 `Is*Enabled()`, fix `OnConfigurationChange` |
| `server/api_config.go` | Modify | Add `feature_flags` to both config endpoint responses |
| `plugin.json` | Modify | Add 10 `type: "bool"` feature flag settings |
| `server/configuration_test.go` | Create | Unit tests for all new configuration methods |
| `server/api_config_test.go` | Modify | Add feature_flags assertions to existing tests |
| `aidlc-docs/construction/unit-5-admin-config/code/code-summary.md` | Create | Code summary documentation |

---

## Steps

### Step 1: Modify server/configuration.go — Struct Fields + GetEffective*()
- [x] Add 10 `*bool` feature flag fields to `configuration` struct
- [x] Update `GetEffectiveOrgID()` to use `os.LookupEnv("RTK_ORG_ID")`
- [x] Update `GetEffectiveAPIKey()` to use `os.LookupEnv("RTK_API_KEY")`
- [x] Add `import "os"` and `import "strings"` to imports
- [x] Story: US-001 (credential env var override)

### Step 2: Modify server/configuration.go — Is*Enabled() Methods
- [x] Add `IsRecordingEnabled()` method
- [x] Add `IsScreenShareEnabled()` method
- [x] Add `IsPollsEnabled()` method
- [x] Add `IsTranscriptionEnabled()` method
- [x] Add `IsWaitingRoomEnabled()` method
- [x] Add `IsVideoEnabled()` method
- [x] Add `IsChatEnabled()` method
- [x] Add `IsPluginsEnabled()` method
- [x] Add `IsParticipantsEnabled()` method
- [x] Add `IsRaiseHandEnabled()` method
- [x] Story: US-002 (feature flag env var override + default ON)

### Step 3: Modify server/configuration.go — OnConfigurationChange
- [x] Update credential change detection to use `GetEffective*()` instead of raw fields
- [x] Story: US-001 (credential change detection uses effective values)

### Step 4: Modify server/api_config.go — Feature Flags in Responses
- [x] Add `featureFlags()` helper method on `configuration` that returns `map[string]bool`
- [x] Update `handleConfigStatus` to include `feature_flags` in response
- [x] Update `handleAdminConfigStatus` to include `feature_flags` in response
- [x] Verify API Key is NOT included in any response (SEC-03)
- [x] Story: US-002 (feature flags exposed via API)

### Step 5: Modify plugin.json — Feature Flag Settings
- [x] Add `RecordingEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `ScreenShareEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `PollsEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `TranscriptionEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `WaitingRoomEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `VideoEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `ChatEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `PluginsEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `ParticipantsEnabled` (`type: "bool"`, `default: "true"`)
- [x] Add `RaiseHandEnabled` (`type: "bool"`, `default: "true"`)
- [x] Story: US-002 (System Console UI for feature flags)

### Step 6: Create server/configuration_test.go — Unit Tests
- [x] `TestGetEffectiveOrgID_EnvSet` — env var present → env var wins
- [x] `TestGetEffectiveOrgID_EnvEmpty` — env var set to empty → empty wins
- [x] `TestGetEffectiveOrgID_NoEnv` — env var absent → config value
- [x] `TestGetEffectiveAPIKey_EnvSet` — env var present → env var wins
- [x] `TestGetEffectiveAPIKey_EnvEmpty` — env var set to empty → empty wins
- [x] `TestGetEffectiveAPIKey_NoEnv` — env var absent → config value
- [x] `TestIsRecordingEnabled_EnvTrue` — env var "true" → true
- [x] `TestIsRecordingEnabled_EnvFalse` — env var "false" → false
- [x] `TestIsRecordingEnabled_NilDefault` — nil *bool → true (default ON)
- [x] `TestIsRecordingEnabled_ExplicitFalse` — &false → false
- [x] Repeat 4 cases for remaining 9 feature flags (abbreviated in plan, all generated)
- [x] Story: US-001, US-002 (test coverage for effective value logic)

### Step 7: Modify server/api_config_test.go — Feature Flags Assertions
- [x] Update `TestHandleConfigStatus_Enabled` to assert `feature_flags` present and all true (default)
- [x] Update `TestHandleConfigStatus_Disabled` to assert `feature_flags` present and all true (default ON even when plugin disabled)
- [x] Update `TestHandleAdminConfigStatus_Admin` to assert `feature_flags` present
- [x] Add `TestHandleConfigStatus_FeatureFlagDisabled` — single flag disabled via *bool &false
- [x] Story: US-002 (feature flags in API response)

### Step 8: Create aidlc-docs/construction/unit-5-admin-config/code/code-summary.md
- [x] Document all modified and created files
- [x] Note key design decisions implemented
- [x] Story coverage summary

---

## Story Coverage

| Story | Steps |
|---|---|
| US-001: Configure Cloudflare Credentials | Steps 1, 3, 6 |
| US-002: Toggle Feature Flags | Steps 2, 4, 5, 6, 7 |
