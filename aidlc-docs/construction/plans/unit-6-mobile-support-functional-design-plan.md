# Unit 6: Mobile Support — Functional Design Plan

## Objective

Design the business logic for the mobile push notification subsystem:
- `server/push/push.go` — `Sender` interface + implementation with `SendIncomingCall()` and `SendCallEnded()`
- `server/plugin.go` — initialize Sender on `OnActivate`, invoke from `CreateCall` / `endCallInternal`

## Stories Covered

- **US-018** (Primary): Receive Incoming Call Push Notification
- **US-019** (Primary): Join a Call from Push Notification
- **US-020** (Supporting): Dismiss Incoming Call Notification (already implemented in Unit 2; no changes here)

## Execution Steps

- [x] Step 1: Analyze unit definition and assigned stories
- [x] Step 2: Identify clarifying questions
- [x] Step 3: Collect answers from user
- [x] Step 4: Generate functional design artifacts
  - [x] `domain-entities.md`
  - [x] `business-logic-model.md`
  - [x] `business-rules.md`

---

## Clarifying Questions

Please fill in the `[Answer]:` tags below and return the file.

---

### Q1: Push notification delivery — best-effort or blocking?

[Answer]: A — Best-effort (aligns with Mattermost Calls plugin; push errors logged as warnings, do not fail the operation)

---

### Q2: Member enumeration for push — scope

[Answer]: A — All channel members except the caller (Mattermost Calls plugin spec)

---

### Q3: Large channel handling — pagination

[Answer]: C — Single call `GetChannelMembers(page=0, perPage=8)`; no pagination (aligns with Mattermost Calls plugin limit of 8 members)

---

### Q4: `team_id` for DM/GM channels

[Answer]: A — Empty string ""

---

### Q5: `sender_name` field value

[Answer]: C — user.Username

---

### Q6: `SendCallEnded` target scope

[Answer]: A — Same scope as SendIncomingCall (all channel members except caller)

---

### Q7: Push sender interface location

[Answer]: B — server/push/interface.go (separate file)
