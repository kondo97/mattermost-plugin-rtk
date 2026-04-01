# Unit 6: Mobile Support — Domain Entities

> **Updated 2026-04-01**: Push notifications reinstated. The implementation uses
> plugin-level methods on the `Plugin` struct (no separate package).
> All entities below reflect the current codebase.

## Overview

Unit 6 adds push notification delivery to the Plugin Core. There is no separate
package — all logic lives in `server/push.go` as methods on `*Plugin`.

---

## Entity 1: PushNotification (model.PushNotification)

**Package**: `github.com/mattermost/mattermost/server/public/model`
**File**: used in `server/push.go`

The standard Mattermost push notification payload dispatched via `plugin.API.SendPushNotification`.

| Field | Type | Value for RTK calls |
|---|---|---|
| `Version` | `string` | `model.PushMessageV2` (`"v2"`) |
| `Type` | `string` | `model.PushTypeMessage` (`"message"`) |
| `SubType` | `PushSubType` | `model.PushSubTypeCalls` (`"calls"`) |
| `TeamId` | `string` | `channel.TeamId` (empty for DM/GM) |
| `ChannelId` | `string` | Call's channel ID |
| `PostId` | `string` | Call post ID |
| `SenderId` | `string` | Caller's user ID |
| `ChannelType` | `model.ChannelType` | `D` or `G` |
| `Message` | `string` | `"\u200b<Name> is calling you"` or `"Incoming call"` |
| `SenderName` | `string` | Caller's display name |
| `ChannelName` | `string` | Sender name (DM) or comma-joined member names (GM) |
| `IsIdLoaded` | `bool` | `true` when IdLoaded is configured and licensed |

---

## Entity 2: Plugin (Modified — push methods added)

**File**: `server/push.go`

New methods added to the existing `Plugin` struct:

```
Plugin
  ...existing fields (unchanged)...

  + NotificationWillBePushed(notification *model.PushNotification, userID string)
        (*model.PushNotification, string)
  + sendPushNotifications(channelID, postID string, sender *model.User)
  + checkIDLoadedLicense() bool
  + getNameFormat(userID string) string
```

No new fields are added to `Plugin`. The push sender uses `p.API` directly.

---

## Entity 3: Helper Functions (package-level)

**File**: `server/push.go`

| Function | Signature | Description |
|---|---|---|
| `getChannelNameForNotification` | `(channel, sender, members, nameFormat, receiverID) string` | Builds channel name for notification |
| `buildPushMessage` | `(senderName string) string` | Returns `"\u200b<name> is calling you"` |
| `buildGenericPushMessage` | `() string` | Returns `"Incoming call"` |

---

## Comparison with Previous Design (Archived)

The original Unit 6 design used a separate `server/push/` package with a `PushSender`
interface and `Sender` struct to enable mock injection. That design was removed
(2026-03-31) because the separate package added complexity without benefit — the
`plugintest.API` mock already covers `SendPushNotification` in tests.

The reinstated design (2026-04-01) places all logic directly on `*Plugin`, which is
consistent with the rest of the codebase (`calls.go`, `api_mobile.go`, etc.).

## Overview

~~Unit 6 introduces a push notification subsystem as a standalone package (`server/push`).
The subsystem is integrated into the Plugin Core via dependency injection.~~ **REMOVED.**

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
| `SendIncomingCall` | Called by `CreateCall` after KVStore save | `error` — non-nil on first delivery failure (blocking) |
| `SendCallEnded` | Called by `endCallInternal` after KVStore end | `error` — logged as warning (best-effort in caller) |

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
- `GetChannelMembers(channelID, page, perPage)` — paginate recipients
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
