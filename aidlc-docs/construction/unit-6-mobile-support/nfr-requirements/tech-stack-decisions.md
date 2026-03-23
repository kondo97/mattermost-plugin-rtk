# Unit 6: Mobile Support — Tech Stack Decisions

## Push Notification Delivery

| Decision | Choice | Rationale |
|---|---|---|
| Push API | `p.API.SendPushNotification(notification, userID)` | Mattermost Plugin API — platform routing (APNs/FCM) handled transparently by Mattermost push proxy |
| Channel member fetch | `p.API.GetChannelMembers(channelID, 0, 8)` | Single call, max 8 members; aligns with Mattermost Calls plugin |
| Concurrency | Sequential | DM/GM channels have ≤ 8 members; simplicity outweighs parallelism benefit |
| Error strategy | Best-effort (log warn, continue) | Aligns with Mattermost Calls plugin; push must not block call operations |

## Mock Generation

| Decision | Choice | Rationale |
|---|---|---|
| Mock tool | `mockery` | Consistent with `server/rtkclient/mocks/` and `server/store/kvstore/mocks/` |
| Mock output | `server/push/mocks/mock_push.go` | Standard location pattern |
| Interface | `PushSender` in `server/push/interface.go` | Consistent with `RTKClient` interface pattern |

## Testing

| Decision | Choice | Rationale |
|---|---|---|
| API mock | `plugintest.API` from `github.com/mattermost/mattermost/server/public/plugin/plugintest` | Already used in `calls_test.go`; no new test dependency |
| Test package | `package push` (white-box) | Direct access to Sender internals |

## No New Dependencies

Unit 6 introduces no new Go module dependencies. All required packages are already
available in the existing `go.mod`:
- `github.com/mattermost/mattermost/server/public/model` — `PushNotification`, `ChannelTypeDirect`, etc.
- `github.com/mattermost/mattermost/server/public/plugin` — `plugin.API`
- `github.com/mattermost/mattermost/server/public/plugin/plugintest` — test mock
