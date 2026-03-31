# Unit 6: Mobile Support — Business Rules

> **Updated 2026-03-31**: The `server/push/` package has been REMOVED. Mobile clients receive call notifications via WebSocket events (`custom_cf_call_started`, `custom_cf_call_ended`) instead of push notifications. All rules below are no longer enforced. This document is retained for historical reference.

## Push Notification Rules (REMOVED)

### BR-P01: Both SendIncomingCall and SendCallEnded are best-effort

Both methods return an error on failure, but callers log a warning and continue.
Neither `CreateCall` nor `endCallInternal` is blocked by push delivery failures.

**Rationale**: Aligns with Mattermost Calls plugin pattern. Push notification delivery
must not affect call creation or termination reliability.

---

### BR-P02: Push is limited to DM and GM channels only

`SendIncomingCall` and `SendCallEnded` check `channel.Type`:
- `model.ChannelTypeDirect` — send
- `model.ChannelTypeGroup` — send
- All other types — return nil immediately (no push sent)

**Rationale**: Aligns with Mattermost Calls plugin behavior.

---

### BR-P03: Recipient scope — up to 8 channel members except the caller

`GetChannelMembers(channelID, 0, 8)` is called once (no pagination).
Members where `member.UserId == session.CreatorID` are skipped.

**Rationale**: Aligns with Mattermost Calls plugin limit (8 members).
Platform-level filtering (device availability, online status) is handled by the
Mattermost push proxy.

---

### BR-P04: Per-member send failure does not abort the loop

If `SendPushNotification` fails for one member, a warning is logged and the loop
continues to the next member. The method returns nil after all members are processed.

**Rationale**: Consistent with best-effort delivery model (BR-P01).

---

### BR-P05: team_id is empty string for DM/GM channels

DM and GM channels have `channel.TeamId == ""`. The empty string is passed directly
in the push notification payload.

**Rationale**: The Mattermost push proxy handles routing for DM/GM channels without
requiring a team ID.

---

### BR-P06: sender_name uses Username

The `SenderName` field in the push notification is set to `caller.Username`.

---

### BR-P07: channel_name fallback

`ChannelName` is set to `channel.DisplayName`. If `channel.DisplayName` is empty
(e.g., some DM channels), falls back to `channel.Name`.

---

### BR-P08: Interface and implementation are in separate files

The `PushSender` interface is defined in `server/push/interface.go`.
The `Sender` struct and methods are in `server/push/push.go`.

**Rationale**: Consistent with `server/rtkclient` package pattern.
Enables mock generation via `mockgen` (`go.uber.org/mock/mockgen`).

---

### BR-P09: pushSender is always initialized on OnActivate

`push.NewSender(p.API)` is called unconditionally during `OnActivate`, regardless
of RTK credential configuration. Push notifications use only the Mattermost API.

---

### BR-P10: US-019 requires no new server-side logic

A mobile user joining a call from a push notification calls
`POST /api/v1/calls/{callId}/token` (Unit 2). No additional endpoint or logic
is required for Unit 6.

---

## Security Compliance Summary

| Rule | Status | Notes |
|---|---|---|
| SECURITY-01 | N/A | No new data stores |
| SECURITY-02 | N/A | No new network intermediaries |
| SECURITY-03 | Compliant | Errors logged via `LogWarn`; no tokens or PII in log messages |
| SECURITY-04 | N/A | No new HTML-serving endpoints |
| SECURITY-05 | Compliant | No user-supplied input in push sender; inputs are internal session data |
| SECURITY-06 | N/A | No IAM policies |
| SECURITY-07 | N/A | No network configuration |
| SECURITY-08 | Compliant | Push targets derived from channel membership; no user-controlled routing |
| SECURITY-09 | N/A | No new deployments or default credentials |
| SECURITY-10 | N/A | No new dependencies |
| SECURITY-11 | Compliant | Push sender is an isolated module; DM/GM-only restriction limits misuse scope |
| SECURITY-12 | N/A | No authentication logic |
| SECURITY-13 | N/A | No deserialization of untrusted data |
| SECURITY-14 | N/A | No new alerting infrastructure |
| SECURITY-15 | Compliant | All external API calls have explicit error handling; fail-safe per BR-P01/BR-P04 |
