# Unit 2: Server API & WebSocket — Logical Components

## Component Overview

```
Plugin (server/plugin.go)
├── callMu sync.Mutex          [NEW — concurrency guard]
├── router *mux.Router         [existing — extended]
├── kvStore KVStore            [existing — extended with webhook + meetingID methods]
├── rtkClient RTKClient        [existing — extended with webhook registration]
└── configurationLock          [existing]

HTTP Layer (server/)
├── api.go                     [Modified — new router init, writeError, webhook route]
├── api_calls.go               [NEW — CreateCall, JoinCall, LeaveCall, EndCall handlers]
├── api_config.go              [NEW — ConfigStatus, AdminConfigStatus handlers]
├── api_mobile.go              [NEW — Dismiss handler]
├── api_static.go              [NEW — serveCallHTML, serveCallJS, serveWorkerJS]
└── api_webhook.go             [NEW — handleRTKWebhook]

Assets (server/assets/)        [populated by webapp build — Unit 4]
├── call.html
├── call.js
└── worker.js
```

---

## Component: Router (api.go)

| Aspect | Design |
|---|---|
| Static paths | Root router — 3 paths allowlist |
| Webhook path | Root router — `/api/v1/webhook/rtk` (no Mattermost auth) |
| API paths | Subrouter at `/api/v1/` with `MattermostAuthorizationRequired` |
| Error helper | `writeError(w, status, msg)` |

---

## Component: Calls Handler (api_calls.go)

| Handler | Method | Path | Mutex |
|---|---|---|---|
| `handleCreateCall` | POST | `/api/v1/calls` | Lock |
| `handleJoinCall` | POST | `/api/v1/calls/{id}/token` | Lock |
| `handleLeaveCall` | POST | `/api/v1/calls/{id}/leave` | Lock |
| `handleEndCall` | DELETE | `/api/v1/calls/{id}` | Lock |

---

## Component: Config Handler (api_config.go)

| Handler | Method | Path | Auth Check |
|---|---|---|---|
| `handleConfigStatus` | GET | `/api/v1/config/status` | User auth |
| `handleAdminConfigStatus` | GET | `/api/v1/config/admin-status` | System Admin |

---

## Component: Mobile Handler (api_mobile.go)

| Handler | Method | Path | Notes |
|---|---|---|---|
| `handleDismiss` | POST | `/api/v1/calls/{id}/dismiss` | Idempotent, user-scoped WS event |

---

## Component: Webhook Handler (api_webhook.go)

| Handler | Method | Path | Auth |
|---|---|---|---|
| `handleRTKWebhook` | POST | `/api/v1/webhook/rtk` | RTK signature (not Mattermost) |

**Event handling:**

| Event | Action |
|---|---|
| `meeting.participantLeft` | `GetCallByMeetingID` → `callMu.Lock` → `LeaveCall` |
| `meeting.ended` | `GetCallByMeetingID` → `callMu.Lock` → `endCallInternal` |
| Others | 200 OK, ignored |

---

## Component: Static Handler (api_static.go)

| Handler | Path | Security Headers |
|---|---|---|
| `serveCallHTML` | `/call` | Full set (CSP, HSTS, etc.) |
| `serveCallJS` | `/call.js` | `X-Content-Type-Options: nosniff` |
| `serveWorkerJS` | `/worker.js` | `X-Content-Type-Options: nosniff` |

---

## KVStore Extensions (Unit 2)

| Method | Key | Description |
|---|---|---|
| `GetCallByMeetingID(meetingID string)` | `call:meeting:{meetingID}` | O(1) webhook lookup |
| `StoreWebhookID(id string)` | `webhook:id` | Persist registered webhook ID |
| `GetWebhookID() string` | `webhook:id` | Read webhook ID |
| `StoreWebhookSecret(secret string)` | `webhook:secret` | Persist signing secret |
| `GetWebhookSecret() string` | `webhook:secret` | Read for signature verification |

`SaveCall` also writes `call:meeting:{meetingID}` for every new/updated session.

---

## RTKClient Extensions (Unit 2)

| Method | Description |
|---|---|
| `RegisterWebhook(url string, events []string) (id, secret string, err error)` | Register webhook endpoint with RTK |
| `DeleteWebhook(webhookID string) error` | Remove webhook on credential change |

---

## Webhook Signature Verification

```
raw body bytes (read before json.Decode)
    +
"dyte-signature" header value
    +
webhookSecret (from KVStore)
    │
    ▼
HMAC-SHA256(secret, body) == signature → proceed
                                       → 401 if mismatch
```
