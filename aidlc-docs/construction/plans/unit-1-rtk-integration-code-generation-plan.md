# Unit 1: RTK Integration — Code Generation Plan

## Unit Context

- **Stories**: US-005, US-009, US-013, US-015, US-022, US-025
- **Workspace root**: `/Users/sei.kondo/git/mattermost-plugin-rtk`
- **Project type**: Brownfield — Go plugin (module: `github.com/kondo97/mattermost-plugin-rtk`)
- **Mock tool**: `go.uber.org/mock` (mockgen)
- **Dependencies**: None (Unit 1 has no unit dependencies)

## Execution Checklist

### Step 1: Update Module Path and Plugin Identity
- [x] Modify `go.mod` — rename module from `github.com/mattermost/mattermost-plugin-starter-template` to `github.com/kondo97/mattermost-plugin-rtk`
- [x] Modify `server/plugin.go` — update import path
- [x] Modify `server/api.go` — update import path (no self-import; no change needed)
- [x] Modify `server/command/command.go` — update import path (no self-import; no change needed)
- [x] Modify `server/command/command_test.go` — update import path (no self-import; no change needed)
- [x] Modify `server/command/mocks/mock_commands.go` — update import path
- [x] Modify `server/store/kvstore/startertemplate.go` — update import path (no self-import; no change needed)
- [x] Modify `plugin.json` — update id, name, description, min_server_version to 10.11.0

### Step 2: Create Domain Models
- [x] Create `server/store/kvstore/models.go` — `CallSession` struct with all fields

### Step 3: Create Sentinel Errors
- [x] Create `server/errors.go` — `ErrCallAlreadyActive`, `ErrCallNotFound`, `ErrNotParticipant`, `ErrUnauthorized`, `ErrRTKNotConfigured`

### Step 4: Extend KVStore Interface
- [x] Modify `server/store/kvstore/kvstore.go` — add 10 call session methods to interface (keep `GetTemplateData`)

### Step 5: Implement KVStore Call Methods
- [x] Create `server/store/kvstore/calls.go` — implement all new call session methods on `Client` struct

### Step 6: Generate KVStore Mock
- [x] Create `server/store/kvstore/mocks/mock_kvstore.go` — mockgen for extended KVStore interface

### Step 7: Create RTKClient Interface and Types
- [x] Create `server/rtkclient/interface.go` — `RTKClient` interface, `Meeting` struct, `Token` struct

### Step 8: Implement RTKClient HTTP Client
- [x] Create `server/rtkclient/client.go` — `client` struct with 10s timeout, Basic Auth, HTTPS

### Step 9: Generate RTKClient Mock
- [x] Create `server/rtkclient/mocks/mock_rtkclient.go` — mockgen for RTKClient interface

### Step 10: Update Plugin Struct and OnActivate
- [x] Modify `server/plugin.go` — add `rtkClient rtkclient.RTKClient` field; update `OnActivate` to initialize rtkClient and change background job interval to 30s
- [x] Modify `server/configuration.go` — add `CloudflareOrgID`, `CloudflareAPIKey` fields and accessor methods

### Step 11: Implement Call Lifecycle Methods
- [x] Create `server/calls.go` — `CreateCall`, `JoinCall`, `LeaveCall`, `EndCall`, `endCallInternal`, `HeartbeatCall`

### Step 12: Implement Background Job Cleanup
- [x] Modify `server/job.go` — replace stub `runJob` with `CleanupStaleParticipants` logic

### Step 13: Write Unit Tests
- [x] Create `server/calls_test.go` — unit tests for all call lifecycle methods using mocks

### Step 14: Write Code Summary Documentation
- [x] Create `aidlc-docs/construction/unit-1-rtk-integration/code/code-summary.md`

---

## Story Traceability

| Story | Implemented In |
|---|---|
| US-005 (Start a Call) | Step 11 `CreateCall` |
| US-009 (Join a Call) | Step 11 `JoinCall` |
| US-013 (Leave by tab close) | Step 11 `LeaveCall` |
| US-015 (Host ends call) | Step 11 `EndCall` |
| US-022 (No duplicate call) | Step 11 `CreateCall` BR-01 check |
| US-025 (Auto-end on last leave) | Step 11 `LeaveCall` → `endCallInternal` |

## File Summary

| File | Action |
|---|---|
| `go.mod` | Modified |
| `plugin.json` | Modified |
| `server/plugin.go` | Modified |
| `server/configuration.go` | Modified |
| `server/job.go` | Modified |
| `server/errors.go` | Created |
| `server/calls.go` | Created |
| `server/calls_test.go` | Created |
| `server/rtkclient/interface.go` | Created |
| `server/rtkclient/client.go` | Created |
| `server/rtkclient/mocks/mock_rtkclient.go` | Created |
| `server/store/kvstore/kvstore.go` | Modified |
| `server/store/kvstore/models.go` | Created |
| `server/store/kvstore/calls.go` | Created |
| `server/store/kvstore/mocks/mock_kvstore.go` | Created |
| `server/command/mocks/mock_commands.go` | Modified (import path only) |
