# Component Inventory

## Design Decisions Applied

| Question | Decision |
|---|---|
| Floating widget | Yes — Mattermost-side widget with Open-in-new-tab |
| Backend service layer | Flat — logic in Plugin struct + api/ handlers |
| Frontend state | Redux slice via `registry.registerReducer()` |
| Call page auth | Token (JWT) + Mattermost session cookie (automatic, same domain) |
| Leave detection | fetch+keepalive on beforeunload (heartbeat deferred) |

---

## Backend Components

### B-01: Plugin Core (`server/plugin.go`)

**Type**: Existing — major extension
**Responsibility**: Plugin lifecycle, component wiring, call business logic
**Key additions**:
- Initialize RTKClient on `OnActivate`
- Call lifecycle methods: `CreateCall`, `JoinCall`, `LeaveCall`, `EndCall`
- WebSocket event emission helpers
- Cleanup loop placeholder (future: RTK participant reconciliation)

---

### B-02: API Handler (`server/api*.go`)

**Type**: New files (flat structure)
**Responsibility**: All HTTP endpoint handlers, authentication middleware, static file serving
**Files**:

| File | Endpoints |
|---|---|
| `api.go` | Router setup, auth middleware |
| `api_calls.go` | `POST /api/v1/calls`, `GET /api/v1/calls/{callId}`, `POST /api/v1/calls/{callId}/token`, `POST /api/v1/calls/{callId}/leave`, `DELETE /api/v1/calls/{callId}` |
| `api_config.go` | `GET /api/v1/config/status`, `GET /api/v1/config/admin-status` |
| `api_mobile.go` | `POST /api/v1/mobile/voip-token`, `POST /api/v1/calls/{callId}/dismiss` |
| `api_static.go` | `GET /call` (call page HTML), `GET /call.js`, `GET /worker.js` |
| `api_webhook.go` | RTK webhook handler |

---

### B-03: Configuration (`server/configuration.go`)

**Type**: Existing — significant extension
**Responsibility**: Thread-safe plugin configuration with env var override support
**Key additions**:
- `CloudflareOrgID`, `CloudflareAPIKey` fields
- 10 feature flag fields (Polls, Plugins, Chat, ScreenShare, Participants, Recording, AITranscription, WaitingRoom, Video, RaiseHand)
- `GetEffective*()` methods applying env var overrides
- `Clone()` pattern for safe concurrent reads

---

### B-04: RTK Client (`server/rtkclient/`)

**Type**: New package
**Responsibility**: Cloudflare RTK API communication
**Files**:

| File | Content |
|---|---|
| `interface.go` | `RTKClient` interface |
| `client.go` | HTTP implementation (Basic Auth, HTTPS) |

**Interface methods**:
- `CreateMeeting(preset string) (*Meeting, error)`
- `GenerateToken(meetingID, userID, preset string) (*Token, error)`
- `EndMeeting(meetingID string) error`

---

### B-05: KV Store (`server/store/kvstore/`)

**Type**: Existing — extension
**Responsibility**: Call session persistence
**Key additions to interface**:
- `GetCallByChannel(channelID string) (*CallSession, error)`
- `GetCallByID(callID string) (*CallSession, error)`
- `SaveCall(session *CallSession) error`
- `UpdateCallParticipants(callID string, participants []string) error`
- `EndCall(callID string, endAt int64) error`
- `StoreVoIPToken(userID, token string) error`
- `GetVoIPToken(userID string) (string, error)`

---

### ~~B-06: Push Sender (`server/push/`)~~ — REMOVED

> **Updated 2026-03-31**: Push notification subsystem removed. Mobile clients receive call notifications via WebSocket events.

---

### B-07: Cleanup Loop (`server/cleanup.go`)

**Type**: New (placeholder)
**Responsibility**: Placeholder for future RTK participant reconciliation via `GetMeetingParticipants()`
**Current state**: Stub — waits for stop signal only

---

## Frontend Components (Main Bundle)

### F-01: Plugin Entry (`webapp/src/index.tsx`)

