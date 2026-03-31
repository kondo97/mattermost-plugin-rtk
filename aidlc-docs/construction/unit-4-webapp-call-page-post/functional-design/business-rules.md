# Unit 4: Webapp - Call Page & Post — Business Rules

## CallPost Rules

### BR-U4-001: Active vs Ended State Detection
The CallPost determines its state from `end_at`:
- `end_at === 0` → active state (green indicator, "Call started", "Join" button)
- `end_at > 0` → ended state (gray indicator, "Call ended", duration, no "Join" button)
- Transition is triggered by `custom_com.kondo97.mattermost-plugin-rtk_call_ended` WS event updating Redux

### BR-U4-002: Join Button Disabled for Current Participant
The "Join call" button MUST be `disabled={true}` when `selectMyActiveCall?.callId === post.props.call_id`.
The button is still rendered (not hidden) so the user understands their current state.
Source: US-010

### BR-U4-003: No Join Button in Ended State
The "Join call" button MUST NOT be rendered when `end_at > 0`.
No other action buttons appear in the ended state.
Source: US-016

### BR-U4-004: Participant Avatars — Max 3 Visible
Show at most 3 participant avatars. If `participants.length > 3`, show overflow count `(+N)`.
Same pattern as Unit 3 (FloatingWidget, ToastBar).
Source: US-007, US-016

### BR-U4-005: Post Props as Initial Data, Redux as Live Data (Q4=C)
The CallPost MUST use `post.props` for initial render and fall back to it if Redux has no data yet.
Redux `selectCallByChannel(channelId)` provides live updates once WS events populate the slice.
This prevents empty renders while waiting for the first WS event.

### BR-U4-006: SwitchCallModal Required Before Switching Calls from Post
If the user is in call A and clicks "Join" on a different call B's post, a `SwitchCallModal` MUST be shown before any API call is made.
Source: US-023

---

## Call Page Rules

### BR-U4-007: URL Parameters Are Required
The call page MUST parse `token` and `call_id` from `window.location.search`.
If `token` is absent, show an error screen ("Missing call token") and do not initialize the RTK SDK.
If `call_id` is absent, the call page MAY still initialize RTK but leave-on-close will not work — log a warning.

### BR-U4-008: Browser Tab Title Format
`document.title` MUST be set to `'Call in #' + channelName` where `channelName` comes from the `channel_name` URL param.
If `channel_name` is absent, use the static title `'RTK Call'`.
Source: US-006

### BR-U4-009: Call Page Requires No Mattermost Session Cookie
The call page MUST function without a Mattermost session cookie.
RTK SDK authentication uses the `token` URL parameter only.
Leave endpoint authenticates via `call_id` in KVStore (server-side, no cookie required).
Source: US-006

### BR-U4-010: ~~Heartbeat Interval — 15 Seconds~~ — Deferred / Not Implemented
> **Updated 2026-03-30**: Heartbeat mechanism is deferred. RTK webhook (`meeting.participantLeft`) handles participant cleanup instead. No heartbeat endpoint or client-side heartbeat loop exists in the current implementation.

### BR-U4-011: fetch+keepalive on Tab Close
> **Updated 2026-03-30**: Changed from `navigator.sendBeacon` to `fetch` with `keepalive: true` and custom `X-Requested-With` header. This allows setting custom headers (which `sendBeacon` cannot do) while still surviving tab close.

`fetch('/plugins/{id}/api/v1/calls/{call_id}/leave', {method: 'POST', keepalive: true, headers: {'X-Requested-With': 'XMLHttpRequest'}})` MUST be called in the `beforeunload` event handler.
Source: US-013

### BR-U4-012: RTK UI Kit — No Custom UI Construction
The call page MUST use `<RtkMeeting mode="fill" />` from `@cloudflare/realtimekit-react-ui`.
Custom in-call UI (video grid, controls) MUST NOT be built in this unit.
Source: Q2=A decision

### BR-U4-013: Feature Flags Not Passed to Call Page
Feature flags MUST NOT be passed as URL parameters to the call page.
The RTK UI Kit reads preset configuration from the Cloudflare server side automatically.
Source: Q3=C decision

---

## Build System Rules

### BR-U4-014: `main.js` Must Externalize Mattermost Globals
The `main` Vite entry MUST declare the following as externals (provided as globals by Mattermost host):
- `react` → `window.React`
- `react-dom` → `window.ReactDOM`
- `redux` → `window.Redux`
- `react-redux` → `window.ReactRedux`
- `prop-types` → `window.PropTypes`
- `react-bootstrap` → `window.ReactBootstrap`
- `react-router-dom` → `window.ReactRouterDom`

### BR-U4-015: `call.js` Must Bundle React Independently
The `call` Vite entry MUST NOT externalize React or any other library.
It runs in a standalone page with no Mattermost framework.
Source: Q8=A decision

### BR-U4-016: Makefile Copy Step Sequencing
`server/assets/call.js` MUST be up to date before `go build`.
The Makefile MUST copy `webapp/dist/call.js` → `server/assets/call.js` after the webapp build and before the Go build.
Source: Q1=B decision

### BR-U4-017: webpack Replaced by Vite
After Unit 4 implementation, `webpack.config.js` is removed and `vite.config.ts` replaces it.
`package.json` scripts are updated: `webpack` → `vite build`.
Jest continues to use Babel (not Vite) — no change to test infrastructure.

---

## URL Parameter Propagation Rules

### BR-U4-018: All Call Tab Opens Must Include `call_id` and `channel_name`
Every `window.open(...)` call for the call tab MUST include `call_id` and `channel_name` parameters.
This applies to all Unit 3 components: `ChannelHeaderButton`, `ToastBar`, `FloatingWidget`, `IncomingCallNotification`.

### BR-U4-019: `channel_name` Resolution
`channel_name` MUST be resolved from `store.getState().entities.channels.channels[channelId].display_name` at the time of tab opening.
If the value is unavailable, use an empty string (the call page will fall back to "RTK Call" title).

### BR-U4-020: Token Not Logged (Carry-over SEC-U3-01)
The `token` parameter MUST NOT be logged in console output on the call page or in any component.
`call_id` and `channel_name` MAY be logged for debugging.
