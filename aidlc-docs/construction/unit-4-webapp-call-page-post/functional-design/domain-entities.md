# Unit 4: Webapp - Call Page & Post — Domain Entities

## Summary

Unit 4 introduces two new rendering surfaces:
1. **CallPost** — a `custom_cf_call` post renderer embedded inside Mattermost
2. **CallPage** — a standalone React bundle (`call.js`) served from `server/assets/call.js`, rendering the Cloudflare RTK UI Kit in a new browser tab

---

## Post Props — `CallPostProps`

The Mattermost server sets these fields in `post.props` when creating/updating a `custom_cf_call` post.
The CallPost component reads them as the initial/fallback data source.

```typescript
interface CallPostProps {
    call_id: string;           // Mattermost-internal call UUID
    channel_id: string;        // Channel where the call is happening
    creator_id: string;        // User who started the call
    start_at: number;          // Unix ms timestamp — when call started
    end_at: number;            // Unix ms timestamp — 0 means call is active
    participants: string[];    // Current participant user IDs
}
```

**Active state**: `end_at === 0`
**Ended state**: `end_at > 0`

---

## Call Page URL Parameters — `CallPageParams`

All parameters are passed as URL query parameters when opening the call tab.
The standalone call page reads them via `window.location.search` (no Mattermost framework available).

```typescript
interface CallPageParams {
    token: string;          // Cloudflare RTK JWT — used to authenticate with RTK SDK
    call_id: string;        // Mattermost call UUID — used for heartbeat + leave API calls
    channel_name: string;   // Channel display name — used for browser tab title
}
```

**URL format**: `/plugins/{id}/call?token=<jwt>&call_id=<uuid>&channel_name=<name>`

---

## Call Page State

Internal state managed by the standalone call page React component.

```typescript
interface CallPageState {
    token: string;
    callId: string;
    channelName: string;
    heartbeatIntervalId: ReturnType<typeof setInterval> | null;
}
```

---

## Redux State Dependencies (from Unit 3)

CallPost reads the following from the Unit 3 Redux slice:

| Selector | Used For |
|----------|----------|
| `selectCallByChannel(channelId)` | Live participant updates, real-time state |
| `selectMyActiveCall` | Determine if "Join" button should be disabled |

Post `props` serves as the initial/fallback data (Q4=C). If Redux has no data yet (before first WS event), `props` is used. Redux state takes over once WS events arrive.

---

## Build Artifacts

| Artifact | Entry | Target | Externals |
|----------|-------|--------|-----------|
| `main.js` | `webapp/src/index.tsx` | `webapp/dist/main.js` | React, ReactDOM, Redux, ReactRedux, PropTypes, ReactBootstrap, ReactRouterDom |
| `call.js` | `webapp/src/call_page/main.tsx` | `webapp/dist/call.js` | None — bundles React independently |

Makefile copies `webapp/dist/call.js` → `server/assets/call.js` before the Go build (Q1=B).
