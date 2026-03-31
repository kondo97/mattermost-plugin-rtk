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
| Q5 | Leave detection | fetch+keepalive on beforeunload | Custom headers (CSRF) not possible with sendBeacon; heartbeat deferred |

---

## System Architecture

```
Browser (Mattermost SPA)                    Browser (New Tab)
+---------------------------------+         +------------------+
| Main Bundle (main.js)           |         | Call Bundle      |
|  ChannelHeaderButton            |         | (call.js)        |
|  CallPost                       |         |  RtkProvider     |
|  ToastBar                       |  opens  |  fetch+keepalive |
|  FloatingWidget  +--------------+-------->|  on beforeunload |
|  SwitchCallModal |  /call?token=|         +--------+---------+
|  IncomingCallNotification       |                  |
|  AdminSettings                  |    leave API     |
|  Calls Redux (slice + WS)       |                  |
+---------------+--+--------------+                  |
                |  ^ WebSocket events                 |
                |  |                                  |
                v  |                                  v
+----------------------------------------------------------+
|                  Mattermost Server                        |
|  +----------------------------------------------------+  |
|  |  mattermost-plugin-rtk                             |  |
|  |                                                    |  |
|  |  Plugin Core (plugin.go, calls.go)                 |  |
|  |  +- CreateCall / JoinCall / LeaveCall / EndCall    |  |
|  |                                                    |  |
|  |  API Handler (api.go, api_calls.go, ...)           |  |
|  |  +- /api/v1/calls (CRUD + token + leave)           |  |
|  |  +- /api/v1/config/status                          |  |
|  |  +- /api/v1/mobile/voip-token                      |  |
|  |  +- /call, /call.js, /worker.js (static)           |  |
|  |                                                    |  |
|  |  Configuration (configuration.go)                  |  |
|  |  +- Cloudflare credentials + 10 feature flags      |  |
|  |  +- Env var overrides                              |  |
|  |                                                    |  |
|  |  Cleanup Loop (cleanup.go)                         |  |
|  |  +- Placeholder for future RTK participant sync    |  |
|  +----------------------------------------------------+  |
+-------+------------------+-------------------------------+
        |                  |
        v                  v
+-------+------+  +--------+-----------+
| Cloudflare   |  | Mattermost KVStore |
| RTK API      |  | call:channel:{id}  |
| (rtkclient/) |  | call:id:{id}       |
|              |  | voip:{userID}      |
| CreateMeeting|  +--------------------+
| GenToken     |
| EndMeeting   |
| GetParticipants |
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
       -> PublishWebSocketEvent(custom_cf_call_started)
       <- {call_id, token, feature_flags}
  -> Redux: startCallSuccess(channelID, callID, token)
  -> FloatingWidget renders (inline RtkMeeting via RTK React SDK)
  -> CallPage: RtkProvider initializes with token
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

### Flow 3: Leave (Tab / Widget Close — Normal)

```
User closes call tab or clicks Leave in floating widget
  -> beforeunload fires (tab) / handleClose called (widget)
  -> fetch+keepalive POST /api/v1/calls/{callId}/leave
       (uses fetch instead of sendBeacon — custom CSRF header required)
       -> KVStore: remove userID from participants
       -> PublishWebSocketEvent(custom_cf_user_left)
       -> If participants empty: EndCallInternal(callID)
  -> Redux (via WS event): userLeft / callEnded
  -> FloatingWidget disappears
```

### Flow 4: Leave (Crash / Network Drop)

```
NOTE: Currently no automatic fallback for crash or network drop.
If the browser crashes or network drops, the user remains listed
as a participant until the call is manually ended by the creator.

Future: cleanup.go will periodically call RTKClient.GetMeetingParticipants()
to reconcile stale participants against RTK's actual active participants.
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
  plugin.go              (modified — OnActivate, ServeHTTP, cleanup loop)
  configuration.go       (modified — add RTK fields + feature flags)
  calls.go               (new — CreateCall, JoinCall, LeaveCall, EndCall)
  cleanup.go             (new — placeholder for RTK participant reconciliation)
  api.go                 (new — router setup, auth middleware)
  api_calls.go           (new — call CRUD + token + leave endpoints)
  api_config.go          (new — GET /config/status, /config/admin-status)
  api_mobile.go          (new — POST /mobile/voip-token, /calls/{id}/dismiss)
  api_static.go          (new — /call HTML, /call.js, /worker.js)
  api_webhook.go         (new — RTK webhook handler)
  rtkclient/
    interface.go         (new)
    client.go            (new)
  store/kvstore/
    kvstore.go           (modified — extend interface + implementation)
    calls.go             (new — call session KVStore methods)

webapp/src/
  index.tsx              (modified — register all components + reducer)
  utils/
    rtk_lang_ja.ts       (new — Japanese translations for RTK SDK UI)
  components/
    channel_header_button/  (new)
    call_post/              (new)
    toast_bar/              (new)
    floating_widget/        (new)
    switch_call_modal/      (new)
    incoming_call_notification/  (new)
    admin_config/           (new — EnvVarCredentialSetting)
  redux/
    calls_slice.ts          (new)
    websocket_handlers.ts   (new)
    selectors.ts            (new)

webapp/src/call_page/       (new — separate Vite bundle entry)
  main.tsx
  CallPage.tsx
```

---

## Security Considerations

- All API endpoints except `/call`, `/call.js`, `/worker.js` require `Mattermost-User-ID` header (NFR-02)
- Admin endpoints additionally verify admin role server-side
- Cloudflare API credentials never returned to frontend
- RTK JWT tokens are short-lived (RTK SDK enforces expiry)

---

## Artifacts

| Document | Content |
|---|---|
| `components.md` | Full component inventory (backend + frontend) |
| `component-methods.md` | Method signatures per component |
| `services.md` | RTKClient, KVStore, PluginAPI interfaces; WebSocket event contracts |
| `component-dependency.md` | Dependency graphs (backend, frontend main, call bundle) |
| `application-design.md` | This document — consolidated design and interaction flows |