**Type**: Existing — major extension
**Responsibility**: Register all components and Redux reducer with Mattermost
**Registers**:
- `registerReducer(callsReducer)`
- `registerChannelHeaderButtonAction(ChannelHeaderButton)`
- `registerPostTypeComponent('custom_cf_call', CallPost)`
- `registerWebSocketEventHandler` for all 5 custom WS events
- `registerAdminConsoleCustomSetting` for custom admin UI fields

---

### F-02: Channel Header Button (`webapp/src/components/channel_header_button/`)

**Type**: New
**Responsibility**: Call button in channel header with 4 states
**States**: "Start call" / "Join call" / "In call" (disabled) / "Starting call..." (spinner)
**Behavior**: Opens SwitchCallModal when user is already in a different call

---

### F-03: Call Post (`webapp/src/components/call_post/`)

**Type**: New
**Responsibility**: Custom post renderer for `custom_cf_call` type
**States**:
- Active: green indicator, "Call started", start time, participant avatars (max 3 + overflow), "Join call" / "Join call" (disabled)
- Ended: gray indicator, "Call ended", end time, duration

---

### F-04: Toast Bar (`webapp/src/components/toast_bar/`)

**Type**: New
**Responsibility**: Channel call toast bar above message input
**Shows**: Call start time, participant avatars, "Join" button (non-members), dismiss button
**Dismisses**: Locally on user dismiss; globally on `custom_com.kondo97.mattermost-plugin-rtk_call_ended` WS event

---

### F-05: Floating Widget (`webapp/src/components/floating_widget/`)

**Type**: New
**Responsibility**: Persistent in-call indicator within Mattermost window
**Shows**: Participant count/avatars, call duration (live timer), mute/unmute control, "Open in new tab" button
**Behavior**: Draggable; persists while browsing other channels; triggers new tab with `/plugins/{id}/call?token=<jwt>`

---

### F-06: Switch Call Modal (`webapp/src/components/switch_call_modal/`)

**Type**: New
**Responsibility**: Confirmation dialog when joining a different call
**Actions**: "Cancel" / "Leave and join new call"

---

### F-07: Incoming Call Notification (`webapp/src/components/incoming_call_notification/`)

**Type**: New
**Responsibility**: DM/GM in-app ringing notification
**Shows**: Caller info, "Ignore" and "Join" actions
**Auto-dismisses**: After 30 seconds
**Trigger**: `custom_com.kondo97.mattermost-plugin-rtk_call_started` WS event on DM/GM channels

---

### F-08: Admin Settings (`webapp/src/components/admin_settings/`)

**Type**: New
**Responsibility**: Custom System Console fields for credentials and feature flags
**Shows**: Env var override indicator (read-only field + label) when applicable
**Fields**: Organization ID, API Key (masked), 10 feature flag toggles

---

### F-09: Calls Redux (`webapp/src/redux/`)

**Type**: New
**Responsibility**: Call state management across components
**Files**:

| File | Content |
|---|---|
| `calls_slice.ts` | Redux slice: `callsByChannel`, `myActiveCall`, `incomingCall` state |
| `websocket_handlers.ts` | Handle 5 custom WS events → dispatch slice actions |
| `selectors.ts` | Typed selector hooks |

---

## Frontend Components (Standalone Call Bundle)

### F-10: Call Page (`webapp/src/call_page/`)

**Type**: New — separate Vite entry point (`webapp/dist/call.js`)
**Responsibility**: Full RTK SDK call UI in a dedicated browser tab
**Files**:

| File | Content |
|---|---|
| `main.tsx` | Page bootstrap, read `?token` from URL |
| `CallPage.tsx` | RealtimeKitProvider initialization, fetch+keepalive on unload |

**Authentication**: JWT token from URL `?token=` for RTK SDK; Mattermost session cookie automatic (same domain) for leave API calls
**Leave detection**: `beforeunload` → `fetch('/api/v1/calls/{id}/leave', {keepalive: true, headers: {'X-Requested-With': 'XMLHttpRequest'}})`
