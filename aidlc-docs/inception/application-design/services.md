# Service Interfaces

This document defines the external-facing service interfaces used by the plugin.

## RTKClient Interface

**Package**: `server/rtkclient`
**Purpose**: Abstracts all communication with the Cloudflare RTK REST API, enabling mock-based unit testing without real API calls (NFR-06).

```go
type RTKClient interface {
    // CreateMeeting creates a new RTK meeting session.
    // preset: the host preset name (e.g., "group_call_host")
    // Returns the created meeting with its RTK-assigned ID.
    CreateMeeting(preset string) (*Meeting, error)

    // GenerateToken generates a participant JWT for an existing meeting.
    // meetingID: the RTK meeting ID returned by CreateMeeting
    // userID: Mattermost user ID (used as participant identifier)
    // preset: "group_call_host" or "group_call_participant"
    // Returns a signed JWT token string for use with the RTK SDK.
    GenerateToken(meetingID, userID, preset string) (string, error)

    // EndMeeting terminates an RTK meeting and disconnects all participants.
    EndMeeting(meetingID string) error
}

type Meeting struct {
    ID string `json:"id"`
}
```

**Transport**: HTTPS only (NFR-01)
**Auth**: HTTP Basic Auth — `orgID:apiKey` (NFR-01)
**Base URL**: `https://api.realtime.cloudflare.com/v2`

---

## KVStore Interface

**Package**: `server/store/kvstore`
**Purpose**: Abstracts all Mattermost KVStore access for call session persistence and heartbeat tracking (NFR-06).

```go
type KVStore interface {
    // --- Call Session ---

    // GetCallByChannel returns the active call for a channel, or nil if none.
    GetCallByChannel(channelID string) (*CallSession, error)

    // GetCallByID returns a call session by its RTK meeting ID.
    GetCallByID(callID string) (*CallSession, error)

    // SaveCall persists a new call session under both key patterns:
    //   call:channel:{channelID}
    //   call:id:{callID}
    SaveCall(session *CallSession) error

    // UpdateCallParticipants overwrites the participants list for a call.
    UpdateCallParticipants(callID string, participants []string) error

    // EndCall sets end_at to the provided Unix timestamp (ms).
    EndCall(callID string, endAt int64) error

    // --- Heartbeat ---

    // SetHeartbeat records the latest heartbeat timestamp for a participant.
    // Key: heartbeat:{callID}:{userID}
    SetHeartbeat(callID, userID string, ts int64) error

    // GetStaleParticipants returns userIDs whose last heartbeat is older than cutoff (Unix ms).
    GetStaleParticipants(callID string, cutoff int64) ([]string, error)

    // --- Mobile VoIP Tokens ---

    // StoreVoIPToken persists an iOS VoIP push token for a user.
    // Key: voip:{userID}, Value: "apple_voip:{token}"
    StoreVoIPToken(userID, token string) error

    // GetVoIPToken retrieves the stored VoIP token for a user.
    GetVoIPToken(userID string) (string, error)
}

type CallSession struct {
    CallID       string   `json:"call_id"`
    ChannelID    string   `json:"channel_id"`
    PostID       string   `json:"post_id"`
    CreatorID    string   `json:"creator_id"`
    StartAt      int64    `json:"start_at"`   // Unix ms
    EndAt        int64    `json:"end_at"`     // Unix ms; 0 = active
    Participants []string `json:"participants"`
}
```

---

## PluginAPI Interface (internal)

**Purpose**: Allows API handlers to call Plugin business logic without importing the Plugin struct directly, enabling unit testing of handlers.

```go
// Located in server/api/handler.go
type PluginAPI interface {
    CreateCall(userID, channelID string) (*CreateCallResponse, error)
    JoinCall(userID, callID string) (*JoinCallResponse, error)
    LeaveCall(userID, callID string) error
    EndCall(userID, callID string) error
    HeartbeatCall(userID, callID string) error
    GetConfigStatus() ConfigStatus
    GetAdminConfigStatus() AdminConfigStatus
    RegisterVoIPToken(userID, token string) error
    DismissNotification(userID, callID string) error
}

type CreateCallResponse struct {
    CallID       string       `json:"call_id"`
    Token        string       `json:"token"`
    FeatureFlags FeatureFlags `json:"feature_flags"`
}

type JoinCallResponse struct {
    Token        string       `json:"token"`
    FeatureFlags FeatureFlags `json:"feature_flags"`
}

type FeatureFlags struct {
    Polls          bool `json:"polls_enabled"`
    Plugins        bool `json:"plugins_enabled"`
    Chat           bool `json:"chat_enabled"`
    ScreenShare    bool `json:"screenshare_enabled"`
    Participants   bool `json:"participants_enabled"`
    Recording      bool `json:"recording_enabled"`
    AITranscription bool `json:"ai_transcription_enabled"`
    WaitingRoom    bool `json:"waiting_room_enabled"`
    Video          bool `json:"video_enabled"`
    RaiseHand      bool `json:"raise_hand_enabled"`
}
```

---

## WebSocket Event Contracts

Events emitted by `p.API.PublishWebSocketEvent`:

| Event Name | Payload Fields |
|---|---|
| `custom_cf_call_started` | `call_id`, `channel_id`, `creator_id`, `start_at` |
| `custom_cf_call_ended` | `call_id`, `channel_id`, `end_at`, `duration_ms` |
| `custom_cf_user_joined` | `call_id`, `user_id` |
| `custom_cf_user_left` | `call_id`, `user_id` |
| `custom_cf_notification_dismissed` | `call_id`, `user_id` |

All events are broadcast to the entire channel (`broadcast.ChannelId`).
