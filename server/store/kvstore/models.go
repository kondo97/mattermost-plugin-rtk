package kvstore

// CallSession represents an active or ended call within a Mattermost channel.
type CallSession struct {
	// ID is the unique call identifier (UUID).
	ID string `json:"id"`
	// ChannelID is the Mattermost channel the call belongs to.
	ChannelID string `json:"channel_id"`
	// CreatorID is the UserID of the call host.
	CreatorID string `json:"creator_id"`
	// MeetingID is the Cloudflare RTK meeting identifier.
	MeetingID string `json:"meeting_id"`
	// Participants holds the current participant UserIDs.
	Participants []string `json:"participants"`
	// StartAt is the Unix timestamp (ms) when the call was created.
	StartAt int64 `json:"start_at"`
	// EndAt is the Unix timestamp (ms) when the call ended; 0 means active.
	EndAt int64 `json:"end_at"`
	// PostID is the ID of the custom_cf_call post in the channel.
	PostID string `json:"post_id"`
	// CleanupFailCount tracks how many consecutive cleanup cycles found this
	// meeting missing from the RTK API. The call is force-ended only after
	// reaching the threshold, to guard against transient API 404s.
	CleanupFailCount int `json:"cleanup_fail_count,omitempty"`
}
