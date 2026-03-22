# Application Design

## Overview

This document consolidates the application design for `mattermost-plugin-rtk`. It defines the system structure, component responsibilities, and key interaction flows based on the INCEPTION phase artifacts.

---

## Design Decisions

| # | Decision | Choice | Rationale |
|---|---|---|---|
| Q1 | Floating widget | Yes (Mattermost-side widget) | Aligns with Mattermost Calls UX; US-006 |
| Q2 | Backend service layer | Flat (Plugin struct + api/) | Acceptable scope; avoids over-engineering |
| Q3 | Frontend state | Redux slice | Mattermost plugin standard; survives navigation |
| Q4 | Call page auth | Token + Mattermost session | Session cookie automatic (same domain); Calls plugin pattern |
| Q5 | Leave detection | sendBeacon + heartbeat (60s timeout) | Balances reliability vs false-positive risk |

---

## System Architecture

```
Browser (Mattermost SPA)                    Browser (New Tab)
+---------------------------------+         +------------------+
| Main Bundle (main.js)           |         | Call Bundle      |
|  ChannelHeaderButton            |         | (call.js)        |
|  CallPost                       |         |  DyteProvider    |
|  ToastBar                       |  opens  |  heartbeat loop  |
|  FloatingWidget  +--------------+-------->|  sendBeacon      |
|  SwitchCallModal |  /call?token=|         +--------+---------+
|  IncomingCallNotification       |                  |
|  AdminSettings                  |    heartbeat /   |
|  Calls Redux (slice + WS)       |    leave API     |
+---------------+--+--------------+                  |
                |  ^ WebSocket events                 |
                |  |                                  |
                v  |                                  v
+----------------------------------------------------------+
|                  Mattermost Server                        |
|  +----------------------------------------------------+  |
|  |  mattermost-plugin-rtk                             |  |
|  |                                                    |  |
|  |  Plugin Core (plugin.go)                           |  |
|  |  +- CreateCall / JoinCall / LeaveCall / EndCall    |  |
|  |  +- HeartbeatCall / CleanupStaleParticipants       |  |
|  |                                                    |  |
|  |  API Handler (api/)                                |  |
|  |  +- /api/v1/calls (CRUD + token + leave)           |  |
|  |  +- /api/v1/calls/{id}/heartbeat                   |  |
|  |  +- /api/v1/config/status                          |  |
|  |  +- /api/v1/mobile/voip-token                      |  |
|  |  +- /call, /call.js, /worker.js (static)           |  |
|  |                                                    |  |
|  |  Configuration (configuration.go)                  |  |
|  |  +- Cloudflare credentials + 10 feature flags      |  |
|  |  +- Env var overrides                              |  |
|  |                                                    |  |
|  |  Background Job (job.go)                           |  |
|  |  +- Heartbeat cleanup every 30s                    |  |
|  +----------------------------------------------------+  |
+-------+------------------+-------------------------------+
        |                  |
        v                  v
+-------+------+  +--------+-----------+
| Cloudflare   |  | Mattermost KVStore |
| RTK API      |  | call:channel:{id}  |
| (rtkclient/) |  | call:id:{id}       |
|              |  | heartbeat:{id}:{u} |
| CreateMeeting|  | voip:{userID}      |
| GenToken     |  +--------------------+
| EndMeeting   |
+--------------+
```

---

## Key Interaction Flows

### Flow 1: Start a Call

```
User clicks "Start call"
  -> POST /api/v1/calls  (channelID in body)
       -> Check KVStore: no active call for channel
       -> RTKClient.CreateMeeting("group_call_host")
       -> RTKClient.GenerateToken(meetingID, userID, "group_call_host")
       -> KVStore.SaveCall(session)
       -> Post custom_cf_call to channel
       -> Push.SendIncomingCall (to channel members)
       -> PublishWebSocketEvent(custom_cf_call_started)
       <- {call_id, token, feature_flags}
  -> Redux: startCallSuccess(channelID, callID, token)
  -> FloatingWidget renders
  -> New tab opens: /plugins/{id}/call?token=<jwt>
  -> CallPage: DyteProvider initializes with token
  -> CallPage: heartbeat interval starts
```

