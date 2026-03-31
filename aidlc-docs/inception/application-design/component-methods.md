# Component Methods

## B-01: Plugin Core (`server/plugin.go`)

### Lifecycle Methods

```go
func (p *Plugin) OnActivate() error
// Initializes: API handler, RTKClient, cleanup loop (stub)
// Registers: HTTP router, slash commands (if any)

func (p *Plugin) OnDeactivate() error
// Stops background job cleanly

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request)
// Delegates to gorilla/mux router

func (p *Plugin) OnConfigurationChange() error
// Reloads configuration, updates effective values
```

### Call Business Logic Methods

```go
func (p *Plugin) CreateCall(userID, channelID string) (*CreateCallResponse, error)
// 1. Check no active call exists for channelID (KVStore)
// 2. Call RTKClient.CreateMeeting("group_call_host")
// 3. Generate JWT token via RTKClient.GenerateToken
// 4. Persist CallSession to KVStore
// 5. Post custom_cf_call post to channel
// 6. Send push notifications (REMOVED — mobile uses WebSocket events)
// 7. Emit custom_com.kondo97.mattermost-plugin-rtk_call_started WebSocket event
// Returns: call_id, token, feature_flags

func (p *Plugin) JoinCall(userID, callID string) (*JoinCallResponse, error)
// 1. Load CallSession from KVStore by callID
// 2. Verify call is active (end_at == 0)
// 3. Call RTKClient.GenerateToken(callID, userID, "group_call_participant")
// 4. Add userID to participants in KVStore
// 5. Emit custom_com.kondo97.mattermost-plugin-rtk_user_joined WebSocket event
// Returns: token, feature_flags

func (p *Plugin) LeaveCall(userID, callID string) error
// 1. Remove userID from participants in KVStore
// 2. Emit custom_com.kondo97.mattermost-plugin-rtk_user_left WebSocket event
// 3. If participants list empty: call p.EndCallInternal(callID)

func (p *Plugin) EndCall(userID, callID string) error
// 1. Verify userID is creator (host check)
// 2. Call p.EndCallInternal(callID)

func (p *Plugin) EndCallInternal(callID string) error
// 1. Set end_at = now in KVStore
// 2. Update custom_cf_call post to ended state
// 3. Emit custom_com.kondo97.mattermost-plugin-rtk_call_ended WebSocket event
// 4. Call RTKClient.EndMeeting(meetingID)
// (Called by EndCall and LeaveCall auto-end)

// server/cleanup.go — placeholder stub
// Waits for stop signal only; no active participant reconciliation implemented
```

### WebSocket Helpers

```go
func (p *Plugin) publishCallStarted(channelID string, session *CallSession)
func (p *Plugin) publishCallEnded(channelID string, session *CallSession)
func (p *Plugin) publishUserJoined(channelID, userID, callID string)
func (p *Plugin) publishUserLeft(channelID, userID, callID string)
func (p *Plugin) publishNotificationDismissed(callID, userID string)
// All use p.API.PublishWebSocketEvent with "custom_" prefix
```

---

## B-02: API Handler (`server/api.go`)

```go
// server/api.go — router setup and auth middleware (flat structure, not a separate package)
func (p *Plugin) initAPI()
// Registers all routes on p.router with auth middleware

func (p *Plugin) authRequired(next http.HandlerFunc) http.HandlerFunc
// Validates Mattermost-User-ID header; returns 401 if missing
```

### calls.go

```go
func (p *Plugin) handleCreateCall(w http.ResponseWriter, r *http.Request)
// POST /api/v1/calls
// Calls plugin.CreateCall(userID, channelID)
// Returns: {call_id, token, feature_flags}

func (p *Plugin) handleJoinCall(w http.ResponseWriter, r *http.Request)
// POST /api/v1/calls/{callId}/token
// Calls plugin.JoinCall(userID, callID)
// Returns: {token, feature_flags}

func (p *Plugin) handleLeaveCall(w http.ResponseWriter, r *http.Request)
// POST /api/v1/calls/{callId}/leave
// Calls plugin.LeaveCall(userID, callID)

func (p *Plugin) handleEndCall(w http.ResponseWriter, r *http.Request)
// DELETE /api/v1/calls/{callId}
// Calls plugin.EndCall(userID, callID)
```

### config.go

```go
func (p *Plugin) handleConfigStatus(w http.ResponseWriter, r *http.Request)
// GET /api/v1/config/status
// Returns: {configured: bool}

func (p *Plugin) handleAdminConfigStatus(w http.ResponseWriter, r *http.Request)
// GET /api/v1/config/admin-status (admin only)
// Returns: {org_id_configured, api_key_configured, org_id_source, api_key_source}
```

### mobile.go

```go
func (p *Plugin) handleRegisterVoIPToken(w http.ResponseWriter, r *http.Request)
// POST /api/v1/mobile/voip-token
// Body: {token: string}
// Stores VoIP token keyed by userID in KVStore

func (p *Plugin) handleDismissNotification(w http.ResponseWriter, r *http.Request)
// POST /api/v1/calls/{callId}/dismiss
// Calls plugin.publishNotificationDismissed(callID, userID)
```

### static.go

