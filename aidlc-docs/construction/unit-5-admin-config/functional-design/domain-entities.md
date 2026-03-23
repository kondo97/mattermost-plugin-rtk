# Domain Entities — Unit 5: Admin & Config

## Entity: configuration

The central configuration struct for the plugin. Stored in Mattermost server config and loaded via `LoadPluginConfiguration`.

### Fields

#### Credentials

| Field | Go Type | JSON Key | Env Var | Description |
|---|---|---|---|---|
| `CloudflareOrgID` | `string` | `CloudflareOrgID` | `RTK_ORG_ID` | Cloudflare Organization ID |
| `CloudflareAPIKey` | `string` | `CloudflareAPIKey` | `RTK_API_KEY` | Cloudflare API Key (stored encrypted via `secret: true`) |

#### Feature Flags

| Field | Go Type | JSON Key | Env Var | Default |
|---|---|---|---|---|
| `RecordingEnabled` | `*bool` | `RecordingEnabled` | `RTK_RECORDING_ENABLED` | ON (`nil`) |
| `ScreenShareEnabled` | `*bool` | `ScreenShareEnabled` | `RTK_SCREEN_SHARE_ENABLED` | ON (`nil`) |
| `PollsEnabled` | `*bool` | `PollsEnabled` | `RTK_POLLS_ENABLED` | ON (`nil`) |
| `TranscriptionEnabled` | `*bool` | `TranscriptionEnabled` | `RTK_TRANSCRIPTION_ENABLED` | ON (`nil`) |
| `WaitingRoomEnabled` | `*bool` | `WaitingRoomEnabled` | `RTK_WAITING_ROOM_ENABLED` | ON (`nil`) |
| `VideoEnabled` | `*bool` | `VideoEnabled` | `RTK_VIDEO_ENABLED` | ON (`nil`) |
| `ChatEnabled` | `*bool` | `ChatEnabled` | `RTK_CHAT_ENABLED` | ON (`nil`) |
| `PluginsEnabled` | `*bool` | `PluginsEnabled` | `RTK_PLUGINS_ENABLED` | ON (`nil`) |
| `ParticipantsEnabled` | `*bool` | `ParticipantsEnabled` | `RTK_PARTICIPANTS_ENABLED` | ON (`nil`) |
| `RaiseHandEnabled` | `*bool` | `RaiseHandEnabled` | `RTK_RAISE_HAND_ENABLED` | ON (`nil`) |

### Semantics of `*bool` for Feature Flags

| Stored Value | Meaning | `IsXxxEnabled()` returns |
|---|---|---|
| `nil` (not in JSON) | Never configured — default ON | `true` |
| `&true` | Explicitly enabled by admin | `true` |
| `&false` | Explicitly disabled by admin | `false` |
| env var `"true"` (any case) | Overridden ON by env var | `true` |
| env var `"false"` (any case) | Overridden OFF by env var | `false` |

### Env Var Precedence Rule

- **Credentials**: If `os.LookupEnv(envVar)` reports the variable is present (even if empty), the env var value is used. Config value is ignored.
- **Feature Flags**: If env var is set (present in environment), parse as `true`/`false` (case-insensitive). If parsing fails, log a warning and fall back to config value.

---

## Value Object: FeatureFlags

A read-only snapshot of all feature flag states, returned by `/config/status` and `/config/admin-status` API endpoints.

```go
type FeatureFlags struct {
    Recording    bool `json:"recording"`
    ScreenShare  bool `json:"screenShare"`
    Polls        bool `json:"polls"`
    Transcription bool `json:"transcription"`
    WaitingRoom  bool `json:"waitingRoom"`
    Video        bool `json:"video"`
    Chat         bool `json:"chat"`
    Plugins      bool `json:"plugins"`
    Participants bool `json:"participants"`
    RaiseHand    bool `json:"raiseHand"`
}
```

Populated by calling `Is*Enabled()` on the current configuration.
