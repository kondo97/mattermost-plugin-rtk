# Application Design Plan

## Execution Checklist

- [x] Analyze context (requirements, stories, reverse engineering artifacts)
- [x] Identify components and responsibilities
- [x] Ask clarifying questions and collect answers
- [x] Analyze answers for ambiguities
- [x] Generate application design artifacts:
  - [x] `components.md`
  - [x] `component-methods.md`
  - [x] `services.md`
  - [x] `component-dependency.md`
  - [x] `application-design.md` (consolidation)

---

## Context Analysis Summary

### Key Functional Capabilities Identified
1. **Call Lifecycle Management** — Create, join, leave, end calls (FR-02, FR-03, FR-14)
2. **Cloudflare RTK API Integration** — Meeting creation, token generation (NFR-01)
3. **Admin Configuration** — Credentials + feature flags + env var overrides (FR-07, FR-08, FR-09)
4. **Session State** — KVStore-backed call sessions with participant tracking (FR-12)
5. **WebSocket Events** — Real-time UI synchronization (FR-15)
6. **Custom Post Type** — `custom_cf_call` with active/ended states (FR-13)
7. **Mobile Support** — VoIP token registration, push notifications, dismiss API (FR-16)
8. **Frontend UI** — Channel header button, call post, toast bar, modals (FR-02–FR-05)
9. **Standalone Call Page** — Token-auth page that loads RTK React SDK (FR-04)
10. **Web Worker Endpoint** — CSP-compliant `/worker.js` endpoint (FR-11)

### Existing Components (from Reverse Engineering)
- `server/plugin.go` — Plugin core (lifecycle, HTTP routing, config) — needs major extension
- `server/api.go` — Minimal HTTP routing stub
- `server/configuration.go` — Config struct (currently minimal)
- `server/command/` — Slash command handler
- `server/store/kvstore/` — KVStore interface and implementation
- `webapp/src/index.tsx` — Plugin entry point (minimal)

### Proposed New Components (Backend)
- `server/api/` — Dedicated API handler package with all HTTP endpoints
- `server/rtkclient/` — Cloudflare RTK API client (interface + implementation)
- `server/push/` — Mobile push notification delivery

### Proposed New Components (Frontend)
- `webapp/src/components/` — All UI components (channel header, custom post, toast bar, modals)
- `webapp/src/redux/` — Redux slices for call state and WebSocket events
- `webapp/src/call/` — Standalone call page (separate Vite bundle entry point)

---

## Clarifying Questions

### Q1: Floating Widget — Contradiction Between Requirements and User Stories

The requirements document (FR-05) states:
> "No floating widget is required. The call UI is fully contained in the new browser tab."

However, User Story US-006 describes a floating widget appearing within Mattermost with participant count, call duration, mute control, and an "Open in new tab" button.

**Which specification is correct?**

A) **No floating widget** (follow FR-05): The in-call indicator is limited to the channel header button showing "In call" and the channel call toast bar. No persistent Mattermost-side widget.

B) **Floating widget required** (follow US-006): A floating widget appears within the Mattermost window showing call status, allowing the user to open the full call UI in a new tab from within Mattermost.

[Answer]: B

---

### Q2: Backend Service Layer Organization

The call business logic (session creation, participant management, auto-end logic) can be organized in two ways:

A) **Flat: Logic in `plugin.go` / `api/` directly** — Handler functions call KVStore and RTK client directly. Simpler structure, acceptable for this scope.

B) **Service layer: Separate `server/service/` package** — A `CallService` struct encapsulates all call lifecycle logic and is called by the API handlers. Better separation of concerns, easier to unit-test.

[Answer]: A

---

### Q3: Frontend State Management

The frontend needs to track active call state (call ID, participants, user's in-call status) across multiple components. Two approaches:

A) **Redux slice** — Add a `calls` Redux slice to the existing Mattermost Redux store (consistent with how Mattermost plugins typically work). State persists across route changes.

B) **React Context** — Provide call state via a React Context provider. Simpler but state may not survive all navigation events.

[Answer]: A

---

### Q4: Standalone Call Page — Authentication Strategy

The standalone call page at `/plugins/{id}/call` is opened in a new browser tab with a JWT token in the URL. How should this page authenticate with the Mattermost server?

A) **Token-only** — The page uses only the JWT token passed in the URL to initialize the RTK SDK. No calls back to the Mattermost API are needed from the call page itself. (Simplest approach — matches FR-04.)

B) **Token + Mattermost session** — The call page also uses the Mattermost session cookie to make plugin API calls (e.g., to fetch call metadata or leave cleanly on tab close). Requires the page to be loaded with a valid session.

[Answer]: B

---

### Q5: Leave-on-Tab-Close Detection Strategy

When a user closes the standalone call browser tab, the plugin needs to detect the departure and update KVStore (US-013). How should this be implemented?

A) **RTK SDK callback** — The standalone call page uses the RTK SDK's `onParticipantLeft` or `onCallEnd` event to notify the Mattermost backend before the tab closes (via `fetch` or `navigator.sendBeacon`).

B) **Server-side polling/heartbeat** — The call page sends periodic heartbeats to the plugin, and the server marks a participant as left if the heartbeat stops.

C) **Both A + B** — Primary: RTK SDK callback. Fallback: heartbeat timeout for cases where the callback does not fire (e.g., crash).

[Answer]: C — heartbeat timeout set to 60 seconds (to reduce false positives from network blips)

