package app

import (
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/kondo97/mattermost-plugin-rtk/server/store"
)

// JobStateClient mirrors the Calls plugin's JobStateClient shape so that
// existing Calls clients can decode the response without changes. RTK does not
// run recording/transcription/live-captions today, so this type is currently
// unused at runtime — it exists only to document the wire format. The
// corresponding fields on CallStateClient are emitted as JSON null.
type JobStateClient struct {
	Type    string `json:"type"`
	InitAt  int64  `json:"init_at"`
	StartAt int64  `json:"start_at"`
	EndAt   int64  `json:"end_at"`
	Err     string `json:"err,omitempty"`
}

// UserSessionClient mirrors the Calls plugin's per-session participant state.
//
// RTK does not yet track per-session mute / hand / video state. To preserve
// wire compatibility with the Calls plugin's GET /channels response we
// synthesise one entry per participant with SessionID == UserID and all
// boolean state set to false. Callers MUST NOT depend on SessionID being a
// distinct identifier.
type UserSessionClient struct {
	SessionID  string `json:"session_id"`
	UserID     string `json:"user_id"`
	Unmuted    bool   `json:"unmuted"`
	RaisedHand int64  `json:"raised_hand"`
	Video      bool   `json:"video"`
}

// CallStateClient is the per-call payload returned inside ChannelCallState.
// Field names and JSON tags follow the Calls plugin so existing clients work.
type CallStateClient struct {
	ID                     string              `json:"id"`
	StartAt                int64               `json:"start_at"`
	Sessions               []UserSessionClient `json:"sessions"`
	ThreadID               string              `json:"thread_id"`
	PostID                 string              `json:"post_id"`
	ScreenSharingSessionID string              `json:"screen_sharing_session_id"`
	OwnerID                string              `json:"owner_id"`
	HostID                 string              `json:"host_id"`
	Recording              *JobStateClient     `json:"recording"`
	Transcription          *JobStateClient     `json:"transcription"`
	LiveCaptions           *JobStateClient     `json:"live_captions"`
	DismissedNotification  map[string]bool     `json:"dismissed_notification,omitempty"`
}

// ChannelCallState is one element of the GET /api/v1/channels response array.
type ChannelCallState struct {
	ChannelID string           `json:"channel_id"`
	Enabled   *bool            `json:"enabled,omitempty"`
	Call      *CallStateClient `json:"call,omitempty"`
}

// buildCallStateClient converts an internal CallSession into the Calls-compatible
// wire format. Returns nil if session is nil. Always emits the fields that
// callers (Calls clients) expect to be present, using the documented placeholder
// values for fields RTK does not track yet.
func buildCallStateClient(session *store.CallSession) *CallStateClient {
	if session == nil {
		return nil
	}
	sessions := make([]UserSessionClient, 0, len(session.Participants))
	for _, userID := range session.Participants {
		sessions = append(sessions, UserSessionClient{
			SessionID:  userID,
			UserID:     userID,
			Unmuted:    false,
			RaisedHand: 0,
			Video:      false,
		})
	}
	return &CallStateClient{
		ID:                     session.ID,
		StartAt:                session.CreateAt,
		Sessions:               sessions,
		ThreadID:               "",
		PostID:                 session.PostID,
		ScreenSharingSessionID: "",
		OwnerID:                session.CreatorID,
		HostID:                 session.CreatorID,
		Recording:              nil,
		Transcription:          nil,
		LiveCaptions:           nil,
		DismissedNotification:  nil,
	}
}

