# Unit 1: RTK Integration — Logical Components

## Component Map

```
server/
  errors.go                  ← Sentinel error definitions (Pattern 3)
  plugin.go                  ← Plugin struct + call lifecycle methods
  job.go                     ← Background job (CleanupStaleParticipants)
  rtkclient/
    interface.go             ← RTKClient interface (Pattern 1)
    client.go                ← HTTP implementation (Pattern 2: timeout)
  store/kvstore/
    kvstore.go               ← KVStore interface + implementation (Pattern 1)
```

---

## Plugin Struct (relevant fields)

```go
type Plugin struct {
    plugin.MattermostPlugin
    configurationLock sync.RWMutex
    configuration     *configuration
    rtkClient         rtkclient.RTKClient   // injected in OnActivate
    kvStore           kvstore.KVStore       // injected in OnActivate
}
```

---

## OnActivate Wiring

```
OnActivate()
  ├── p.kvStore = kvstore.New(p.API)
  ├── cfg = p.getConfiguration()
  ├── if cfg.CloudflareOrgID != "" && cfg.CloudflareAPIKey != "":
  │     p.rtkClient = rtkclient.NewClient(cfg.GetEffectiveOrgID(), cfg.GetEffectiveAPIKey())
  └── start background job
```

Note: `rtkClient` may be nil if credentials are not yet configured. All call lifecycle methods must check for nil and return `ErrRTKNotConfigured`.

---

## Error Flow

```
Plugin method (e.g. CreateCall)
  │
  ├── Authorization / precondition check
  │     └── fail → return sentinel error (ErrCallAlreadyActive etc.)
  │
  ├── External call (RTKClient / KVStore)
  │     ├── success → continue
  │     └── fail → log error + return wrapped error
  │
  └── Best-effort operations (CreatePost, EndMeeting)
        ├── success → update state
        └── fail → log warning, continue
```

---

## Background Job Flow

```
job.go ticker (30s interval)
  └── p.CleanupStaleParticipants()
        ├── kvStore.GetAllActiveCalls()
        │     └── error → LogError, return (skip this run)
        ├── for each session:
        │     for each participant:
        │       heartbeat = kvStore.GetHeartbeat(callID, userID)
        │       if stale → p.LeaveCall(callID, userID)
        │             └── error → LogError, continue to next participant
        └── done
```

---

## Dependency Injection for Testing

```go
// Test setup
func newTestPlugin(rtkClient rtkclient.RTKClient, store kvstore.KVStore) *Plugin {
    p := &Plugin{}
    p.rtkClient = rtkClient
    p.kvStore = store
    return p
}

// Usage
mockRTK := &mocks.RTKClient{}
mockStore := &mocks.KVStore{}
p := newTestPlugin(mockRTK, mockStore)
```
