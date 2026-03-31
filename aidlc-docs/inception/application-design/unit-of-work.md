# Unit of Work Definitions

## Overview

This document defines the 6 units of work for `mattermost-plugin-rtk`. All units compile into a single Mattermost plugin binary (monolith). Units are logical groupings of work ordered by data and API dependencies.

---

## Unit 1: RTK Integration

**Purpose**: Establish the foundational backend layer — RTK API client, call session storage, and call lifecycle business logic.

**Responsibilities**:
- Implement the Cloudflare RTK API client (create meeting, generate token, end meeting)
- Extend KVStore interface and implementation with call session methods
- Add call lifecycle methods to Plugin struct (`CreateCall`, `JoinCall`, `LeaveCall`, `EndCall`)
- Add placeholder cleanup loop for future RTK participant reconciliation

**In-Scope Files**:

| File | Change |
|---|---|
| `server/rtkclient/interface.go` | New — RTKClient interface |
| `server/rtkclient/client.go` | New — HTTP implementation (Basic Auth) |
| `server/store/kvstore/kvstore.go` | Modified — add 9 call session methods to interface + implementation |
| `server/plugin.go` | Modified — initialize RTKClient on `OnActivate`, start cleanup loop |
| `server/calls.go` | New — `CreateCall`, `JoinCall`, `LeaveCall`, `EndCall` |
| `server/cleanup.go` | New — placeholder cleanup loop (future: RTK participant reconciliation) |

**KVStore Key Pattern**:
- `call:channel:{channelID}` — active call per channel
- `call:id:{callID}` — call session by ID
- `voip:{userID}` — VoIP device token

**Success Criteria**:
- RTKClient interface is mockable (used in unit tests)
- KVStore interface is mockable (used in unit tests)
- All call lifecycle methods have unit tests

---

## Unit 2: Server API & WebSocket

**Purpose**: Expose all HTTP endpoints and WebSocket events. Depends on Unit 1 (calls business logic).

**Responsibilities**:
- Implement HTTP router with authentication middleware
- Implement all 8 REST endpoints + static file serving
- Emit WebSocket events for all call state changes
- Serve static assets (`/call`, `/call.js`, `/worker.js`)

**In-Scope Files**:

| File | Change |
|---|---|
| `server/api.go` | New — router setup (gorilla/mux), `Mattermost-User-ID` auth middleware |
| `server/api_calls.go` | New — `POST /calls`, `GET /calls/{id}`, `POST /calls/{id}/token`, `POST /calls/{id}/leave`, `DELETE /calls/{id}` |
| `server/api_config.go` | New — `GET /config/status`, `GET /config/admin-status` |
| `server/api_mobile.go` | New — `POST /mobile/voip-token`, `POST /calls/{id}/dismiss` |
| `server/api_static.go` | New — `GET /call` (HTML), `GET /call.js`, `GET /worker.js` |
| `server/api_webhook.go` | New — RTK webhook handler |
| `server/plugin.go` | Modified — register API handler in `ServeHTTP` |

**WebSocket Events Emitted**:
- `custom_com.kondo97.mattermost-plugin-rtk_call_started` — on CreateCall
- `custom_com.kondo97.mattermost-plugin-rtk_user_joined` — on JoinCall
- `custom_com.kondo97.mattermost-plugin-rtk_user_left` — on LeaveCall
- `custom_com.kondo97.mattermost-plugin-rtk_call_ended` — on EndCall / auto-end
- `custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed` — on dismiss

**Carry-over from Unit 1**:
- Add a `sync.Mutex` (or `sync.RWMutex`) to the `Plugin` struct to guard concurrent KVStore read-modify-write operations (`UpdateCallParticipants`, `EndCall`). The Mattermost plugin runs as a single process, so a plugin-level mutex is the appropriate solution. Without this, concurrent Join/Leave requests on the same call can cause lost updates (last-write-wins race).

**Success Criteria**:
- All endpoints return correct HTTP status codes
- Auth middleware rejects requests missing `Mattermost-User-ID` header (except `/call`, `/call.js`, `/worker.js`)
- Admin-only endpoints verify admin role server-side
- WebSocket events emitted for all state transitions
- Concurrent Join/Leave requests on the same call do not cause lost participant updates
- Unit tests for all handlers using mock Plugin interface

---

## Unit 3: Webapp - Channel UI

**Purpose**: All Mattermost-side UI components that react to call state, plus Redux state management.

**Responsibilities**:
- Implement Redux slice for call state (`callsByChannel`, `myActiveCall`, `incomingCall`)
- Handle 5 custom WebSocket events → dispatch Redux actions
- Render channel header button with 4 states
- Render toast bar for non-participants
- Render floating widget for participants
- Render Switch Call Modal and Incoming Call Notification
- Register all components in plugin entry point

**In-Scope Files**:

| File | Change |
|---|---|
| `webapp/src/redux/calls_slice.ts` | New — Redux slice |
| `webapp/src/redux/websocket_handlers.ts` | New — WS event → Redux dispatch |
| `webapp/src/redux/selectors.ts` | New — typed selector hooks |
| `webapp/src/components/channel_header_button/` | New — 4-state call button |
| `webapp/src/components/toast_bar/` | New — channel call toast bar |
| `webapp/src/components/floating_widget/` | New — draggable in-call indicator |
| `webapp/src/components/switch_call_modal/` | New — leave-and-join confirmation |
| `webapp/src/components/incoming_call_notification/` | New — DM/GM ringing (30s auto-dismiss) |
| `webapp/src/index.tsx` | Modified — register reducer + components (shared edit with Unit 4) |