// GetAllCallChannels returns one entry per channel for which the requesting
// user has ReadChannel permission and either:
//   - A row exists in rtk_calls_channels (the channel is explicitly registered),
//     or
//   - An active call exists in the channel.
//
// This mirrors the semantics of the Calls plugin's GET /channels endpoint so
// that existing Calls clients can be repointed at the RTK plugin without code
// changes. See ARCHITECTURE.md for the migration story from `calls_channels`.
//
// The response is always non-nil; an empty result is returned as an empty slice
// to keep JSON marshalling deterministic.
func (a *App) GetAllCallChannels(userID string) ([]ChannelCallState, error) {
	if userID == "" {
		return nil, errInvalidUser
	}

	memberChannels, err := a.collectUserChannelMemberships(userID)
	if err != nil {
		return nil, err
	}

	channels, err := a.store.GetAllCallsChannels()
	if err != nil {
		return nil, err
	}

	calls, err := a.store.GetAllActiveCalls()
	if err != nil {
		return nil, err
	}

	callsByChannel := make(map[string]*store.CallSession, len(calls))
	for _, call := range calls {
		// Only include calls in channels the user can read. We additionally
		// gate on ReadChannel below when emitting; checking here lets us avoid
		// redundant work for channels the user cannot see.
		if !memberChannels[call.ChannelID] {
			continue
		}
		callsByChannel[call.ChannelID] = call
	}

	out := make([]ChannelCallState, 0, len(channels)+len(callsByChannel))
	seen := make(map[string]struct{}, len(channels))

	for _, ch := range channels {
		if !memberChannels[ch.ChannelID] {
			continue
		}
		seen[ch.ChannelID] = struct{}{}
		enabled := ch.Enabled
		entry := ChannelCallState{
			ChannelID: ch.ChannelID,
			Enabled:   &enabled,
		}
		if call, ok := callsByChannel[ch.ChannelID]; ok {
			entry.Call = buildCallStateClient(call)
			delete(callsByChannel, ch.ChannelID)
		}
		out = append(out, entry)
	}

	// Active calls in channels with no rtk_calls_channels row.
	for channelID, call := range callsByChannel {
		if _, dup := seen[channelID]; dup {
			continue
		}
		out = append(out, ChannelCallState{
			ChannelID: channelID,
			Call:      buildCallStateClient(call),
		})
	}

	return out, nil
}

// GetCallsChannel returns the calls-enablement state for a channel.
// Returns nil (with no error) when no row exists — the caller should treat nil
// as "default enabled".
func (a *App) GetCallsChannel(channelID string) (*store.CallsChannel, error) {
	if channelID == "" {
		return nil, errInvalidUser
	}
	return a.store.GetCallsChannel(channelID)
}

// UpdateCallsChannelEnabled sets the calls-enabled flag for a channel.
// Only users with the channel-type-appropriate manage-properties permission
// (channel admin, team admin, or system admin) may call this.
// DM and GM channels are not supported.
func (a *App) UpdateCallsChannelEnabled(channelID string, enabled bool, userID string) error {
	if channelID == "" || userID == "" {
		return errInvalidUser
	}

	ch, appErr := a.api.GetChannel(channelID)
	if appErr != nil {
		return appErr
	}

	var perm *model.Permission
	switch ch.Type {
	case model.ChannelTypeOpen:
		perm = model.PermissionManagePublicChannelProperties
	case model.ChannelTypePrivate:
		perm = model.PermissionManagePrivateChannelProperties
	default:
		// DM and GM channels do not support call disable/enable management.
		return ErrForbidden
	}

	if !a.api.HasPermissionToChannel(userID, channelID, perm) {
		return ErrForbidden
	}

	if err := a.store.UpsertCallsChannel(&store.CallsChannel{
		ChannelID: channelID,
		Enabled:   enabled,
	}); err != nil {
		return err
	}

	return nil
}

// collectUserChannelMemberships returns the set of channel IDs the user is a
// member of. Used to filter the GET /channels response to channels the user
// can actually read. Pages through GetChannelMembersForUser at 200 per page,
// matching the Calls plugin's behaviour.
func (a *App) collectUserChannelMemberships(userID string) (map[string]bool, error) {
	memberChannels := map[string]bool{}
	const perPage = 200
	for page := 0; ; page++ {
		members, appErr := a.api.GetChannelMembersForUser("", userID, page, perPage)
		if appErr != nil {
			return nil, appErr
		}
		for i := range members {
			memberChannels[members[i].ChannelId] = true
		}
		if len(members) < perPage {
			break
		}
	}
	return memberChannels, nil
}
