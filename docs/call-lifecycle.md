# Call Lifecycle: Use Case Implementations

This document describes the implementation flow for each major use case in the RTK plugin. For each use case, it covers: API endpoints, app-layer functions, database operations, RTK API calls, WebSocket events, and edge cases.

---

## Table of Contents

1. [Start Call](#1-start-call)
2. [Join Call](#2-join-call)
3. [Leave Call](#3-leave-call)
4. [End Call (by creator)](#4-end-call-by-creator)
5. [Get Call (with reconciliation)](#5-get-call-with-reconciliation)
6. [RTK Webhook: Participant Joined](#6-rtk-webhook-participant-joined)
7. [RTK Webhook: Participant Left](#7-rtk-webhook-participant-left)
8. [RTK Webhook: Meeting Ended](#8-rtk-webhook-meeting-ended)
9. [Edge Cases](#9-edge-cases)

---

## 1. Start Call

**Trigger**: User initiates a call in a Mattermost channel.

### API

```
POST /api/v1/calls
Header: Mattermost-User-ID: <userID>
Body:   { "channel_id": "<channelID>" }

Response 201: { "call": CallSession, "token": "<rtk-jwt>" }
Response 403: user is not a channel member
Response 409: a call is already active in the channel
Response 503: RTK not configured
```

### App Layer: `CreateCall(channelID, userID)`

Steps executed inside `callMu` lock:

| Step | Action | Detail |
|------|--------|--------|
| 1 | Check channel membership | MM API: `GetChannelMember`; returns `ErrNotChannelMember` if absent |
| 2 | Check for existing active call | `store.GetCallByChannel(channelID)` |
| 2a | ↳ Active call found → verify alive | `rtkclient.GetMeeting(meetingID)` |
| 2b | ↳ RTK returns 404 → force-end stale call | `store.EndCall(callID, now)` + emit `call_ended` WS event |
| 2c | ↳ RTK call is live → return `ErrCallAlreadyActive` | |
| 3 | Resolve RTK meeting for channel | `store.GetChannelMeeting(channelID)` |
| 3a | ↳ No meeting or app config changed → create new meeting | `rtkclient.CreateMeeting()` → `store.SaveChannelMeeting(...)` |
| 4 | Mint `session.ID` | 26-char NewID (Mattermost format) |
| 5 | Generate RTK host token | `rtkclient.GenerateToken(meetingID, callID, userID, displayName, "group_call_host")` |
| 6 | Create Mattermost post | `CreatePost(custom_cf_call)` — must succeed before DB insert |
| 7 | Persist call + creator participant | `store.CreateCallSession(session)` — atomic insert into both tables |
| 8 | Send push notifications | `sendPushNotifications(channelID)` — notify channel members |
| 9 | Emit WebSocket event | `PublishWebSocketEvent("call_started", {...})` |

### Database Writes

| Table | Operation | Notes |
|-------|-----------|-------|
| `rtk_channel_meetings` | `INSERT ... ON CONFLICT DO UPDATE` | Only when meeting is created or app config changed |
| `rtk_call_sessions` | `INSERT` | `endat=0`, `post_id` set, `session_id=""` (filled later by webhook) |
| `rtk_call_participants` | `INSERT` | Creator added with `joined_at=now` |

### RTK API Calls

| Call | Endpoint | Condition |
|------|----------|-----------|
| `GetMeeting` | `GET /meetings/{meetingID}` | Only if an active call record already exists |
| `CreateMeeting` | `POST /meetings` | Only if channel has no meeting or app config changed |
| `GenerateToken` | `POST /meetings/{meetingID}/participants` | Always; preset = `group_call_host` |

### WebSocket Events Emitted

| Event | Payload |
|-------|---------|
| `call_ended` | `{ call_id, channel_id, end_at, duration_ms }` — only if a stale call was force-ended |
| `call_started` | `{ call_id, channel_id, creator_id, participants, start_at, post_id }` |

### Sequence Diagram

```
Client                  API                 App               Store              RTK
  |                      |                   |                  |                  |
  |-- POST /calls -----→ |                   |                  |                  |
  |                      |-- CreateCall() →  |                  |                  |
  |                      |                   |-- GetCallByChannel() →              |
  |                      |                   |                  |←- nil / session  |
  |                      |                   |-- [if active] GetMeeting() ------→  |
  |                      |                   |                  |         ←- 200/404
  |                      |                   |-- GetChannelMeeting() →             |
  |                      |                   |                  |←- meetingID      |
  |                      |                   |-- [if new] CreateMeeting() ------→  |
  |                      |                   |                  |         ←- meetingID
  |                      |                   |-- SaveChannelMeeting() →            |
  |                      |                   |-- GenerateToken() ---------------→  |
  |                      |                   |                  |         ←- JWT   |
  |                      |                   |-- CreatePost() →                    |
  |                      |                   |-- CreateCallSession() →             |
  |                      |                   |-- sendPushNotifications()           |
  |                      |                   |-- PublishWS("call_started")         |
  |                      |←- 201 { call, token }                                  |
  |←- { call, token } - |                   |                  |                  |
  |                      |                   |                  |                  |
  |-- connect RTK WS ----------------------------------------→  |                  |
```

---

## 2. Join Call

**Trigger**: A user (not the creator) joins an ongoing call.

### API

```
POST /api/v1/calls/{callID}/token
Header: Mattermost-User-ID: <userID>

Response 200: { "call": CallSession, "token": "<rtk-jwt>" }
Response 403: user is not a channel member
Response 404: call not found or already ended
Response 503: RTK not configured
```

### App Layer: `JoinCall(callID, userID)`

Steps executed inside `callMu` lock:

| Step | Action | Detail |
|------|--------|--------|
| 1 | Load call | `store.GetCallByID(callID)`; returns `ErrCallNotFound` if `EndAt != 0` |
| 2 | Check channel membership | MM API: `GetChannelMember` |
| 3 | Verify RTK meeting alive | `rtkclient.GetMeeting(meetingID)` |
| 3a | ↳ 404 → force-end stale call, return error | `store.EndCall(...)` + emit `call_ended` |
| 4 | Add participant (atomic) | `store.AddCallParticipant(callID, userID)` — `SELECT FOR UPDATE` serializes concurrent joins |
| 5 | Generate RTK participant token | `rtkclient.GenerateToken(meetingID, callID, userID, displayName, "group_call_participant")` |
| 5a | ↳ Token failure → compensate | `store.RemoveCallParticipant(callID, userID)` only if step 4 actually inserted a new row |
| 6 | Update Mattermost post | `updatePostParticipants(session)` — best-effort |
| 7 | Emit WebSocket event | `PublishWebSocketEvent("user_joined", {...})` |

### Database Writes

| Table | Operation | Notes |
|-------|-----------|-------|
| `rtk_call_participants` | `INSERT ... ON CONFLICT DO NOTHING` | Idempotent; `added=true` only if a new row was inserted |
| `rtk_call_sessions` | `UPDATE` (via post update) | Indirect: post content updated to reflect new participants list |

### RTK API Calls

| Call | Endpoint | Notes |
|------|----------|-------|
| `GetMeeting` | `GET /meetings/{meetingID}` | Verifies meeting still exists |
| `GenerateToken` | `POST /meetings/{meetingID}/participants` | preset = `group_call_participant` |

### WebSocket Events Emitted

| Event | Payload |
|-------|---------|
| `call_ended` | Only if stale call was detected and force-ended |
| `user_joined` | `{ call_id, channel_id, user_id, participants }` |

---

## 3. Leave Call

**Trigger**: A participant disconnects from a call (user action or client disconnect).

### API

```
POST /api/v1/calls/{callID}/leave
Header: Mattermost-User-ID: <userID>

Response 200: {}
```

### App Layer: `LeaveCall(callID, userID)`

Steps executed inside `callMu` lock:

| Step | Action | Detail |
|------|--------|--------|
| 1 | Load call | `store.GetCallByID(callID)`; returns `nil` (idempotent) if not found or already ended |
| 2 | Remove participant (atomic) | `store.RemoveCallParticipant(callID, userID)` |
| 2a | ↳ Returns `(participants, endedNow, endAt, err)` | `endedNow=true` if this transaction emptied the participants and set `EndAt` |
| 3 | Update Mattermost post | `updatePostParticipants(session)` |
| 4 | Emit `user_left` event | `PublishWebSocketEvent("user_left", {...})` |
| 5 | [If `endedNow`] Emit `call_ended` | `emitCallEnded(session, endAt, "last_participant_left")` |

### Database Writes

| Table | Operation | Notes |
|-------|-----------|-------|
| `rtk_call_participants` | `DELETE WHERE rtk_call_sessions_id=? AND user_id=?` | Inside `SELECT FOR UPDATE` transaction |
| `rtk_call_sessions` | `UPDATE SET endat=now` | Only when last participant leaves (`endedNow=true`) |

### RTK API Calls

None. The client disconnects from the RTK WebSocket directly; no server-side RTK call is needed.

### WebSocket Events Emitted

| Event | Payload | Condition |
|-------|---------|-----------|
| `user_left` | `{ call_id, channel_id, user_id, participants }` | Always |
| `call_ended` | `{ call_id, channel_id, end_at, duration_ms }` | Only if last participant left |

---

## 4. End Call (by creator)

**Trigger**: The call creator explicitly ends the call for all participants.

### API

```
DELETE /api/v1/calls/{callID}
Header: Mattermost-User-ID: <userID>

Response 200: {}
Response 403: user is not the call creator
Response 404: call not found or already ended
```

### App Layer: `EndCall(callID, requestingUserID)`

Steps executed inside `callMu` lock:

| Step | Action | Detail |
|------|--------|--------|
| 1 | Load call | `store.GetCallByID(callID)`; `ErrCallNotFound` if `EndAt != 0` |
| 2 | Verify creator | `session.CreatorID == requestingUserID`; else `ErrUnauthorized` |
| 3 | End call internally | `endCallInternal(session, "creator_ended")` |

### `endCallInternal(session, reason)`

| Step | Action |
|------|--------|
| 1 | `store.EndCall(callID, now)` — sets `EndAt` |
| 2 | `emitCallEnded(session, endAt, reason)` |

### `emitCallEnded(session, endAt, reason)`

| Step | Action |
|------|--------|
| 1 | Update Mattermost post with `end_at` and `duration_ms` |
| 2 | Send push notifications (dismiss ringing UI on mobile) |
| 3 | `PublishWebSocketEvent("call_ended", { call_id, channel_id, end_at, duration_ms })` |

### Database Writes

| Table | Operation | Notes |
|-------|-----------|-------|
| `rtk_call_sessions` | `UPDATE SET endat=now WHERE id=? AND endat=0` | Idempotent (partial condition on `endat=0`) |

### RTK API Calls

None. RTK meetings are **not deleted** — they are reused per channel across multiple calls. Participants disconnect from the RTK WebSocket after receiving the `call_ended` WebSocket event.

### WebSocket Events Emitted

| Event | Payload |
|-------|---------|
| `call_ended` | `{ call_id, channel_id, end_at, duration_ms }` |

---

## 5. Get Call (with reconciliation)

**Trigger**: A client fetches call state (e.g., on page load or reconnect).

### API

```
GET /api/v1/calls/{callID}
Header: Mattermost-User-ID: <userID>

Response 200: { "call": CallSession }
Response 404: call not found
```

### App Layer: `GetCallByID` + `ReconcileCallOnDemand`

| Step | Action | Detail |
|------|--------|--------|
| 1 | Load call | `store.GetCallByID(callID)` |
| 2 | [If active] Reconcile | `ReconcileCallOnDemand(session)` |
| 2a | ↳ `GetMeeting` → 404 | `endCallInternal(session, "rtk_meeting_gone")` |
| 2b | ↳ Transient error | Ignored — treat call as still alive |

### RTK API Calls

| Call | Condition |
|------|-----------|
| `GetMeeting` | Only if `EndAt == 0` (call is still active) |

---

## 6. RTK Webhook: Participant Joined

**Trigger**: Cloudflare RTK notifies that a participant connected to the meeting WebSocket.

### Webhook Payload

```json
{
  "event": "meeting.participantJoined",
  "meeting": { "id": "<meetingID>", "sessionId": "<rtkSessionID>" },
  "participant": { "customParticipantId": "<callID>:<userID>" }
}
```

### API Layer: `handleRTKWebhook`

Parses `customParticipantId` with `ParseCustomParticipantID(s)` → `(callID, userID, ok)`.

### App Layer: `HandleWebhookParticipantJoined(meetingID, callID, userID, sessionID)`

| Step | Action | Detail |
|------|--------|--------|
| 1 | Validate callID | Reject if empty (legacy token without callID binding) |
| 2 | Load active call | `store.GetCallByMeetingID(meetingID)` |
| 3 | Verify callID matches | Reject delayed webhooks from a prior call on the same reused meeting |
| 4 | Check channel membership | Reject participants removed from channel |
| 5 | Rescue-add participant | `store.AddCallParticipant(callID, userID)` — idempotent; syncs DB with RTK truth |
| 6 | Backfill session ID | `store.UpdateCallSessionID(callID, sessionID)` if `session.SessionID == ""` |
| 7 | Update post | `updatePostParticipants(session)` |
| 8 | Emit WebSocket event | `PublishWebSocketEvent("user_joined", {...})` |

**Purpose**: Acts as a rescue mechanism. If step 4 (`AddCallParticipant`) in [Join Call](#2-join-call) succeeded but the RTK token call failed and was rolled back, the webhook re-adds the participant once they actually connect.

### Database Writes

| Table | Operation | Notes |
|-------|-----------|-------|
| `rtk_call_participants` | `INSERT ... ON CONFLICT DO NOTHING` | Idempotent |
| `rtk_call_sessions` | `UPDATE SET session_id=?` | Only if `session_id` was not yet set |

---

## 7. RTK Webhook: Participant Left

**Trigger**: Cloudflare RTK notifies that a participant disconnected.

### Webhook Payload

```json
{
  "event": "meeting.participantLeft",
  "meeting": { "id": "<meetingID>" },
  "participant": { "customParticipantId": "<callID>:<userID>" }
}
```

### App Layer: `HandleWebhookParticipantLeft(meetingID, callID, userID)`

| Step | Action | Detail |
|------|--------|--------|
| 1 | Validate callID | Reject if empty |
| 2 | Load active call | `store.GetCallByMeetingID(meetingID)` |
| 3 | Verify callID matches | Reject delayed webhooks |
| 4 | Call `LeaveCall` | Delegates to the same logic as [Leave Call](#3-leave-call) — fully idempotent |

**Purpose**: Handles cases where the client disconnects without calling `POST /calls/{id}/leave` (e.g., browser crash, network drop).

---

## 8. RTK Webhook: Meeting Ended

**Trigger**: Cloudflare RTK notifies that the meeting session was terminated on the RTK side.

### Webhook Payload

```json
{
  "event": "meeting.ended",
  "meeting": { "id": "<meetingID>" }
}
```

### App Layer: `HandleWebhookMeetingEnded(meetingID)`

| Step | Action | Detail |
|------|--------|--------|
| 1 | Load active call | `store.GetCallByMeetingID(meetingID)`; return if none found |
| 2 | Acquire `callMu` lock | Prevents race with concurrent `EndCall` or `LeaveCall` |
| 3 | Re-check under lock | TOCTOU prevention: re-load call and confirm `EndAt == 0` |
| 4 | End call internally | `endCallInternal(session, "rtk_webhook")` |

**Purpose**: Handles RTK-initiated termination (e.g., Cloudflare killed the session due to inactivity or policy). The call is idempotently ended — if `EndCall` already ran, the `UPDATE ... WHERE endat=0` is a no-op.

---

## 9. Edge Cases

### Stale Call Detection

A call record may survive in the database with `EndAt=0` while the RTK meeting no longer exists (e.g., plugin restart, RTK service issue).

| Trigger | Detection | Resolution |
|---------|-----------|------------|
| `CreateCall` — active call exists | `GetMeeting` → 404 | `store.EndCall` + emit `call_ended`; proceed to create new call |
| `JoinCall` | `GetMeeting` → 404 | `store.EndCall` + emit `call_ended`; return `ErrCallNotFound` |
| `GET /calls/{id}` | `GetMeeting` → 404 | `endCallInternal`; return updated (ended) call state |
| Transient RTK error | `GetMeeting` → non-404 error | Treat as alive; do not force-end |

### Delayed Webhooks on Reused Meetings

RTK meetings are permanent per channel and reused across calls. A webhook from a previous call's participant may arrive after a new call has started on the same meeting.

**Prevention**: `customParticipantId` embeds `callID`. Webhook handlers parse this ID and verify it matches the currently active call's ID before taking any action.

### Concurrent Join/Leave/End

Multiple Mattermost nodes may handle requests simultaneously.

| Mechanism | Scope | Protects |
|-----------|-------|---------|
| `callMu` (in-memory mutex) | Single node | Prevents interleaved state reads/writes within one process |
| `SELECT FOR UPDATE` | Cross-node (DB) | Serializes `AddCallParticipant` / `RemoveCallParticipant` across all nodes |
| Unique index `(channel_id) WHERE endat=0` | Cross-node (DB) | Prevents two concurrent `CreateCall` attempts from both succeeding |

### Token Generation Failure After Participant Added

In `JoinCall`, `AddCallParticipant` runs before `GenerateToken`. If the token call fails:

- If `added=true` (this call inserted the participant row): compensate by calling `RemoveCallParticipant`.
- If `added=false` (participant was already in DB): do not remove — the user may be reconnecting.

### Auto-End on Last Participant Leave

`RemoveCallParticipant` atomically checks the participant count inside the `SELECT FOR UPDATE` transaction. If it reaches zero, it sets `EndAt=now` and returns `endedNow=true`. Only the transaction that sets `EndAt` emits the `call_ended` event — preventing double-emission if concurrent requests both try to be the last to leave.
