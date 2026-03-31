# Unit 4: Webapp - Call Page & Post — Business Logic Model

## BL-001: CallPost — Active State Rendering

**Trigger**: Mattermost renders a post with `type: 'custom_cf_call'` and `end_at === 0`

**Inputs**: `post.props` (CallPostProps) + Redux `selectCallByChannel(channelId)` + `selectMyActiveCall`

**Logic**:
1. Read initial data from `post.props` (always available)
2. If Redux `callsByChannel[channelId]` exists, use it for live participant data (overrides props)
3. Render active state:
   - Green indicator + "Call started" label
   - Call start time (formatted)
   - Participant avatars (up to 3) + overflow count (`+N`)
   - "Join call" button — enabled unless `selectMyActiveCall?.callId === post.props.call_id`

---

## BL-002: CallPost — Ended State Rendering

**Trigger**: `end_at > 0` in `post.props` OR `custom_com.kondo97.mattermost-plugin-rtk_call_ended` WS event updates Redux

**Logic**:
1. Detect ended state: `end_at > 0` (from props or Redux)
2. Calculate duration: `end_at - start_at` (ms → formatted as "Xh Ym Zs" or "Xm Zs")
3. Render ended state:
   - Gray indicator + "Call ended" label
   - End time (formatted)
   - Duration string ("Lasted X minutes")
   - No "Join call" button

---

## BL-003: CallPost — Join Button Click

**Trigger**: User clicks "Join call" button in active CallPost

**Pre-condition**: `myActiveCall?.callId !== post.props.call_id` (otherwise button is disabled)

**Logic**:
1. If `myActiveCall` exists and `myActiveCall.callId !== post.props.call_id`:
   - Show `SwitchCallModal`
   - On confirm: `POST /calls/{myActiveCall.callId}/leave` (fire-and-forget) → proceed to step 2
   - On cancel: dismiss modal, no action
2. `POST /calls/{post.props.call_id}/token`
3. On success: dispatch `setMyActiveCall`, open call tab with `token`, `call_id`, `channel_name` params
4. On error: show error modal

---

## BL-004: CallPost — Join Button Disabled State

**Trigger**: Component renders when `selectMyActiveCall?.callId === post.props.call_id`

**Logic**:
- Button is rendered but `disabled={true}`
- Tooltip: "You are already in this call"
- No click handler needed

---

## BL-005: Call Page — Initialization

**Trigger**: User navigates to `/plugins/{id}/call?token=...&call_id=...&channel_name=...`

**Logic**:
1. Parse URL params: `token`, `call_id`, `channel_name` from `window.location.search`
2. Validate presence of `token` and `call_id`; if missing, show error screen
3. Set `document.title = 'Call in #' + channelName` (or "RTK Call" if `channel_name` missing)
4. Initialize Cloudflare RTK React Provider with `token`
5. Render `<RtkMeeting mode="fill" />` — UI Kit handles all in-call state automatically
6. Start heartbeat loop (→ BL-006)
7. Register `beforeunload` handler (→ BL-007)

---

## BL-006: Call Page — Heartbeat Loop

**Trigger**: Successful call page initialization (BL-005)

**Logic**:
1. `setInterval(() => { POST /api/v1/calls/{call_id}/heartbeat }, 15_000)`
2. Heartbeat is fire-and-forget (`fetch`, no await)
3. Interval persists for the lifetime of the tab
4. No retry on failure — heartbeat is best-effort

---

## BL-007: Call Page — sendBeacon on Tab Close

**Trigger**: `window.addEventListener('beforeunload', ...)`

**Logic**:
1. `navigator.sendBeacon('/plugins/{id}/api/v1/calls/{call_id}/leave')`
2. `sendBeacon` is used (not `fetch`) because it survives tab close
3. Server handles the leave and emits `custom_com.kondo97.mattermost-plugin-rtk_user_left` WS event

**Note**: `sendBeacon` uses POST with no body and no auth header. The server authenticates via the call session in KVStore using the `call_id`.

---

## BL-008: Build System — Vite Dual Bundle

**Trigger**: `npm run build` in webapp directory

**Logic**:
1. Vite builds two entries simultaneously:
   - `main` entry (`src/index.tsx`) → `dist/main.js` with externals (React/Redux from Mattermost globals)
   - `call` entry (`src/call_page/main.tsx`) → `dist/call.js` with React bundled independently
2. Makefile step (post-webapp-build): `cp webapp/dist/call.js server/assets/call.js`
3. Go build embeds `server/assets/call.js` via `//go:embed assets/call.js`

---

## BL-009: URL Parameter Propagation (Unit 3 Update)

**Context**: Unit 3 code currently opens the call tab as:
```
window.open(`/plugins/${manifest.id}/call?token=${data.token}`, ...)
```
This must be updated to include `call_id` and `channel_name`.

**Update required in**:
- `channel_header_button/index.tsx` — `openCallTab()` helper
- `toast_bar/index.tsx` — `joinCall()` helper
- `floating_widget/index.tsx` — `handleOpenInNewTab()`
- `incoming_call_notification/index.tsx` — `joinCall()` helper

**New URL format**:
```
/plugins/${manifest.id}/call?token=${token}&call_id=${callId}&channel_name=${channelName}
```

**`channel_name` source**: Resolved from Mattermost Redux state via `state.entities.channels.channels[channelId].display_name`.

---

## BL-010: Call Page — Host "End Call" Action (US-015)

**Trigger**: User in call page is `creator_id` — UI Kit renders "End call" option

**Logic**:
- The RTK UI Kit provides host controls automatically when the user has the host role
- Server-side: `DELETE /api/v1/calls/{id}` (Unit 2) handles the end-call event
- The call page does not need custom logic for this — the UI Kit handles it via RTK SDK

---

## WS Event → CallPost Update Flow

```
WS event arrives → Unit 3 WS handler dispatches Redux action
    ↓
Redux state updated (callsByChannel[channelId])
    ↓
CallPost re-renders (useSelector triggers re-render)
    ↓
Live participant list shown / ended state shown
```

This flow is driven entirely by Unit 3 infrastructure. Unit 4's CallPost only needs to subscribe to the correct selectors.
