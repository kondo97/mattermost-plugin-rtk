# Logical Components — Unit 5: Admin & Config

## Component Map

```
plugin.json (settings_schema)
    |
    | Mattermost config load
    v
configuration struct (server/configuration.go)
    |
    +-- GetEffectiveOrgID()      os.LookupEnv("RTK_ORG_ID")
    +-- GetEffectiveAPIKey()     os.LookupEnv("RTK_API_KEY")
    +-- IsRecordingEnabled()     os.LookupEnv("RTK_RECORDING_ENABLED") + *bool nil check
    +-- IsScreenShareEnabled()   os.LookupEnv("RTK_SCREEN_SHARE_ENABLED") + *bool nil check
    +-- IsPollsEnabled()         os.LookupEnv("RTK_POLLS_ENABLED") + *bool nil check
    +-- IsTranscriptionEnabled() os.LookupEnv("RTK_TRANSCRIPTION_ENABLED") + *bool nil check
    +-- IsWaitingRoomEnabled()   os.LookupEnv("RTK_WAITING_ROOM_ENABLED") + *bool nil check
    +-- IsVideoEnabled()         os.LookupEnv("RTK_VIDEO_ENABLED") + *bool nil check
    +-- IsChatEnabled()          os.LookupEnv("RTK_CHAT_ENABLED") + *bool nil check
    +-- IsPluginsEnabled()       os.LookupEnv("RTK_PLUGINS_ENABLED") + *bool nil check
    +-- IsParticipantsEnabled()  os.LookupEnv("RTK_PARTICIPANTS_ENABLED") + *bool nil check
    +-- IsRaiseHandEnabled()     os.LookupEnv("RTK_RAISE_HAND_ENABLED") + *bool nil check
    +-- Clone()                  shallow copy (extended, no logic change)
    |
    v
OnConfigurationChange (server/configuration.go)
    |
    +-- Credential change detection via GetEffective*()
    +-- RTK client re-init (if credentials changed and non-empty)
    +-- Webhook re-registration (if credentials changed and non-empty)
    |
    v
API handlers (server/api/config.go) — existing, consuming GetEffective*() / Is*Enabled()
```

---

## Component: configuration struct (extended)

**File**: `server/configuration.go`

**New fields added**:

```go
type configuration struct {
    // Existing fields
    CloudflareOrgID  string `json:"CloudflareOrgID"`
    CloudflareAPIKey string `json:"CloudflareAPIKey"`

    // New feature flag fields (all *bool, default nil = ON)
    RecordingEnabled    *bool `json:"RecordingEnabled"`
    ScreenShareEnabled  *bool `json:"ScreenShareEnabled"`
    PollsEnabled        *bool `json:"PollsEnabled"`
    TranscriptionEnabled *bool `json:"TranscriptionEnabled"`
    WaitingRoomEnabled  *bool `json:"WaitingRoomEnabled"`
    VideoEnabled        *bool `json:"VideoEnabled"`
    ChatEnabled         *bool `json:"ChatEnabled"`
    PluginsEnabled      *bool `json:"PluginsEnabled"`
    ParticipantsEnabled *bool `json:"ParticipantsEnabled"`
    RaiseHandEnabled    *bool `json:"RaiseHandEnabled"`
}
```

**New methods added** (12 total):

| Method | Env Var | Returns |
|---|---|---|
| `GetEffectiveOrgID() string` | `RTK_ORG_ID` | env var if present, else config |
| `GetEffectiveAPIKey() string` | `RTK_API_KEY` | env var if present, else config |
| `IsRecordingEnabled() bool` | `RTK_RECORDING_ENABLED` | env var if present, else *bool or true |
| `IsScreenShareEnabled() bool` | `RTK_SCREEN_SHARE_ENABLED` | env var if present, else *bool or true |
| `IsPollsEnabled() bool` | `RTK_POLLS_ENABLED` | env var if present, else *bool or true |
| `IsTranscriptionEnabled() bool` | `RTK_TRANSCRIPTION_ENABLED` | env var if present, else *bool or true |
| `IsWaitingRoomEnabled() bool` | `RTK_WAITING_ROOM_ENABLED` | env var if present, else *bool or true |
| `IsVideoEnabled() bool` | `RTK_VIDEO_ENABLED` | env var if present, else *bool or true |
| `IsChatEnabled() bool` | `RTK_CHAT_ENABLED` | env var if present, else *bool or true |
| `IsPluginsEnabled() bool` | `RTK_PLUGINS_ENABLED` | env var if present, else *bool or true |
| `IsParticipantsEnabled() bool` | `RTK_PARTICIPANTS_ENABLED` | env var if present, else *bool or true |
| `IsRaiseHandEnabled() bool` | `RTK_RAISE_HAND_ENABLED` | env var if present, else *bool or true |

**Existing methods updated**:
- `GetEffectiveOrgID()` — add `os.LookupEnv` check (currently returns raw field)
- `GetEffectiveAPIKey()` — add `os.LookupEnv` check (currently returns raw field)
- `Clone()` — no logic change needed; shallow copy valid for `*bool` fields
- `OnConfigurationChange()` — update credential change detection to use `GetEffective*()`

---

## Component: plugin.json (settings_schema)

**File**: `plugin.json`

**Changes**:
- `CloudflareAPIKey` — already has `"secret": true` (no change needed)
- Add 10 `type: "bool"` entries with `"default": "true"` for feature flags

**No new React components.** No changes to `webapp/src/index.tsx`.

---

## Component: configuration_test.go (new)

**File**: `server/configuration_test.go`

**Test cases per method**:
- Credential methods: 3 cases each (env set non-empty, env set empty, env not set)
- Feature flag methods: 4 cases each (env "true", env "false", nil pointer, &false)
- `OnConfigurationChange`: credential change detection test

**Total test cases**: ~50 (12 methods × ~3-4 cases + OnConfigurationChange)
