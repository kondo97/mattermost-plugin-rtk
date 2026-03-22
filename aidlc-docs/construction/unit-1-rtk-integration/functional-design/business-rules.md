# Unit 1: RTK Integration — Business Rules

## CreateCall Rules

| Rule | Description |
|---|---|
| BR-01 | Only one active call allowed per channel. If `call:channel:{channelID}` exists with `EndAt == 0`, return error. |
| BR-02 | Creator is assigned `group_call_host` preset in RTK. |
| BR-03 | Creator is automatically added to `Participants` on creation. |
| BR-04 | On success: create `custom_cf_call` post in channel, emit `custom_cf_call_started` WebSocket event, send push notification (delegated to push sender in Unit 6). |
| BR-05 | If RTKClient.CreateMeeting fails, abort — do not write to KVStore or post to channel. |

## JoinCall Rules

| Rule | Description |
|---|---|
| BR-06 | Call must be active (`EndAt == 0`). If not found or ended, return error. |
| BR-07 | No participant limit — any channel member may join. |
| BR-08 | Joining user is assigned `group_call_participant` preset in RTK. |
| BR-09 | UserID is appended to `Participants` in KVStore (deduplicated — no duplicate entries). |
| BR-09a | Set initial heartbeat (`SetHeartbeat(callID, userID, now())`) immediately after adding to participants, to prevent stale-cleanup race condition before the call page sends its first heartbeat. |
| BR-10 | Emit `custom_cf_user_joined` WebSocket event on success. |

## LeaveCall Rules

| Rule | Description |
|---|---|
| BR-11 | UserID is removed from `Participants`. If userID not present, operation is a no-op (idempotent). |
| BR-12 | Emit `custom_cf_user_left` WebSocket event after removal. |
| BR-13 | If `Participants` becomes empty after removal, auto-trigger `EndCallInternal`. |

## EndCall Rules

| Rule | Description |
|---|---|
| BR-14 | Only the creator (`CreatorID == requestingUserID`) may call EndCall. Others receive an authorization error. |
| BR-15 | Set `EndAt` to current Unix timestamp (ms) in KVStore. |
| BR-16 | Call `RTKClient.EndMeeting(meetingID)` — best effort: log failure but do not abort the end-call flow. |
| BR-17 | Update `custom_cf_call` post to ended state (set `EndAt` and `DurationMs` in post props). |
| BR-18 | Emit `custom_cf_call_ended` WebSocket event with `end_at` and `duration_ms`. |

## HeartbeatCall Rules

| Rule | Description |
|---|---|
| BR-19 | Call must be active (`EndAt == 0`). If not found or ended, return error. |
| BR-20 | UserID must be present in `Participants`. If not, return error (prevents unauthorized heartbeat registration). |
| BR-21 | Update `heartbeat:{callID}:{userID}` with current Unix timestamp (ms). |

## CleanupStaleParticipants Rules

| Rule | Description |
|---|---|
| BR-22 | Executed by Background Job every 30 seconds. |
| BR-23 | Scan all active calls (all KVStore keys matching `call:channel:*` where `EndAt == 0`). |
| BR-24 | For each participant in each active call: if `heartbeat:{callID}:{userID}` is older than 60 seconds (or missing), invoke `LeaveCall(userID, callID)`. |
| BR-25 | LeaveCall triggered from cleanup follows the same rules as BR-11 through BR-13 (including auto-end if last participant). |

## EndCallInternal (shared logic for auto-end and host-end)

`EndCallInternal` is the shared implementation called by both `EndCall` (host-initiated) and `LeaveCall` (auto-end when last participant leaves).

| Rule | Description |
|---|---|
| BR-26 | Set `EndAt` in KVStore. |
| BR-27 | Call `RTKClient.EndMeeting` — best effort. |
| BR-28 | Update post to ended state. |
| BR-29 | Emit `custom_cf_call_ended` WebSocket event. |
