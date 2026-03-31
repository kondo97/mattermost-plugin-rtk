# Unit 6: Mobile Support — Business Logic Model

> **Status**: REMOVED. This document describes the original push notification design
> that was implemented and subsequently deleted. It is retained for historical reference.

## Removal Rationale

Push notifications were removed because the mobile app handles call notifications
through the same WebSocket events used by the desktop client. The dedicated push
notification subsystem (`server/push/`) was no longer needed.

## Current Mobile Support

Mobile clients receive call events through existing WebSocket channels:
- `custom_cf_call_started` — triggers incoming call UI
- `custom_cf_call_ended` — dismisses incoming call UI
- `custom_cf_user_joined` / `custom_cf_user_left` — participant updates

Joining a call from mobile uses the existing `POST /api/v1/calls/{callId}/token`
endpoint (implemented in Unit 2). No dedicated mobile-specific server code is required.

---

## Original Design (Archived)

The following describes the push notification system as originally designed and implemented,
before its removal. This section is kept for historical context only.

### Overview

Two business operations defined Unit 6:
1. **SendIncomingCall** — dispatch call-started push to DM/GM channel members (max 8, except caller)
2. **SendCallEnded** — dispatch call-ended clear push to same scope

Both were invoked from Plugin Core (`calls.go`) after the corresponding call state change.
Both were **best-effort** — errors were logged but did not block the call operation.

### Integration Points (Removed)

- `CreateCall`: called `SendIncomingCall` after `PublishWebSocketEvent`
- `endCallInternal`: called `SendCallEnded` after `PublishWebSocketEvent`
- `OnActivate`: initialized `pushSender = push.NewSender(p.API)`
