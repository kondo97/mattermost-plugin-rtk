# Unit 1: RTK Integration — NFR Design Patterns

## Pattern 1: Interface Abstraction (MAINT-01, MAINT-02)

Both external dependencies are accessed exclusively through interfaces, enabling mock-based unit testing.

```go
// server/rtkclient/interface.go
type RTKClient interface {
    CreateMeeting(preset string) (*Meeting, error)
    GenerateToken(meetingID, userID, preset string) (*Token, error)
    EndMeeting(meetingID string) error
}

// server/store/kvstore/kvstore.go (extended)
type KVStore interface {
    // existing methods...
    GetCallByChannel(channelID string) (*CallSession, error)
    GetCallByID(callID string) (*CallSession, error)
    GetAllActiveCalls() ([]*CallSession, error)
    SaveCall(session *CallSession) error
    UpdateCallParticipants(callID string, participants []string) error
    EndCall(callID string, endAt int64) error
    SetHeartbeat(callID, userID string, ts int64) error
    GetHeartbeat(callID, userID string) (int64, error)
    StoreVoIPToken(userID, token string) error
    GetVoIPToken(userID string) (string, error)
}
```

**Usage**: `Plugin` struct holds `rtkClient RTKClient` and `kvStore KVStore` fields, injected in `OnActivate`.

---

## Pattern 2: HTTP Timeout (PERF-03)

Dedicated HTTP client with explicit timeout, shared across all RTK API calls.

```go
// server/rtkclient/client.go
type client struct {
    httpClient *http.Client
    baseURL    string
    orgID      string
    apiKey     string
}

func NewClient(orgID, apiKey string) RTKClient {
    return &client{
        httpClient: &http.Client{Timeout: 10 * time.Second},
        baseURL:    "https://api.realtime.cloudflare.com/v2",
        orgID:      orgID,
        apiKey:     apiKey,
    }
}
```

---

## Pattern 3: Sentinel Errors (SECURITY-08, SECURITY-15)

Domain errors defined as package-level variables. API handlers use `errors.Is()` to map to HTTP status codes.

```go
// server/errors.go
var (
    ErrCallAlreadyActive = errors.New("call already active in channel")
    ErrCallNotFound      = errors.New("call not found or already ended")
    ErrNotParticipant    = errors.New("user is not a participant of this call")
    ErrUnauthorized      = errors.New("only the call creator can perform this action")
)
```

**Mapping in Unit 2 API handlers:**

| Error | HTTP Status |
|---|---|
| `ErrCallAlreadyActive` | 409 Conflict |
| `ErrCallNotFound` | 404 Not Found |
| `ErrNotParticipant` | 403 Forbidden |
| `ErrUnauthorized` | 403 Forbidden |

---

## Pattern 4: Authorization Guard (SEC-03, SEC-04, SECURITY-08)

Explicit authorization checks at the start of each privileged operation, before any state mutation.

```go
// EndCall: creator check
func (p *Plugin) EndCall(callID, requestingUserID string) error {
    session, err := p.kvStore.GetCallByID(callID)
    if err != nil || session == nil || session.EndAt != 0 {
        return ErrCallNotFound
    }
    if session.CreatorID != requestingUserID {
        return ErrUnauthorized  // fail closed — deny before any mutation
    }
    return p.endCallInternal(session)
}

// HeartbeatCall: participant check
func (p *Plugin) HeartbeatCall(callID, userID string) error {
    session, err := p.kvStore.GetCallByID(callID)
    if err != nil || session == nil || session.EndAt != 0 {
        return ErrCallNotFound
    }
    if !containsUser(session.Participants, userID) {
        return ErrNotParticipant  // fail closed
    }
    return p.kvStore.SetHeartbeat(callID, userID, now())
}
```

---

## Pattern 5: Best-Effort / Fire-and-Continue (REL-02, REL-03)

Non-critical operations are attempted but failures do not abort the primary flow.

```go
// EndMeeting — best effort
if err := p.rtkClient.EndMeeting(session.MeetingID); err != nil {
    p.API.LogWarn("EndMeeting failed (best effort)", "call_id", session.ID, "err", err.Error())
    // continue — do not return error
}

// CreatePost — best effort
post, err := p.API.CreatePost(callPost)
if err != nil {
    p.API.LogWarn("CreatePost failed (best effort)", "call_id", session.ID, "err", err.Error())
    // continue — session already saved, call is active
} else {
    session.PostID = post.Id
    _ = p.kvStore.SaveCall(session)
}
```

---

## Pattern 6: Structured Logging (MAINT-04, SECURITY-03)

Every significant event and error is logged with consistent structured fields. Sensitive data is never logged.

```go
// Call lifecycle event
p.API.LogInfo("call started",
    "call_id", session.ID,
    "channel_id", session.ChannelID,
    "creator_id", session.CreatorID,
)

// Error with context
p.API.LogError("CreateMeeting failed",
    "channel_id", channelID,
    "user_id", userID,
    "err", err.Error(),
)
```

**Never log**: `orgID`, `apiKey`, RTK JWT tokens, user passwords.

---

## Pattern 7: Explicit Error Handling on External Calls (SECURITY-15)

Every call to RTKClient or KVStore has an explicit error check. No silent failures.

```go
// Every external call follows this pattern:
result, err := p.rtkClient.SomeMethod(...)
if err != nil {
    p.API.LogError("SomeMethod failed", "err", err.Error())
    return fmt.Errorf("operation failed: %w", err)  // wrap for context
}
```

---

## Pattern 8: Generic Error Messages to Callers (SECURITY-09)

Internal error details are never returned to API callers.

```go
// In API handler (Unit 2):
if err != nil {
    p.API.LogError("CreateCall failed", "err", err.Error())
    http.Error(w, "Failed to start call", http.StatusInternalServerError)
    // NOT: http.Error(w, err.Error(), ...) — would expose internals
}
```
