# Unit 2: Server API & WebSocket — Functional Design Plan

## Unit Context

**Unit**: Unit 2 — Server API & WebSocket
**Stories**: US-005, US-009, US-013, US-015, US-021, US-022, US-025, US-032
**Dependencies**: Unit 1 (RTK Integration — call lifecycle methods in `plugin.go`)

**Unit Responsibilities** (from unit-of-work.md):
- HTTP router with `Mattermost-User-ID` auth middleware
- 8 REST endpoints + static file serving
- WebSocket events emitted for all call state changes (already emitted by Unit 1 `calls.go` — Unit 2 wires the router)
- Serve static assets (`/call`, `/call.js`, `/worker.js`)
- Carry-over: Add `sync.Mutex` to Plugin struct for concurrent KVStore protection

## Execution Checklist

- [x] Step 1: Analyze Unit Context (read unit-of-work.md, existing code)
- [x] Step 2: Create Functional Design Plan (this file)
- [x] Step 3: Generate clarifying questions
- [x] Step 4: Store Plan (this file)
- [x] Step 5: Collect and analyze answers
- [x] Step 6: Generate Functional Design artifacts
- [x] Step 7: Present completion message
- [x] Step 8: Wait for explicit approval

---

## Clarifying Questions

Please answer the questions below by filling in the `[Answer]:` tags. Use the letter choices provided.

---

### Q1: Response Schema — POST /calls (CreateCall)

When a call is successfully created, what should the JSON response body include?

A) `{ "call_id": "...", "token": "...", "channel_id": "..." }` — minimal
B) `{ "call_id": "...", "token": "...", "channel_id": "...", "start_at": 1234 }` — include timestamp
C) Full CallSession object plus token: `{ "call": { ...all CallSession fields... }, "token": "..." }`
D) Only the RTK token: `{ "token": "..." }` — client already knows channel context
E) Other (describe below)

[Answer]:

---

### Q2: Response Schema — POST /calls/{id}/token (JoinCall)

When a user joins and gets an RTK token, what should the response include?

A) `{ "token": "..." }` only
B) `{ "token": "...", "meeting_id": "..." }` — include RTK meeting ID for SDK init
C) `{ "token": "...", "call_id": "...", "channel_id": "..." }` — full context
D) Other (describe below)

[Answer]:

---

### Q3: Already-In-Call Join Behavior

If a user calls `POST /calls/{id}/token` and they are already in `Participants`, what should happen?

A) Return a fresh token (regenerate) — idempotent join, 200 OK
B) Return 409 Conflict — user must leave first
C) Return a fresh token AND re-emit `custom_com.kondo97.mattermost-plugin-rtk_user_joined` WebSocket event
D) Return the same token as before (requires token caching — complex)
E) Other (describe below)

[Answer]:

---

### Q4: Admin Role Check — GET /config/admin-status

The `GET /config/admin-status` endpoint is admin-only. What constitutes "admin" for this plugin?

A) Mattermost System Admin role only (`model.SystemAdminRoleId`)
B) System Admin OR Plugin Admin (custom role)
C) Channel Admin within the requested channel
D) Other (describe below)

[Answer]:

---

### Q5: Error Response Format

What format should error responses use?

A) Plain text string body (e.g., `"Not found"`) with appropriate HTTP status code
B) JSON with `{"error": "message"}` and appropriate HTTP status code
C) JSON with `{"error": "message", "code": "ERROR_CODE"}` for machine-readable codes
D) Other (describe below)

[Answer]:

---

### Q6: Static File Auth Bypass — Scope

The unit-of-work says `/call`, `/call.js`, `/worker.js` should bypass the `Mattermost-User-ID` auth middleware. Should this bypass also apply to other static assets (e.g., CSS, images)?

A) Only the three files listed (`/call`, `/call.js`, `/worker.js`) — strict allowlist
B) Any path under `/static/` prefix bypasses auth
C) All non-API paths bypass auth (anything not under `/api/v1/`)
D) Other (describe below)

[Answer]:

---

### Q7: Dismiss Endpoint — Authorization

`POST /calls/{id}/dismiss` is the mobile dismiss notification endpoint. Should the server validate that the requesting user can only dismiss their own notification?

A) Yes — the user can only dismiss their own notification; server uses `Mattermost-User-ID` header to scope the dismiss to that user
B) No — any authenticated user may dismiss any notification (channel-level dismiss)
C) Yes — and if the call is not found or already ended, return 200 (idempotent no-op)
D) Other (describe below)

[Answer]:

---

### Q8: VoIP Token Endpoint — POST /mobile/voip-token

This stores the user's VoIP device token. What validation should be applied?

A) Require a non-empty `token` string in the request body; reject empty
B) Require a non-empty `token` and validate format (e.g., length bounds only)
C) Accept any non-null value; no format validation (device token format varies by platform)
D) Other (describe below)

[Answer]:

---

### Q9: Concurrency Mutex Scope (Carry-over from Unit 1)

The unit-of-work specifies adding a `sync.Mutex` (or `sync.RWMutex`) to the Plugin struct to guard concurrent KVStore read-modify-write for `JoinCall` and `LeaveCall`. What scope should the mutex cover?

A) A single `callMu sync.Mutex` on Plugin — all call operations serialize through it
B) A `callMu sync.RWMutex` on Plugin — reads share, writes exclusive (more concurrent for read-heavy paths)
C) Per-call mutex map `map[callID]*sync.Mutex` — finer grained, only serialize per-call
D) Other (describe below)

[Answer]:

---

### Q10: GET /config/status — What does it return?

The non-admin config status endpoint. What information should it expose to any authenticated user?

A) `{ "enabled": true/false }` — just whether the plugin is ready to use
B) `{ "enabled": true/false, "org_id": "..." }` — include Cloudflare org ID (no API key)
C) `{ "enabled": true/false, "feature_flags": { "recording": true, ... } }` — include all feature flags
D) Other (describe below)

[Answer]:
