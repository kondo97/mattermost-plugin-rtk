# Known Constraints

This document records known technical constraints and limitations that have been investigated and accepted. Each entry includes the investigation result and the rationale for acceptance.

---

## KC-001: JoinCall — KVStore failure after RTK token generation leaves orphaned token

**Status**: Accepted (no fix available)
**Affected code**: `server/calls.go` — `JoinCall`
**Discovered**: 2026-03-22

### Description

`JoinCall` performs two external operations in sequence:

1. `rtkClient.GenerateToken()` — registers the participant on Cloudflare RTK and issues a JWT
2. `kvStore.UpdateCallParticipants()` — records the participant in Mattermost KVStore and emits a WebSocket event

If step 2 fails after step 1 succeeds, the user holds a valid RTK token but is absent from the Mattermost participant list. The WebSocket event is not emitted.

### Investigation: Can the RTK token be invalidated?

The Cloudflare RTK API provides `DELETE /apps/{app_id}/meetings/{meeting_id}/participants/{participant_id}`, but this only **prevents future joins** — it does not revoke an already-issued JWT. There is no token invalidation endpoint. Issued tokens remain valid until their natural JWT expiry.

Therefore, atomic rollback is not possible.

### Why the current order (GenerateToken → UpdateCallParticipants) is correct

Reversing the order (UpdateCallParticipants → GenerateToken) would be worse: if GenerateToken fails, a ghost entry remains in the participant list. Ghost participants prevent auto-end logic (`len(participants) == 0`) from triggering, causing the call to never terminate automatically.

The current order ensures that the critical failure mode (ghost participant) cannot occur. The orphaned-token failure is self-healing: the JWT expires naturally, and no persistent state is corrupted.

### Practical impact

- Likelihood: very low (Mattermost KVStore failures are rare)
- User impact: the affected user can enter the RTK meeting but is not visible in the Mattermost participant list until they leave and rejoin
- Auto-end safety: not affected (orphaned token does not create a ghost participant in KVStore)

### Acceptance rationale

No atomic solution exists without RTK token invalidation support. The failure mode is self-healing and low-likelihood. Accepted as a known limitation.

---
