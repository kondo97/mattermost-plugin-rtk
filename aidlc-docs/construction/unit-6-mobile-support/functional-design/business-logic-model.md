# Unit 6: Mobile Support — Business Logic Model

> **Updated 2026-04-01**: Push notifications reinstated. The plugin now implements
> `server/push.go` directly (no separate package), using Mattermost's
> `SendPushNotification` API and `NotificationWillBePushed` hook.

## Overview

Unit 6 delivers mobile push notifications for incoming calls. When a call starts in
a DM or GM channel, all channel members except the caller receive a push notification
with `SubType: calls`, which triggers the native call ringing UI on iOS and Android.

The implementation is a single file (`server/push.go`) on the Plugin struct — no
separate package or interface is required, since there is no need to mock the
push sender in tests (the plugin API is already mockable via `plugintest.API`).

---

## Business Operations

### Operation 1: sendPushNotifications

**Trigger**: Called by `CreateCall` (in `calls.go`) after the call post is created.

**Preconditions**:
- Server has push notifications enabled (`SendPushNotifications = true`)
- Push notification server URL is configured
- Channel type is DM (`D`) or GM (`G`)

**Steps**:
1. Fetch `model.Config` — exit if push is disabled or server URL is empty
2. Fetch channel — exit if not DM/GM
3. Fetch up to 8 channel members sorted by username
4. For each member except the caller:
   a. Build `model.PushNotification` with `SubType: calls`
   b. Populate `SenderName`, `ChannelName`, and `Message` based on `PushNotificationContents` setting
   c. Call `p.API.SendPushNotification(msg, member.Id)` — log error on failure, continue
5. Return (best-effort; no error propagation)

**Postcondition**: Push notification queued by Mattermost push proxy for each eligible member.

---

### Operation 2: NotificationWillBePushed (Hook)

**Trigger**: Called by Mattermost server before sending any push notification.

**Logic**:
- If `notification.PostType != callPostType` → return `(nil, "")` (pass through)
- If `ChannelType` is DM or GM → return `(nil, "rtk plugin will handle this notification")`
  (suppress; plugin sends its own notification with correct SubType)
- Otherwise → return `(nil, "")` (pass through)

---

## Integration Points

| Location | Change |
|---|---|
| `server/calls.go` — `CreateCall` | Calls `sendPushNotifications(channelID, createdPost.Id, senderUser)` after post creation |
| `server/plugin.go` | No change — `NotificationWillBePushed` is auto-registered as a plugin hook |

---

## Mobile Notification Flow

```
CreateCall() in DM/GM channel
    └─► CreatePost (custom_cf_call)
           ├─► Mattermost server queues default push notification
           │      └─► NotificationWillBePushed hook fires
           │             └─► DM/GM? → suppress (return nil, reason)
           └─► sendPushNotifications() called
                  └─► SendPushNotification(msg{SubType:calls}, memberID)
                         └─► Mattermost push proxy → iOS/Android
```

---

## WebSocket Events (unchanged)

Mobile clients also receive WebSocket events for real-time UI updates:
- `custom_com.kondo97.mattermost-plugin-rtk_call_started` — triggers incoming call UI (web)
- `custom_com.kondo97.mattermost-plugin-rtk_call_ended` — dismisses incoming call UI
- `custom_com.kondo97.mattermost-plugin-rtk_user_joined` / `_user_left` — participant updates

Push notifications complement (not replace) WebSocket events for offline/background delivery.

## Removal Rationale

Push notifications were removed because the mobile app handles call notifications
through the same WebSocket events used by the desktop client. The dedicated push
notification subsystem (`server/push/`) was no longer needed.

## Current Mobile Support

Mobile clients receive call events through existing WebSocket channels:
- `custom_com.kondo97.mattermost-plugin-rtk_call_started` — triggers incoming call UI
- `custom_com.kondo97.mattermost-plugin-rtk_call_ended` — dismisses incoming call UI
- `custom_com.kondo97.mattermost-plugin-rtk_user_joined` / `custom_com.kondo97.mattermost-plugin-rtk_user_left` — participant updates

Joining a call from mobile uses the existing `POST /api/v1/calls/{callId}/token`
endpoint (implemented in Unit 2). No dedicated mobile-specific server code is required.

---

## Original Design (Archived)

The following describes the push notification system as originally designed and implemented,
before its removal. This section is kept for historical context only.

### Overview

Two business operations defined Unit 6:
1. **SendIncomingCall** — dispatch call-started push to DM/GM channel members (max 8, except caller)
2. **SendCallEnded** — dispatch call-ended clear push to same scope

Both were invoked from Plugin Core (`calls.go`) after the corresponding call state change.
Both were **best-effort** — errors were logged but did not block the call operation.

### Integration Points (Removed)

- `CreateCall`: called `SendIncomingCall` after `PublishWebSocketEvent`
- `endCallInternal`: called `SendCallEnded` after `PublishWebSocketEvent`
- `OnActivate`: initialized `pushSender = push.NewSender(p.API)`
