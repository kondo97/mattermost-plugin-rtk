# Unit 6: Mobile Support — Code Generation Plan

## Unit Context

**Stories**: US-018 (Primary), US-019 (Primary — satisfied by Unit 2), US-020 (Supporting — satisfied by Unit 2)
**Dependencies**: Unit 1 (CallSession), Unit 2 (no new deps)
**New package**: `server/push/`
**Modified files**: `server/plugin.go`, `server/calls.go`, `server/calls_test.go`

## Key Design Decisions

- Push is **best-effort** — errors logged, callers continue
- **DM/GM only** — `model.ChannelTypeDirect` and `model.ChannelTypeGroup`
- **Max 8 members** — `GetChannelMembers(channelID, 0, 8)`
- **Sequential dispatch** — no goroutines
- Internal `sendToMembers` helper shared by both public methods
- Mock uses `go.uber.org/mock/gomock` (consistent with existing mocks)

---

## Steps

- [x] Step 1: Create `server/push/interface.go` — PushSender interface
- [x] Step 2: Create `server/push/push.go` — Sender struct + sendToMembers + SendIncomingCall + SendCallEnded
- [x] Step 3: Create `server/push/mocks/mock_push.go` — mockgen-generated MockPushSender
- [x] Step 4: Create `server/push/push_test.go` — unit tests using plugintest.API
- [x] Step 5: Modify `server/plugin.go` — add pushSender field + OnActivate initialization
- [x] Step 6: Modify `server/calls.go` — integrate push calls in CreateCall and endCallInternal
- [x] Step 7: Modify `server/calls_test.go` — add MockPushSender expectations to existing tests
- [x] Step 8: Create `aidlc-docs/construction/unit-6-mobile-support/code/code-summary.md`

---

## Story Traceability

| Step | Story |
|---|---|
| Steps 1–4 | US-018 (push sender implementation + tests) |
| Steps 5–7 | US-018 (integration into call lifecycle) |
| Step 5–7 | US-019 (no new code — satisfied by existing handleToken endpoint) |

---

## File Locations (Application Code in Workspace Root)

| Step | File | Action |
|---|---|---|
| 1 | `server/push/interface.go` | Create |
| 2 | `server/push/push.go` | Create |
| 3 | `server/push/mocks/mock_push.go` | Create |
| 4 | `server/push/push_test.go` | Create |
| 5 | `server/plugin.go` | Modify |
| 6 | `server/calls.go` | Modify |
| 7 | `server/calls_test.go` | Modify |
| 8 | `aidlc-docs/construction/unit-6-mobile-support/code/code-summary.md` | Create |
