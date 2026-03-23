package push

import "github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"

// PushSender dispatches mobile push notifications for call events.
// Implementations must be safe for concurrent use.
//
//go:generate mockgen -destination=mocks/mock_push.go -package=mocks github.com/kondo97/mattermost-plugin-rtk/server/push PushSender
type PushSender interface {
	// SendIncomingCall sends a "message"/"calls" push notification to all
	// DM/GM channel members (up to 8) except the call creator.
	// Returns an error if channel/user/member lookup fails; individual
	// SendPushNotification failures are logged and do not cause an error return.
	SendIncomingCall(session *kvstore.CallSession) error

	// SendCallEnded sends a "clear"/"calls_ended" push notification to dismiss
	// any incoming call UI on DM/GM channel members (up to 8) except the call creator.
	// Returns an error if channel/user/member lookup fails; individual
	// SendPushNotification failures are logged and do not cause an error return.
	SendCallEnded(session *kvstore.CallSession) error
}
