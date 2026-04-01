# Unit 6: Mobile Support — Code Summary

> **Status**: COMPLETE (Reinstated 2026-04-01). Mobile push notifications are implemented
> directly in `server/push.go` using Mattermost's standard `SendPushNotification` API,
> following the same pattern as the official Mattermost Calls plugin.

## Implementation Overview

Push notifications are sent to mobile clients when a call starts in a DM or GM channel.
The implementation intercepts Mattermost's default notification for call posts
(`NotificationWillBePushed` hook) and replaces it with a notification that carries
`SubType: calls` — which triggers the mobile call ringing UI (iOS/Android).

## Files

| File | Description |
|---|---|
| `server/push.go` | All push notification logic: hook, sender, helpers |
| `server/push_test.go` | Unit tests (14 tests) |

## Key Changes to Existing Files

| File | Change |
|---|---|
| `server/calls.go` | `CreateCall()` calls `sendPushNotifications()` after post creation (best-effort) |
| `server/calls_test.go` | Added `GetConfig` mock (push disabled) to `TestCreateCall_Success` |
| `server/api_calls_test.go` | Added `GetConfig` mock (push disabled) to `TestHandleCreateCall_Success` |

## Functions in `server/push.go`

| Function | Role |
|---|---|
| `NotificationWillBePushed(notification, userID)` | Plugin hook — suppresses Mattermost's default notification for DM/GM call posts; plugin sends its own |
| `sendPushNotifications(channelID, postID, sender)` | Sends push notification to all DM/GM channel members except the caller |
| `checkIDLoadedLicense()` | Reports whether the server license supports ID-loaded push notifications |
| `getNameFormat(userID)` | Returns the display name format for a user (preference → server default → username) |
| `getChannelNameForNotification(...)` | Builds the channel name field for push notifications |
| `buildPushMessage(senderName)` | Full message with `\u200b` prefix (triggers mobile call ringing UI) |
| `buildGenericPushMessage()` | Generic "Incoming call" message for privacy-sensitive configurations |

## Push Notification Payload

| Field | Value |
|---|---|
| `Version` | `model.PushMessageV2` (`"v2"`) |
| `Type` | `model.PushTypeMessage` (`"message"`) |
| `SubType` | `model.PushSubTypeCalls` (`"calls"`) — triggers call ringing UI |
| `TeamId` | `channel.TeamId` (empty string for DM/GM) |
| `ChannelId` | Call's channel ID |
| `PostId` | Call post ID |
| `SenderId` | Caller's user ID |
| `ChannelType` | `D` or `G` |
| `Message` | `\u200b<SenderName> is calling you` (full) or `"Incoming call"` (generic) |
| `SenderName` | Caller's display name (format depends on server/user preference) |
| `ChannelName` | Sender name (DM) or comma-joined member names (GM) |
| `IsIdLoaded` | `true` when IdLoaded notification is configured and licensed |

## PushNotificationContents Handling

| Server Setting | Behaviour |
|---|---|
| `full` (default) | `Message = "\u200b<SenderName> is calling you"`, `SenderName` and `ChannelName` set |
| `generic_no_channel` | Same as full but `ChannelName = ""` for GM channels |
| `generic` | `Message = "Incoming call"`, `SenderName` and `ChannelName` set |
| `id_loaded` (+ license) | `IsIdLoaded = true`; server fetches details from proxy |

