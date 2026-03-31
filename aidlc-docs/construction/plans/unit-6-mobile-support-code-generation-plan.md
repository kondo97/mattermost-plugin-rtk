# Unit 6: Mobile Support — Code Generation Plan

> **Updated 2026-03-31**: The `server/push/` package has been **REMOVED**. Mobile clients receive call notifications via WebSocket events (`custom_cf_call_started`, `custom_cf_call_ended`) instead of push notifications. Steps 1-4 below are historical and no longer apply. Steps 5-7 were simplified (no push integration needed).

## Unit Context

**Stories**: US-018 (Primary — now satisfied by WebSocket events), US-019 (Primary — satisfied by Unit 2), US-020 (Supporting — satisfied by Unit 2)
**Dependencies**: Unit 1 (CallSession), Unit 2 (no new deps)
**~~New package~~**: ~~`server/push/`~~ — REMOVED
**Modified files**: `server/plugin.go`, `server/calls.go`, `server/calls_test.go`

## Key Design Decisions

- ~~Push is **best-effort** — errors logged, callers continue~~ — REMOVED: no push subsystem
- Mobile clients rely on existing WebSocket event broadcast for call notifications
- ~~**DM/GM only** — `model.ChannelTypeDirect` and `model.ChannelTypeGroup`~~ — N/A
- ~~**Max 8 members** — `GetChannelMembers(channelID, 0, 8)`~~ — N/A
- ~~**Sequential dispatch** — no goroutines~~ — N/A
- ~~Internal `sendToMembers` helper shared by both public methods~~ — N/A
- ~~Mock uses `go.uber.org/mock/gomock` (consistent with existing mocks)~~ — N/A

---

## Steps

- ~~[x] Step 1: Create `server/push/interface.go` — PushSender interface~~ — REMOVED
- ~~[x] Step 2: Create `server/push/push.go` — Sender struct + sendToMembers + SendIncomingCall + SendCallEnded~~ — REMOVED
- ~~[x] Step 3: Create `server/push/mocks/mock_push.go` — mockgen-generated MockPushSender~~ — REMOVED
- ~~[x] Step 4: Create `server/push/push_test.go` — unit tests using plugintest.API~~ — REMOVED
- [x] Step 5: Modify `server/plugin.go` — ~~add pushSender field + OnActivate initialization~~ simplified (no push integration)
- [x] Step 6: Modify `server/calls.go` — ~~integrate push calls in CreateCall and endCallInternal~~ simplified (no push integration)
- [x] Step 7: Modify `server/calls_test.go` — ~~add MockPushSender expectations to existing tests~~ simplified (no push mocks)
- [x] Step 8: Create `aidlc-docs/construction/unit-6-mobile-support/code/code-summary.md`

---

## Story Traceability

| Step | Story |
|---|---|
| ~~Steps 1–4~~ | ~~US-018 (push sender implementation + tests)~~ — REMOVED: `server/push/` deleted |
| Steps 5–7 | US-018 (now satisfied by WebSocket events, no push integration) |
| Step 5–7 | US-019 (no new code — satisfied by existing handleToken endpoint) |

---

## File Locations (Application Code in Workspace Root)

| Step | File | Action |
|---|---|---|
| ~~1~~ | ~~`server/push/interface.go`~~ | ~~Create~~ — REMOVED |
| ~~2~~ | ~~`server/push/push.go`~~ | ~~Create~~ — REMOVED |
| ~~3~~ | ~~`server/push/mocks/mock_push.go`~~ | ~~Create~~ — REMOVED |
| ~~4~~ | ~~`server/push/push_test.go`~~ | ~~Create~~ — REMOVED |
| 5 | `server/plugin.go` | Modify |
| 6 | `server/calls.go` | Modify |
| 7 | `server/calls_test.go` | Modify |
| 8 | `aidlc-docs/construction/unit-6-mobile-support/code/code-summary.md` | Create |
