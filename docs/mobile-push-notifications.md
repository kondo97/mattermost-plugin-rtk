# Mobile Push Notifications

This document describes how the RTK plugin sends push notifications to iOS and Android devices, covering prerequisites, notification timing, payload fields, and the default-notification suppression mechanism.

---

## Prerequisites

Push notifications are sent only when **all** of the following conditions are met:

1. **Push notifications enabled** — `EmailSettings.SendPushNotifications = true` in the Mattermost server config.
2. **Push notification server configured** — `EmailSettings.PushNotificationServer` is set to a non-empty URL (e.g. `https://push.mattermost.com`).
3. **Channel type is DM or GM** — notifications are scoped to Direct Message and Group Message channels only. Public/private channels are not affected.

---

## Notification Types

The plugin sends two distinct push notifications during a call lifecycle.

### 1. Call Started (`SubType = calls`)

**When**: Immediately after the call-start post (`custom_cf_call`) is created in a DM/GM channel.

**Who receives it**: All channel members **except the call creator**.

**Purpose**: Trigger the native incoming-call ringing UI on the mobile device.

**Payload**:

| Field | Value | Notes |
|---|---|---|
| `version` | `"v2"` | Mattermost Push Message V2 |
| `type` | `"message"` | Standard message notification |
| `sub_type` | `"calls"` | Triggers native call ringing UI on iOS/Android |
| `team_id` | Team ID of the channel | |
| `channel_id` | Channel ID | |
| `post_id` | Call start post ID | |
| `root_id` | Call start post ID | Same as `post_id` (post is its own thread root) |
| `sender_id` | Call creator's user ID | |
| `channel_type` | `"D"` or `"G"` | Direct or Group |
| `message` | See table below | Varies by `PushNotificationContents` setting |
| `sender_name` | Caller's display name | Set unless `id_loaded` is active |
| `channel_name` | See table below | Set unless `id_loaded` is active |
| `is_id_loaded` | `true` | Only when `id_loaded` mode is active (requires license) |

#### `message` and `channel_name` by `PushNotificationContents` setting

| Setting | `message` | `channel_name` |
|---|---|---|
| `full` | `"\u200b<SenderName> is calling you"` | Sender name (DM) or other members' names (GM) |
| `generic` | `"\u200bIncoming call"` | Sender name (DM) or other members' names (GM) |
| `generic_no_channel` | `"\u200bIncoming call"` | Sender name (DM) or `""` (GM) |
| `id_loaded` | `"\u200bIncoming call"` | Not set — mobile fetches details via API |

> **Note on the zero-width space (`\u200b`)**: The leading zero-width space is a signal to the Mattermost mobile app to activate the call ringing UI (ringtone + incoming call screen). It is always prepended to the message regardless of the `PushNotificationContents` setting.

---

### 2. Call Ended (`SubType = calls_ended`)

**When**: When the call ends (last participant leaves, creator explicitly ends, or cleanup loop triggers).

**Who receives it**: All channel members **except the call creator**.

**Purpose**: Allow the mobile app to dismiss the ringing or incoming-call UI when the call has already ended before the user responded.

**Payload**:

| Field | Value | Notes |
|---|---|---|
| `version` | `"v2"` | Mattermost Push Message V2 |
| `type` | `"clear"` | Clear/dismiss notification (not a new alert) |
| `sub_type` | `"calls_ended"` | Signals the mobile app to dismiss the call UI |
| `team_id` | Team ID of the channel | |
| `channel_id` | Channel ID | |
| `post_id` | Call start post ID | |
| `channel_name` | Channel display name | |

> Unlike the call-started notification, the call-ended notification uses `type = "clear"` and does not include `message`, `sender_id`, `sender_name`, or `root_id`.

---

## Default Notification Suppression

Mattermost's server also generates a default push notification when a `custom_cf_call` post is created. Without intervention, users would receive two notifications for the same event.

The plugin implements the `NotificationWillBePushed` hook to suppress the default notification when the plugin is capable of sending its own:

```
Mattermost server generates default notification for custom_cf_call post
    └─► NotificationWillBePushed() hook fires
            ├─ Post type is not custom_cf_call? → pass through (return nil, "")
            ├─ Channel is not DM/GM?            → pass through (return nil, "")
            ├─ Push disabled or no server?      → pass through (return original notification, "")
            └─ All checks pass (DM/GM, push enabled)
                → suppress default notification (return nil, "rtk plugin will handle this notification")
                → plugin sends its own calls-specific notification via sendPushNotifications()
```

If push notifications are not configured on the server, the default Mattermost notification is passed through unchanged as a fallback.

---

## Call Lifecycle and Notification Flow

```
User A starts a call (DM/GM)
    │
    ├─► CreateCall()
    │       ├─► SQL DB: save CallSession (rtk_call_sessions)
    │       ├─► CreatePost (custom_cf_call)
    │       ├─► sendPushNotifications()         ← SubType=calls  →  User B's device rings
    │       └─► WebSocket: call_started         ← IncomingCallNotification UI on desktop/web
    │
    │   ... call in progress ...
    │
    └─► endCallInternal() (via LeaveCall / EndCall / cleanup)
            ├─► SQL DB: mark EndAt (rtk_call_sessions)
            ├─► EndMeeting (RTK API)
            ├─► UpdatePost (set end_at, duration_ms)
            ├─► sendEndCallPushNotifications()  ← SubType=calls_ended  →  User B's device dismisses
            └─► WebSocket: call_ended           ← clears IncomingCallNotification on desktop/web
```

---

## Implementation Reference

| Component | File | Description |
|---|---|---|
| `NotificationWillBePushed` | `server/push.go` | Plugin hook — suppresses default notification for DM/GM call posts |
| `sendPushNotifications` | `server/push.go` | Sends `SubType=calls` notification on call start |
| `sendEndCallPushNotifications` | `server/push.go` | Sends `SubType=calls_ended` notification on call end |
| `CreateCall` | `server/calls.go` | Calls `sendPushNotifications` after post creation |
| `endCallInternal` | `server/calls.go` | Calls `sendEndCallPushNotifications` after post update |
