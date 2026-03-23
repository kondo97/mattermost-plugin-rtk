# Business Logic Model — Unit 5: Admin & Config

## Overview

Unit 5 extends the plugin configuration system with:
1. Env var override support for all credential and feature flag fields
2. Ten `*bool` feature flag fields with default-ON semantics
3. Per-field accessor methods (`GetEffective*`, `Is*Enabled`) that encapsulate override logic
4. Admin console UI fields in `plugin.json` settings_schema (no custom components)

---

## BL-01: Credential Resolution

```
GetEffectiveOrgID():
  present, val = os.LookupEnv("RTK_ORG_ID")
  if present:
    return val          // env var wins (even if empty)
  return c.CloudflareOrgID

GetEffectiveAPIKey():
  present, val = os.LookupEnv("RTK_API_KEY")
  if present:
    return val          // env var wins (even if empty)
  return c.CloudflareAPIKey
```

**Key difference from current implementation**: `os.LookupEnv` (not `os.Getenv`) is used so that an explicitly-set empty env var (`RTK_ORG_ID=""`) is still respected as an override.

---

## BL-02: Feature Flag Resolution

Each feature flag follows the same pattern:

```
IsRecordingEnabled():
  val, present = os.LookupEnv("RTK_RECORDING_ENABLED")
  if present:
    parsed = strings.EqualFold(val, "true")
    return parsed       // "false" (any case) → false; anything else → false
  if c.RecordingEnabled == nil:
    return true         // nil = never configured = default ON
  return *c.RecordingEnabled
```

Env var parsing rule:
- Case-insensitive `"true"` → `true`
- Anything else (including `"false"`, `"1"`, `"0"`, empty) → `false`

---

## BL-03: Thread-Safe Configuration Access

The existing `configurationLock sync.RWMutex` + `Clone()` pattern is preserved and extended:

```
getConfiguration() → *configuration   // RLock, returns pointer to immutable snapshot
setConfiguration(*configuration)      // Lock, replaces pointer
Clone() → *configuration              // shallow copy (all new fields are value types or pointers)
```

`Clone()` is valid for all new fields:
- `string` fields: copied by value
- `*bool` fields: pointer is copied (the bool value it points to is immutable — never mutated after load)

---

## BL-04: Configuration Change Handling

`OnConfigurationChange` behavior with new fields:

```
OnConfigurationChange():
  prev = getConfiguration()
  configuration = new(configuration)
  LoadPluginConfiguration(configuration)
  setConfiguration(configuration)

  credentialsChanged = prev.GetEffectiveOrgID() != configuration.GetEffectiveOrgID()
                    || prev.GetEffectiveAPIKey() != configuration.GetEffectiveAPIKey()
  if credentialsChanged:
    if GetEffectiveOrgID() != "" && GetEffectiveAPIKey() != "":
      p.rtkClient = rtkclient.NewClient(...)
      p.reRegisterWebhook()
    else:
      p.rtkClient = nil

  // Feature flag changes require no additional action.
  // Flags are read on-demand via getConfiguration().
```

---

## BL-05: Plugin Enabled Check

The `/config/status` endpoint returns whether the plugin is ready (credentials configured). This check must use `GetEffective*()` methods to respect env var overrides:

```
IsPluginEnabled():
  return GetEffectiveOrgID() != "" && GetEffectiveAPIKey() != ""
```

---

## BL-06: FeatureFlags Snapshot Construction

When building the API response for `/config/status` or `/config/admin-status`:

```
BuildFeatureFlags(cfg *configuration) FeatureFlags:
  return FeatureFlags{
    Recording:    cfg.IsRecordingEnabled(),
    ScreenShare:  cfg.IsScreenShareEnabled(),
    Polls:        cfg.IsPollsEnabled(),
    Transcription: cfg.IsTranscriptionEnabled(),
    WaitingRoom:  cfg.IsWaitingRoomEnabled(),
    Video:        cfg.IsVideoEnabled(),
    Chat:         cfg.IsChatEnabled(),
    Plugins:      cfg.IsPluginsEnabled(),
    Participants: cfg.IsParticipantsEnabled(),
    RaiseHand:    cfg.IsRaiseHandEnabled(),
  }
```

---

## BL-07: Admin Console UI (plugin.json settings_schema)

All admin console settings are declared in `plugin.json`. No custom React components are registered.

### API Key Masking

`CloudflareAPIKey` uses `"secret": true` in `plugin.json`. Mattermost's built-in behavior:
- The value is stored encrypted in the Mattermost server config
- The System Console never sends the plaintext value back to the browser
- The admin must re-enter the key to change it (field appears empty)

This satisfies Q4 (always `********`, re-enter to change) **without any custom component**.

### Feature Flag Toggles

Each feature flag is declared as `type: "bool"` with a `default_value` of `"true"`. This ensures the System Console shows the toggle as ON when the admin has not yet configured the flag.

### Env Var Indicator

Env var override indicators (Q5) are **backend-only** for this unit. The System Console does not show which fields are overridden by env vars — the admin must consult server logs or documentation. This is acceptable given the low UI requirement (user feedback: "UIにあまりこだわりはない").