```go
func (p *Plugin) handleCallPage(w http.ResponseWriter, r *http.Request)
// GET /call
// Serves embedded call.html (no auth required)

func (p *Plugin) handleCallJS(w http.ResponseWriter, r *http.Request)
// GET /call.js
// Serves embedded call.js bundle (no auth required)

func (p *Plugin) handleWorkerJS(w http.ResponseWriter, r *http.Request)
// GET /worker.js
// Serves embedded worker.js (no auth required)
// Sets Content-Type: application/javascript
```

---

## B-03: Configuration (`server/configuration.go`)

```go
type configuration struct {
    CloudflareOrgID    string
    CloudflareAPIKey   string
    PollsEnabled       bool
    PluginsEnabled     bool
    ChatEnabled        bool
    ScreenShareEnabled bool
    ParticipantsEnabled bool
    RecordingEnabled   bool
    AITranscriptionEnabled bool
    WaitingRoomEnabled bool
    VideoEnabled       bool
    RaiseHandEnabled   bool
}

func (c *configuration) Clone() *configuration
// Returns deep copy for safe concurrent reads

func (p *Plugin) getConfiguration() *configuration
// Thread-safe read (sync.RWMutex)

func (p *Plugin) setConfiguration(c *configuration)
// Thread-safe write

func (p *Plugin) getEffectiveOrgID() string
// Returns RTK_ORG_ID env var if set, else CloudflareOrgID

func (p *Plugin) getEffectiveAPIKey() string
// Returns RTK_API_KEY env var if set, else CloudflareAPIKey

func (p *Plugin) getEffectiveFeatureFlags() FeatureFlags
// Returns all 10 flags with env var overrides applied
```

---

## B-04: RTK Client (`server/rtkclient/`)

### interface.go

```go
type RTKClient interface {
    CreateMeeting(preset string) (*Meeting, error)
    GenerateToken(meetingID, userID, preset string) (string, error)
    EndMeeting(meetingID string) error
}

type Meeting struct {
    ID string
}
```

### client.go

```go
type Client struct {
    orgID   string
    apiKey  string
    baseURL string
    http    *http.Client
}

func NewClient(orgID, apiKey string) *Client
// Sets baseURL = "https://api.realtime.cloudflare.com/v2"
// Configures HTTP Basic Auth header

func (c *Client) CreateMeeting(preset string) (*Meeting, error)
// POST {baseURL}/meetings with Basic Auth

func (c *Client) GenerateToken(meetingID, userID, preset string) (string, error)
// POST {baseURL}/meetings/{meetingID}/participants with Basic Auth
// Returns JWT token string

func (c *Client) EndMeeting(meetingID string) error
// DELETE {baseURL}/meetings/{meetingID} with Basic Auth
```

---

## B-05: KV Store (`server/store/kvstore/`)

```go
type KVStore interface {
    // Call session methods
    GetCallByChannel(channelID string) (*CallSession, error)
    GetCallByID(callID string) (*CallSession, error)
    SaveCall(session *CallSession) error
    UpdateCallParticipants(callID string, participants []string) error
    EndCall(callID string, endAt int64) error

    // Mobile VoIP token methods
    StoreVoIPToken(userID, token string) error
    GetVoIPToken(userID string) (string, error)
}

type CallSession struct {
    CallID       string   `json:"call_id"`
    ChannelID    string   `json:"channel_id"`
    PostID       string   `json:"post_id"`
    CreatorID    string   `json:"creator_id"`
    StartAt      int64    `json:"start_at"`
    EndAt        int64    `json:"end_at"`
    Participants []string `json:"participants"`
}
// KV key patterns:
//   call:channel:{channelID} -> CallSession JSON
//   call:id:{callID}         -> CallSession JSON
//   voip:{userID}            -> token string
```

---

## ~~B-06: Push Sender~~ — REMOVED

> Push notification subsystem removed. Mobile clients receive call notifications via WebSocket events.

---

## F-09: Calls Redux (`webapp/src/redux/calls_slice.ts`)

```typescript
interface CallsState {
    callsByChannel: Record<string, CallSession | null>  // channelID -> session
    myActiveCall: { callID: string; channelID: string; token: string } | null
    incomingCall: { callID: string; channelID: string; callerName: string } | null
}

// Actions
startCallSuccess(channelID, callID, token)
joinCallSuccess(channelID, callID, token)
leaveCall()
endCall(channelID)
callStarted(channelID, session)   // from WS event
callEnded(channelID, callID)      // from WS event
userJoined(callID, userID)        // from WS event
userLeft(callID, userID)          // from WS event
notificationDismissed(callID)     // from WS event
```

---

## F-10: Call Page (`webapp/src/call_page/CallPage.tsx`)

```typescript
function CallPage(): JSX.Element
// 1. Parse token from URL search params
// 2. Parse callID and channelID from token or URL params
// 3. Initialize RealtimeKitProvider with token and feature flag config
// 4. Register beforeunload handler:
//    fetch(`/plugins/{id}/api/v1/calls/${callID}/leave`, { method: 'POST', keepalive: true })
// 5. Set document.title = `Call in #${channelName}`
// 6. Render <RealtimeKitProvider> with RTK meeting config and feature flags
```
