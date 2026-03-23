# Unit 3: Webapp - Channel UI — Domain Entities

## Redux State Shape

The plugin reducer is mounted at `state['plugins-com.mattermost.plugin-rtk']`.

```typescript
interface CallsPluginState {
    callsByChannel: Record<string, ActiveCall>; // channelId → active call
    myActiveCall: MyActiveCall | null;           // current user's participation
    incomingCall: IncomingCall | null;           // DM/GM ringing notification
    pluginEnabled: boolean;                      // from GET /config/status
}
```

---

## ActiveCall

Represents a call that is currently in progress in a channel. Populated from WebSocket events. Only active calls are stored; ended calls are removed.

| Field | Type | Source |
|---|---|---|
| `id` | `string` | `call_id` from `custom_cf_call_started` |
| `channelId` | `string` | `channel_id` from WS events |
| `creatorId` | `string` | `creator_id` from `custom_cf_call_started` |
| `participants` | `string[]` | `participants` array (full list on each WS event) |
| `startAt` | `number` | `start_at` (Unix ms) from `custom_cf_call_started` |
| `postId` | `string` | `post_id` from `custom_cf_call_started` |

---

## MyActiveCall

Tracks the call the current user is actively participating in. Set from API response handlers and certain WebSocket events. Contains the JWT token obtained at join time for reopening the call tab.

| Field | Type | Source |
|---|---|---|
| `callId` | `string` | From API response or `custom_cf_user_joined` |
| `channelId` | `string` | From API response or `custom_cf_user_joined` |
| `token` | `string` | From API response (`POST /calls` or `POST /calls/{id}/token`) |

**Note**: `token` is only set when the current user triggers the join via the UI. WebSocket events received for the current user (e.g., from another session) update `callId`/`channelId` but do not set `token`.

---

## IncomingCall

Represents a ringing DM/GM call notification shown to non-creator members.

| Field | Type | Source |
|---|---|---|
| `callId` | `string` | `call_id` from `custom_cf_call_started` |
| `channelId` | `string` | `channel_id` from `custom_cf_call_started` |
| `creatorId` | `string` | `creator_id` from `custom_cf_call_started` |
| `startAt` | `number` | `start_at` from `custom_cf_call_started` |

**Lifecycle**: Set on `custom_cf_call_started` for DM/GM; cleared on `custom_cf_call_ended`, `custom_cf_notification_dismissed` (for current user), or after 30-second auto-dismiss timer.

---

## WebSocket Event → Redux Mapping

| WS Event | Redux Update |
|---|---|
| `custom_cf_call_started` | Add entry to `callsByChannel`; if channel type is `D`/`G` and `creator_id != currentUser.id`, set `incomingCall`; if `creator_id == currentUser.id`, set `myActiveCall` (token comes from API response, not WS) |
| `custom_cf_user_joined` | Update `callsByChannel[channelId].participants`; if `user_id == currentUser.id` and `myActiveCall` not yet set, set `myActiveCall` (no token — secondary path for multi-session) |
| `custom_cf_user_left` | Update `callsByChannel[channelId].participants`; if `user_id == currentUser.id`, clear `myActiveCall` |
| `custom_cf_call_ended` | Remove from `callsByChannel`; if `call_id == myActiveCall?.callId`, clear `myActiveCall`; if `call_id == incomingCall?.callId`, clear `incomingCall` |
| `custom_cf_notification_dismissed` | If `user_id == currentUser.id`, clear `incomingCall` |

---

## Selectors

Typed selector hooks used by components:

| Selector | Returns | Purpose |
|---|---|---|
| `selectPluginEnabled` | `boolean` | Whether credentials are configured |
| `selectCallByChannel(channelId)` | `ActiveCall \| undefined` | Active call for a specific channel |
| `selectMyActiveCall` | `MyActiveCall \| null` | Current user's participation |
| `selectIncomingCall` | `IncomingCall \| null` | Ringing DM/GM notification state |
| `selectIsCurrentUserParticipant(channelId)` | `boolean` | Whether current user is in the call for this channel |

---

## Channel Type Constants

Used to determine whether to show `IncomingCallNotification`:

| Constant | Value | Meaning |
|---|---|---|
| `CHANNEL_TYPE_DM` | `'D'` | Direct Message channel |
| `CHANNEL_TYPE_GM` | `'G'` | Group Message channel |

Only channels of type `D` or `G` trigger `IncomingCallNotification`.
