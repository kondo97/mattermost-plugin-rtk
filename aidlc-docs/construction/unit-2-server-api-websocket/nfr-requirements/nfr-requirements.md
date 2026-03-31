# Unit 2: Server API & WebSocket — NFR Requirements

## Performance

| ID | Requirement | Target |
|---|---|---|
| PERF-U2-01 | HTTP handler overhead (excluding KVStore and RTK API latency) | < 100ms |
| PERF-U2-02 | End-to-end API response time (including RTK API) | Inherits PERF-01 from Unit 1: < 1s under normal conditions |
| PERF-U2-03 | Concurrent request handling | Gorilla/mux handles each request in its own goroutine; `callMu` serializes only call-mutating operations |

---

## Reliability

| ID | Requirement | Decision |
|---|---|---|
| REL-U2-01 | Request body size limits | Not enforced at plugin level; Mattermost server controls body size |
| REL-U2-02 | Dismiss endpoint resilience | Idempotent — always returns 200 regardless of call state; no KVStore read |
| REL-U2-03 | Static file serving | Files embedded in Go binary via `go:embed`; no runtime I/O failure possible |
| REL-U2-04 | Error propagation | All Plugin method errors are caught and mapped to HTTP status codes; no unhandled panics |

---

## Security

| ID | Requirement | Source |
|---|---|---|
| SEC-U2-01 | All `/api/v1/` endpoints require non-empty `Mattermost-User-ID` header | BR-U2-01, SECURITY-08 |
| SEC-U2-02 | Static endpoints (`/call`, `/call.js`, `/worker.js`) bypass auth via explicit allowlist only | BR-U2-02, SECURITY-08 |
| SEC-U2-03 | `GET /config/admin-status` enforces System Admin role check server-side | BR-U2-27, SECURITY-08 |
| SEC-U2-04 | `POST /calls/{id}/dismiss` scoped to requesting user only | BR-U2-30, SECURITY-08 (IDOR prevention) |
| SEC-U2-05 | HTTP security headers set on `GET /call` (HTML response) | SECURITY-04 |
| SEC-U2-06 | Error responses use generic messages — no stack traces or internal details | BR-U2-38, SECURITY-09, SECURITY-15 |
| SEC-U2-07 | Cloudflare credentials never returned to clients in any response | Inherits SEC-02 from Unit 1 |
| SEC-U2-08 | All handler errors explicitly caught and handled (fail-closed) | SECURITY-15 |

### HTTP Security Headers for GET /call (SECURITY-04)

| Header | Value |
|---|---|
| `Content-Security-Policy` | `default-src 'self'; connect-src *` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` |

**CSP rationale**: `connect-src *` allows WebRTC/WebSocket connections to Cloudflare RTK infrastructure without hardcoding specific Cloudflare domain patterns, providing resilience to Cloudflare URL changes.

---

## Availability

| ID | Requirement |
|---|---|
| AVA-U2-01 | Plugin availability follows Mattermost server availability (inherits AVA-01 from Unit 1) |
| AVA-U2-02 | No independent HA requirement for the HTTP handler layer |

---

## Maintainability

| ID | Requirement | Source |
|---|---|---|
| MAINT-U2-01 | Unit tests for all handlers using `httptest.NewRecorder` with existing `mock_kvstore.go` and `mock_rtkclient.go` | NFR Q4 decision |
| MAINT-U2-02 | Handler files use flat structure: `server/api.go` (router setup + middleware), `server/api_calls.go`, `server/api_config.go`, `server/api_static.go`, etc. **Updated 2026-03-30**: Changed from `server/api/` subdirectory to flat files in `server/`. Heartbeat handler (`api/heartbeat.go`) not implemented. | unit-of-work.md |
| MAINT-U2-03 | Structured logging in all handlers using `p.API.LogInfo`, `p.API.LogWarn`, `p.API.LogError` | Inherits MAINT-04 from Unit 1 |
| MAINT-U2-04 | Log entries include: call_id (where applicable), user_id, handler name, error | Inherits MAINT-05 from Unit 1 |

---

## Security Compliance Summary (SECURITY Extension)

| Rule | Status | Rationale |
|---|---|---|
| SECURITY-01 | N/A | No new data stores introduced in Unit 2; KVStore encryption managed by Mattermost |
| SECURITY-02 | N/A | No load balancers or API gateways owned by this unit |
| SECURITY-03 | Compliant | Structured logging required in all handlers (MAINT-U2-03/04); no credentials in logs |
| SECURITY-04 | Compliant | HTTP security headers set on `GET /call` HTML response (SEC-U2-05) |
| SECURITY-05 | Compliant | `channel_id` validated as non-empty (BR-U2-03); all path params extracted via gorilla/mux (typed) |
| SECURITY-06 | N/A | No IAM policies; plugin uses Mattermost Plugin API |
| SECURITY-07 | N/A | No networking configuration owned by this unit |
| SECURITY-08 | Compliant | Auth middleware on all API routes (SEC-U2-01); admin role check (SEC-U2-03); dismiss user-scoped (SEC-U2-04) |
| SECURITY-09 | Compliant | Generic error messages only; no internal details in responses (SEC-U2-06) |
| SECURITY-10 | N/A | Dependency management is a project-level concern addressed in Build and Test |
| SECURITY-11 | Compliant | Auth logic isolated in middleware; dismiss misuse addressed (user-scoped, idempotent) |
| SECURITY-12 | N/A | No user authentication managed in this unit; Mattermost handles auth |
| SECURITY-13 | Compliant | Request JSON deserialization uses typed structs; no unsafe deserialization |
| SECURITY-14 | Compliant | Handler errors logged per MAINT-U2-03/04; alerting is Mattermost server responsibility |
| SECURITY-15 | Compliant | All handler errors explicitly caught and mapped (SEC-U2-08); fail-closed on auth errors |
