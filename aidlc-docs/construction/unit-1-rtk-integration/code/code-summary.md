# Unit 1: RTK Integration — Code Summary

## Files Modified

| File | Action | Description |
|---|---|---|
| `go.mod` | Modified | Renamed module from `github.com/mattermost/mattermost-plugin-starter-template` to `github.com/kondo97/mattermost-plugin-rtk` |
| `plugin.json` | Modified | Updated id, name, description, min_server_version to 10.11.0; added Cloudflare settings schema |
| `server/plugin.go` | Modified | Added `rtkClient` field, updated `OnActivate` to initialize RTK client and set 30s job interval; renamed `kvstore` → `kvStore` |
| `server/configuration.go` | Modified | Added `CloudflareOrgID`, `CloudflareAPIKey` fields and `GetEffectiveOrgID`/`GetEffectiveAPIKey` accessors |
| `server/job.go` | Modified | Replaced stub `runJob` with `CleanupStaleParticipants` call |
| `server/store/kvstore/kvstore.go` | Modified | Extended `KVStore` interface with 10 call session methods |
| `server/command/mocks/mock_commands.go` | Modified | Updated module import path |

## Files Created

| File | Description |
|---|---|
| `server/errors.go` | Sentinel errors: `ErrCallAlreadyActive`, `ErrCallNotFound`, `ErrNotParticipant`, `ErrUnauthorized`, `ErrRTKNotConfigured` |
| `server/calls.go` | Call lifecycle methods: `CreateCall`, `JoinCall`, `LeaveCall`, `EndCall`, `endCallInternal`, `HeartbeatCall`; helpers: `containsUser`, `removeUser`, `nowMs` |
| `server/calls_test.go` | Unit tests for all call lifecycle methods using gomock (KVStore/RTKClient) and plugintest.API |
| `server/store/kvstore/models.go` | `CallSession` struct with all domain fields |
| `server/store/kvstore/calls.go` | KVStore implementation of call session methods: CRUD, heartbeat, VoIP token, active channel index |
| `server/store/kvstore/mocks/mock_kvstore.go` | gomock-generated mock for `KVStore` interface |
| `server/rtkclient/interface.go` | `RTKClient` interface, `Meeting` struct, `Token` struct |
| `server/rtkclient/client.go` | HTTP implementation using Basic Auth, 10s timeout, base URL `https://api.realtime.cloudflare.com/v2` |
| `server/rtkclient/mocks/mock_rtkclient.go` | gomock-generated mock for `RTKClient` interface |

## Story Traceability

| Story | Implemented In |
|---|---|
| US-005 (Start a Call) | `server/calls.go` → `CreateCall` |
| US-009 (Join a Call) | `server/calls.go` → `JoinCall` |
| US-013 (Leave by tab close) | `server/calls.go` → `LeaveCall` |
| US-015 (Host ends call) | `server/calls.go` → `EndCall` |
| US-022 (No duplicate call) | `server/calls.go` → `CreateCall` BR-01 check |
| US-025 (Auto-end on last leave) | `server/calls.go` → `LeaveCall` → `endCallInternal` |

## Key Design Decisions

- **Active call index**: `GetAllActiveCalls` uses a maintained index key (`calls:index:active_channels`) to avoid full KV scan; updated by `SaveCall` and `EndCall`.
- **Dual-key storage**: Each `CallSession` is stored under both `call:channel:{channelID}` and `call:id:{callID}` for O(1) lookup by either key.
- **Initial heartbeat on join** (BR-09a): `JoinCall` sets a heartbeat immediately after participant is added to prevent race with the 30s cleanup job.
- **Best-effort operations**: `CreatePost`, `UpdatePost`, `EndMeeting` failures are logged as warnings but do not abort the primary flow (REL-02, REL-03).
- **Nil RTK client guard**: All call lifecycle methods check `p.rtkClient == nil` and return `ErrRTKNotConfigured` if credentials are not configured.
