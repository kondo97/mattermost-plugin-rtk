package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	"github.com/kondo97/mattermost-plugin-rtk/server/store"
)

const (
	WSEventCallStarted   = "call_started"
	WSEventUserJoined    = "user_joined"
	WSEventUserLeft      = "user_left"
	WSEventCallEnded     = "call_ended"
	CallPostType         = "custom_cf_call"
	RTKPresetHost        = "group_call_host"
	RTKPresetParticipant = "group_call_participant"
)

// getUserDisplayName returns the display name for a Mattermost user.
func (a *App) getUserDisplayName(userID string) string {
	user, appErr := a.api.GetUser(userID)
	if appErr != nil || user == nil {
		return "Someone"
	}
	if name := user.GetDisplayName(model.ShowFullName); name != "" {
		return name
	}
	return "@" + user.Username
}

// nowMs returns the current time as Unix milliseconds.
func nowMs() int64 {
	return time.Now().UnixMilli()
}

// updatePostParticipants updates the participants list stored in the call post props.
// Best-effort: errors are logged but not returned to avoid blocking call state transitions.
func (a *App) updatePostParticipants(postID string, participants []string) {
	if postID == "" {
		return
	}
	post, appErr := a.api.GetPost(postID)
	if appErr != nil {
		a.api.LogWarn("updatePostParticipants: GetPost failed", "post_id", postID, "err", appErr.Error())
		return
	}
	if post.Props == nil {
		post.Props = make(model.StringInterface)
	}
	post.Props["participants"] = participants
	if _, appErr := a.api.UpdatePost(post); appErr != nil {
		a.api.LogWarn("updatePostParticipants: UpdatePost failed", "post_id", postID, "err", appErr.Error())
	}
}

