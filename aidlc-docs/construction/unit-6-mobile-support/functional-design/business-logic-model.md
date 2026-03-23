# Unit 6: Mobile Support — Business Logic Model

## Overview

Two business operations define Unit 6:
1. **SendIncomingCall** — dispatch call-started push to DM/GM channel members (max 8, except caller)
2. **SendCallEnded** — dispatch call-ended clear push to same scope

Both are invoked from Plugin Core (`calls.go`) after the corresponding call state change.
Both are **best-effort** — errors are logged but do not block the call operation.

**Alignment**: This design follows the Mattermost Calls plugin pattern
(`sendPushNotifications` in `push_notifications.go`).

---

## Operation 1: SendIncomingCall

**Trigger**: `CreateCall` (after `PublishWebSocketEvent`)
**Failure behavior**: Error logged as warning; `CreateCall` continues and succeeds (best-effort)

### Flow

```
SendIncomingCall(session)
  |
  +-- 1. GetChannel(session.ChannelID)
  |       -> channel
  |       -> error: log warn, return error (caller ignores)
  |
  +-- 2. Check channel type:
  |       if channel.Type != model.ChannelTypeDirect &&
  |          channel.Type != model.ChannelTypeGroup:
  |         return nil   // push only for DM/GM channels
  |
  +-- 3. GetUser(session.CreatorID)
  |       -> caller (for Username)
  |       -> error: log warn, return error
  |
  +-- 4. channelName = channel.DisplayName
  |       if channelName == "": channelName = channel.Name
  |
  +-- 5. members = GetChannelMembers(session.ChannelID, 0, 8)
  |       -> error: log warn, return error
  |
  +-- 6. for each member (sequential):
  |         if member.UserId == session.CreatorID: skip
  |         notification = build IncomingCallNotification(session, channel, caller, channelName)
  |         err = SendPushNotification(notification, member.UserId)
  |         if err != nil: log warn, continue   // per-member failure does not abort loop
  |
  +-- 7. return nil
```

### Notification Construction

```
notification.Type        = "message"
notification.SubType     = "calls"
notification.ChannelId   = session.ChannelID
notification.TeamId      = channel.TeamId        // empty string for DM/GM
notification.SenderId    = session.CreatorID
notification.SenderName  = caller.Username
notification.ChannelName = channelName
notification.RootId      = session.PostID
```

---

## Operation 2: SendCallEnded

**Trigger**: `endCallInternal` (after `PublishWebSocketEvent`)
**Failure behavior**: Error logged as warning; `endCallInternal` continues (best-effort)

### Flow

```
SendCallEnded(session)
  |
  +-- 1. GetChannel(session.ChannelID)
  |       -> channel
  |       -> error: log warn, return error
  |
  +-- 2. Check channel type:
  |       if channel.Type != model.ChannelTypeDirect &&
  |          channel.Type != model.ChannelTypeGroup:
  |         return nil   // push only for DM/GM channels
  |
  +-- 3. GetUser(session.CreatorID)
  |       -> caller (for Username)
  |       -> error: log warn, return error
  |
  +-- 4. channelName = channel.DisplayName
  |       if channelName == "": channelName = channel.Name
  |
  +-- 5. members = GetChannelMembers(session.ChannelID, 0, 8)
  |       -> error: log warn, return error
  |
  +-- 6. for each member (sequential):
  |         if member.UserId == session.CreatorID: skip
  |         notification = build CallEndedNotification(session, channel, caller, channelName)
  |         err = SendPushNotification(notification, member.UserId)
  |         if err != nil: log warn, continue
  |
  +-- 7. return nil
```

### Notification Construction

```
notification.Type        = "clear"
notification.SubType     = "calls_ended"
notification.ChannelId   = session.ChannelID
notification.TeamId      = channel.TeamId
notification.SenderId    = session.CreatorID
notification.SenderName  = caller.Username
notification.ChannelName = channelName
notification.RootId      = session.PostID
```

---

## Integration: CreateCall (modified)

After `PublishWebSocketEvent` (existing), add:

```
// Push notification — best-effort (DM/GM only, max 8 members)
if err := p.pushSender.SendIncomingCall(session); err != nil {
    p.API.LogWarn("CreateCall: SendIncomingCall failed (best effort)", ...)
}
```

---

## Integration: endCallInternal (modified)

After `PublishWebSocketEvent` (existing), add:

```
// Push notification — best-effort
if err := p.pushSender.SendCallEnded(session); err != nil {
    p.API.LogWarn("endCallInternal: SendCallEnded failed (best effort)", ...)
}
```

---

## OnActivate Initialization

```
p.pushSender = push.NewSender(p.API)
```

`pushSender` is always initialized regardless of RTK credential configuration,
because push notifications use only the Mattermost API (not the RTK API).

---

## US-019: Join a Call from Push Notification

No new server-side logic required. The mobile user taps "Join" in the notification
and calls `POST /api/v1/calls/{callId}/token` (already implemented in Unit 2).
The response includes the auth token and all feature flag values.
US-019 is fully satisfied by the existing `handleToken` endpoint.
