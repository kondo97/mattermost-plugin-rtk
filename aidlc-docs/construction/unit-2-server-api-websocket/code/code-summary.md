# Unit 2: Server API & WebSocket — Code Summary

## Overview

Unit 2 adds the full HTTP API layer, RTK webhook receiver, static asset serving, and concurrency guard to the Mattermost RTK plugin. It delegates all call state mutations to the Unit 1 Plugin methods.

---

## Files Modified

### `server/store/kvstore/kvstore.go`
Added 5 new interface methods:
- `GetCallByMeetingID(meetingID string) (*CallSession, error)`
- `StoreWebhookID(id string) error`
- `GetWebhookID() (string, error)`
- `StoreWebhookSecret(secret string) error`
- `GetWebhookSecret() (string, error)`

### `server/store/kvstore/calls.go`
- Added constants: `keyCallMeeting`, `keyWebhookID`, `keyWebhookSecret`
- Implemented `GetCallByMeetingID`
- Updated `SaveCall`, `UpdateCallParticipants`, `EndCall` to also write `call:meeting:{meetingID}` key
- Implemented `StoreWebhookID`, `GetWebhookID`, `StoreWebhookSecret`, `GetWebhookSecret`

### `server/store/kvstore/mocks/mock_kvstore.go`
Regenerated with mock methods for all 5 new interface methods.

### `server/rtkclient/interface.go`
Added:
- `RegisterWebhook(url string, events []string) (id, secret string, err error)`
- `DeleteWebhook(webhookID string) error`

### `server/rtkclient/client.go`
Implemented `RegisterWebhook` (POST `/apps/{orgID}/webhooks`) and `DeleteWebhook` (DELETE `/apps/{orgID}/webhooks/{webhookID}`).

### `server/rtkclient/mocks/mock_rtkclient.go`
Regenerated with mock methods for `RegisterWebhook` and `DeleteWebhook`.

### `server/plugin.go`
- Added `callMu sync.Mutex` field to `Plugin` struct
- Added `rtkWebhookEvents` constant (`meeting.participantLeft`, `meeting.ended`)
- Added `registerWebhookIfNeeded()` — called from `OnActivate`; best-effort webhook registration; skips if webhook ID already stored
- Added `reRegisterWebhook()` — deletes existing webhook and re-registers; called from `OnConfigurationChange`
- Updated `OnActivate` to call `registerWebhookIfNeeded()` after RTK client init

### `server/configuration.go`
Updated `OnConfigurationChange` to detect credential changes and call `reRegisterWebhook()`.

### `server/calls.go`
Added `p.callMu.Lock(); defer p.callMu.Unlock()` at the start of `CreateCall`, `JoinCall`, `LeaveCall`, and `EndCall`.

### `server/api.go`
Complete rewrite:
- Removed `HelloWorld` handler
- Added `writeError(w, status, msg)` helper
- Router structure:
  - Static routes (no auth): `/call`, `/call.js`, `/worker.js`
  - Webhook route (RTK signature auth): `POST /api/v1/webhook/rtk`
  - Authenticated subrouter (`/api/v1/`): all call, config, and mobile routes

---

## Files Created

### `server/api_calls.go`
HTTP handlers for call management:
- `handleCreateCall` — `POST /api/v1/calls` → 201 with `{call, token}`
- `handleJoinCall` — `POST /api/v1/calls/{id}/token` → 200 with `{call, token}`
- `handleLeaveCall` — `POST /api/v1/calls/{id}/leave` → 200 (idempotent)
- `handleEndCall` — `DELETE /api/v1/calls/{id}` → 200

### `server/api_config.go`
- `handleConfigStatus` — `GET /api/v1/config/status` → `{enabled: bool}`
- `handleAdminConfigStatus` — `GET /api/v1/config/admin-status` → system admin only

### `server/api_mobile.go`
- `handleDismiss` — `POST /api/v1/calls/{id}/dismiss` → emits `custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed` WS event to requesting user; always 200

### `server/api_static.go`
Serves embedded static assets using `//go:embed`:
- `serveCallHTML` — `/call` — full HTTP security headers (CSP `connect-src *`, HSTS, etc.)
- `serveCallJS` — `/call.js`
- `serveWorkerJS` — `/worker.js`

### `server/api_webhook.go`
RTK webhook receiver:
- `handleRTKWebhook` — reads raw body, verifies `dyte-signature` HMAC-SHA256, dispatches events
- `handleWebhookParticipantLeft` — calls `LeaveCall` (mutex acquired inside)
- `handleWebhookMeetingEnded` — acquires `callMu` explicitly, calls `endCallInternal`
- `verifyRTKSignature` — `HMAC-SHA256(secret, body) == hex(signature)`

### `server/assets/call.html`, `server/assets/call.js`, `server/assets/worker.js`
Placeholder files; replaced by webapp build in Unit 4.

---

## Test Files Created

### `server/api_calls_test.go`
Tests: CreateCall success, missing channel_id (400), already active (409), no auth (401), JoinCall success/not-found, LeaveCall success (with auto-end), EndCall success/unauthorized.

### `server/api_config_test.go`
Tests: ConfigStatus enabled/disabled, AdminConfigStatus admin/forbidden.

### `server/api_mobile_test.go`
Tests: Dismiss emits correct WS event.

### `server/api_webhook_test.go`
Tests: Invalid signature (401), unknown event (200), participantLeft success/session-not-found, meetingEnded success/already-ended.

### `server/plugin_test.go`
Updated: tests unknown route (404) and unauthenticated request (401) against new router.

---

## Build & Test Results

```
go build ./server/...  → OK
go test ./server/...   → OK (all tests pass)
```

---

## Key Design Decisions

| Decision | Rationale |
|---|---|
| `callMu` in business methods (not handlers) | Consistent locking regardless of caller (HTTP handler or webhook) |
| Webhook handler locks `callMu` explicitly for `meeting.ended` | `endCallInternal` is called without holding mutex; only `LeaveCall`/`EndCall` lock it |
| Best-effort webhook registration | Activation should not fail due to RTK API unavailability |
| `go:embed` for static assets | Avoids separate file distribution; assets replaced at build time by Unit 4 |
| CSP `connect-src *` | RTK SDK requires connections to Cloudflare infrastructure (wildcard approved in NFR requirements) |
