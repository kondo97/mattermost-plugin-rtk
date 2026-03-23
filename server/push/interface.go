package push

import "github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"

// PushSender dispatches mobile push notifications for call events.
// Implementations must be safe for concurrent use.
//
//go:generate mockgen -destination=mocks/mock_push.go -package=mocks github.com/kondo97/mattermost-plugin-rtk/server/push PushSender
type PushSender interface {
	// SendIncomingCall sends a "message"/"calls" push notification to all
	// DM/GM channel members (up to 8) except the call creator.
	// Returns an error if required metadata (e.g., channel, user, or channel members)
	// cannot be retrieved. Per-recipient push send failures are treated as best-effort:
	// they are logged and skipped, but do not cause a non-nil error return.
	SendIncomingCall(session *kvstore.CallSession) error

	// SendCallEnded sends a "clear"/"calls_ended" push notification to dismiss
	// any incoming call UI on DM/GM channel members (up to 8) except the call creator.
	// Returns an error if required metadata (e.g., channel, user, or channel members)
	// cannot be retrieved. Per-recipient push send failures are treated as best-effort:
	// they are logged and skipped, but do not cause a non-nil error return.
	SendCallEnded(session *kvstore.CallSession) error
}
