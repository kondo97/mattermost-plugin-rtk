package main

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
)

const (
	wsEventCallStarted   = "custom_cf_call_started"
	wsEventUserJoined    = "custom_cf_user_joined"
	wsEventUserLeft      = "custom_cf_user_left"
	wsEventCallEnded     = "custom_cf_call_ended"
	callPostType         = "custom_cf_call"
	rtkPresetHost        = "group_call_host"
	rtkPresetParticipant = "group_call_participant"
)

// nowMs returns the current time as Unix milliseconds.
func nowMs() int64 {
	return time.Now().UnixMilli()
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
func (p *Plugin) CreateCall(channelID, userID string) (*kvstore.CallSession, string, error) {
	p.callMu.Lock()
	defer p.callMu.Unlock()

	if p.rtkClient == nil {
		return nil, "", ErrRTKNotConfigured
	}

	// BR-01: only one active call per channel
	existing, err := p.kvStore.GetCallByChannel(channelID)
	if err != nil {
		p.API.LogError("CreateCall: failed to check existing call", "channel_id", channelID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to check existing call: %w", err)
	}
	if existing != nil {
		return nil, "", ErrCallAlreadyActive
	}

	// BR-02/BR-05: create RTK meeting — abort on failure
	meeting, err := p.rtkClient.CreateMeeting()
	if err != nil {
		p.API.LogError("CreateCall: CreateMeeting failed", "channel_id", channelID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to create meeting: %w", err)
	}

	token, err := p.rtkClient.GenerateToken(meeting.ID, userID, rtkPresetHost)
	if err != nil {
		p.API.LogError("CreateCall: GenerateToken failed", "channel_id", channelID, "user_id", userID, "err", err.Error())
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

	if err := p.kvStore.SaveCall(session); err != nil {
		p.API.LogError("CreateCall: SaveCall failed", "call_id", session.ID, "channel_id", channelID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to save call: %w", err)
	}

	// BR-04: create post — best effort
	post := &model.Post{
		ChannelId: channelID,
		UserId:    userID,
		Type:      callPostType,
		Props: map[string]any{
			"call_id":      session.ID,
			"channel_id":   channelID,
			"creator_id":   userID,
			"participants": session.Participants,
			"start_at":     session.StartAt,
		},
	}
	createdPost, appErr := p.API.CreatePost(post)
	if appErr != nil {
		p.API.LogWarn("CreateCall: CreatePost failed (best effort)", "call_id", session.ID, "err", appErr.Error())
	} else {
		session.PostID = createdPost.Id
		if err := p.kvStore.SaveCall(session); err != nil {
			p.API.LogWarn("CreateCall: failed to update PostID (best effort)", "call_id", session.ID, "err", err.Error())
		}
	}

	// BR-04: emit WebSocket event
	p.API.PublishWebSocketEvent(wsEventCallStarted, map[string]any{
		"call_id":      session.ID,
		"channel_id":   channelID,
		"creator_id":   userID,
		"participants": session.Participants,
		"start_at":     session.StartAt,
		"post_id":      session.PostID,
	}, &model.WebsocketBroadcast{ChannelId: channelID})

	// BR-P01: best-effort push notification (DM/GM only, max 8 members)
	if p.pushSender != nil {
		if err := p.pushSender.SendIncomingCall(session); err != nil {
			p.API.LogWarn("CreateCall: SendIncomingCall failed (best effort)",
				"call_id", session.ID, "channel_id", channelID, "err", err.Error())
		}
	}

	p.API.LogInfo("call started", "call_id", session.ID, "channel_id", channelID, "creator_id", userID)

	return session, token.Token, nil
}

// JoinCall adds a user to an existing call and returns the updated session and an RTK auth token.
func (p *Plugin) JoinCall(callID, userID string) (*kvstore.CallSession, string, error) {
	p.callMu.Lock()
	defer p.callMu.Unlock()

	if p.rtkClient == nil {
		return nil, "", ErrRTKNotConfigured
	}

	// BR-06: call must be active
	session, err := p.kvStore.GetCallByID(callID)
	if err != nil {
		p.API.LogError("JoinCall: GetCallByID failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to get call: %w", err)
	}
	if session == nil || session.EndAt != 0 {
		return nil, "", ErrCallNotFound
	}

	// BR-08: generate participant token
	token, err := p.rtkClient.GenerateToken(session.MeetingID, userID, rtkPresetParticipant)
	if err != nil {
		p.API.LogError("JoinCall: GenerateToken failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// BR-09: add userID deduplicated
	if !containsUser(session.Participants, userID) {
		session.Participants = append(session.Participants, userID)
	}
	if err := p.kvStore.UpdateCallParticipants(callID, session.Participants); err != nil {
		p.API.LogError("JoinCall: UpdateCallParticipants failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return nil, "", fmt.Errorf("failed to update participants: %w", err)
	}

	// BR-10: emit WebSocket event
	p.API.PublishWebSocketEvent(wsEventUserJoined, map[string]any{
		"call_id":      callID,
		"channel_id":   session.ChannelID,
		"user_id":      userID,
		"participants": session.Participants,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	p.API.LogInfo("user joined call", "call_id", callID, "user_id", userID, "channel_id", session.ChannelID)

	return session, token.Token, nil
}

// LeaveCall removes a user from a call. If the last participant leaves, the call is ended.
func (p *Plugin) LeaveCall(callID, userID string) error {
	p.callMu.Lock()
	defer p.callMu.Unlock()

	// BR-11: idempotent — if call not found or already ended, no-op
	session, err := p.kvStore.GetCallByID(callID)
	if err != nil {
		p.API.LogError("LeaveCall: GetCallByID failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return fmt.Errorf("failed to get call: %w", err)
	}
	if session == nil || session.EndAt != 0 {
		return nil
	}

	// BR-11: remove userID (no-op if not present)
	updated := removeUser(session.Participants, userID)
	if err := p.kvStore.UpdateCallParticipants(callID, updated); err != nil {
		p.API.LogError("LeaveCall: UpdateCallParticipants failed", "call_id", callID, "user_id", userID, "err", err.Error())
		return fmt.Errorf("failed to update participants: %w", err)
	}
	session.Participants = updated

	// BR-12: emit WebSocket event
	p.API.PublishWebSocketEvent(wsEventUserLeft, map[string]any{
		"call_id":      callID,
		"channel_id":   session.ChannelID,
		"user_id":      userID,
		"participants": updated,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	p.API.LogInfo("user left call", "call_id", callID, "user_id", userID, "channel_id", session.ChannelID)

	// BR-13: auto-end if last participant left
	if len(updated) == 0 {
		if err := p.endCallInternal(session); err != nil {
			p.API.LogError("LeaveCall: endCallInternal failed", "call_id", callID, "err", err.Error())
			return fmt.Errorf("failed to end call: %w", err)
		}
	}

	return nil
}

// EndCall ends a call. Only the call creator may end the call.
func (p *Plugin) EndCall(callID, requestingUserID string) error {
	p.callMu.Lock()
	defer p.callMu.Unlock()

	session, err := p.kvStore.GetCallByID(callID)
	if err != nil {
		p.API.LogError("EndCall: GetCallByID failed", "call_id", callID, "user_id", requestingUserID, "err", err.Error())
		return fmt.Errorf("failed to get call: %w", err)
	}
	if session == nil || session.EndAt != 0 {
		return ErrCallNotFound
	}

	// BR-14: only creator may end call
	if session.CreatorID != requestingUserID {
		return ErrUnauthorized
	}

	return p.endCallInternal(session)
}

// endCallInternal is the shared implementation called by EndCall and LeaveCall (auto-end).
func (p *Plugin) endCallInternal(session *kvstore.CallSession) error {
	// BR-26: set EndAt
	endAt := nowMs()
	if err := p.kvStore.EndCall(session.ID, endAt); err != nil {
		p.API.LogError("endCallInternal: EndCall KVStore failed", "call_id", session.ID, "err", err.Error())
		return fmt.Errorf("failed to end call in store: %w", err)
	}

	durationMs := endAt - session.StartAt

	// BR-27: end RTK meeting — best effort
	if p.rtkClient != nil {
		if err := p.rtkClient.EndMeeting(session.MeetingID); err != nil {
			p.API.LogWarn("endCallInternal: EndMeeting failed (best effort)", "call_id", session.ID, "err", err.Error())
		}
	}

	// BR-28: update post to ended state — best effort
	if session.PostID != "" {
		post, appErr := p.API.GetPost(session.PostID)
		if appErr != nil {
			p.API.LogWarn("endCallInternal: GetPost failed (best effort)", "call_id", session.ID, "post_id", session.PostID, "err", appErr.Error())
		} else {
			if post.Props == nil {
				post.Props = make(model.StringInterface)
			}
			post.Props["end_at"] = endAt
			post.Props["duration_ms"] = durationMs
			if _, appErr := p.API.UpdatePost(post); appErr != nil {
				p.API.LogWarn("endCallInternal: UpdatePost failed (best effort)", "call_id", session.ID, "err", appErr.Error())
			}
		}
	}

	// BR-29: emit WebSocket event
	p.API.PublishWebSocketEvent(wsEventCallEnded, map[string]any{
		"call_id":     session.ID,
		"channel_id":  session.ChannelID,
		"end_at":      endAt,
		"duration_ms": durationMs,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	// BR-P01: best-effort push notification to dismiss incoming call UI
	if p.pushSender != nil {
		if err := p.pushSender.SendCallEnded(session); err != nil {
			p.API.LogWarn("endCallInternal: SendCallEnded failed (best effort)",
				"call_id", session.ID, "channel_id", session.ChannelID, "err", err.Error())
		}
	}

	p.API.LogInfo("call ended", "call_id", session.ID, "channel_id", session.ChannelID, "duration_ms", durationMs)

	return nil
}
