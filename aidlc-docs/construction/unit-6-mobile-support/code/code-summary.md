# Unit 6: Mobile Support — Code Summary

## Files Created

| File | Description |
|---|---|
| `server/push/interface.go` | `PushSender` interface with `SendIncomingCall` and `SendCallEnded` |
| `server/push/push.go` | `Sender` struct + `sendToMembers` helper + both method implementations |
| `server/push/mocks/mock_push.go` | mockgen-generated `MockPushSender` (go.uber.org/mock/gomock) |
| `server/push/push_test.go` | 8 unit tests using `plugintest.API` |

## Files Modified

| File | Changes |
|---|---|
| `server/plugin.go` | Added `pushSender push.PushSender` field; `push.NewSender(p.API)` in `OnActivate` |
| `server/calls.go` | `CreateCall`: call `SendIncomingCall` (best-effort, nil-guarded); `endCallInternal`: call `SendCallEnded` (best-effort, nil-guarded) |
| `server/calls_test.go` | Added `newTestPluginWithPush` helper; 3 new push integration tests |

## Key Implementation Notes

- **Best-effort**: nil guards (`if p.pushSender != nil`) allow existing tests to run without a push mock
- **DM/GM only**: `channel.Type` check in `sendToMembers` — non-DM/GM channels return nil immediately
- **Max 8 members**: `GetChannelMembers(channelID, 0, 8)` — single call, no pagination
- **Sequential dispatch**: simple for-loop, no goroutines
- **Per-member failure**: logged as warning, loop continues
- **US-019**: fully satisfied by existing `handleToken` endpoint in Unit 2 — no new code required

## Test Coverage

### server/push package (push_test.go)
- `TestSendIncomingCall_DMChannel_Success` — sends to non-caller members, skips caller
- `TestSendIncomingCall_NonDMChannel_Skipped` — open channel returns nil without API calls
- `TestSendIncomingCall_GetChannelFails` — returns error
- `TestSendIncomingCall_SendPushFails_ContinuesLoop` — per-member failure does not abort
- `TestSendCallEnded_DMChannel_Success` — sends clear notification
- `TestSendCallEnded_NonDMChannel_Skipped` — private channel skipped
- `TestSendCallEnded_GMChannel_Success` — GM channel supported

### server package (calls_test.go additions)
- `TestCreateCall_InvokesPushSender` — push is called on successful CreateCall
- `TestCreateCall_PushSenderError_CallSucceeds` — push failure does not fail CreateCall
- `TestEndCall_InvokesPushSender` — push is called on successful EndCall

## Build & Test Results

```
ok  github.com/kondo97/mattermost-plugin-rtk/server       0.539s
ok  github.com/kondo97/mattermost-plugin-rtk/server/push  0.294s
```

All tests pass. Build clean.
