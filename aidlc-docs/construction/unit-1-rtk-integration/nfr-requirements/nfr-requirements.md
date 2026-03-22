# Unit 1: RTK Integration — NFR Requirements

## Performance

| ID | Requirement | Target |
|---|---|---|
| PERF-01 | Token generation API response time | < 1 second under normal Cloudflare API conditions (NFR-03) |
| PERF-02 | Concurrent user scale | Up to 200 concurrent users (signaling only; media handled by Cloudflare) |
| PERF-03 | RTK HTTP client timeout | 10 seconds (applies to CreateMeeting, GenerateToken, EndMeeting) |
| PERF-04 | Background job interval | Every 30 seconds |
| PERF-05 | Heartbeat stale timeout | 60 seconds (participant removed if no heartbeat within this window) |

## Reliability

| ID | Requirement | Decision |
|---|---|---|
| REL-01 | RTK API retry policy | No automatic retry — return error immediately on failure; user retries via UI interaction |
| REL-02 | EndMeeting failure handling | Best effort — log warning and continue; call end proceeds regardless |
| REL-03 | CreatePost failure handling | No rollback — log warning and continue; call session remains in KVStore |
| REL-04 | Background job partial failure | Continue to next call on per-call error (best effort); log each failure |
| REL-05 | Concurrent CreateCall race | Accept race condition — duplicate active call per channel is prevented by KVStore check; simultaneous requests may rarely result in two sessions, but cleanup handles orphans |

## Security

| ID | Requirement | Source |
|---|---|---|
| SEC-01 | All Cloudflare RTK API calls use HTTPS only | NFR-01, SECURITY-01 |
| SEC-02 | Cloudflare credentials (orgID, apiKey) never logged or returned to clients | NFR-02, SECURITY-03, SECURITY-09 |
| SEC-03 | HeartbeatCall validates caller is in participants list | NFR-02, SECURITY-08 (IDOR prevention) |
| SEC-04 | EndCall validates caller is the session creator | NFR-02, SECURITY-08 (function-level authorization) |
| SEC-05 | All KVStore values deserialized with explicit JSON schema validation | SECURITY-13 |
| SEC-06 | Structured logging for all call lifecycle events; no sensitive data in logs | NFR-06, SECURITY-03 |
| SEC-07 | Error responses use generic messages — no internal details exposed | SECURITY-09, SECURITY-15 |
| SEC-08 | All external calls (RTK API, KVStore) have explicit error handling | SECURITY-15 |
| SEC-09 | Credentials not validated on activation; validated implicitly on first API call | —  |

## Availability

| ID | Requirement |
|---|---|
| AVA-01 | Plugin availability follows Mattermost server availability (no independent HA requirement) |
| AVA-02 | Background job runs on a single goroutine; no distributed coordination required |

## Maintainability

| ID | Requirement | Source |
|---|---|---|
| MAINT-01 | RTKClient abstracted behind interface for mock-based unit testing | NFR-06 |
| MAINT-02 | KVStore access abstracted behind interface for mock-based unit testing | NFR-06 |
| MAINT-03 | Unit tests for all call lifecycle methods (CreateCall, JoinCall, LeaveCall, EndCall, HeartbeatCall, CleanupStaleParticipants) | NFR-06 |
| MAINT-04 | Structured logging using Mattermost plugin logger (`p.API.LogInfo`, `p.API.LogWarn`, `p.API.LogError`) | NFR-06 |
| MAINT-05 | Log entries include: call_id, channel_id, user_id (where applicable), operation name, and error (where applicable) | SECURITY-03 |

## Security Compliance Summary (SECURITY Extension)

| Rule | Status | Rationale |
|---|---|---|
| SECURITY-01 | Compliant | HTTPS enforced on all Cloudflare API calls; KVStore encryption managed by Mattermost server |
| SECURITY-02 | N/A | No load balancers or API gateways owned by this unit |
| SECURITY-03 | Compliant | Structured logging required; no credentials/tokens in logs (SEC-02, SEC-06) |
| SECURITY-04 | N/A | No HTML-serving endpoints in Unit 1 |
| SECURITY-05 | Compliant | Input validation on callID/userID/channelID required; KVStore JSON deserialization validated (SEC-05) |
| SECURITY-06 | N/A | No IAM policies; plugin uses Mattermost Plugin API with fixed permissions |
| SECURITY-07 | N/A | No networking configuration owned by this unit |
| SECURITY-08 | Compliant | HeartbeatCall participant check (SEC-03); EndCall creator check (SEC-04) enforced |
| SECURITY-09 | Compliant | No credentials in logs or error responses (SEC-02, SEC-07) |
| SECURITY-10 | N/A | Dependency lock files are a project-level concern, addressed in Build and Test stage |
| SECURITY-11 | Compliant | Auth/authz logic isolated in Plugin Core methods; misuse case (concurrent CreateCall) addressed in REL-05 |
| SECURITY-12 | N/A | No user authentication managed in this unit; Mattermost handles auth |
| SECURITY-13 | Compliant | KVStore deserialization uses explicit JSON schema (SEC-05); no unsafe deserialization |
| SECURITY-14 | Compliant | Lifecycle events logged per MAINT-04/05; alerting is Mattermost server responsibility |
| SECURITY-15 | Compliant | All external calls (RTK API, KVStore) require explicit error handling (SEC-08); fail-closed pattern on authorization errors |
