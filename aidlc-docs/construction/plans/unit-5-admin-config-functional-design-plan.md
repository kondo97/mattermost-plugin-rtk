# Unit 5: Admin & Config — Functional Design Plan

## Objective

Design the business logic for plugin configuration management: credentials storage with env var overrides, 10 feature flags, thread-safe clone pattern, and admin console custom UI.

## Plan Steps

- [x] Step 1: Analyze existing configuration.go and identify what already exists vs. what is missing
- [x] Step 2: Design env var override logic for `GetEffective*()` methods
- [x] Step 3: Define 10 feature flag fields and their env var names
- [x] Step 4: Design frontend admin settings component structure
- [x] Step 5: Answer clarifying questions from user
- [x] Step 6: Generate functional design artifacts (domain-entities.md, business-logic-model.md, business-rules.md, frontend-components.md)

---

## Clarifying Questions

### Q1: Env Var Override — Precedence for Credentials

The unit spec lists env vars `RTK_ORG_ID` and `RTK_API_KEY` for credential override.
Currently `GetEffectiveOrgID()` and `GetEffectiveAPIKey()` return the config value directly with no env var check.

What should the env var override behavior be?

A) Env var takes strict precedence: if set (even to empty string), use env var value
B) Env var takes precedence only if non-empty: use env var when `os.Getenv("RTK_ORG_ID") != ""`
C) Other (describe after [Answer]:)

[Answer]:

---

### Q2: Feature Flags — Default Value

The unit spec states "all 10 feature flags default to enabled (ON)". In Go, boolean struct fields default to `false`. What is the preferred strategy?

A) Use `*bool` (pointer to bool) — `nil` means "not set → default ON", `false` means explicitly disabled
B) Use inverted flag names (e.g., `DisableRecording bool`) — `false` means enabled by default
C) Use bool with OnConfigurationChange defaulting logic — if field is false after load, force-set to true (but this prevents users from disabling them)
D) Other (describe after [Answer]:)

[Answer]:

---

### Q3: Feature Flag Env Var — True/False Format

When a feature flag env var is set (e.g., `RTK_RECORDING_ENABLED=false`), what format should be accepted?

A) Only `"true"` and `"false"` (case-insensitive)
B) `"1"`/`"0"` and `"true"`/`"false"` (Go's `strconv.ParseBool` semantics)
C) Other (describe after [Answer]:)

[Answer]:

---

### Q4: Admin UI — API Key Masking

In the admin console custom UI, the Cloudflare API Key should be masked. What behavior is expected?

A) Display as `********` (never show the actual value); admin must re-enter to change
B) Display masked by default, with a "show" toggle button to reveal the value
C) Use a standard password-type input (browser-masked); show value on toggle
D) Other (describe after [Answer]:)

[Answer]:

---

### Q5: Admin UI — Env Var Indicator

When a field is overridden by an env var, the unit spec says to show a "read-only field with indicator label". What should the indicator look like?

A) A text label like `"Set via environment variable RTK_ORG_ID"` next to the field; field is disabled
B) A tooltip icon (ⓘ) with hover text explaining the env var name; field is disabled
C) A banner above the field: `"This field is controlled by environment variable RTK_ORG_ID and cannot be edited here"`
D) Other (describe after [Answer]:)

[Answer]:

---

### Q6: Feature Flags — Admin UI Layout

10 feature flags need to be presented in the admin console. What is the preferred layout?

A) Individual toggle rows, one per flag, each with a label and description
B) A grouped section titled "Feature Flags" with all toggles listed
C) Two columns of toggles (5 left, 5 right) to reduce vertical space
D) Other (describe after [Answer]:)

[Answer]:

---

### Q7: Configuration Change Handling

When feature flags change in `OnConfigurationChange`, is any action required beyond updating `p.configuration`?

A) No — feature flags are read on demand via `getConfiguration()`, no additional action needed
B) Yes — emit a WebSocket event so webapp clients can reload their feature flag state
C) Other (describe after [Answer]:)

[Answer]:

---

### Q8: Webapp Admin Settings — Mattermost System Console Integration

Admin settings in Mattermost plugins can be implemented in two ways:

A) **Custom component** (`registerAdminConsoleCustomSetting`) — full custom React component per field, maximum control over UI
B) **Plugin manifest settings schema** (`plugin.json` `settings_schema`) — declarative JSON with limited field types (text, bool, dropdown), no custom masking or env var indicators
C) **Hybrid** — use manifest schema for simple flags, and a custom component for the API key field only

Given that we need masking and env var indicators, which approach?

[Answer]:

---
