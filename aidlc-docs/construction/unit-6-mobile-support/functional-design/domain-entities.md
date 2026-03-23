# Unit 6: Mobile Support — Domain Entities

## Overview

Unit 6 introduces a push notification subsystem as a standalone package (`server/push`).
The subsystem is integrated into the Plugin Core via dependency injection.

---

## Entity 1: PushSender (Interface)

**Package**: `server/push`
**File**: `server/push/interface.go`

The mockable interface used by Plugin Core to dispatch push notifications.

```
PushSender
  + SendIncomingCall(session *kvstore.CallSession) error
  + SendCallEnded(session *kvstore.CallSession) error
```

| Method | Trigger | Returns |
|---|---|---|
| `SendIncomingCall` | Called by `CreateCall` after KVStore save | `error` — non-nil only if metadata fetch fails; per-recipient send failures are logged and skipped (best-effort) |
| `SendCallEnded` | Called by `endCallInternal` after KVStore end | `error` — non-nil only if metadata fetch fails; per-recipient send failures are logged and skipped (best-effort) |

---

## Entity 2: Sender (Struct)

**Package**: `server/push`
**File**: `server/push/push.go`

Concrete implementation of `PushSender`. Receives `plugin.API` via constructor.

```
Sender
  - api  plugin.API

  + SendIncomingCall(session *kvstore.CallSession) error
  + SendCallEnded(session *kvstore.CallSession) error
```

**Constructor**: `NewSender(api plugin.API) *Sender`

**Dependencies** (via `plugin.API`):
- `GetChannel(channelID)` — fetch `TeamId` and `DisplayName`
- `GetUser(userID)` — fetch caller `Username`
- `GetChannelMembers(channelID, 0, 8)` — fetch up to 8 recipients (single call, no pagination)
- `SendPushNotification(notification, userID)` — dispatch to Mattermost push proxy

---

## Entity 3: IncomingCallNotification (Conceptual)

Represents the push payload sent when a call starts.

| Field | Value | Source |
|---|---|---|
| `Type` | `"message"` | Constant |
| `SubType` | `"calls"` | Constant |
| `ChannelId` | `session.ChannelID` | CallSession |
| `TeamId` | `channel.TeamId` (empty string for DM/GM) | Channel |
| `SenderId` | `session.CreatorID` | CallSession |
| `SenderName` | `caller.Username` | User |
| `ChannelName` | `channel.DisplayName` (fallback: `channel.Name`) | Channel |
| `RootId` | `session.PostID` | CallSession |

**Platform routing**: handled transparently by Mattermost push proxy
(APNs PushKit for iOS VoIP tokens, FCM for Android).

---

## Entity 4: CallEndedNotification (Conceptual)

Represents the push payload sent when a call ends (dismiss incoming call UI).

| Field | Value | Source |
|---|---|---|
| `Type` | `"clear"` | Constant |
| `SubType` | `"calls_ended"` | Constant |
| `ChannelId` | `session.ChannelID` | CallSession |
| `TeamId` | `channel.TeamId` | Channel |
| `SenderId` | `session.CreatorID` | CallSession |
| `SenderName` | `caller.Username` | User |
| `ChannelName` | `channel.DisplayName` (fallback: `channel.Name`) | Channel |
| `RootId` | `session.PostID` | CallSession |

**Recipient scope**: same as `IncomingCallNotification` — all channel members except the caller.

---

## Entity 5: Plugin (Modified)

**File**: `server/plugin.go`

New field added to the Plugin struct:

```
Plugin
  ...existing fields...
  + pushSender push.PushSender   // nil if push is not initialized
```

`pushSender` is initialized on `OnActivate`. It is always non-nil after activation
(uses `plugin.API` which is always available).
