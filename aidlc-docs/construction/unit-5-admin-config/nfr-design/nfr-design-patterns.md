# NFR Design Patterns — Unit 5: Admin & Config

## Pattern 1: Env Var Override (Per-Call)

**Addresses**: PERF-01, SEC-04, alignment with mattermost-plugin-calls

**Description**: Each `GetEffective*()` and `Is*Enabled()` method calls `os.LookupEnv` on every invocation. No caching.

**Rationale**: Consistent with `mattermost-plugin-calls` (`getRTCDURL()`, `getJobServiceURL()`). Plugin processes typically restart when env vars change, so caching provides no practical benefit while adding complexity.

**Implementation pattern for credentials**:

```go
func (c *configuration) GetEffectiveOrgID() string {
    if val, ok := os.LookupEnv("RTK_ORG_ID"); ok {
        return val
    }
    return c.CloudflareOrgID
}
```

**Implementation pattern for feature flags**:

```go
func (c *configuration) IsRecordingEnabled() bool {
    if val, ok := os.LookupEnv("RTK_RECORDING_ENABLED"); ok {
        return strings.EqualFold(val, "true")
    }
    if c.RecordingEnabled == nil {
        return true // default ON
    }
    return *c.RecordingEnabled
}
```

---

## Pattern 2: Nil Pointer Default (Default-ON Feature Flags)

**Addresses**: BR-02, TEST-02, REL-02

**Description**: Feature flag fields use `*bool`. The three-state semantics:

| State | Meaning | `Is*Enabled()` |
|---|---|---|
| `nil` | Never configured | `true` (default ON) |
| `&true` | Explicitly enabled | `true` |
| `&false` | Explicitly disabled | `false` |

**Why not `bool` with defaulting in `OnConfigurationChange`**: Would prevent admins from disabling features (any `false` value would be overwritten back to `true`).

**Why not inverted names (`DisableX`)**: Confusing with env var names like `RTK_RECORDING_ENABLED=false`.

---

## Pattern 3: Thread-Safe Config Snapshot (Extended)

**Addresses**: Existing concurrency requirement, extended with new fields

**Description**: The existing `configurationLock sync.RWMutex` + `Clone()` pattern is unchanged. Unit 5 only adds fields — all new fields are value types (`string`) or pointer types (`*bool`) that are valid for shallow copy.

**No change required to `getConfiguration()` or `setConfiguration()`.**

`Clone()` shallow copy validity:
- `string` fields: immutable value type, copied by value
- `*bool` fields: pointer is copied; pointed-to value is never mutated post-deserialization

---

## Pattern 4: Secret Field (Mattermost plugin.json)

**Addresses**: SEC-02, Q4 (API Key masking)

**Description**: `CloudflareAPIKey` in `plugin.json` uses `"secret": true`. Mattermost framework behavior:
- Value is stored encrypted in Mattermost server config
- System Console never sends the plaintext value to the browser
- Field appears empty when admin views the settings page
- Admin must re-enter the key to change it

**No custom React component needed**. This is purely a `plugin.json` configuration change.

---

## Pattern 5: t.Setenv Test Isolation

**Addresses**: TEST-01, TEST-03

**Description**: All unit tests that exercise env var override behavior use `t.Setenv(key, value)`. Go 1.17+ automatically restores the original env var value after the test completes, ensuring test isolation.

```go
func TestGetEffectiveOrgID_EnvOverride(t *testing.T) {
    t.Setenv("RTK_ORG_ID", "env-org-id")
    cfg := &configuration{CloudflareOrgID: "config-org-id"}
    assert.Equal(t, "env-org-id", cfg.GetEffectiveOrgID())
}

func TestGetEffectiveOrgID_EnvEmpty(t *testing.T) {
    t.Setenv("RTK_ORG_ID", "") // explicitly set to empty
    cfg := &configuration{CloudflareOrgID: "config-org-id"}
    assert.Equal(t, "", cfg.GetEffectiveOrgID()) // empty env var wins
}

func TestGetEffectiveOrgID_NoEnv(t *testing.T) {
    // RTK_ORG_ID not set
    cfg := &configuration{CloudflareOrgID: "config-org-id"}
    assert.Equal(t, "config-org-id", cfg.GetEffectiveOrgID())
}
```

---

## Pattern 6: Credential Guard in OnConfigurationChange

**Addresses**: BR-04, REL-03

**Description**: RTK client re-initialization in `OnConfigurationChange` uses `GetEffective*()` (not raw fields) for both change detection and the empty-check gate.

```go
credentialsChanged := prev.GetEffectiveOrgID() != configuration.GetEffectiveOrgID() ||
    prev.GetEffectiveAPIKey() != configuration.GetEffectiveAPIKey()

if credentialsChanged {
    orgID := configuration.GetEffectiveOrgID()
    apiKey := configuration.GetEffectiveAPIKey()
    if orgID != "" && apiKey != "" {
        p.rtkClient = rtkclient.NewClient(orgID, apiKey)
        p.reRegisterWebhook()
    } else {
        p.rtkClient = nil
    }
}
```
