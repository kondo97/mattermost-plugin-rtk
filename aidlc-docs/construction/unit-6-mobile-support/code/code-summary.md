# Unit 6: Mobile Support — Code Summary

> **Status**: REMOVED. The push notification subsystem was implemented and subsequently
> deleted. Mobile call notifications are now handled through WebSocket events directly.

## Removal Rationale

The `server/push/` package and all related integration code were removed because
the mobile app handles call notifications through the same WebSocket events
(`custom_com.kondo97.mattermost-plugin-rtk_call_started`, `custom_com.kondo97.mattermost-plugin-rtk_call_ended`) used by the desktop client.
Dedicated push notifications were no longer needed.

## Removed Files

| File | Original Description |
|---|---|
| `server/push/interface.go` | `PushSender` interface with `SendIncomingCall` and `SendCallEnded` |
| `server/push/push.go` | `Sender` struct + `sendToMembers` helper + both method implementations |
| `server/push/mocks/mock_push.go` | mockgen-generated `MockPushSender` |
| `server/push/push_test.go` | 8 unit tests |

## Reverted Modifications

| File | Reverted Change |
|---|---|
| `server/plugin.go` | Removed `pushSender push.PushSender` field and `push.NewSender(p.API)` initialization |
| `server/calls.go` | Removed `SendIncomingCall` / `SendCallEnded` calls from `CreateCall` / `endCallInternal` |
| `server/calls_test.go` | Removed `newTestPluginWithPush` helper and 3 push integration tests |

## Current Mobile Support

Mobile clients receive call events through existing WebSocket channels:
- `custom_com.kondo97.mattermost-plugin-rtk_call_started` — triggers incoming call UI
- `custom_com.kondo97.mattermost-plugin-rtk_call_ended` — dismisses incoming call UI
- `custom_com.kondo97.mattermost-plugin-rtk_user_joined` / `custom_com.kondo97.mattermost-plugin-rtk_user_left` — participant updates

No dedicated mobile-specific server code is required.
