# Unit 2: Server API & WebSocket — NFR Design Patterns

## Inherited Patterns from Unit 1

The following patterns from Unit 1 apply unchanged to Unit 2:

| Pattern | Description | Applied In |
|---|---|---|
| Pattern 3: Sentinel Errors | `errors.Is()` maps domain errors to HTTP status codes | All handlers |
| Pattern 6: Structured Logging | `p.API.LogInfo/Warn/Error` with context fields; no sensitive data | All handlers |
| Pattern 7: Explicit Error Handling | Every KVStore/Plugin method call has explicit error check | All handlers |
| Pattern 8: Generic Error Messages | Internal errors logged, generic message returned to caller | All handlers |

---

## Pattern U2-1: Auth Middleware (SEC-U2-01, BR-U2-01)

The `Mattermost-User-ID` middleware is applied as a subrouter-level middleware. Static paths are registered on the root router, outside the subrouter.

```go
func (p *Plugin) initRouter() *mux.Router {
    router := mux.NewRouter()

    // Static files: no auth
    router.HandleFunc("/call", p.serveCallHTML).Methods(http.MethodGet)
    router.HandleFunc("/call.js", p.serveCallJS).Methods(http.MethodGet)
    router.HandleFunc("/worker.js", p.serveWorkerJS).Methods(http.MethodGet)

    // API: auth required
    api := router.PathPrefix("/api/v1").Subrouter()
    api.Use(p.MattermostAuthorizationRequired)

    api.HandleFunc("/calls", p.handleCreateCall).Methods(http.MethodPost)
    api.HandleFunc("/calls/{id}/token", p.handleJoinCall).Methods(http.MethodPost)
    api.HandleFunc("/calls/{id}/leave", p.handleLeaveCall).Methods(http.MethodPost)
    api.HandleFunc("/calls/{id}", p.handleEndCall).Methods(http.MethodDelete)
    // heartbeat endpoint deferred / not implemented — RTK webhook handles cleanup
    api.HandleFunc("/calls/{id}/dismiss", p.handleDismiss).Methods(http.MethodPost)
    api.HandleFunc("/config/status", p.handleConfigStatus).Methods(http.MethodGet)
    api.HandleFunc("/config/admin-status", p.handleAdminConfigStatus).Methods(http.MethodGet)

    return router
}
```

**Key design decision**: Static paths use explicit registration (allowlist), not a blanket exception. Any new path added to the router requires a conscious decision about auth.

---

## Pattern U2-2: HTTP Security Headers on /call (SEC-U2-05, SECURITY-04)

Security headers are set only on the HTML-serving endpoint. JS files do not need the full set (no framing, no navigation).

```go
func (p *Plugin) serveCallHTML(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src *")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
    w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
    // serve embedded HTML...
}
```

**CSP rationale**: `connect-src *` allows WebRTC/WebSocket connections to Cloudflare RTK infrastructure without hardcoding domain patterns.

---

## Pattern U2-3: Concurrency Mutex (BR-U2-39 to BR-U2-41)

A single `callMu sync.Mutex` on the Plugin struct guards all call-mutating HTTP handlers. The mutex is acquired in the handler, wrapping the Plugin method call.

```go
// Plugin struct (server/plugin.go)
type Plugin struct {
    plugin.MattermostPlugin
    kvStore    kvstore.KVStore
    rtkClient  rtkclient.RTKClient
    client     *pluginapi.Client
    router     *mux.Router
    callMu     sync.Mutex        // guards concurrent JoinCall/LeaveCall/EndCall/CreateCall
    configurationLock sync.RWMutex
    configuration     *configuration
}

// Handler usage pattern:
func (p *Plugin) handleJoinCall(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("Mattermost-User-ID")
    callID := mux.Vars(r)["id"]

    p.callMu.Lock()
    defer p.callMu.Unlock()

    token, err := p.JoinCall(callID, userID)
    // ...
}
```

**Scope**: `handleCreateCall`, `handleJoinCall`, `handleLeaveCall`, `handleEndCall` acquire `callMu`.
**Excluded**: `handleDismiss`, `handleConfigStatus`, `handleAdminConfigStatus` — no read-modify-write on participants. (Heartbeat handler deferred / not implemented.)

---

## Pattern U2-4: Static File Embedding (REL-U2-03)

Assets are embedded into the Go binary at compile time using `//go:embed`. No runtime file I/O, no deployment dependency on asset files.

```go
// server/api_static.go (flat structure, not api/ subdirectory)
import "embed"

//go:embed assets/call.html
var callHTML []byte

//go:embed assets/call.js
var callJS []byte

//go:embed assets/worker.js
var workerJS []byte

func (p *Plugin) serveCallHTML(w http.ResponseWriter, r *http.Request) {
    // set security headers...
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write(callHTML)
}
```

**Build dependency**: `assets/call.js` and `assets/worker.js` are produced by the webapp build (Unit 4). If assets are absent at compile time, the build fails — making missing assets a compile-time error rather than a runtime error.

---

## Pattern U2-5: Error Response Helper (Pattern 8 applied to HTTP)

A consistent JSON error writer prevents format drift across handlers.

```go
// server/api.go (flat structure, not api/ subdirectory)
func writeError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Usage in handlers:
if errors.Is(err, ErrCallNotFound) {
    writeError(w, http.StatusNotFound, "call not found")
    return
}
```

---

## Pattern U2-6: Admin Role Guard (SEC-U2-03, SECURITY-08)

Admin-only endpoints check system admin permission server-side before returning any data.

```go
func (p *Plugin) handleAdminConfigStatus(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("Mattermost-User-ID")
    if !p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
        writeError(w, http.StatusForbidden, "Forbidden")
        return
    }
    // return admin config...
}
```

**Fail-closed**: Permission check failure immediately returns 403 before any data access.
