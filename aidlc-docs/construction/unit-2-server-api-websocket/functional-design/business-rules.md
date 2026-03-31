# Unit 2: Server API & WebSocket — Business Rules

## Authentication Rules

| Rule | Description |
|---|---|
| BR-U2-01 | All API endpoints under `/api/v1/` (except `/api/v1/webhook/rtk`) MUST require a non-empty `Mattermost-User-ID` header. Missing or empty → 401 Unauthorized. |
| BR-U2-02 | Static endpoints (`GET /call`, `GET /call.js`, `GET /worker.js`) bypass auth middleware via explicit allowlist. |
| BR-U2-03 | `POST /api/v1/webhook/rtk` bypasses Mattermost auth middleware. Authentication is performed via RTK signature verification in the handler. |

---

## POST /calls (CreateCall)

| Rule | Description |
|---|---|
| BR-U2-04 | Request body MUST contain a non-empty `channel_id` field → 400 if missing. |
| BR-U2-05 | Acquire `callMu` lock before calling `CreateCall`. |
| BR-U2-06 | On success: 201 Created with `{ "call": CallSession, "token": string }`. |
| BR-U2-07 | `ErrRTKNotConfigured` → 503. `ErrCallAlreadyActive` → 409. |

---

## POST /calls/{id}/token (JoinCall)

| Rule | Description |
|---|---|
| BR-U2-08 | Acquire `callMu` lock before calling `JoinCall`. |
| BR-U2-09 | If user already in Participants: regenerate token, no WS re-emit, 200 OK. |
| BR-U2-10 | On success: fetch updated session, respond `{ "call": CallSession, "token": string }`. |
| BR-U2-11 | `ErrCallNotFound` → 404. `ErrRTKNotConfigured` → 503. |

---

## POST /calls/{id}/leave (LeaveCall)

| Rule | Description |
|---|---|
| BR-U2-12 | Acquire `callMu` lock before calling `LeaveCall`. |
| BR-U2-13 | Not found or already ended: no-op, 200 OK (idempotent). |

---

## DELETE /calls/{id} (EndCall)

| Rule | Description |
|---|---|
| BR-U2-14 | Acquire `callMu` lock before calling `EndCall`. |
| BR-U2-15 | `ErrUnauthorized` → 403. `ErrCallNotFound` → 404. |

---

## GET /config/status

| Rule | Description |
|---|---|
| BR-U2-16 | `enabled = GetEffectiveOrgID() != "" && GetEffectiveAPIKey() != ""`. |
| BR-U2-17 | 200 OK: `{ "enabled": bool }`. |

---

## GET /config/admin-status

| Rule | Description |
|---|---|
| BR-U2-18 | Caller MUST have `model.PermissionManageSystem` → 403 if not. |
| BR-U2-19 | Full response schema defined in Unit 5. |

---

## POST /calls/{id}/dismiss

| Rule | Description |
|---|---|
| BR-U2-20 | Scoped to requesting user only. Emit `custom_cf_notification_dismissed` to UserID. |
| BR-U2-21 | Always 200 OK (idempotent). |

---

## POST /api/v1/webhook/rtk (RTK Webhook Receiver)

| Rule | Description |
|---|---|
| BR-U2-22 | Read raw body bytes BEFORE parsing JSON (signature verification requires the raw body). |
| BR-U2-23 | Verify RTK signature using stored `webhook:secret` from KVStore. Invalid signature → 401 Unauthorized. |
| BR-U2-24 | `meeting.participantLeft`: look up CallSession by `meeting.id` via `GetCallByMeetingID`. If not found or already ended → 200 OK (idempotent). |
| BR-U2-25 | `meeting.participantLeft`: extract Mattermost userID from `participant.customParticipantId`, acquire `callMu`, call `LeaveCall`. |
| BR-U2-26 | `meeting.ended`: look up CallSession by `meeting.id`. If not found or already ended → 200 OK (idempotent). Acquire `callMu`, call `endCallInternal`. |
| BR-U2-27 | Unknown event types: 200 OK (ignored). |
| BR-U2-28 | Always return 200 OK within 3 seconds (RTK retry policy). |

---

## Static File Serving

| Rule | Description |
|---|---|
| BR-U2-29 | `GET /call` → `assets/call.html`, `Content-Type: text/html`, full HTTP security headers. |
| BR-U2-30 | `GET /call.js` → `assets/call.js`, `Content-Type: application/javascript`. |
| BR-U2-31 | `GET /worker.js` → `assets/worker.js`, `Content-Type: application/javascript`. |

---

## OnActivate: Webhook Registration

| Rule | Description |
|---|---|
| BR-U2-32 | On `OnActivate`, if `rtkClient != nil` and no `webhook:id` in KVStore, register webhook with events `["meeting.participantLeft", "meeting.ended"]`. |
| BR-U2-33 | Webhook URL = `{SiteURL}/plugins/{pluginID}/api/v1/webhook/rtk`. |
| BR-U2-34 | Store returned `webhookID` and `webhookSecret` in KVStore. |
| BR-U2-35 | Registration failure is best-effort — log warning, plugin continues without webhook. |
| BR-U2-36 | If `webhook:id` already exists in KVStore, skip registration (avoid duplicates). |
| BR-U2-37 | On `OnConfigurationChange` with new credentials: delete old webhook, register new one, update KVStore. |

---

## Error Response Format

| Rule | Description |
|---|---|
| BR-U2-38 | All error responses: `Content-Type: application/json`, body `{ "error": "message" }`. |
| BR-U2-39 | No stack traces, internal paths, or system details in error messages. |

---

## Concurrency

| Rule | Description |
|---|---|
| BR-U2-40 | `callMu sync.Mutex` field on Plugin struct. |
| BR-U2-41 | Handlers for `/calls`, `/calls/{id}/token`, `/calls/{id}/leave`, `/calls/{id}` (DELETE), and webhook `meeting.participantLeft`/`meeting.ended` MUST acquire `callMu`. |
| BR-U2-42 | `/calls/{id}/dismiss`, `/config/status`, `/config/admin-status`, static handlers do NOT require `callMu`. |

---

## Out of Scope

| Item | Reason |
|---|---|
| `POST /calls/{id}/heartbeat` | Deferred / not implemented — RTK webhook (`meeting.participantLeft`) handles cleanup |
| `POST /mobile/voip-token` | Handled by Mattermost server at login time |
| `CleanupStaleParticipants` background job | Deferred / not implemented — RTK webhook handles cleanup |
| ~~sendBeacon~~ fetch+keepalive / ~~heartbeat loop~~ (Unit 4) | Heartbeat deferred; leave uses `fetch` with `keepalive: true`; `POST /calls/{id}/leave` retained as explicit leave path |
