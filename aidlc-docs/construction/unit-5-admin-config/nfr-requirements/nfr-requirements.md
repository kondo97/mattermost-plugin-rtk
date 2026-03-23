# Unit 5: Admin & Config — NFR Requirements

## Performance

| ID | Requirement | Target |
|---|---|---|
| PERF-01 | Env var read latency | Per-call `os.LookupEnv` — acceptable (syscall overhead < 1μs, not on hot path) |
| PERF-02 | Configuration clone cost | O(1) shallow copy — all fields are value types or pointers |
| PERF-03 | `OnConfigurationChange` execution time | < 50ms (no I/O except Mattermost config load + optional RTK client init) |

**Pattern alignment**: Per-call `os.LookupEnv` in `GetEffective*()` and `Is*Enabled()` — consistent with `mattermost-plugin-calls` (`getRTCDURL()`, `getJobServiceURL()` patterns).

---

## Security

| ID | Requirement | Detail |
|---|---|---|
| SEC-01 | API Key never logged | `GetEffectiveAPIKey()` return value MUST NOT appear in any log output |
| SEC-02 | API Key stored encrypted | `"secret": true` in `plugin.json` — Mattermost encrypts the value in server config |
| SEC-03 | API Key never returned in API responses | `/config/status` and `/config/admin-status` MUST NOT include credential values |
| SEC-04 | Env var presence check via `os.LookupEnv` | Distinguishes "not set" from "set to empty" for strict override semantics |

### SECURITY Extension Compliance

| Rule | Status | Notes |
|---|---|---|
| SECURITY-01 | N/A | No data persistence store in this unit (config stored by Mattermost server) |
| SECURITY-02 | N/A | No network intermediaries owned by this unit |
| SECURITY-03 | Compliant | API Key never logged (SEC-01); structured logging via Mattermost plugin API |
| SECURITY-04 | N/A | No HTML-serving endpoints in this unit |
| SECURITY-05 | Compliant | No user-supplied string concatenation; empty check only (no injection risk) |
| SECURITY-06 | N/A | No IAM policies in this unit |
| SECURITY-07 | N/A | No network configuration in this unit |
| SECURITY-08 | Compliant | Admin-only endpoints use server-side role check (existing Unit 2 middleware) |
| SECURITY-09 | Compliant | No default credentials; error responses are generic |
| SECURITY-10 | N/A | No new dependencies introduced in this unit |
| SECURITY-11 | Compliant | Credential logic isolated in `GetEffective*()` methods |
| SECURITY-12 | N/A | No user authentication in this unit (delegated to Mattermost) |
| SECURITY-13 | N/A | No deserialization of untrusted data beyond Mattermost config load |
| SECURITY-14 | N/A | No alerting infrastructure in this unit |
| SECURITY-15 | Compliant | `OnConfigurationChange` returns error on load failure; no fail-open paths |

---

## Reliability

| ID | Requirement | Detail |
|---|---|---|
| REL-01 | `OnConfigurationChange` failure handling | Return error on `LoadPluginConfiguration` failure; do not update `p.configuration` |
| REL-02 | Invalid feature flag env var value | Log warning, fall back to config value (fail safe) |
| REL-03 | Credentials cleared via env var | If `RTK_ORG_ID` or `RTK_API_KEY` is set to empty string, treat as intentional — set `rtkClient = nil` |

---

## Testability

| ID | Requirement | Detail |
|---|---|---|
| TEST-01 | Env var tests use `t.Setenv` | Go 1.17+ `t.Setenv` automatically restores env var after test — no manual cleanup needed |
| TEST-02 | Feature flag nil/true/false coverage | Each `Is*Enabled()` method has 3 test cases: nil pointer (default ON), `&true`, `&false` |
| TEST-03 | Env var override coverage | Each `GetEffective*()` and `Is*Enabled()` tested with env var set AND unset |
| TEST-04 | `OnConfigurationChange` credential change detection | Test: credential change → rtkClient re-initialized; no change → rtkClient unchanged |

---

## Input Validation

| ID | Requirement | Detail |
|---|---|---|
| VAL-01 | `CloudflareOrgID` | No format validation — empty check only |
| VAL-02 | `CloudflareAPIKey` | No format validation — empty check only |
| VAL-03 | Feature flag env var values | Case-insensitive `"true"` only maps to enabled; all other values map to disabled |
