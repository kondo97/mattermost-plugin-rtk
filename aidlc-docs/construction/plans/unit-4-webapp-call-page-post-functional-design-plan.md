# Functional Design Plan — Unit 4: Webapp - Call Page & Post

## Unit Summary

**Unit**: Unit 4: Webapp - Call Page & Post
**Primary Stories**: US-007, US-010, US-013, US-016
**Supporting Stories**: US-006, US-009, US-012, US-015, US-025

**Responsibilities**:
- Implement `custom_cf_call` post renderer (active and ended states)
- Implement standalone call page (separate bundle entry — `call.js`)
- Initialize Cloudflare RTK React SDK (`RealtimeKitProvider`)
- Implement heartbeat loop (every 15s) and `sendBeacon` on tab close
- Migrate build system from webpack to Vite (dual bundle: `main.js` + `call.js`)

---

## Execution Plan

- [x] Step 1: Analyze unit context (unit definition + story map reviewed)
- [x] Step 2: Generate clarifying questions
- [x] Step 3: Collect and analyze answers
- [x] Step 4: Generate domain-entities.md
- [x] Step 5: Generate business-logic-model.md
- [x] Step 6: Generate business-rules.md
- [x] Step 7: Generate frontend-components.md

---

## Clarifying Questions

### Q1: Build System — Vite Output Target for `call.js`

The server already has `go:embed` directives that embed `server/assets/call.js` into the Go binary. The Vite dual-bundle build needs to output `call.js` to a location the `go:embed` can find.

Which approach should be used?

A) Vite outputs `call.js` directly to `server/assets/call.js` (configured via `build.rollupOptions.output` in `vite.config.ts`)
B) Vite outputs `call.js` to `webapp/dist/call.js`, and a Makefile target copies it to `server/assets/call.js` before the server build
C) Other (describe after [Answer]:)

[Answer]:

---

### Q2: RTK SDK — UI Kit vs Core SDK

The call page needs to render the full Cloudflare RTK meeting UI. Two options:

A) **UI Kit** (`@cloudflare/realtimekit-react-ui`) — Pre-built React component `<RtkMeeting mode="fill" />` handles all state transitions automatically; fastest to implement; provides complete meeting UI out of the box
B) **Core SDK** (`@cloudflare/realtimekit-react`) — Headless SDK; full UI must be built from scratch; maximum customization but significantly more development effort
C) Other (describe after [Answer]:)

[Answer]:

---

### Q3: Feature Flags — How Call Page Receives Them

The Mattermost main bundle receives feature flags in the `POST /calls` and `POST /calls/{id}/token` API responses. The call page (new tab) needs feature flags to configure the RTK SDK (e.g., to disable recording, screen share, etc.).

The call page only has the `?token=` URL parameter by default. Options:

A) Feature flags are passed as additional URL query parameters when opening the tab (e.g., `?token=...&recording=1&screen_share=0&polls=1...`). The call page reads them from `window.location.search`.
B) Feature flags are written to `sessionStorage` or `localStorage` before opening the tab, and the call page reads them on load. (Same origin required; works because both are served from the same Mattermost server.)
C) Feature flags are not needed in the call page for this phase — the RTK UI Kit applies the preset configuration server-side and does not need client-side feature flags.
D) Other (describe after [Answer]:)

[Answer]:

---

### Q4: CallPost — Participant Data Source

The `custom_cf_call` post renderer (US-007, US-010, US-012, US-016) needs to display participant avatars and call state. Data can come from different sources:

A) **Redux state only** — The CallPost reads live call state from the Redux slice (`callsByChannel`) via selectors. This assumes Unit 3's Redux slice is in place when this post is rendered. Initial render may be empty until WebSocket events populate Redux.
B) **Post props only** — The CallPost reads from the Mattermost post's `props` field (set by the server when creating/updating the custom post). This is always available even before WebSocket state arrives.
C) **Both** — Post `props` provides the initial/fallback data; Redux state provides live updates (re-renders when participants change). This is the most robust approach and aligns with how the Mattermost Calls plugin works.
D) Other (describe after [Answer]:)

[Answer]:

---

### Q5: CallPost — "Join" Button Disabled State for Active Participant

US-010 requires the "Join call" button to be disabled when the user is already in this call. To determine this, the CallPost needs to know if `myActiveCall.callID === post.callID`.

How should this be implemented given Unit 3 (Redux slice) may not be complete?

A) The CallPost imports the `selectMyActiveCall` selector from the Redux slice path (to be created in Unit 3). The UI renders correctly once Unit 3 is integrated. Unit 4 defines the selector interface/contract; Unit 3 implements it.
B) The CallPost maintains its own minimal local check: the post `props` server side includes a `participants` array, and the CallPost checks if the current Mattermost user ID is in that list. No Redux dependency.
C) Other (describe after [Answer]:)

[Answer]:

---

### Q6: Call Page — Channel Name for Browser Tab Title

US-006 requires the browser tab title to show the channel name (e.g., "Call in #general"). The call page is a standalone bundle (no Mattermost framework). The `?token=` is a Cloudflare RTK JWT and does not contain Mattermost-specific metadata.

How should the channel name reach the call page?

A) Pass `channel_name` as a URL query parameter when opening the tab (e.g., `?token=...&channel_name=general`). The call page reads it from `window.location.search`.
B) The channel name is not needed in MVP — the tab title is a static string like "RTK Call". Channel name can be added as a follow-up.
C) Other (describe after [Answer]:)

[Answer]:

---

### Q7: Call Page — `call_id` Availability for Heartbeat and sendBeacon

The heartbeat (`POST /api/v1/calls/{id}/heartbeat`) and sendBeacon on tab close (`POST /api/v1/calls/{id}/leave`) both require the `call_id`. The call page receives `?token=` but `call_id` is the Mattermost-internal call ID (UUID), not the Cloudflare meeting ID.

How should `call_id` be available in the call page?

A) Pass `call_id` as a URL query parameter when opening the tab (e.g., `?token=...&call_id=<uuid>`).
B) The call page uses the Cloudflare token to infer the meeting ID (from the JWT payload) and the heartbeat/leave API accepts the meeting ID in addition to the call ID.
C) Other (describe after [Answer]:)

[Answer]:

---

### Q8: Vite — Mattermost Plugin Externals

The main bundle (`main.js`) must NOT bundle React, Redux, etc. — the Mattermost host provides them as globals (`window.React`, `window.ReactRedux`, etc.). The call bundle (`call.js`) runs in a standalone page (no Mattermost framework) and MUST bundle React.

Is this understanding correct?

A) Yes — `main.js` externalizes React/Redux (same as current webpack config); `call.js` includes React in its bundle. Vite config should mark externals only for the `main` entry, not for the `call` entry.
B) No — both bundles should include React (clarify in answer).
C) Other (describe after [Answer]:)

[Answer]:
