# Unit 2: Server API & WebSocket — Business Logic Model

## Architecture

Unit 2 is a thin HTTP adapter layer plus a webhook receiver. Each handler delegates to Unit 1 Plugin methods.

```
HTTP Request (client)
    │
    ▼
Auth Middleware (Mattermost-User-ID check)
    │
    ▼
Handler → Plugin Method (Unit 1)

RTK Webhook POST (Cloudflare → plugin)
    │
    ▼
Signature Verification (dyte-signature header)
    │
    ▼
Webhook Handler → Plugin Method (LeaveCall / endCallInternal)
```

---

## Endpoint Flows

### POST /calls — CreateCall

```
1. Auth middleware: userID from Mattermost-User-ID header
2. Parse body → channel_id (required, non-empty → 400 if missing)
3. p.callMu.Lock(); defer p.callMu.Unlock()
4. session, token, err = p.CreateCall(channel_id, userID)
5. Map error → HTTP status
6. 201 Created: { "call": session, "token": token }
```

---

### POST /calls/{id}/token — JoinCall

```
1. Auth middleware: userID from header
2. Extract callID from path
3. p.callMu.Lock(); defer p.callMu.Unlock()
4. token, err = p.JoinCall(callID, userID)
5. Map error → HTTP status
6. Fetch updated session: p.kvStore.GetCallByID(callID)
7. 200 OK: { "call": session, "token": token }
```

---

### POST /calls/{id}/leave — LeaveCall

```
1. Auth middleware: userID from header
2. Extract callID from path
3. p.callMu.Lock(); defer p.callMu.Unlock()
4. err = p.LeaveCall(callID, userID)
5. 200 OK (not found / ended is a no-op)
```

---

### DELETE /calls/{id} — EndCall

```
1. Auth middleware: userID from header
2. Extract callID from path
3. p.callMu.Lock(); defer p.callMu.Unlock()
4. err = p.EndCall(callID, userID)
5. Map error → HTTP status
6. 200 OK
```

---

### GET /config/status

```
1. Auth middleware: userID from header
2. enabled = GetEffectiveOrgID() != "" && GetEffectiveAPIKey() != ""
3. 200 OK: { "enabled": enabled }
```

---

### GET /config/admin-status

```
1. Auth middleware: userID from header
2. HasPermissionTo(userID, model.PermissionManageSystem) → 403 if false
3. 200 OK: { ...admin config... }  (schema defined in Unit 5)
```

---

### POST /calls/{id}/dismiss

```
1. Auth middleware: userID from header
2. Extract callID from path
3. Emit WebSocket event to userID only:
   custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed: { call_id, user_id }
4. 200 OK (always — idempotent)
```

---

### POST /api/v1/webhook/rtk — RTK Webhook Receiver

```
1. NO Mattermost auth middleware (Cloudflare sends this, not Mattermost clients)
2. Read raw body bytes (needed for signature verification)
3. Verify signature:
   secret = p.kvStore.GetWebhookSecret()
   if !verifyRTKSignature(r.Header.Get("dyte-signature"), body, secret):
     → 401 Unauthorized
4. Parse JSON event
5. Switch event.event:
   "meeting.participantLeft":
     meetingID = event.meeting.id
     userID = event.participant.customParticipantId
     session = p.kvStore.GetCallByMeetingID(meetingID)
     if session == nil || session.EndAt != 0: → 200 OK (idempotent)
     p.callMu.Lock()
     p.LeaveCall(session.ID, userID)
     p.callMu.Unlock()

   "meeting.ended":
     meetingID = event.meeting.id
     session = p.kvStore.GetCallByMeetingID(meetingID)
     if session == nil || session.EndAt != 0: → 200 OK (idempotent)
     p.callMu.Lock()
     p.endCallInternal(session)
     p.callMu.Unlock()

   default: → 200 OK (unknown events ignored)
6. 200 OK
```

**Deduplication**: RTK may retry failed webhooks (up to 5 times). The `session.EndAt != 0` check ensures idempotency.

---

### Static Files (no auth)

```
GET /call       → serve embedded call.html (with HTTP security headers)
GET /call.js    → serve embedded call.js
GET /worker.js  → serve embedded worker.js
```

---

## OnActivate: Webhook Registration

```
OnActivate()
  ...existing init...
  if rtkClient != nil:
    existingID = p.kvStore.GetWebhookID()
    if existingID == "":
      siteURL = p.API.GetConfig().ServiceSettings.SiteURL
      webhookURL = siteURL + "/plugins/{pluginID}/api/v1/webhook/rtk"
      id, secret, err = p.rtkClient.RegisterWebhook(webhookURL, [
        "meeting.participantLeft",
        "meeting.ended"
      ])
      if err: LogWarn (best effort — plugin still works without webhook)
      else:
        p.kvStore.StoreWebhookID(id)
        p.kvStore.StoreWebhookSecret(secret)
```

**Best-effort**: Webhook registration failure is logged but does not abort plugin activation.
**Re-registration**: If webhookID exists in KVStore, skip registration (avoid duplicates).
**Credential change**: If credentials change via `OnConfigurationChange`, re-register webhook with new credentials.

---

## Router Structure

```
mux.Router
├── Static (no auth):
│   ├── GET  /call
│   ├── GET  /call.js
│   └── GET  /worker.js
│
├── Webhook (no Mattermost auth — RTK signature verified in handler):
│   └── POST /api/v1/webhook/rtk    → handleRTKWebhook
│
└── API subrouter (MattermostAuthorizationRequired middleware):
    └── /api/v1/
        ├── POST   /calls                    → handleCreateCall
        ├── POST   /calls/{id}/token         → handleJoinCall
        ├── POST   /calls/{id}/leave         → handleLeaveCall
        ├── DELETE /calls/{id}               → handleEndCall
        ├── POST   /calls/{id}/dismiss       → handleDismiss
        ├── GET    /config/status            → handleConfigStatus
        └── GET    /config/admin-status      → handleAdminConfigStatus
```

Note: `/api/v1/webhook/rtk` is registered on the root router (outside auth subrouter) because Cloudflare does not send `Mattermost-User-ID`.
