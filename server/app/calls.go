package app

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
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

// containsUser returns true if userID is in participants.
func containsUser(participants []string, userID string) bool {
	return slices.Contains(participants, userID)
}

// removeUser returns participants with userID removed (all occurrences).
func removeUser(participants []string, userID string) []string {
	result := make([]string, 0, len(participants))
	for _, p := range participants {
		if p != userID {
			result = append(result, p)
		}
	}
	return result
}

// CreateCall creates a new call in the given channel for the user.
// Returns the CallSession, the RTK auth token, and any error.
func (a *App) CreateCall(channelID, userID string) (*kvstore.CallSession, string, error) {
	if _, appErr := a.api.GetChannelMember(channelID, userID); appErr != nil {
		return nil, "", ErrNotChannelMember
	}

	a.callMu.Lock()
	defer a.callMu.Unlock()

	if a.rtk == nil {
		return nil, "", ErrRTKNotConfigured
	}

	// BR-01: only one active call per channel
	existing, err := a.store.GetCallByChannel(channelID)
	if err != nil {
		a.api.LogError("CreateCall: failed to check existing call", "channel_id", channelID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to check existing call: %w", err)
	}
	if existing != nil {
		// BR-01: verify the existing call is still alive on the RTK side.
		// If the meeting no longer exists (404), force-end the stale record and proceed.
		_, rtkErr := a.rtk.GetMeetingParticipants(existing.MeetingID)
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
			a.api.LogWarn("CreateCall: GetMeetingParticipants transient error, treating existing call as active",
				"call_id", existing.ID, "err", rtkErr.Error())
			return nil, "", ErrCallAlreadyActive
		default:
			// Meeting is alive — normal conflict.
			return nil, "", ErrCallAlreadyActive
		}
	}

	// BR-02/BR-05: create RTK meeting — abort on failure
	meeting, err := a.rtk.CreateMeeting(rtkclient.CreateMeetingOptions{})
	if err != nil {
		a.api.LogError("CreateCall: CreateMeeting failed", "channel_id", channelID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to create meeting: %w", err)
	}

	displayName := a.getUserDisplayName(userID)
	token, err := a.rtk.GenerateToken(meeting.ID, userID, displayName, RTKPresetHost)
	if err != nil {
		a.api.LogError("CreateCall: GenerateToken failed", "channel_id", channelID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// BR-03: creator is added to participants
	session := &kvstore.CallSession{
		ID:           uuid.New().String(),
		ChannelID:    channelID,
		CreatorID:    userID,
		MeetingID:    meeting.ID,
		Participants: []string{userID},
		StartAt:      nowMs(),
		EndAt:        0,
	}

	if err := a.store.SaveCall(session); err != nil {
		a.api.LogError("CreateCall: SaveCall failed", "call_id", session.ID, "channel_id", channelID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to save call: %w", err)
	}

	// BR-04: create post — best effort
	post := &model.Post{
		ChannelId: channelID,
		UserId:    userID,
		Type:      CallPostType,
		Props: map[string]any{
			"call_id":      session.ID,
			"channel_id":   channelID,
			"creator_id":   userID,
			"participants": session.Participants,
			"start_at":     session.StartAt,
		},
	}
	createdPost, appErr := a.api.CreatePost(post)
	if appErr != nil {
		a.api.LogWarn("CreateCall: CreatePost failed (best effort)", "call_id", session.ID, "err", appErr.Error())
	} else {
		session.PostID = createdPost.Id
		if err := a.store.SaveCall(session); err != nil {
			a.api.LogWarn("CreateCall: failed to update PostID (best effort)", "call_id", session.ID, "err", err.Error())
		}
		// Send mobile push notifications synchronously before emitting the WebSocket event.
		senderUser, senderErr := a.api.GetUser(userID)
		if senderErr == nil {
			a.sendPushNotifications(channelID, createdPost.Id, createdPost.Id, senderUser)
		} else {
			a.api.LogWarn("CreateCall: GetUser failed for push notifications (best effort)", "call_id", session.ID, "err", senderErr.Error())
		}
	}

	// BR-04: emit WebSocket event
	a.api.PublishWebSocketEvent(WSEventCallStarted, map[string]any{
		"call_id":      session.ID,
		"channel_id":   channelID,
		"creator_id":   userID,
		"participants": session.Participants,
		"start_at":     session.StartAt,
		"post_id":      session.PostID,
	}, &model.WebsocketBroadcast{ChannelId: channelID})

	a.api.LogInfo("call started", "call_id", session.ID, "channel_id", channelID, "creator_id", userID)

	return session, token.Token, nil
}

// JoinCall adds a user to an existing call and returns the updated session and an RTK auth token.
func (a *App) JoinCall(callID, userID string) (*kvstore.CallSession, string, error) {
	a.callMu.Lock()
	defer a.callMu.Unlock()

	if a.rtk == nil {
		return nil, "", ErrRTKNotConfigured
	}

	// BR-06: call must be active
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
	_, rtkErr := a.rtk.GetMeetingParticipants(session.MeetingID)
	if errors.Is(rtkErr, rtkclient.ErrMeetingNotFound) {
		a.api.LogInfo("JoinCall: RTK meeting not found, force-ending stale call",
			"call_id", callID, "meeting_id", session.MeetingID)
		if err := a.endCallInternal(session, "stale_on_join"); err != nil {
			a.api.LogError("JoinCall: failed to end stale call", "call_id", callID, "err", err.Error())
		}
		return nil, "", ErrCallNotFound
	} else if rtkErr != nil {
		a.api.LogWarn("JoinCall: GetMeetingParticipants transient error (continuing)",
			"call_id", callID, "err", rtkErr.Error())
	}

	// BR-08: generate participant token
	displayName := a.getUserDisplayName(userID)
	token, err := a.rtk.GenerateToken(session.MeetingID, userID, displayName, RTKPresetParticipant)
	if err != nil {
		a.api.LogError("JoinCall: GenerateToken failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// BR-09: add userID deduplicated
	if !containsUser(session.Participants, userID) {
		session.Participants = append(session.Participants, userID)
	}
	if err := a.store.UpdateCallParticipants(callID, session.Participants); err != nil {
		a.api.LogError("JoinCall: UpdateCallParticipants failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to update participants: %w", err)
	}

	// BR-10: update post participants — best effort
	a.updatePostParticipants(session.PostID, session.Participants)

	// BR-10: emit WebSocket event so all clients see the updated participant list immediately.
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

	// BR-11: idempotent — if call not found or already ended, no-op
	session, err := a.store.GetCallByID(callID)
	if err != nil {
		a.api.LogError("LeaveCall: GetCallByID failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return fmt.Errorf("failed to get call: %w", err)
	}
	if session == nil || session.EndAt != 0 {
		return nil
	}

	// BR-11: remove userID (no-op if not present)
	updated := removeUser(session.Participants, userID)
	if err := a.store.UpdateCallParticipants(callID, updated); err != nil {
		a.api.LogError("LeaveCall: UpdateCallParticipants failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return fmt.Errorf("failed to update participants: %w", err)
	}
	session.Participants = updated

	// Update post participants — best effort
	a.updatePostParticipants(session.PostID, updated)

	// BR-12: emit WebSocket event
	a.api.PublishWebSocketEvent(WSEventUserLeft, map[string]any{
		"call_id":      callID,
		"channel_id":   session.ChannelID,
		"user_id":      userID,
		"participants": updated,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	a.api.LogInfo("user left call", "call_id", callID, "user_id", userID, "channel_id", session.ChannelID)

	// BR-13: auto-end if last participant left
	if len(updated) == 0 {
		if err := a.endCallInternal(session, "last_participant_left"); err != nil {
			a.api.LogError("LeaveCall: endCallInternal failed", "call_id", callID, "err", err.Error())
			return fmt.Errorf("failed to end call: %w", err)
		}
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

	// BR-14: only creator may end call
	if session.CreatorID != requestingUserID {
		return ErrUnauthorized
	}

	return a.endCallInternal(session, "explicit_end")
}

// endCallInternal is the shared implementation called by EndCall, LeaveCall (auto-end),
// on-demand reconciliation, and webhook handlers.
// reason identifies the code path that triggered the end (for diagnostics).
// Caller must hold callMu when invoking this function.
func (a *App) endCallInternal(session *kvstore.CallSession, reason string) error {
	a.api.LogInfo("endCallInternal triggered", "call_id", session.ID, "channel_id", session.ChannelID, "reason", reason)

	// BR-26: set EndAt
	endAt := nowMs()
	if err := a.store.EndCall(session.ID, endAt); err != nil {
		a.api.LogError("endCallInternal: EndCall failed", "call_id", session.ID, "err", err.Error())
		return fmt.Errorf("failed to end call in store: %w", err)
	}

	durationMs := endAt - session.StartAt

	// BR-27: end RTK meeting — best effort
	if a.rtk != nil {
		if err := a.rtk.EndMeeting(session.MeetingID); err != nil {
			a.api.LogWarn("endCallInternal: EndMeeting failed (best effort)", "call_id", session.ID, "err", err.Error())
		}
	}

	// BR-28: update post to ended state — best effort
	if session.PostID != "" {
		post, appErr := a.api.GetPost(session.PostID)
		if appErr != nil {
			a.api.LogWarn("endCallInternal: GetPost failed (best effort)", "call_id", session.ID, "post_id", session.PostID, "err", appErr.Error())
		} else {
			if post.Props == nil {
				post.Props = make(model.StringInterface)
			}
			post.Props["end_at"] = endAt
			post.Props["duration_ms"] = durationMs
			if _, appErr := a.api.UpdatePost(post); appErr != nil {
				a.api.LogWarn("endCallInternal: UpdatePost failed (best effort)", "call_id", session.ID, "err", appErr.Error())
			}
		}
		// Send mobile push notifications to dismiss the ringing UI on all member devices.
		a.sendEndCallPushNotifications(session.ChannelID, session.PostID, session.CreatorID)
	}

	// BR-29: emit WebSocket event
	a.api.PublishWebSocketEvent(WSEventCallEnded, map[string]any{
		"call_id":     session.ID,
		"channel_id":  session.ChannelID,
		"end_at":      endAt,
		"duration_ms": durationMs,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	a.api.LogInfo("call ended", "call_id", session.ID, "channel_id", session.ChannelID, "duration_ms", durationMs)

	return nil
}

// reconcileCallOnDemand checks a single active call against the RTK API and
// force-ends it if the meeting no longer exists. This is an on-demand, single-cycle
// check without a failure threshold — it fires on user requests instead of on a timer,
// so a definitive 404 is acted on immediately.
// Transient RTK errors are ignored to avoid accidentally terminating live calls.
func (a *App) ReconcileCallOnDemand(session *kvstore.CallSession) {
	if a.rtk == nil {
		return
	}

	_, err := a.rtk.GetMeetingParticipants(session.MeetingID)
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
func (a *App) GetCallByID(callID string) (*kvstore.CallSession, error) {
	return a.store.GetCallByID(callID)
}
