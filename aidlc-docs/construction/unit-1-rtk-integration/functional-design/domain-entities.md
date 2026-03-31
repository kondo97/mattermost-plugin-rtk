# Unit 1: RTK Integration — Domain Entities

## CallSession

The central entity representing an active or ended call.

| Field | Type | Description |
|---|---|---|
| `ID` | string (UUID) | Unique call identifier |
| `ChannelID` | string | Mattermost channel the call belongs to |
| `CreatorID` | string | UserID of the call host (used for EndCall authorization) |
| `MeetingID` | string | Cloudflare RTK meeting identifier |
| `Participants` | []string | Current participant UserIDs |
| `StartAt` | int64 | Unix timestamp (ms) when call was created |
| `EndAt` | int64 | Unix timestamp (ms) when call ended; 0 = active |
| `PostID` | string | ID of the `custom_cf_call` post in the channel |

**Active call**: `EndAt == 0`
**Ended call**: `EndAt > 0`

---

## RTKMeeting (external — Cloudflare)

Returned by `RTKClient.CreateMeeting()`. Not persisted directly; `MeetingID` is stored in `CallSession`.

| Field | Type | Description |
|---|---|---|
| `ID` | string | Cloudflare meeting identifier |

---

## RTKToken (external — Cloudflare)

Returned by `RTKClient.GenerateToken()`. Passed to the client; not stored in KVStore.

| Field | Type | Description |
|---|---|---|
| `Token` | string | JWT token for RTK SDK initialization |

---

## KVStore Key Schema

| Key Pattern | Value | TTL |
|---|---|---|
| `call:channel:{channelID}` | CallSession (JSON) | None (manually managed) |
| `call:id:{callID}` | CallSession (JSON) | None (manually managed) |
| ~~`heartbeat:{callID}:{userID}`~~ | ~~int64 (Unix ms timestamp)~~ | ~~None~~ | **Deferred / not implemented** — heartbeat replaced by RTK webhook |
| `voip:{userID}` | string (device token) | None |

Both `call:channel:` and `call:id:` point to the same CallSession; kept in sync on all writes.

---

## WebSocket Event Payloads

> **Naming convention**: The server publishes events using short names (e.g., `call_started`).
> The Mattermost plugin framework automatically prepends `custom_{pluginID}_` when delivering
> events to clients. With plugin ID `com.kondo97.mattermost-plugin-rtk`, the full client-side
> event name is `custom_com.kondo97.mattermost-plugin-rtk_call_started`, etc.
> The webapp subscribes using `custom_${manifest.id}_call_started` (see `webapp/src/index.tsx`).

### custom_com.kondo97.mattermost-plugin-rtk_call_started
Server publishes as: `call_started`
```json
{
  "call_id": "string",
  "channel_id": "string",
  "creator_id": "string",
  "participants": ["string"],
  "start_at": 1234567890000,
  "post_id": "string"
}
```

### custom_com.kondo97.mattermost-plugin-rtk_user_joined
Server publishes as: `user_joined`
```json
{
  "call_id": "string",
  "channel_id": "string",
  "user_id": "string",
  "participants": ["string"]
}
```

### custom_com.kondo97.mattermost-plugin-rtk_user_left
Server publishes as: `user_left`
```json
{
  "call_id": "string",
  "channel_id": "string",
  "user_id": "string",
  "participants": ["string"]
}
```

### custom_com.kondo97.mattermost-plugin-rtk_call_ended
Server publishes as: `call_ended`
```json
{
  "call_id": "string",
  "channel_id": "string",
  "end_at": 1234567890000,
  "duration_ms": 720000
}
```

### custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed
Server publishes as: `notification_dismissed`
```json
{
  "call_id": "string",
  "user_id": "string"
}
```