### Flow 2: Join a Call

```
User clicks "Join call" in CallPost
  -> POST /api/v1/calls/{callId}/token
       -> KVStore.GetCallByID(callID) — verify active
       -> RTKClient.GenerateToken(callID, userID, "group_call_participant")
       -> KVStore.UpdateCallParticipants (add userID)
       -> PublishWebSocketEvent(custom_cf_user_joined)
       <- {token, feature_flags}
  -> Redux: joinCallSuccess
  -> New tab opens: /plugins/{id}/call?token=<jwt>
```

### Flow 3: Leave (Tab Close — Normal)

```
User closes call tab
  -> beforeunload fires
  -> navigator.sendBeacon POST /api/v1/calls/{callId}/leave
       -> KVStore: remove userID from participants
       -> PublishWebSocketEvent(custom_cf_user_left)
       -> If participants empty: EndCallInternal(callID)
  -> Redux (via WS event): userLeft / callEnded
  -> FloatingWidget disappears
```

### Flow 4: Leave (Tab Close — Crash / Network Drop Fallback)

```
sendBeacon fails (crash, kill -9, severe network drop)
  -> heartbeat stops arriving
  -> Background job (every 30s) checks heartbeat:{callID}:{userID}
  -> After 60s without heartbeat: LeaveCall(userID, callID)
       -> Same flow as Flow 3 from server side
```

### Flow 5: End Call (Host)

```
Host clicks "End call" in call UI
  -> DELETE /api/v1/calls/{callId}
       -> Verify userID == session.CreatorID
       -> EndCallInternal(callID):
            KVStore.EndCall (set end_at)
            Update custom_cf_call post to ended state
            PublishWebSocketEvent(custom_cf_call_ended)
  -> All clients: Redux callEnded -> ToastBar dismissed, CallPost -> ended state
```

---

## File Structure (New / Modified)

```
server/
  plugin.go              (modified — add call logic methods)
  configuration.go       (modified — add RTK fields + feature flags)
  job.go                 (modified — add heartbeat cleanup)
  api/
    handler.go           (new)
    calls.go             (new)
    heartbeat.go         (new)
    config.go            (new)
    mobile.go            (new)
    static.go            (new)
  rtkclient/
    interface.go         (new)
    client.go            (new)
  push/
    push.go              (new)
  store/kvstore/
    kvstore.go           (modified — extend interface + implementation)

webapp/src/
  index.tsx              (modified — register all components + reducer)
  components/
    channel_header_button/  (new)
    call_post/              (new)
    toast_bar/              (new)
    floating_widget/        (new)
    switch_call_modal/      (new)
    incoming_call_notification/  (new)
    admin_settings/         (new)
  redux/
    calls_slice.ts          (new)
    websocket_handlers.ts   (new)
    selectors.ts            (new)

webapp/src/call/            (new — separate Vite bundle entry)
  index.tsx
  CallPage.tsx
```

---

## Security Considerations

- All API endpoints except `/call`, `/call.js`, `/worker.js` require `Mattermost-User-ID` header (NFR-02)
- Admin endpoints additionally verify admin role server-side
- Cloudflare API credentials never returned to frontend
- RTK JWT tokens are short-lived (RTK SDK enforces expiry)
- Heartbeat endpoint verifies user is a participant of the specified call before updating

---

## Artifacts

| Document | Content |
|---|---|
| `components.md` | Full component inventory (backend + frontend) |
| `component-methods.md` | Method signatures per component |
| `services.md` | RTKClient, KVStore, PluginAPI interfaces; WebSocket event contracts |
| `component-dependency.md` | Dependency graphs (backend, frontend main, call bundle) |
| `application-design.md` | This document — consolidated design and interaction flows |
