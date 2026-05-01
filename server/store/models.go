package store

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
	// Participants holds the current participant UserIDs, ordered by join time.
	// Persisted in the rtk_call_participants table; on read it is populated by the
	// store, on CreateCallSession the slice is used as the initial participant set.
	Participants []string `json:"participants"`
	// CreateAt is the Unix timestamp (ms) when the call was created.
	CreateAt int64 `json:"create_at"`
	// UpdateAt is the Unix timestamp (ms) when the call was last updated.
	UpdateAt int64 `json:"update_at"`
	// EndAt is the Unix timestamp (ms) when the call ended; 0 means active.
	EndAt int64 `json:"end_at"`
	// PostID is the ID of the custom_cf_call post in the channel.
	PostID string `json:"post_id"`
	// ChannelMeetingID is the ID of the rtk_channel_meetings entry that this call was created against.
	ChannelMeetingID string `json:"rtk_channel_meeting_id"`
	// SessionID is the Cloudflare RTK session identifier.
	// Populated when the first participant connects (via webhook).
	// Empty string means the webhook has not been received yet.
	SessionID string `json:"session_id"`
}
