# Application Design Plan

## Context Analysis Summary

- **Project**: Mattermost plugin for Cloudflare RealtimeKit (brownfield, starter template → production)
- **Scope**: Go backend + React/TypeScript frontend + dual-bundle build system
- **Key constraints from requirements**:
  - `RTKClient` and `KVStore` behind interfaces (NFR-06)
  - Config via thread-safe struct with `sync.RWMutex` + Clone pattern (NFR-06)
  - Vite dual-bundle: `main.js` (React/Redux external) + `call.js` (self-contained)
  - SECURITY-01–15 enforced as blocking constraints
  - No Infrastructure Design needed

---

## Design Plan Checkboxes

- [x] Analyze requirements and user stories
- [ ] Generate questions and collect answers
- [ ] Generate `components.md`
- [ ] Generate `component-methods.md`
- [ ] Generate `services.md`
- [ ] Generate `component-dependency.md`
- [ ] Generate `application-design.md` (consolidated)
- [ ] Validate all design artifacts for consistency
- [ ] Update `aidlc-state.md`

---

## Design Questions

The following questions address ambiguities in the design scope. Please fill in each `[Answer]:` tag.

---

### Q1: Go Package Structure

The backend needs to organize the following responsibilities:
- RTK API client (`RTKClient` interface + Cloudflare HTTP implementation)
- Call lifecycle business logic (create, join, end, leave, auto-end)
- KVStore access abstraction (`KVStore` interface)
- HTTP route handlers (REST API endpoints)
- WebSocket event emission
- Push notification delivery
- Plugin lifecycle + main `Plugin` struct

**Option A — Flat (single `server` package):**
All Go code lives in `server/` as a single package. Sub-files are named by responsibility: `rtk_client.go`, `call_service.go`, `kvstore.go`, `api.go`, etc.

**Option B — Sub-packages:**
```
server/
  plugin.go          (Plugin struct, hooks)
  api.go             (HTTP route registration)
  rtkclient/         (RTKClient interface + implementation)
  callservice/       (CallService struct, call lifecycle)
  kvstore/           (KVStore interface + KV implementation)
```

[Answer]:

---

### Q2: Frontend State Management

The Mattermost plugin webapp needs to manage:
- Current call state per channel (active call, participants, local user's join state)
- Global "is local user in a call" state (drives the floating widget and SwitchCallModal)
- WebSocket event handling

**Option A — Redux (Mattermost pattern):**
Use Redux actions, reducers, and selectors registered via `registry.registerReducer`. Consistent with the Mattermost Calls plugin architecture.

**Option B — React Context + hooks:**
A single `CallContext` wraps the webapp, managing call state and WebSocket listeners. Simpler, no Redux dependency beyond what Mattermost provides.

[Answer]:

---

### Q3: Standalone Call Page Delivery

The standalone call page (opened in a new tab) needs its own React bundle that is NOT the main Mattermost plugin bundle. NFR-04 requires `call.js` bundled with RTK SDK included.

**Option A — Served directly by the plugin Go binary (embedded):**
`call.js` and `call.html` are embedded via `//go:embed` in the Go binary and served at `GET /plugins/{id}/call` and `GET /plugins/{id}/call.js`.

**Option B — Served as static files via Mattermost static asset serving:**
Plugin registers static files for the call page through the Mattermost plugin static file mechanism.

Which approach should be used?

[Answer]:

---

### Q4: WebSocket Event Handling in Frontend

WebSocket events from the plugin need to reach all React components that care about call state updates.

**Option A — Centralized subscription in Redux middleware / store:**
Register one `WebSocket.registerEventHandler` call per event type in a central location. Events dispatch Redux actions that update the store. Components use selectors.

**Option B — Per-component subscriptions:**
Each component registers its own WebSocket handler (e.g., `CallPost` handles `custom_cf_user_joined` independently).

[Answer]:

---

### Q5: Call Session State Storage (KVStore dual-key design)

Requirements specify two KVStore keys per call:
- `call:channel:{channelID}` → active call lookup by channel
- `call:id:{callID}` → call lookup by RTK meeting ID

**Option A — Duplicate data (both keys hold the full `CallSession` struct):**
Each key stores the full session JSON. Writes must update both keys atomically (best-effort; no distributed transactions).

**Option B — Primary + index (one key holds data, other holds reference):**
`call:id:{callID}` holds the full `CallSession`. `call:channel:{channelID}` holds only the `callID` string as a reference (lookup requires two KV reads for channel-based lookups).

[Answer]:

