# Unit 2: Server API & WebSocket — Tech Stack Decisions

## HTTP Router

| Decision | Choice | Rationale |
|---|---|---|
| Router library | `github.com/gorilla/mux` | Already used in existing `server/api.go`; supports path variables (`{id}`), subrouters, and middleware chaining |
| Auth middleware | Custom `MattermostAuthorizationRequired` | Already implemented; checks `Mattermost-User-ID` header |
| Static file bypass | Explicit allowlist on root router | Three paths registered directly on root router, outside auth subrouter |

## Response Serialization

| Decision | Choice | Rationale |
|---|---|---|
| JSON encoding | `encoding/json` stdlib | Consistent with Unit 1; no external dependency needed |
| Error response helper | Inline `json.NewEncoder(w).Encode` | Simple; no abstraction needed for a thin adapter layer |

## Static File Embedding

| Decision | Choice | Rationale |
|---|---|---|
| Embedding mechanism | `//go:embed` directive (Go 1.16+) | Embeds `assets/` directory into Go binary at compile time; no runtime I/O |
| Asset directory | `server/assets/` | Standard location; populated by webapp build (Unit 4) |

## Testing

| Decision | Choice | Rationale |
|---|---|---|
| HTTP testing | `net/http/httptest` stdlib | Standard Go HTTP handler testing; no additional dependency |
| Mocking | Existing `mock_kvstore.go` + `mock_rtkclient.go` (go.uber.org/mock) | Consistent with Unit 1 test strategy; no new interfaces needed |
| Test framework | `github.com/stretchr/testify` | Already used in Unit 1 tests |

## Concurrency

| Decision | Choice | Rationale |
|---|---|---|
| Call mutation guard | `sync.Mutex` (`callMu` on Plugin struct) | Simple and correct; low call volume means serialization overhead is acceptable; consistent with Plugin-level mutex pattern |

## No New Dependencies

Unit 2 introduces no new Go module dependencies. All required packages are already present:
- `github.com/gorilla/mux` — routing
- `encoding/json` — serialization
- `net/http` — HTTP primitives
- `sync` — mutex
- `github.com/mattermost/mattermost/server/public/model` — WebSocket broadcast, permission check
