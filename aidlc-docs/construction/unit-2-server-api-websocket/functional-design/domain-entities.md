# Unit 2: Server API & WebSocket — Domain Entities

## Request DTOs

### CreateCallRequest
Request body for `POST /calls`.

| Field | Type | Required | Description |
|---|---|---|---|
| `channel_id` | string | Yes | Mattermost channel to start the call in |

---

### DismissCallRequest
No request body. Call ID provided via path parameter `{id}`.

---

### RTKWebhookEvent
Incoming JSON body from Cloudflare RTK webhook.

```json
{
  "event": "meeting.participantLeft",
  "meeting": {
    "id": "string (RTK meeting ID)",
    "sessionId": "string",
    "status": "LIVE",
    "startedAt": "2022-12-13T06:57:09.736Z"
  },
  "participant": {
    "peerId": "string",
    "userDisplayName": "string",
    "customParticipantId": "string (Mattermost userID)",
    "joinedAt": "2022-12-13T07:01:41.535Z",
    "leftAt": "2022-12-13T07:03:42.420Z"
  }
}
```

For `meeting.ended`:
```json
{
  "event": "meeting.ended",
  "meeting": {
    "id": "string (RTK meeting ID)",
    "sessionId": "string",
    "startedAt": "string",
    "endedAt": "string",
    "status": "ENDED"
  },
  "reason": "ALL_PARTICIPANTS_LEFT"
}
```

**Key mapping**: `participant.customParticipantId` = Mattermost `userID` (set at token generation time)
**Key mapping**: `meeting.id` = `CallSession.MeetingID` (RTK meeting ID, used to look up CallSession)

---

## Response DTOs

### CallResponse
Shared response shape for `POST /calls` and `POST /calls/{id}/token`.

```json
{
  "call": {
    "id": "string (UUID)",
    "channel_id": "string",
    "creator_id": "string",
    "meeting_id": "string",
    "participants": ["string"],
    "start_at": 1234567890000,
    "end_at": 0,
    "post_id": "string"
  },
  "token": "string (RTK JWT)"
}
```

---

### ConfigStatusResponse
Response for `GET /config/status`.

```json
{
  "enabled": true
}
```

---

### ErrorResponse
Shared error response for all endpoints.

```json
{
  "error": "human-readable error message"
}
```

---

## Error → HTTP Status Code Mapping

| Sentinel Error | HTTP Status | Description |
|---|---|---|
| `ErrCallAlreadyActive` | 409 Conflict | Channel already has an active call |
| `ErrCallNotFound` | 404 Not Found | Call not found or already ended |
| `ErrNotParticipant` | 403 Forbidden | User is not a participant (not used in Unit 2 — heartbeat deferred / not implemented) |
| `ErrUnauthorized` | 403 Forbidden | User is not the call creator (EndCall) |
| `ErrRTKNotConfigured` | 503 Service Unavailable | Cloudflare credentials not set |
| Missing required field | 400 Bad Request | Request body validation failure |
| Not system admin | 403 Forbidden | Admin-only endpoint |
| Other / internal | 500 Internal Server Error | Unexpected error |

---

## KVStore Key Schema Additions (Unit 2)

| Key Pattern | Value | Description |
|---|---|---|
| `call:meeting:{meetingID}` | CallSession (JSON) | Index for O(1) lookup by RTK meeting ID (used by webhook handler) |
| `webhook:id` | string | Registered RTK webhook ID |
| `webhook:secret` | string | RTK webhook signing secret (for signature verification) |

---

## WebSocket Event Payloads

All WebSocket events are emitted by Unit 1 (`calls.go`) via `p.API.PublishWebSocketEvent`. Unit 2 wires the HTTP triggers and webhook triggers that cause these emissions.

| Event | Trigger | Broadcast Scope |
|---|---|---|
| `custom_cf_call_started` | `POST /calls` → `CreateCall` | Channel |
| `custom_cf_user_joined` | `POST /calls/{id}/token` → `JoinCall` (new participant) | Channel |
| `custom_cf_user_left` | `meeting.participantLeft` webhook → `LeaveCall` | Channel |
| `custom_cf_call_ended` | `DELETE /calls/{id}` or `meeting.ended` webhook → `endCallInternal` | Channel |
| `custom_cf_notification_dismissed` | `POST /calls/{id}/dismiss` | User (all sessions) |

### custom_cf_notification_dismissed
```json
{
  "call_id": "string",
  "user_id": "string"
}
```