**Success Criteria**:
- Redux state updates correctly for all 5 WebSocket events
- Channel header button reflects all 4 states
- Floating widget persists while browsing other channels
- Switch Call Modal appears when joining different call while in one
- Incoming Call Notification auto-dismisses after 30s
- Jest unit tests for Redux slice and selectors

---

## Unit 4: Webapp - Call Page & Post

**Purpose**: Custom post renderer for call announcements, and standalone call page in a separate browser tab.

**Responsibilities**:
- Implement `custom_cf_call` post renderer (active and ended states)
- Implement standalone call page (separate Vite bundle entry)
- Initialize Cloudflare RTK React SDK (`RealtimeKitProvider`)
- Implement `fetch+keepalive` on tab close for leave detection
- Migrate build system from webpack to Vite (dual bundle: `main.js` + `call.js`)

**In-Scope Files**:

| File | Change |
|---|---|
| `webapp/src/components/call_post/` | New — `custom_cf_call` post renderer |
| `webapp/src/call_page/main.tsx` | New — call page bootstrap, read `?token` from URL |
| `webapp/src/call_page/CallPage.tsx` | New — RealtimeKitProvider, fetch+keepalive on unload |
| `webapp/vite.config.ts` | New — dual entry point configuration |
| `webapp/package.json` | Modified — replace webpack deps with Vite |
| `webapp/src/index.tsx` | Modified — register `custom_cf_call` post type (shared edit with Unit 3) |

**Success Criteria**:
- Call post shows active state (green indicator, participants, Join button)
- Call post shows ended state (gray indicator, duration, no buttons)
- Call page loads RTK SDK with `?token` parameter
- `fetch+keepalive` fires on `beforeunload` with CSRF header
- `make` produces `call.js` embedded in Go binary
- Jest unit tests for CallPost component states

---

## Unit 5: Admin & Config

**Purpose**: Plugin configuration with Cloudflare credentials, feature flags, and env var override support.

**Responsibilities**:
- Extend configuration struct with RTK credentials and 10 feature flags
- Implement `GetEffective*()` methods for env var overrides
- Implement `Clone()` pattern for thread-safe reads
- Implement admin console custom UI with masked API key and env var indicators

**In-Scope Files**:

| File | Change |
|---|---|
| `server/configuration.go` | Modified — `CloudflareOrgID`, `CloudflareAPIKey`, 10 feature flag fields, `GetEffective*()`, `Clone()` |
| `webapp/src/components/admin_settings/` | New — System Console custom fields |

**Feature Flag Env Vars**:
`RTK_ORG_ID`, `RTK_API_KEY`, `RTK_RECORDING_ENABLED`, `RTK_SCREEN_SHARE_ENABLED`, `RTK_POLLS_ENABLED`, `RTK_TRANSCRIPTION_ENABLED`, `RTK_WAITING_ROOM_ENABLED`, `RTK_VIDEO_ENABLED`, `RTK_CHAT_ENABLED`, `RTK_PLUGINS_ENABLED`, `RTK_PARTICIPANTS_ENABLED`, `RTK_RAISE_HAND_ENABLED`

**Success Criteria**:
- All 10 feature flags default to enabled (ON)
- API Key field is masked in the UI
- Env var override shows read-only field with indicator label
- `GetEffective*()` methods return env var value when set, config value otherwise
- Unit tests for `GetEffective*()` methods

---

## Unit 6: Mobile Support

> **Status**: Push notification subsystem was implemented and then **intentionally removed**. Mobile call notifications are now handled through WebSocket events directly. The `server/push/` package and all related integration code have been deleted.

**Original Purpose**: Push notification delivery for mobile users when calls start and end.

**Current State**: No dedicated mobile support code. Mobile clients receive call events through the same WebSocket channel as desktop clients. The `custom_com.kondo97.mattermost-plugin-rtk_call_started` and `custom_com.kondo97.mattermost-plugin-rtk_call_ended` events carry sufficient data for mobile apps to display call notifications.

**Removed Files**:

| File | Status |
|---|---|
| `server/push/interface.go` | Deleted |
| `server/push/push.go` | Deleted |
| `server/push/push_test.go` | Deleted |
| `server/push/mocks/mock_push.go` | Deleted |

---

## Implementation Order

```
Unit 1: RTK Integration          (no plugin dependencies)
  |
  v
Unit 2: Server API & WebSocket   (depends on Unit 1)
  |
  +---> Unit 3: Webapp - Channel UI     (depends on Unit 2 API contracts)
  +---> Unit 4: Webapp - Call Page & Post (depends on Unit 2 API contracts)
  +---> Unit 5: Admin & Config          (depends on Unit 2 config endpoints)
  +---> Unit 6: Mobile Support          (REMOVED — WS events used instead of push)
```

Units 3, 4, 5 can be designed and reviewed in parallel; they are independent of each other.
Unit 6 push notification subsystem was removed; mobile clients use WebSocket events.
