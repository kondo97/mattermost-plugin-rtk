# Unit 1: RTK Integration — Tech Stack Decisions

## Language & Runtime

| Component | Decision | Rationale |
|---|---|---|
| Language | Go 1.25 | Existing project; Mattermost Plugin SDK requirement |
| Plugin SDK | `github.com/mattermost/mattermost/server/public` | Official SDK; provides PluginAPI, KVStore, WebSocket |

## HTTP Client (Cloudflare RTK API)

| Decision | Value | Rationale |
|---|---|---|
| Client | `net/http` standard library | No external dependency needed for simple REST calls |
| Timeout | 10 seconds | Balances Cloudflare latency tolerance vs. user experience |
| TLS | Default Go TLS (1.2+) | Enforced by `https://` scheme; satisfies SECURITY-01 |
| Auth | HTTP Basic Auth (`orgID:apiKey`) | Per NFR-01 / Cloudflare RTK API spec |
| Retry | None | Return error immediately; user retries via UI (REL-01) |
| Base URL | `https://api.realtime.cloudflare.com/v2` | Per NFR-01 |

## Data Persistence

| Component | Decision | Rationale |
|---|---|---|
| Store | Mattermost KVStore (via PluginAPI) | Plugin constraint; no external DB |
| Serialization | `encoding/json` | Standard library; no extra dependency |
| Key schema | `call:channel:{id}`, `call:id:{id}`, `heartbeat:{callID}:{userID}`, `voip:{userID}` | Per FR-12 / domain-entities.md |

## Concurrency

| Component | Decision | Rationale |
|---|---|---|
| Configuration access | `sync.RWMutex` + Clone pattern | Existing pattern in `configuration.go`; thread-safe reads (NFR-06) |
| Background job | Single goroutine via `job.go` ticker | Existing infrastructure; no distributed coordination needed |
| CreateCall race | Accept (no lock) | KVStore does not support atomic CAS; race is rare and recoverable (REL-05) |

## Testing

| Component | Decision | Rationale |
|---|---|---|
| Test framework | `github.com/stretchr/testify` | Existing project dependency |
| Mocks | `github.com/stretchr/testify/mock` or `go-mock` | Existing project pattern (see `server/command/`) |
| RTKClient mock | Interface-based mock | Enables unit testing without real Cloudflare API calls (MAINT-01) |
| KVStore mock | Interface-based mock | Existing pattern; enables isolated unit tests (MAINT-02) |

## Logging

| Component | Decision | Rationale |
|---|---|---|
| Logger | `p.API.LogInfo` / `p.API.LogWarn` / `p.API.LogError` | Mattermost plugin standard; routes to MM server log (MAINT-04) |
| Format | Structured key-value pairs | Per SECURITY-03; includes call_id, user_id, operation |
| Sensitive data | Never log orgID, apiKey, or RTK tokens | Per SECURITY-03, SEC-02 |
