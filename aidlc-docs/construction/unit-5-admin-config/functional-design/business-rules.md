# Business Rules — Unit 5: Admin & Config

## BR-01: Env Var Presence Check (Credentials)

**Rule**: Credential env vars (`RTK_ORG_ID`, `RTK_API_KEY`) must be checked with `os.LookupEnv`, not `os.Getenv`.

**Rationale**: An explicitly-set empty env var (`RTK_ORG_ID=""`) must override the config value (Q1: strict precedence). `os.Getenv` cannot distinguish "not set" from "set to empty".

**Violation**: Using `os.Getenv` and comparing to `""` would silently fall through to config value when an empty override is intended.

---

## BR-02: Feature Flag Default ON

**Rule**: All 10 feature flag fields are `*bool`. A `nil` value MUST be treated as `true` (enabled) by all `Is*Enabled()` methods.

**Rationale**: Go zero value for pointer is `nil`. A freshly-loaded config with no admin-configured flags must default all features to ON.

---

## BR-03: Feature Flag Env Var Parsing

**Rule**: Feature flag env var values are parsed case-insensitively. Only `"true"` (any case) maps to `true`. All other non-empty values (including `"false"`, `"0"`, `"1"`, `"yes"`) map to `false`.

**Rationale**: Simple and predictable. Avoids `strconv.ParseBool` ambiguity where `"1"` and `"t"` would mean different things to different administrators.

**Implementation**:
```go
strings.EqualFold(val, "true")  // true only for "true", "True", "TRUE", etc.
```

---

## BR-04: Credential Change Detection

**Rule**: `OnConfigurationChange` MUST compare effective values (using `GetEffective*()`) — not raw struct fields — to detect credential changes.

**Rationale**: If credentials are overridden by env vars, the raw struct fields never change even if the admin edits the System Console. Comparing raw fields would incorrectly skip RTK client re-initialization.

---

## BR-05: Clone Validity

**Rule**: `Clone()` MUST produce a valid independent copy such that mutations to the original do not affect the clone, and vice versa.

**Current implementation** (shallow copy) is valid because:
- `string` fields: immutable value types
- `*bool` fields: pointer is copied; the pointed-to value is never mutated after deserialization

No deep copy is required for the new fields.

---

## BR-06: API Key Never Logged

**Rule**: The effective API key value (`GetEffectiveAPIKey()`) MUST NOT appear in any log output, including debug logs.

**Verification**: No call site logs the return value of `GetEffectiveAPIKey()` or `c.CloudflareAPIKey`.

---

## BR-07: settings_schema Default Values

**Rule**: Each `type: "bool"` feature flag entry in `plugin.json` MUST declare `"default": "true"` so the System Console displays the toggle as ON before the admin saves any configuration.

**Note**: This `default` only affects the System Console display. The actual stored value is `nil` (*bool) until the admin saves, and `nil` is treated as ON by BR-02.

---

## BR-08: Plugin Enabled Check Uses Effective Values

**Rule**: Any check for whether the plugin is fully configured (credentials present) MUST use `GetEffectiveOrgID()` and `GetEffectiveAPIKey()`, not the raw struct fields.

**Applies to**: `/config/status`, `/config/admin-status` handlers, and `OnConfigurationChange` RTK client initialization gate.

---

## BR-09: No Side Effects on Feature Flag Change

**Rule**: When only feature flags change in `OnConfigurationChange` (credentials unchanged), no additional actions are taken. Feature flags are read on-demand via `getConfiguration()`.

**Applies to**: WebSocket events, RTK client re-initialization, webhook re-registration — none of these are triggered by feature flag changes alone.