// CreateCall creates a new call in the given channel for the user.
// Returns the CallSession, the RTK auth token, and any error.
func (a *App) CreateCall(channelID, userID string) (*store.CallSession, string, error) {
	if _, appErr := a.api.GetChannelMember(channelID, userID); appErr != nil {
		return nil, "", ErrNotChannelMember
	}

	a.callMu.Lock()
	defer a.callMu.Unlock()

	if a.rtk == nil {
		return nil, "", ErrRTKNotConfigured
	}

	// Reject when calls are explicitly disabled for this channel. A missing
	// rtk_calls_channels row is treated as enabled by default (mirrors the
	// Calls plugin behavior).
	if ch, err := a.store.GetCallsChannel(channelID); err != nil {
		a.api.LogError("CreateCall: failed to load channel state", "channel_id", channelID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to load channel state: %w", err)
	} else if ch != nil && !ch.Enabled {
		return nil, "", ErrCallsDisabled
	}

	// only one active call per channel
	existing, err := a.store.GetCallByChannel(channelID)
	if err != nil {
		a.api.LogError("CreateCall: failed to check existing call", "channel_id", channelID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to check existing call: %w", err)
	}
	if existing != nil {
		// verify the existing call is still alive on the RTK side.
		// If the meeting no longer exists (404), force-end the stale record and proceed.
		_, rtkErr := a.rtk.GetMeeting(existing.MeetingID)
		switch {
		case errors.Is(rtkErr, rtkclient.ErrMeetingNotFound):
			// Meeting confirmed gone — force-end the stale call and continue to create a new one.
			a.api.LogInfo("CreateCall: stale call detected, force-ending before creating new call",
				"call_id", existing.ID, "channel_id", channelID, "meeting_id", existing.MeetingID)
			if err := a.endCallInternal(existing, "stale_on_create"); err != nil {
				a.api.LogError("CreateCall: failed to end stale call", "call_id", existing.ID, "err", err.Error())
				return nil, "", ErrCallAlreadyActive // safe fallback
			}
		case rtkErr != nil:
			// Transient RTK error — treat the existing call as alive.
			a.api.LogWarn("CreateCall: GetMeeting transient error, treating existing call as active",
				"call_id", existing.ID, "err", rtkErr.Error())
			return nil, "", ErrCallAlreadyActive
		default:
			// Meeting is alive — normal conflict.
			return nil, "", ErrCallAlreadyActive
		}
	}

	// resolve or create RTK meeting for this channel
	savedChannelMeetingID, meetingID, savedAppConfigID, err := a.store.GetChannelMeeting(channelID)
	if err != nil {
		a.api.LogWarn("CreateCall: GetChannelMeeting failed, will create new meeting", "channel_id", channelID, "err", err.Error())
		meetingID = ""
		savedChannelMeetingID = ""
	}

	// Get the current app config ID for staleness check and saving.
	currentAppConfigID, err := a.store.GetActiveAppConfigID()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get app config ID: %w", err)
	}
	if currentAppConfigID == "" {
		return nil, "", fmt.Errorf("RTK app config not initialized")
	}

	var meeting *rtkclient.Meeting
	if meetingID != "" {
		// Stale check: if the app config has changed since this meeting was saved, treat it as invalid.
		if savedAppConfigID != "" && currentAppConfigID != "" && savedAppConfigID != currentAppConfigID {
			a.api.LogInfo("CreateCall: app config changed since meeting creation, creating new one",
				"channel_id", channelID, "old_app_config_id", savedAppConfigID, "current_app_config_id", currentAppConfigID)
			meetingID = ""
			savedChannelMeetingID = ""
		}
	}

	if meetingID != "" {
		// Verify the stored meeting still exists in Cloudflare.
		_, rtkErr := a.rtk.GetMeeting(meetingID)
		if rtkErr != nil && errors.Is(rtkErr, rtkclient.ErrMeetingNotFound) {
			a.api.LogInfo("CreateCall: stored meeting gone (404), creating new one", "channel_id", channelID, "old_meeting_id", meetingID)
			meetingID = ""
		} else if rtkErr != nil {
			// Transient error — reuse stored ID anyway.
			a.api.LogWarn("CreateCall: GetMeeting transient error, reusing stored meeting ID", "channel_id", channelID, "meeting_id", meetingID, "err", rtkErr.Error())
			meeting = &rtkclient.Meeting{ID: meetingID}
		} else {
			meeting = &rtkclient.Meeting{ID: meetingID}
		}
	}

	if meeting == nil {
		meeting, err = a.rtk.CreateMeeting()
		if err != nil {
			a.api.LogError("CreateCall: CreateMeeting failed", "channel_id", channelID, "user_id", userID, "err", err.Error())
			return nil, "", fmt.Errorf("failed to create meeting: %w", err)
		}
		if meeting.ID == "" {
			a.api.LogError("CreateCall: CreateMeeting returned empty meeting ID", "channel_id", channelID, "user_id", userID)
			return nil, "", fmt.Errorf("CreateMeeting returned empty meeting ID")
		}
		newID, saveErr := a.store.SaveChannelMeeting(channelID, meeting.ID, currentAppConfigID)
		if saveErr != nil {
			a.api.LogWarn("CreateCall: SaveChannelMeeting failed (best effort)", "channel_id", channelID, "meeting_id", meeting.ID, "err", saveErr.Error())
		} else {
			savedChannelMeetingID = newID
		}
	}

	// creator is added to participants. Mint session.ID early so it can be
	// embedded into the RTK customParticipantId before token issuance.
	session := &store.CallSession{
		ID:               model.NewId(),
		ChannelID:        channelID,
		CreatorID:        userID,
		MeetingID:        meeting.ID,
		Participants:     []string{userID},
		CreateAt:         nowMs(),
		UpdateAt:         nowMs(),
		EndAt:            0,
		ChannelMeetingID: savedChannelMeetingID,
	}

	// generate the host token. session.ID is bound into the RTK
	// customParticipantId so webhook events can be unambiguously correlated to
	// this specific call (RTK Meetings are reusable across calls in the same
	// channel — without this binding, delayed webhooks from a prior call could
	// be misattributed to a new call sharing the same meetingID).
	displayName := a.getUserDisplayName(userID)
	token, err := a.rtk.GenerateToken(meeting.ID, session.ID, userID, displayName, RTKPresetHost)
	if err != nil {
		a.api.LogError("CreateCall: GenerateToken failed", "channel_id", channelID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// create the post BEFORE inserting the call session so the row is
	// persisted with its real post_id from the start. Inserting first with
	// post_id='' would collide with the UNIQUE(post_id) constraint on any
	// subsequent CreateCall whose CreatePost fails.
	post := &model.Post{
		ChannelId: channelID,
		UserId:    userID,
		Type:      CallPostType,
		Props: map[string]any{
			"call_id":      session.ID,
			"channel_id":   channelID,
			"creator_id":   userID,
			"participants": session.Participants,
			"start_at":     session.CreateAt,
		},
	}
	createdPost, appErr := a.api.CreatePost(post)
	if appErr != nil {
		a.api.LogError("CreateCall: CreatePost failed", "call_id", session.ID, "channel_id", channelID, "err", appErr.Error())
		return nil, "", fmt.Errorf("failed to create call post: %s", appErr.Error())
	}

	session.PostID = createdPost.Id
	session.UpdateAt = nowMs()
	if err := a.store.CreateCallSession(session); err != nil {
		a.api.LogError("CreateCall: CreateCallSession failed", "call_id", session.ID, "channel_id", channelID, "err", err.Error())
		// Best-effort cleanup so the orphaned post does not show a broken call UI.
		if delErr := a.api.DeletePost(createdPost.Id); delErr != nil {
			a.api.LogWarn("CreateCall: DeletePost cleanup failed", "post_id", createdPost.Id, "err", delErr.Error())
		}
		return nil, "", fmt.Errorf("failed to save call: %w", err)
	}

	// Send mobile push notifications synchronously before emitting the WebSocket event.
	senderUser, senderErr := a.api.GetUser(userID)
	if senderErr == nil {
		a.sendPushNotifications(channelID, createdPost.Id, createdPost.Id, senderUser)
	} else {
		a.api.LogWarn("CreateCall: GetUser failed for push notifications (best effort)", "call_id", session.ID, "err", senderErr.Error())
	}

	// emit WebSocket event
	a.api.PublishWebSocketEvent(WSEventCallStarted, map[string]any{
		"call_id":      session.ID,
		"channel_id":   channelID,
		"creator_id":   userID,
		"participants": session.Participants,
		"start_at":     session.CreateAt,
		"post_id":      session.PostID,
	}, &model.WebsocketBroadcast{ChannelId: channelID})

	a.api.LogInfo("call started", "call_id", session.ID, "channel_id", channelID, "creator_id", userID)

	return session, token.Token, nil
}

// JoinCall adds a user to an existing call and returns the updated session and an RTK auth token.
func (a *App) JoinCall(callID, userID string) (*store.CallSession, string, error) {
	a.callMu.Lock()
	defer a.callMu.Unlock()

	if a.rtk == nil {
		return nil, "", ErrRTKNotConfigured
	}

	// call must be active
	session, err := a.store.GetCallByID(callID)
	if err != nil {
		a.api.LogError("JoinCall: GetCallByID failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to get call: %w", err)
	}
	if session == nil {
		a.api.LogError("JoinCall: call not found in KV store", "call_id", callID, "user_id", userID)
		return nil, "", ErrCallNotFound
	}
	if session.EndAt != 0 {
		a.api.LogError("JoinCall: call already ended", "call_id", callID, "user_id", userID, "end_at", session.EndAt)
		return nil, "", ErrCallNotFound
	}

	if _, appErr := a.api.GetChannelMember(session.ChannelID, userID); appErr != nil {
		return nil, "", ErrNotChannelMember
	}

	// Verify the RTK meeting is still alive. A definitive 404 means the call is
	// stale — force-end it and report not found to the caller.
	_, rtkErr := a.rtk.GetMeeting(session.MeetingID)
	if errors.Is(rtkErr, rtkclient.ErrMeetingNotFound) {
		a.api.LogInfo("JoinCall: RTK meeting not found, force-ending stale call",
			"call_id", callID, "meeting_id", session.MeetingID)
		if err := a.endCallInternal(session, "stale_on_join"); err != nil {
			a.api.LogError("JoinCall: failed to end stale call", "call_id", callID, "err", err.Error())
		}
		return nil, "", ErrCallNotFound
	} else if rtkErr != nil {
		a.api.LogWarn("JoinCall: GetMeeting transient error (continuing)",
			"call_id", callID, "err", rtkErr.Error())
	}

	// idempotent insert in the participants table FIRST so that, if RTK
	// token issuance subsequently fails, no token is ever returned for a user
	// who is not in the DB. Order: DB add → token; on token failure, compensate.
	// The store serializes against EndCall and concurrent Add/Remove on other
	// cluster nodes via a SELECT ... FOR UPDATE on the call row, so this cannot
	// produce a ghost participant on a call that is concurrently being ended.
	participants, active, added, err := a.store.AddCallParticipant(callID, userID)
	if err != nil {
		a.api.LogError("JoinCall: AddCallParticipant failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to update participants: %w", err)
	}
	if !active {
		// The call ended between our earlier check and the participant insert.
		return nil, "", ErrCallNotFound
	}
	session.Participants = participants

	// generate participant token. RTK customParticipantId is bound to the
	// (callID, userID) pair so webhook events can be unambiguously correlated to
	// this call (RTK Meetings are reusable across calls in the same channel).
	displayName := a.getUserDisplayName(userID)
	token, err := a.rtk.GenerateToken(session.MeetingID, callID, userID, displayName, RTKPresetParticipant)
	if err != nil {
		a.api.LogError("JoinCall: GenerateToken failed", "call_id", callID, "user_id", userID, "err", err.Error())
		// Compensation: only if THIS invocation actually inserted the row. If the
		// user was already a participant (added=false), removing them would erase
		// legitimate state. Compensation runs in a fresh tx; the store's FOR UPDATE
		// serializes it correctly with concurrent Add/Remove/EndCall on other nodes.
		if added {
			updated, endedNow, endAt, rerr := a.store.RemoveCallParticipant(callID, userID)
			if rerr != nil {
				a.api.LogError("JoinCall: compensating RemoveCallParticipant failed",
					"call_id", callID, "user_id", userID, "err", rerr.Error())
			} else if endedNow {
				// The compensation auto-ended the call (user was the sole participant).
				// No token was issued, so RTK has no session — emitting end side effects
				// is correct. emitCallEnded only updates the post and emits WS/push.
				session.Participants = updated
				session.EndAt = endAt
				a.emitCallEnded(session, endAt, "join_token_failure_compensation")
			}
		}
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// update post participants — best effort
	a.updatePostParticipants(session.PostID, session.Participants)

	// emit WebSocket event so all clients see the updated participant list immediately.
	// handleWebhookParticipantJoined will also fire later (when the RTK SDK actually connects)
	// and performs an idempotent update — this is safe because UpdatePost and PublishWebSocketEvent
	// with the same data are both no-ops from the client's perspective.
	a.api.PublishWebSocketEvent(WSEventUserJoined, map[string]any{
		"call_id":      callID,
		"channel_id":   session.ChannelID,
		"user_id":      userID,
		"participants": session.Participants,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	a.api.LogInfo("user joined call", "call_id", callID, "user_id", userID, "channel_id", session.ChannelID)

	return session, token.Token, nil
}

// LeaveCall removes a user from a call. If the last participant leaves, the call is ended.
func (a *App) LeaveCall(callID, userID string) error {
	a.callMu.Lock()
	defer a.callMu.Unlock()

	// idempotent — if call not found or already ended, no-op
	session, err := a.store.GetCallByID(callID)
	if err != nil {
		a.api.LogError("LeaveCall: GetCallByID failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return fmt.Errorf("failed to get call: %w", err)
	}
	if session == nil || session.EndAt != 0 {
		return nil
	}

	// atomic remove + auto-end. The store transaction holds a row
	// lock on rtk_call_sessions, so a concurrent JoinCall on another node
	// cannot insert a participant between the delete and the endat write.
	updated, endedNow, endAt, err := a.store.RemoveCallParticipant(callID, userID)
	if err != nil {
		a.api.LogError("LeaveCall: RemoveCallParticipant failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return fmt.Errorf("failed to update participants: %w", err)
	}
	session.Participants = updated

	// Update post participants — best effort
	a.updatePostParticipants(session.PostID, updated)

	// emit WebSocket event
	a.api.PublishWebSocketEvent(WSEventUserLeft, map[string]any{
		"call_id":      callID,
		"channel_id":   session.ChannelID,
		"user_id":      userID,
		"participants": updated,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	a.api.LogInfo("user left call", "call_id", callID, "user_id", userID, "channel_id", session.ChannelID)

	// when this remove transitioned the call to ended, run the side effects.
	// We only emit when endedNow=true to avoid double-emission if another path
	// (explicit EndCall, webhook meeting.ended) already ended the call concurrently.
	if endedNow {
		session.EndAt = endAt
		a.emitCallEnded(session, endAt, "last_participant_left")
	}

	return nil
}

// EndCall ends a call. Only the call creator may end the call.
func (a *App) EndCall(callID, requestingUserID string) error {
	a.callMu.Lock()
	defer a.callMu.Unlock()

	session, err := a.store.GetCallByID(callID)
	if err != nil {
		a.api.LogError("EndCall: GetCallByID failed", "call_id", callID, "user_id", requestingUserID, "err", err.Error())
		return fmt.Errorf("failed to get call: %w", err)
	}
	if session == nil || session.EndAt != 0 {
		return ErrCallNotFound
	}

	// only creator may end call
	if session.CreatorID != requestingUserID {
		return ErrUnauthorized
	}

	return a.endCallInternal(session, "explicit_end")
}

// endCallInternal is the shared implementation called by EndCall (explicit end),
// on-demand reconciliation, and webhook handlers. It writes endat to the store
// and runs the side effects.
//
// Note: LeaveCall's auto-end does NOT go through here — RemoveCallParticipant
// performs the endat write atomically with the participant delete to prevent ghost
// participants. The auto-end path calls emitCallEnded directly with the endAt value
// returned by the store.
//
// reason identifies the code path that triggered the end (for diagnostics).
// Caller must hold callMu when invoking this function.
func (a *App) endCallInternal(session *store.CallSession, reason string) error {
	a.api.LogInfo("endCallInternal triggered", "call_id", session.ID, "channel_id", session.ChannelID, "reason", reason)

	// set EndAt
	endAt := nowMs()
	if err := a.store.EndCall(session.ID, endAt); err != nil {
		a.api.LogError("endCallInternal: EndCall failed", "call_id", session.ID, "err", err.Error())
		return fmt.Errorf("failed to end call in store: %w", err)
	}

	a.emitCallEnded(session, endAt, reason)
	return nil
}

// emitCallEnded performs the side effects of ending a call: updating the post
// to its ended state, sending push notifications to dismiss the ringing UI, and
// publishing the WebSocket event. Safe to call multiple times for the same call
// (idempotent at the post and WS layers).
func (a *App) emitCallEnded(session *store.CallSession, endAt int64, reason string) {
	durationMs := endAt - session.CreateAt

	// RTK sessions are auto-ended by Cloudflare when the last participant leaves.
	// Do NOT call EndMeeting — Meeting is a permanent reusable room.

	// update post to ended state — best effort
	if session.PostID != "" {
		post, appErr := a.api.GetPost(session.PostID)
		if appErr != nil {
			a.api.LogWarn("emitCallEnded: GetPost failed (best effort)", "call_id", session.ID, "post_id", session.PostID, "err", appErr.Error())
		} else {
			if post.Props == nil {
				post.Props = make(model.StringInterface)
			}
			post.Props["end_at"] = endAt
			post.Props["duration_ms"] = durationMs
			if _, appErr := a.api.UpdatePost(post); appErr != nil {
				a.api.LogWarn("emitCallEnded: UpdatePost failed (best effort)", "call_id", session.ID, "err", appErr.Error())
			}
		}
		// Send mobile push notifications to dismiss the ringing UI on all member devices.
		a.sendEndCallPushNotifications(session.ChannelID, session.PostID, session.CreatorID)
	}

	// emit WebSocket event
	a.api.PublishWebSocketEvent(WSEventCallEnded, map[string]any{
		"call_id":     session.ID,
		"channel_id":  session.ChannelID,
		"end_at":      endAt,
		"duration_ms": durationMs,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	a.api.LogInfo("call ended", "call_id", session.ID, "channel_id", session.ChannelID, "duration_ms", durationMs, "reason", reason)
}

// reconcileCallOnDemand checks a single active call against the RTK API and
// force-ends it if the meeting no longer exists. This is an on-demand, single-cycle
// check without a failure threshold — it fires on user requests instead of on a timer,
// so a definitive 404 is acted on immediately.
// Transient RTK errors are ignored to avoid accidentally terminating live calls.
func (a *App) ReconcileCallOnDemand(session *store.CallSession) {
	if a.rtk == nil {
		return
	}

	_, err := a.rtk.GetMeeting(session.MeetingID)
	if !errors.Is(err, rtkclient.ErrMeetingNotFound) {
		// Meeting is alive or a transient error occurred — do nothing.
		return
	}

	a.callMu.Lock()
	defer a.callMu.Unlock()

	// Re-read under the lock to avoid a TOCTOU race with concurrent EndCall/LeaveCall.
	fresh, err := a.store.GetCallByID(session.ID)
	if err != nil || fresh == nil || fresh.EndAt != 0 {
		return
	}

	a.api.LogInfo("ReconcileCallOnDemand: RTK meeting not found, force-ending stale call",
		"call_id", session.ID, "meeting_id", session.MeetingID)
	if err := a.endCallInternal(fresh, "stale_on_get"); err != nil {
		a.api.LogError("ReconcileCallOnDemand: endCallInternal failed", "call_id", session.ID, "err", err.Error())
	}
}

// GetCallByID returns a call session by ID.
func (a *App) GetCallByID(callID string) (*store.CallSession, error) {
	return a.store.GetCallByID(callID)
}
