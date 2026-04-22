# Mattermost RTK Plugin — Architecture & Implementation Guide

> **Branch**: `main`
> **Plugin ID**: `com.kondo97.mattermost-plugin-rtk`
> **Minimum Mattermost Version**: 10.11.0

---

## Table of Contents

1. [Overview](#1-overview)
2. [System Architecture](#2-system-architecture)
3. [Server-Side Architecture](#3-server-side-architecture)
   - 3.1 [Plugin Lifecycle](#31-plugin-lifecycle)
   - 3.2 [HTTP Routing](#32-http-routing)
   - 3.3 [Call Business Logic](#33-call-business-logic)
   - 3.4 [Cloudflare RTK Client](#34-cloudflare-rtk-client)
   - 3.5 [KVStore](#35-kvstore)
   - 3.6 [Configuration Management](#36-configuration-management)
   - 3.7 [RTK Webhook](#37-rtk-webhook)
4. [Frontend Architecture](#4-frontend-architecture)
   - 4.1 [Dual-Bundle Build](#41-dual-bundle-build)
   - 4.2 [Redux State Management](#42-redux-state-management)
   - 4.3 [WebSocket Event Handling](#43-websocket-event-handling)
   - 4.4 [UI Components](#44-ui-components)
   - 4.5 [Standalone Call Page](#45-standalone-call-page)
5. [Data Models](#5-data-models)
6. [Key Flow Implementations](#6-key-flow-implementations)
   - 6.1 [Start a Call](#61-start-a-call)
   - 6.2 [Join a Call](#62-join-a-call)
   - 6.3 [Leave / Auto-End](#63-leave--auto-end)
   - 6.4 [Host-Initiated End](#64-host-initiated-end)
   - 6.5 [RTK Webhook Participant Cleanup](#65-rtk-webhook-participant-cleanup)
7. [API Reference](#7-api-reference)
8. [WebSocket Events](#8-websocket-events)
9. [Configuration Reference](#9-configuration-reference)
10. [Security Design](#10-security-design)
11. [Build Configuration](#11-build-configuration)
12. [Directory Structure](#12-directory-structure)

---

## 1. Overview

This plugin adds video and voice calling to Mattermost channels using **Cloudflare RealtimeKit (RTK)**.

| Item | Details |
|------|---------|
| Backend | Go 1.25 |
| Frontend | React 18 + TypeScript (Vite build) |
| Call Engine | Cloudflare RealtimeKit v2 API |
| Session Persistence | Mattermost KVStore |
| Real-time Sync | Mattermost WebSocket events + RTK Webhook |

---

## 2. System Architecture

```
+---------------------------------------------------------------------+
|  Browser (Mattermost Webapp)                                        |
|                                                                     |
|  +-----------------------------+  +------------------------------+  |
|  |  Main Bundle (main.js)      |  |  Call Page (call.js)         |  |
|  |  ─────────────────────────  |  |  ──────────────────────────  |  |
|  |  ChannelHeaderButton        |  |  CallPage.tsx                |  |
|  |  CallPost                   |  |  useRealtimeKitClient()      |  |
|  |  ToastBar                   |  |  RtkMeeting UI               |  |
|  |  FloatingWidget <──RTK SDK──+──+─ /call?token=JWT             |  |
|  |  IncomingCallNotification   |  |  beforeunload -> POST /leave |  |
|  |  AdminConfig                |  +------------------------------+  |
|  |  Redux (calls_slice)        |                                  |  |
|  |  WebSocket handlers         |                                  |  |
|  +-------------+--------------+                                  |  |
+----------------|--------------------------------------------------|--+
                 | REST API / WebSocket
                 v
+---------------------------------------------------------------------+
|  Mattermost Server                                                  |
|                                                                     |
|  +----------------------------------------------------------------+ |
|  |  Go Plugin (com.kondo97.mattermost-plugin-rtk)                 | |
|  |                                                                | |
|  |  plugin.go          OnActivate / OnDeactivate                  | |
|  |  configuration.go   Config management + env var overrides      | |
|  |  calls.go           CreateCall / JoinCall / LeaveCall / End    | |
|  |  api.go             gorilla/mux router                         | |
|  |  api_calls.go       POST/GET /calls, /token, /leave, DELETE    | |
|  |  api_config.go      GET /config/status, /admin-status          | |
|  |  api_mobile.go      POST /calls/{id}/dismiss                   | |
|  |  api_static.go      GET /call, /call.js, /worker.js            | |
|  |  api_webhook.go     POST /api/v1/webhook/rtk (HMAC verify)     | |
|  |  rtkclient/         Cloudflare RTK API client                  | |
|  |  store/kvstore/     KVStore abstraction layer                  | |
|  +----------------+----------------------------+------------------+ |
|                   |                            |                    |
|         Mattermost KVStore          PublishWebSocketEvent           |
+-------------------|----------------------------|--------------------+
                    |                            | WebSocket
                    v                            v (channel members)
        +-----------------------+   +---------------------------+
        |  KVStore              |   |  Browser WebSocket        |
        |  call:id:{id}         |   |  custom_{pluginID}_       |
        |  call:channel:{c}     |   |  call_started, etc.       |
        |  call:meeting:{m}     |   +---------------------------+
        |  webhook:id/secret    |
        +-----------------------+
                    ^
                    | Webhook (HMAC-SHA256)
        +-----------------------+
        |  Cloudflare RTK       |
        |  api.realtime.        |
        |  cloudflare.com/v2    |
        +-----------------------+
```

---

## 3. Server-Side Architecture

### 3.1 Plugin Lifecycle

**Entry point**: `server/plugin.go`

```go
type Plugin struct {
    plugin.MattermostPlugin
    kvStore           kvstore.KVStore     // KVStore client
    rtkClient         rtkclient.RTKClient // Cloudflare RTK API (nil = not configured)
    client            *pluginapi.Client   // Mattermost pluginapi
    commandClient     command.Command     // Slash command handler
    router            *mux.Router         // HTTP router
    callMu            sync.Mutex          // Guards all call state mutations
    stopCleanup       chan struct{}        // Stops cleanup goroutine
    configurationLock sync.RWMutex        // Guards configuration pointer
    configuration     *configuration      // Current config (immutable pointer)
}
```

| Hook | Behavior |
|------|----------|
| `OnActivate()` | Initialize KVStore, RTKClient, router; register webhook; start cleanup goroutine |
| `OnDeactivate()` | Stop cleanup goroutine |
| `OnConfigurationChange()` | Reload config; re-initialize RTKClient and re-register webhook if credentials changed |
| `ExecuteCommand()` | Handle `/hello` slash command (starter template) |

**Webhook registration logic** (`registerWebhookIfNeeded`):
- Skip if both `webhook:id` and `webhook:secret` exist in KVStore
- If either is missing, register with the RTK API and persist the returned ID and secret
- Failures are best-effort (log warning only, does not block activation)

**On credential change** (`reRegisterWebhook`):
1. Delete existing webhook via RTK API
2. Clear `webhook:id` and `webhook:secret` in KVStore
3. Call `registerWebhookIfNeeded`

---

### 3.2 HTTP Routing

**Router**: `server/api.go` (gorilla/mux)

```
/call                          <- No auth (static HTML)
/call.js                       <- No auth (static JS)
/worker.js                     <- No auth (static JS)
/api/v1/webhook/rtk            <- No Mattermost auth (HMAC-SHA256 verified in handler)

/api/v1/* <- MattermostAuthorizationRequired middleware (Mattermost-User-ID header required)
  POST   /calls                    Start a call
  GET    /calls/{id}               Get call state
  POST   /calls/{id}/token         Join a call (issue token)
  POST   /calls/{id}/leave         Leave a call
  DELETE /calls/{id}               End a call (creator only)
  GET    /config/status            Config status (all users)
  GET    /config/admin-status      Config status (system admins only)
  POST   /calls/{id}/dismiss       Dismiss incoming call notification
```

**Auth middleware**:
```go
func MattermostAuthorizationRequired(next http.Handler) http.Handler {
    // Returns 401 if Mattermost-User-ID header is empty.
    // Mattermost Server sets this header on all authenticated requests.
}
```

---

### 3.3 Call Business Logic

**Implementation**: `server/calls.go`

All call state mutations acquire `callMu sync.Mutex` before executing.

#### CreateCall

```
callMu.Lock()
  1. GetCallByChannel -> ErrCallAlreadyActive if active call exists
  2. rtkClient.CreateMeeting() -> abort on failure (no KVStore writes)
  3. rtkClient.GenerateToken(meetingID, userID, displayName, "group_call_host")
  4. Build CallSession (UUID, creator=participants[0], StartAt=nowMs())
  5. kvStore.SaveCall(session) + AddActiveCallID
  6. API.CreatePost(type="custom_cf_call")  [best-effort]
  7. kvStore.SaveCall(session) to persist PostID  [best-effort]
  8. PublishWebSocketEvent("call_started", ...) broadcast to channel
callMu.Unlock()
Returns: (session, token.Token, nil)
```

#### JoinCall

```
callMu.Lock()
  1. GetCallByID -> ErrCallNotFound if nil or EndAt != 0
  2. rtkClient.GenerateToken(meetingID, userID, displayName, "group_call_participant")
  3. Append userID to Participants (deduplicated)
  4. kvStore.UpdateCallParticipants
  5. updatePostParticipants  [best-effort]
  6. PublishWebSocketEvent("user_joined", ...) broadcast to channel
callMu.Unlock()
Returns: (session, token.Token, nil)
```

#### LeaveCall

```
callMu.Lock()
  1. GetCallByID -> no-op if nil or EndAt != 0 (idempotent)
  2. Remove userID from Participants
  3. kvStore.UpdateCallParticipants
  4. updatePostParticipants  [best-effort]
  5. PublishWebSocketEvent("user_left", ...)
  6. If Participants is now empty -> endCallInternal()  [auto-end]
callMu.Unlock()
```

#### EndCall

```
callMu.Lock()
  1. GetCallByID -> ErrCallNotFound if nil or EndAt != 0
  2. ErrUnauthorized if CreatorID != requestingUserID
  3. endCallInternal(session)
callMu.Unlock()
```

#### endCallInternal (shared termination logic)

```
  1. kvStore.EndCall(callID, nowMs()) + RemoveActiveCallID
  2. rtkClient.EndMeeting(meetingID)  [best-effort]
  3. API.UpdatePost: write end_at and duration_ms to post props  [best-effort]
  4. PublishWebSocketEvent("call_ended", {call_id, channel_id, end_at, duration_ms})
```

**Sentinel errors** (`server/errors.go`):

| Constant | HTTP Status | Description |
|----------|-------------|-------------|
| `ErrCallAlreadyActive` | 409 | Active call already exists in the channel |
| `ErrCallNotFound` | 404 | Call not found or already ended |
| `ErrNotParticipant` | 403 | User is not a participant (currently unused) |
| `ErrUnauthorized` | 403 | Non-creator attempted to end the call |
| `ErrRTKNotConfigured` | 503 | Cloudflare credentials not configured |

---

### 3.4 Cloudflare RTK Client

**Interface**: `server/rtkclient/interface.go`
**Implementation**: `server/rtkclient/client.go`

```go
type RTKClient interface {
    CreateMeeting() (*Meeting, error)
    GenerateToken(meetingID, userID, displayName, preset string) (*Token, error)
    EndMeeting(meetingID string) error
    RegisterWebhook(url string, events []string) (id, secret string, err error)
    DeleteWebhook(webhookID string) error
    GetMeetingParticipants(meetingID string) ([]string, error)
}
```

**Transport details**:

| Item | Value |
|------|-------|
| Base URL | `https://api.realtime.cloudflare.com/v2` |
| Authentication | HTTP Basic Auth (`orgID:apiKey`) |
| Timeout | 10 seconds |
| Response format | `{ "success": bool, "data": T }` |

**API endpoint mapping**:

| Method | RTK API | Description |
|--------|---------|-------------|
| `CreateMeeting()` | `POST /meetings` | Create a meeting |
| `GenerateToken()` | `POST /meetings/{id}/participants` | Add participant + issue JWT |
| `EndMeeting()` | `DELETE /meetings/{id}` | Terminate meeting |
| `RegisterWebhook()` | `POST /webhooks` | Register webhook |
| `DeleteWebhook()` | `DELETE /webhooks/{id}` | Remove webhook |
| `GetMeetingParticipants()` | `GET /meetings/{id}/active-participants` | List active participants |

**RTK presets**:

| Role | Preset Name |
|------|-------------|
| Call creator (host) | `group_call_host` |
| Participant | `group_call_participant` |

---

### 3.5 KVStore

**Interface**: `server/store/kvstore/kvstore.go`
**Implementation**: `server/store/kvstore/calls.go`

**Key schema**:

| Key Pattern | Value | Description |
|-------------|-------|-------------|
| `call:channel:{channelID}` | CallSession (JSON) | Active call in a channel |
| `call:id:{callID}` | CallSession (JSON) | Call lookup by ID |
| `call:meeting:{meetingID}` | CallSession (JSON) | Call lookup by RTK meeting ID |
| `active_calls` | []string (JSON) | List of active call IDs |
| `webhook:id` | string | Registered RTK webhook ID |
| `webhook:secret` | string | RTK webhook signing secret |

**Consistency strategy**: `SaveCall`, `UpdateCallParticipants`, and `EndCall` always update all three keys (`call:channel:*`, `call:id:*`, `call:meeting:*`) atomically within the same operation.

---

### 3.6 Configuration Management

**Implementation**: `server/configuration.go`

**Thread-safe design**:
- `sync.RWMutex` + `Clone()` pattern
- `getConfiguration()` acquires RLock and returns an immutable copy
- `setConfiguration()` acquires Lock before writing

**Environment variable overrides** (strict precedence via `os.LookupEnv`):

| Environment Variable | Config Field |
|---------------------|-------------|
| `RTK_ORG_ID` | CloudflareOrgID |
| `RTK_API_KEY` | CloudflareAPIKey |
| `RTK_RECORDING_ENABLED` | RecordingEnabled |
| `RTK_SCREEN_SHARE_ENABLED` | ScreenShareEnabled |
| `RTK_POLLS_ENABLED` | PollsEnabled |
| `RTK_TRANSCRIPTION_ENABLED` | TranscriptionEnabled |
| `RTK_WAITING_ROOM_ENABLED` | WaitingRoomEnabled |
| `RTK_VIDEO_ENABLED` | VideoEnabled |
| `RTK_CHAT_ENABLED` | ChatEnabled |
| `RTK_PLUGINS_ENABLED` | PluginsEnabled |
| `RTK_PARTICIPANTS_ENABLED` | ParticipantsEnabled |
| `RTK_RAISE_HAND_ENABLED` | RaiseHandEnabled |

**Feature flag defaults**: A `nil` `*bool` field is treated as `true` (enabled). Exception: `WaitingRoomEnabled` defaults to `false` (opt-in).

> Feature flag の詳細な管理表（SDK対応状況・未対応モジュール・既知の問題）は [docs/feature-flags.md](../docs/feature-flags.md) を参照。

**`OnConfigurationChange` flow**:
```
Load new config via LoadPluginConfiguration
If credentials changed:
  |- New credentials present -> Re-initialize RTKClient + re-register webhook
  +- Credentials removed     -> Set RTKClient = nil
```

---

### 3.7 RTK Webhook

**Implementation**: `server/api_webhook.go`

RTK delivers the following events to `POST /api/v1/webhook/rtk`:

| Event | Handling |
|-------|---------|
| `meeting.participantLeft` | `GetCallByMeetingID` -> `LeaveCall(callID, userID)` |
| `meeting.ended` | `GetCallByMeetingID` -> `callMu.Lock()` -> re-read from KVStore -> `endCallInternal()` |
| Others | Ignored (200 OK) |

**Signature verification**:
```
HMAC-SHA256(secret, rawBody) == hex(dyte-signature header)
If secret is empty, always reject (401)
```

**Double-end prevention for `meeting.ended`**:
- Check `EndAt != 0` before acquiring the lock
- After acquiring the lock, re-read from KVStore to guard against TOCTOU race conditions

---

## 4. Frontend Architecture

### 4.1 Dual-Bundle Build

Vite produces **two independent bundles**:

| Bundle | Entry Point | Output | Purpose |
|--------|-------------|--------|---------|
| `main.js` | `src/index.tsx` | `webapp/dist/main.js` | Mattermost plugin main bundle |
| `call.js` | `src/call_page/main.tsx` | `webapp/dist/call.js` | Standalone call page |

**`main.js` externals (provided by Mattermost host)**:
`React`, `ReactDOM`, `Redux`, `ReactRedux`, `ReactIntl`, `PropTypes`, `ReactBootstrap`, `ReactRouterDom`

**`call.js` is fully self-contained**: bundles React and all dependencies.

**CSP workaround** (`workerTimersCspPatch` Vite plugin):
`@cloudflare/realtimekit` depends on `worker-timers`, which spawns a Web Worker from a `blob:` URL blocked by Mattermost's CSP. During the Vite build, the plugin patches the blob URL creation to load the worker from the Go plugin's static endpoint `/plugins/{id}/worker.js`, satisfying CSP's `'self'` directive.

---

### 4.2 Redux State Management

**Implementation**: `src/redux/calls_slice.ts`

```typescript
interface CallsPluginState {
    callsByChannel: Record<string, ActiveCall>; // Active calls keyed by channel ID
    myActiveCall:   MyActiveCall | null;         // Current user's active call + JWT
    incomingCall:   IncomingCall | null;         // Ringing call (DM/GM only)
    pluginEnabled:  boolean;                     // Value from /config/status
}
```

**Type definitions**:

```typescript
interface ActiveCall {
    id: string;             // Call UUID
    channelId: string;
    creatorId: string;
    participants: string[]; // Mattermost userID array
    startAt: number;        // Unix ms
    postId: string;
}

interface MyActiveCall {
    callId: string;
    channelId: string;
    token: string;          // RTK JWT — must NOT be logged
}

interface IncomingCall {
    callId: string;
    channelId: string;
    creatorId: string;
    startAt: number;
}
```

**Selectors** (`src/redux/selectors.ts`):

```typescript
// Plugin state is stored at state["plugins-{pluginId}"]
selectPluginEnabled(state)
selectCallByChannel(channelId)(state)
selectMyActiveCall(state)
selectIncomingCall(state)
selectIsCurrentUserParticipant(channelId, currentUserId)(state)
```

---

### 4.3 WebSocket Event Handling

**Implementation**: `src/redux/websocket_handlers.ts`

Each handler is registered in `initialize()` via `registry.registerWebSocketEventHandler`.

| Server publish name | Client receive name | Handler | Primary action |
|--------------------|---------------------|---------|----------------|
| `call_started` | `custom_{pluginID}_call_started` | `handleCallStarted` | `upsertCall` + `setIncomingCall` for DM/GM |
| `user_joined` | `custom_{pluginID}_user_joined` | `handleUserJoined` | Update participants via `upsertCall` |
| `user_left` | `custom_{pluginID}_user_left` | `handleUserLeft` | Update participants; `clearMyActiveCall` if self |
| `call_ended` | `custom_{pluginID}_call_ended` | `handleCallEnded` | `removeCall` + `clearMyActiveCall` + `clearIncomingCall` |
| `notification_dismissed` | `custom_{pluginID}_notification_dismissed` | `handleNotifDismissed` | `clearIncomingCall` if addressed to self |

> **Naming convention**: The Go server publishes short event names (e.g., `call_started`). The Mattermost server automatically prepends `custom_{pluginID}_` before delivering to browsers. The webapp subscribes using `custom_${manifest.id}_call_started`.

**Payload type guards**: Every handler validates the incoming payload at runtime (`isCallStartedPayload`, etc.) and logs a `console.error` on invalid data rather than throwing.

---

### 4.4 UI Components

#### ChannelHeaderButton (`src/components/channel_header_button/`)

Call button displayed in the channel header.

| State | Display |
|-------|---------|
| No active call | "Start call" (phone icon) |
| Active call, not participating | "Join call" + green dot |
| Current user is participating | "In call" (disabled) + green dot |
| Loading | Spinner icon |

- Start call: `POST /api/v1/calls`
- Join call: `POST /api/v1/calls/{id}/token`
- Already in a different call: shows `SwitchCallModal` before leaving and joining

#### CallPost (`src/components/call_post/`)

Custom post renderer for post type `custom_cf_call`.

- On mount: fetches latest state via `GET /api/v1/calls/{id}` (re-syncs after page reload)
- `EndAt == 0`: renders `CallPostActive` (join button, participant count, elapsed time)
- `EndAt > 0`: renders `CallPostEnded` (end time, call duration)
- Live Redux state is only used when `liveCall.id === post.props.call_id` to prevent incorrect state from other calls in the same channel

#### ToastBar (`src/components/toast_bar/`)

Banner above the message input box.

- Shown when: active call in current channel AND current user is not a participant AND not dismissed
- `dismissed` is component-local state (resets on page reload)
- Join button calls `POST /api/v1/calls/{id}/token`

#### FloatingWidget (`src/components/floating_widget/`)

Floating in-call UI widget inside Mattermost.

- Shown when `myActiveCall` is set in Redux
- Embeds the RTK SDK via `useRealtimeKitClient()` + `RtkMeeting` from `@cloudflare/realtimekit-react`
- Japanese locale uses `rtk_lang_ja.ts` dictionary via `useLanguage()`
- Supports minimize, fullscreen (Escape to exit), and drag-to-reposition (`position: fixed`)
- `beforeunload` sends `POST /leave` via `fetch + keepalive` (custom headers required; `sendBeacon` cannot set them)
- On close: `meeting.leaveRoom()` -> `POST /leave` -> `clearMyActiveCall`
- On init failure: retries up to 3 times (2-second intervals); shows error with retry button after exhaustion

#### IncomingCallNotification (`src/components/incoming_call_notification/`)

Ringing notification displayed in the top-right corner.

- Shown when `incomingCall` is set in Redux (DM/GM channels only)
- Auto-dismisses after 30 seconds
- "Ignore" button: fires `POST /api/v1/calls/{id}/dismiss` -> WebSocket event propagates to all user sessions
- "Join" button: calls `POST /api/v1/calls/{id}/token` -> `setMyActiveCall`

#### SwitchCallModal (`src/components/switch_call_modal/`)

Confirmation dialog shown when a user tries to join a call while already in another.
Shared by ChannelHeaderButton, CallPost, ToastBar, and IncomingCallNotification.

#### EnvVarCredentialSetting (`src/components/admin_config/`)

Replacement renderer for `CloudflareOrgID` and `CloudflareAPIKey` in the Admin Console.

- On mount: calls `GET /api/v1/config/admin-status` to check `org_id_via_env` / `api_key_via_env`
- If set via environment variable: shows read-only text with the env var name, disables the input
- Otherwise: renders a normal `text` or `password` input field

---

### 4.5 Standalone Call Page

**Entry**: `src/call_page/main.tsx`
**Component**: `src/call_page/CallPage.tsx`

A standalone SPA with no Mattermost framework dependencies. Parses URL parameters to initialize the call.

**URL format**:
```
/plugins/com.kondo97.mattermost-plugin-rtk/call
  ?token={RTK JWT}
  &call_id={callID}
  &channel_name={channel name}
  [&embedded=1]
  [&locale=ja]
```

**RTK SDK initialization sequence**:
1. Get `[meeting, initMeeting]` from `useRealtimeKitClient()`
2. Call `initMeeting({ authToken: token, defaults: {audio: true, video: true} })`
3. On failure: retry up to 3 times (2-second intervals)
4. Once `meeting` resolves: render `RtkMeeting` UI component

**Leave on tab close**:
- `beforeunload` fires `fetch + keepalive` (`sendBeacon` cannot set `X-Requested-With` header)
- Skipped when `embedded=1` (inside FloatingWidget iframe — parent handles leave)

**Page title**: Set to `Call in #channel-name` from the `channel_name` URL parameter.

---

## 5. Data Models

### CallSession

```go
type CallSession struct {
    ID           string   `json:"id"`            // UUID (call identifier)
    ChannelID    string   `json:"channel_id"`    // Mattermost channel ID
    CreatorID    string   `json:"creator_id"`    // Host's Mattermost userID
    MeetingID    string   `json:"meeting_id"`    // Cloudflare RTK meeting ID
    Participants []string `json:"participants"`  // Current participant userIDs (deduplicated)
    StartAt      int64    `json:"start_at"`      // Start Unix timestamp (ms)
    EndAt        int64    `json:"end_at"`        // End Unix timestamp (ms); 0 = active
    PostID       string   `json:"post_id"`       // ID of the custom_cf_call post
}
```

**Status**:
- `EndAt == 0`: active
- `EndAt > 0`: ended

### Custom Post Props (type: `custom_cf_call`)

```json
{
  "call_id":      "uuid",
  "channel_id":   "string",
  "creator_id":   "string",
  "participants": ["userID"],
  "start_at":     1234567890000,
  "end_at":       0,
  "duration_ms":  720000
}
```

---

## 6. Key Flow Implementations

### 6.1 Start a Call

```
User           ChannelHeaderButton        Go Plugin              Cloudflare RTK
  |                    |                      |                        |
  |--click------------>|                      |                        |
  |                    |--POST /api/v1/calls->|                        |
  |                    |   {channel_id}       |--POST /meetings------->|
  |                    |                      |<--{meetingID}----------|
  |                    |                      |--POST /meetings/{id}/->|
  |                    |                      |    participants        |
  |                    |                      |  (host preset)         |
  |                    |                      |<--{JWT token}----------|
  |                    |                      |                        |
  |                    |                      |--SaveCall--> KVStore
  |                    |                      |--CreatePost (custom_cf_call)
  |                    |                      |--PublishWebSocketEvent("call_started")
  |                    |<--201 {call, token}--|
  |                    |                      |
  |                    | dispatch(upsertCall)  |
  |                    | dispatch(setMyActiveCall{token})
  |                    |                      |
  |<--FloatingWidget---|                      |
  |  (RTK SDK init)    |                      |
```

### 6.2 Join a Call

```
Other user     ChannelHeaderButton/ToastBar   Go Plugin         Cloudflare RTK
    |                    |                        |                    |
    | (call_started WS)  |                        |                    |
    |<------------------- Redux: upsertCall       |                    |
    |                    |                        |                    |
    |--click------------>|                        |                    |
    |                    |--POST /calls/{id}/token->                   |
    |                    |                        |--POST /participants>|
    |                    |                        |  (participant preset)|
    |                    |                        |<--{JWT token}-------|
    |                    |                        |--UpdateCallParticipants
    |                    |                        |--PublishWebSocketEvent("user_joined")
    |                    |<--200 {call, token}----|
    |                    |                        |
    |<--FloatingWidget---|                        |
```

### 6.3 Leave / Auto-End

```
FloatingWidget (x button or beforeunload)
  |
  |--meeting.leaveRoom() --> RTK SDK fires "roomLeft" event
  |--POST /api/v1/calls/{id}/leave
  |     |
  |  LeaveCall():
  |    Remove userID from Participants
  |    UpdateCallParticipants
  |    PublishWebSocketEvent("user_left")
  |    If Participants empty -> endCallInternal()
  |          EndCall in KVStore
  |          rtkClient.EndMeeting (best-effort)
  |          UpdatePost (end_at, duration_ms)
  |          PublishWebSocketEvent("call_ended")
```

### 6.4 Host-Initiated End

```
CallPost end button (or other UI)
  |
  |--DELETE /api/v1/calls/{id}
  |     |
  |  EndCall():
  |    Verify CreatorID (403 if not creator)
  |    endCallInternal()
```

### 6.5 RTK Webhook Participant Cleanup

```
Cloudflare RTK
    |
    |--POST /api/v1/webhook/rtk
    |   dyte-signature: HMAC-SHA256(secret, body)
    |
    +-- meeting.participantLeft:
    |     GetCallByMeetingID
    |     LeaveCall(session.ID, participant.customParticipantId)
    |       (LeaveCall acquires callMu internally)
    |
    +-- meeting.ended:
          GetCallByMeetingID (pre-lock check)
          callMu.Lock()
          GetCallByID (re-read to guard against TOCTOU race)
          endCallInternal()
          callMu.Unlock()
```

---

## 7. API Reference

### POST /api/v1/calls — Start a call

**Request**:
```json
{ "channel_id": "string" }
```

**Response** (201 Created):
```json
{
  "call": {
    "id": "uuid",
    "channel_id": "string",
    "creator_id": "string",
    "meeting_id": "string",
    "participants": ["userID"],
    "start_at": 1234567890000,
    "end_at": 0,
    "post_id": "string"
  },
  "token": "RTK JWT"
}
```

**Errors**: 400 (missing channel_id) / 409 (call already active) / 503 (RTK not configured)

---

### POST /api/v1/calls/{id}/token — Join a call

**Response** (200 OK):
```json
{ "call": { ...CallSession... }, "token": "RTK JWT" }
```

**Errors**: 404 (call not found or ended) / 503 (RTK not configured)

---

### GET /api/v1/calls/{id} — Get call state

**Response** (200 OK): `CallSession` object directly

---

### POST /api/v1/calls/{id}/leave — Leave a call

**Response**: 200 OK (idempotent)

---

### DELETE /api/v1/calls/{id} — End a call (creator only)

**Response**: 200 OK
**Errors**: 403 (not the creator) / 404 (call not found)

---

### GET /api/v1/config/status — Config status (all users)

**Response**:
```json
{
  "enabled": true,
  "feature_flags": {
    "recording": true, "screenShare": true, "polls": true,
    "transcription": true, "waitingRoom": false, "video": true,
    "chat": true, "plugins": true, "participants": true, "raiseHand": true
  }
}
```

---

### GET /api/v1/config/admin-status — Config status (system admins)

**Response**:
```json
{
  "enabled": true,
  "org_id_via_env": false,
  "api_key_via_env": true,
  "cloudflare_org_id": "abc123",
  "feature_flags": { ...same as above... }
}
```

**Errors**: 403 (not a system admin)

---

### POST /api/v1/calls/{id}/dismiss — Dismiss incoming call notification

No RTK JWT required. Publishes `notification_dismissed` WebSocket event to the requesting user only.
**Response**: 200 OK (idempotent)

---

## 8. WebSocket Events

> The server calls `PublishWebSocketEvent` with short names. Mattermost prepends `custom_{pluginID}_` before delivering to clients.
> Full client-side name: `custom_com.kondo97.mattermost-plugin-rtk_{short_name}`

### call_started

```json
{
  "call_id": "string",
  "channel_id": "string",
  "creator_id": "string",
  "participants": ["userID"],
  "start_at": 1234567890000,
  "post_id": "string"
}
```
Broadcast scope: all channel members

### user_joined

```json
{
  "call_id": "string",
  "channel_id": "string",
  "user_id": "string",
  "participants": ["userID"]
}
```
Broadcast scope: all channel members
> **Note**: Also emitted when an already-participating user re-joins (e.g., page reload). The `participants` list in KVStore remains deduplicated.

### user_left

```json
{
  "call_id": "string",
  "channel_id": "string",
  "user_id": "string",
  "participants": ["userID"]
}
```
Broadcast scope: all channel members

### call_ended

```json
{
  "call_id": "string",
  "channel_id": "string",
  "end_at": 1234567890000,
  "duration_ms": 720000
}
```
Broadcast scope: all channel members

### notification_dismissed

```json
{ "call_id": "string", "user_id": "string" }
```
Broadcast scope: requesting user only (all sessions)

---

## 9. Configuration Reference

Defined in `plugin.json` `settings_schema`. Configurable from the System Console.

| Key | Type | Description |
|-----|------|-------------|
| `CloudflareOrgID` | text | Cloudflare Organization ID |
| `CloudflareAPIKey` | text (secret) | Cloudflare API Key |
| `RecordingEnabled` | bool (default: true) | Allow recording |
| `ScreenShareEnabled` | bool (default: true) | Allow screen sharing |
| `PollsEnabled` | bool (default: true) | Enable polls feature |
| `TranscriptionEnabled` | bool (default: true) | Enable real-time transcription |
| `WaitingRoomEnabled` | bool (default: true) | Enable waiting room |
| `VideoEnabled` | bool (default: true) | Allow camera video |
| `ChatEnabled` | bool (default: true) | Enable in-call chat |
| `PluginsEnabled` | bool (default: true) | Allow third-party plugins |
| `ParticipantsEnabled` | bool (default: true) | Show participants panel |
| `RaiseHandEnabled` | bool (default: true) | Allow raise hand |

Environment variables take **strict precedence** over System Console settings (empty string in env var also overrides).

---

## 10. Security Design

### Authentication & Authorization

| Layer | Mechanism |
|-------|-----------|
| General APIs | `Mattermost-User-ID` header (set by Mattermost on authenticated requests) |
| Admin APIs | `model.PermissionManageSystem` check |
| RTK Webhook | HMAC-SHA256 signature verification (`dyte-signature` header) |
| Static files | No authentication (`call.html`, `call.js`, `worker.js`) |

### Credential Protection

- Cloudflare API Key is never returned to the frontend
- RTK JWT tokens must not be logged (only `len(token)` is logged at debug level)
- `GetEffectiveAPIKey()` return value is never included in any log output

### Call Page CSP

```
default-src 'self';
script-src 'self' 'unsafe-eval' 'wasm-unsafe-eval';   <- RTK WebAssembly
connect-src *;                                          <- WebRTC / WebSocket
style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;
font-src 'self' https://fonts.gstatic.com;
img-src 'self' blob: data:;
worker-src 'self' blob:;                               <- Web Worker
media-src *;                                           <- Audio / video streams
```

### CSRF Protection

All API requests include `X-Requested-With: XMLHttpRequest` header (`client.ts`).

### Host Authorization

`DELETE /calls/{id}` strictly checks `CreatorID == requestingUserID`.

---

## 11. Build Configuration

### Server Build

```bash
make build
```

| OS/Arch | Output |
|---------|--------|
| linux/amd64 | `server/dist/plugin-linux-amd64` |
| linux/arm64 | `server/dist/plugin-linux-arm64` |
| darwin/amd64 | `server/dist/plugin-darwin-amd64` |
| darwin/arm64 | `server/dist/plugin-darwin-arm64` |
| windows/amd64 | `server/dist/plugin-windows-amd64.exe` |

### Frontend Build

```bash
cd webapp

# Main bundle (Mattermost plugin)
npm run build

# Standalone call page bundle
VITE_BUILD_TARGET=call npm run build
```

### Static File Embedding

`api_static.go` uses Go's `//go:embed` directive:

```go
//go:embed assets/call.html
var callHTML []byte

//go:embed assets/call.js
var callJS []byte

//go:embed assets/worker.js
var workerJS []byte
```

`assets/call.js` is a copy of the built `webapp/dist/call.js`.

---

## 12. Directory Structure

```
mattermost-plugin-rtk/
├── plugin.json                    # Plugin manifest and settings schema
├── go.mod / go.sum
├── Makefile
│
├── server/
│   ├── main.go                    # Entry point
│   ├── manifest.go                # Auto-generated manifest loader
│   ├── plugin.go                  # Plugin struct, OnActivate, OnDeactivate
│   ├── configuration.go           # Config management and env var overrides
│   ├── calls.go                   # Call business logic
│   ├── errors.go                  # Sentinel error definitions
│   ├── cleanup.go                 # Stub (placeholder for future reconciliation)
│   ├── api.go                     # gorilla/mux router init and auth middleware
│   ├── api_calls.go               # Call API handlers
│   ├── api_config.go              # Config status API handlers
│   ├── api_mobile.go              # Dismiss API handler
│   ├── api_static.go              # Static file serving (go:embed)
│   ├── api_webhook.go             # RTK webhook handler
│   ├── assets/
│   │   ├── call.html              # Call page HTML
│   │   ├── call.js                # Call page JS (copy of webapp/dist/call.js)
│   │   └── worker.js              # Web Worker JS
│   ├── rtkclient/
│   │   ├── interface.go           # RTKClient interface and type definitions
│   │   ├── client.go              # HTTP client implementation
│   │   └── mocks/mock_rtkclient.go
│   ├── store/kvstore/
│   │   ├── kvstore.go             # KVStore interface
│   │   ├── models.go              # CallSession type definition
│   │   ├── calls.go               # KVStore operation implementations
│   │   ├── startertemplate.go     # Methods from the starter template
│   │   └── mocks/mock_kvstore.go
│   └── command/
│       ├── command.go             # /hello slash command
│       └── mocks/mock_commands.go
│
├── webapp/
│   ├── package.json
│   ├── vite.config.ts             # Dual-bundle config and CSP patch plugin
│   ├── src/
│   │   ├── index.tsx              # Plugin entry: initialize, register WS/UI
│   │   ├── manifest.ts            # References plugin.json
│   │   ├── client.ts              # pluginFetch utility
│   │   ├── call_page/
│   │   │   ├── main.tsx           # Standalone call page entry point
│   │   │   └── CallPage.tsx       # RTK SDK init and RtkMeeting UI
│   │   ├── components/
│   │   │   ├── channel_header_button/  # Call button (start/join/in-call states)
│   │   │   ├── call_post/              # custom_cf_call post renderer
│   │   │   ├── toast_bar/              # Channel toast bar
│   │   │   ├── floating_widget/        # Floating in-call widget (RTK UI)
│   │   │   ├── incoming_call_notification/  # Incoming call alert
│   │   │   ├── switch_call_modal/      # Call switch confirmation modal
│   │   │   └── admin_config/           # Admin settings (env var display)
│   │   ├── redux/
│   │   │   ├── calls_slice.ts     # Redux reducer, actions, and type definitions
│   │   │   ├── websocket_handlers.ts  # WebSocket event handlers
│   │   │   └── selectors.ts       # Redux selectors
│   │   └── utils/
│   │       ├── call_tab.ts        # Call page URL builder
│   │       └── rtk_lang_ja.ts     # RTK SDK Japanese i18n dictionary
│   └── i18n/
│       ├── en.json                # English translations
│       └── ja.json                # Japanese translations
│
└── aidlc-docs/                    # AI-DLC methodology design documents
    ├── aidlc-state.md
    ├── audit.md
    ├── inception/
    └── construction/
```
