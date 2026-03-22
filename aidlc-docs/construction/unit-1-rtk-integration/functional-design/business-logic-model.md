# Unit 1: RTK Integration — Business Logic Model

## CreateCall Flow

```
Input: channelID, userID (creator)

1. Check KVStore: GetCallByChannel(channelID)
   - If active call exists (EndAt == 0) → return ErrCallAlreadyActive (BR-01)

2. RTKClient.CreateMeeting("group_call_host")
   - On error → return error, abort (BR-05)
   - On success → meeting.ID

3. RTKClient.GenerateToken(meeting.ID, userID, "group_call_host")
   - On error → return error, abort
   - On success → token.Token

4. Build CallSession:
   - ID = newUUID()
   - ChannelID = channelID
   - CreatorID = userID
   - MeetingID = meeting.ID
   - Participants = [userID]
   - StartAt = now()
   - EndAt = 0

5. KVStore.SaveCall(session)
   - Write call:channel:{channelID} = session
   - Write call:id:{session.ID} = session

6. API.CreatePost(custom_cf_call post with session data)
   - On error: log warning, continue (no rollback — session remains in KVStore)
   - On success: store returned PostID in session → KVStore.SaveCall(session) (update with PostID)

7. API.PublishWebSocketEvent("custom_cf_call_started", payload)

8. Return (session, token.Token, nil)
```

---

## JoinCall Flow

```
Input: callID, userID

1. KVStore.GetCallByID(callID)
   - If not found or EndAt != 0 → return ErrCallNotActive (BR-06)

2. RTKClient.GenerateToken(session.MeetingID, userID, "group_call_participant")
   - On error → return error

3. Add userID to session.Participants (deduplicated) (BR-09)

4. KVStore.UpdateCallParticipants(callID, session.Participants)

4a. KVStore.SetHeartbeat(callID, userID, now())
    - Set initial heartbeat immediately on join to prevent stale-cleanup race condition
   - Update both call:channel:{channelID} and call:id:{callID}

5. API.PublishWebSocketEvent("custom_cf_user_joined", payload) (BR-10)

6. Return (token.Token, nil)
```

---

## LeaveCall Flow

```
Input: callID, userID

1. KVStore.GetCallByID(callID)
   - If not found → return nil (no-op, idempotent) (BR-11)
   - If EndAt != 0 → return nil (already ended, no-op)

2. Remove userID from session.Participants
   - If userID not in list → no-op, continue

3. KVStore.UpdateCallParticipants(callID, updatedParticipants)

4. API.PublishWebSocketEvent("custom_cf_user_left", payload) (BR-12)

5. If len(Participants) == 0 → EndCallInternal(session) (BR-13)

6. Return nil
```

---

## EndCall Flow (host-initiated)

```
Input: callID, requestingUserID

1. KVStore.GetCallByID(callID)
   - If not found or EndAt != 0 → return ErrCallNotActive

2. If session.CreatorID != requestingUserID → return ErrUnauthorized (BR-14)

3. EndCallInternal(session)

4. Return nil
```

---

## EndCallInternal (shared)

```
Input: session *CallSession

1. Set session.EndAt = now() (BR-15)
2. KVStore.EndCall(session.ID, session.EndAt)
   - Update both call:channel:{channelID} and call:id:{callID}

3. RTKClient.EndMeeting(session.MeetingID) — best effort (BR-16)
   - On error: log warning, continue

4. durationMs = session.EndAt - session.StartAt

5. API.UpdatePost(session.PostID, ended state props: EndAt, DurationMs) (BR-17)

6. API.PublishWebSocketEvent("custom_cf_call_ended", {
     call_id, channel_id, end_at, duration_ms
   }) (BR-18)
```

---

## HeartbeatCall Flow

```
Input: callID, userID

1. KVStore.GetCallByID(callID)
   - If not found or EndAt != 0 → return ErrCallNotActive (BR-19)

2. If userID not in session.Participants → return ErrNotParticipant (BR-20)

3. KVStore.SetHeartbeat(callID, userID, now()) (BR-21)

4. Return nil
```

---

## CleanupStaleParticipants Flow (Background Job)

```
Triggered every 30 seconds by job.go (BR-22)

1. KVStore.GetAllActiveCalls() → []CallSession (BR-23)
   - All sessions where EndAt == 0

2. cutoff = now() - 60_000 ms

3. For each session in activeCalls:
   For each userID in session.Participants:
     lastBeat = KVStore.GetHeartbeat(session.ID, userID)
     if lastBeat == 0 || lastBeat < cutoff: (BR-24)
       LeaveCall(session.ID, userID)
       // LeaveCall handles auto-end if last participant (BR-25)
```

---

## RTKClient Interface

```
interface RTKClient {
  CreateMeeting(preset string) (*Meeting, error)
  GenerateToken(meetingID, userID, preset string) (*Token, error)
  EndMeeting(meetingID string) error
}
```

Implementation uses HTTPS with Basic Auth (`orgID:apiKey`).

---

## KVStore Interface Extensions

```
GetCallByChannel(channelID string) (*CallSession, error)
GetCallByID(callID string) (*CallSession, error)
GetAllActiveCalls() ([]*CallSession, error)
SaveCall(session *CallSession) error
UpdateCallParticipants(callID string, participants []string) error
EndCall(callID string, endAt int64) error
SetHeartbeat(callID, userID string, ts int64) error
GetHeartbeat(callID, userID string) (int64, error)
GetStaleParticipants(callID string, cutoff int64) ([]string, error)
StoreVoIPToken(userID, token string) error
GetVoIPToken(userID string) (string, error)
```
