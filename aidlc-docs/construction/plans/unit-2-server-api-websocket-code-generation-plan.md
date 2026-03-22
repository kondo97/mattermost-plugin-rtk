# Unit 2: Server API & WebSocket — Code Generation Plan

## Unit Context

- **Stories (Primary)**: US-020, US-021
- **Stories (Supporting)**: US-005, US-007, US-008, US-009, US-011, US-012, US-013, US-015, US-016, US-019, US-022, US-023, US-024, US-025
- **Workspace root**: `/Users/sei.kondo/git/mattermost-plugin-rtk`
- **Project type**: Brownfield — Go plugin (`package main` in `server/`)

## Key Design Decisions

- **Leave detection**: RTK webhook (`meeting.participantLeft` / `meeting.ended`) — heartbeat polling廃止
- **Webhook registration**: `OnActivate` 自動登録 (best-effort)
- **All handlers**: methods on `*Plugin` in `server/` package main
- **callMu**: Plugin メソッド内でロック取得（JoinCall/LeaveCall/EndCall/CreateCall）
- **Static files**: `//go:embed` — placeholder files now, replaced by Unit 4 build
- **No heartbeat endpoint**: replaced by RTK webhook

## Execution Checklist

### Part A: KVStore Extensions

- [x] Step 1: Extend KVStore interface — add `GetCallByMeetingID`, `StoreWebhookID`, `GetWebhookID`, `StoreWebhookSecret`, `GetWebhookSecret`
- [x] Step 2: Implement KVStore methods in `server/store/kvstore/calls.go` — `GetCallByMeetingID`, `StoreWebhookID`, `GetWebhookID`, `StoreWebhookSecret`, `GetWebhookSecret`; update `SaveCall` to also write `call:meeting:{meetingID}` key
- [x] Step 3: Regenerate KVStore mock — update `server/store/kvstore/mocks/mock_kvstore.go`

### Part B: RTKClient Extensions

- [x] Step 4: Extend RTKClient interface — add `RegisterWebhook(url string, events []string) (id, secret string, err error)` and `DeleteWebhook(webhookID string) error`
- [x] Step 5: Implement RTKClient webhook methods in `server/rtkclient/client.go`
- [x] Step 6: Regenerate RTKClient mock — update `server/rtkclient/mocks/mock_rtkclient.go`

### Part C: Plugin Struct & OnActivate

- [x] Step 7: Modify `server/plugin.go` — add `callMu sync.Mutex` field; update `OnActivate` to auto-register RTK webhook; update `OnConfigurationChange` to re-register webhook on credential change
- [x] Step 8: Modify `server/calls.go` — add `callMu` lock/unlock inside `CreateCall`, `JoinCall`, `LeaveCall`, `EndCall`

### Part D: HTTP Router

- [x] Step 9: Modify `server/api.go` — replace `HelloWorld`, add webhook route and all Unit 2 routes, add `writeError` helper; keep `MattermostAuthorizationRequired`

### Part E: HTTP Handlers

- [x] Step 10: Create `server/api_calls.go` — `handleCreateCall`, `handleJoinCall`, `handleLeaveCall`, `handleEndCall`
- [x] Step 11: Create `server/api_config.go` — `handleConfigStatus`, `handleAdminConfigStatus`
- [x] Step 12: Create `server/api_mobile.go` — `handleDismiss`
- [x] Step 13: Create `server/api_static.go` — static handlers with `//go:embed`
- [x] Step 14: Create `server/api_webhook.go` — `handleRTKWebhook` (signature verification + event dispatch)
- [x] Step 15: Create placeholder assets — `server/assets/call.html`, `server/assets/call.js`, `server/assets/worker.js`

### Part F: Tests

- [x] Step 16: Create `server/api_calls_test.go` — handler tests for CreateCall, JoinCall, LeaveCall, EndCall
- [x] Step 17: Create `server/api_config_test.go` — handler tests for ConfigStatus, AdminConfigStatus
- [x] Step 18: Create `server/api_mobile_test.go` — handler test for Dismiss
- [x] Step 19: Create `server/api_webhook_test.go` — webhook handler tests (valid/invalid signature, participantLeft, ended, unknown event)

### Part G: Documentation

- [x] Step 20: Create `aidlc-docs/construction/unit-2-server-api-websocket/code/code-summary.md`

---

## File Summary

| File | Action | Description |
|---|---|---|
| `server/store/kvstore/kvstore.go` | Modified | Add 5 webhook/meetingID methods to interface |
| `server/store/kvstore/calls.go` | Modified | Implement new methods; update SaveCall for meetingID key |
| `server/store/kvstore/mocks/mock_kvstore.go` | Regenerated | Updated mock |
| `server/rtkclient/interface.go` | Modified | Add `RegisterWebhook`, `DeleteWebhook` |
| `server/rtkclient/client.go` | Modified | Implement webhook methods |
| `server/rtkclient/mocks/mock_rtkclient.go` | Regenerated | Updated mock |
| `server/plugin.go` | Modified | Add `callMu`; webhook registration in OnActivate/OnConfigurationChange |
| `server/calls.go` | Modified | Add `callMu` lock to CreateCall, JoinCall, LeaveCall, EndCall |
| `server/api.go` | Modified | New initRouter; writeError; webhook + all routes |
| `server/api_calls.go` | Created | Call handlers |
| `server/api_config.go` | Created | Config handlers |
| `server/api_mobile.go` | Created | Dismiss handler |
| `server/api_static.go` | Created | Static file handlers |
| `server/api_webhook.go` | Created | RTK webhook receiver |
| `server/assets/call.html` | Created | Placeholder |
| `server/assets/call.js` | Created | Placeholder |
| `server/assets/worker.js` | Created | Placeholder |
| `server/api_calls_test.go` | Created | Call handler tests |
| `server/api_config_test.go` | Created | Config handler tests |
| `server/api_mobile_test.go` | Created | Dismiss handler test |
| `server/api_webhook_test.go` | Created | Webhook handler tests |

---

## Story Traceability

| Story | Implemented In |
|---|---|
| US-005 (Start a Call) | Step 10 `handleCreateCall` |
| US-009 (Join a Call) | Step 10 `handleJoinCall` |
| US-013 (Leave by tab close) | Step 14 `handleRTKWebhook` → `meeting.participantLeft` |
| US-015 (Host ends call) | Step 10 `handleEndCall` |
| US-020 (Dismiss notification) | Step 12 `handleDismiss` |
| US-021 (Web Worker served) | Step 13 `serveWorkerJS` |
| US-022 (No duplicate call) | Step 10 → 409 mapping |
| US-025 (Auto-end) | Step 14 → `meeting.ended` webhook → `endCallInternal` |
