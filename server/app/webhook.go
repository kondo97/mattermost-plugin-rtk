package app

import (
	"github.com/mattermost/mattermost/server/public/model"
)

// GetWebhookSecret returns the stored RTK webhook secret.
func (a *App) GetWebhookSecret() (string, error) {
	return a.store.GetWebhookSecret()
}

// HandleWebhookParticipantJoined processes a meeting.participantJoined event from RTK.
func (a *App) HandleWebhookParticipantJoined(meetingID, userID string) {
	if meetingID == "" || userID == "" {
		return
	}

	session, err := a.store.GetCallByMeetingID(meetingID)
	if err != nil {
		a.api.LogError("HandleWebhookParticipantJoined: GetCallByMeetingID failed", "meeting_id", meetingID, "error", err.Error())
		return
	}
	if session == nil || session.EndAt != 0 {
		return
	}

	// Re-read inside a fresh fetch to get the latest participants list
	// (JoinCall already updated the store before returning the token).
	fresh, err := a.store.GetCallByID(session.ID)
	if err != nil {
		a.api.LogError("HandleWebhookParticipantJoined: GetCallByID failed", "call_id", session.ID, "error", err.Error())
		return
	}
	if fresh == nil || fresh.EndAt != 0 {
		return
	}

	// Update post participants — best effort
	a.updatePostParticipants(fresh.PostID, fresh.Participants)

	// Emit WebSocket event now that the participant is actually in the room
	a.api.PublishWebSocketEvent(WSEventUserJoined, map[string]any{
		"call_id":      fresh.ID,
		"channel_id":   fresh.ChannelID,
		"user_id":      userID,
		"participants": fresh.Participants,
	}, &model.WebsocketBroadcast{ChannelId: fresh.ChannelID})

	a.api.LogInfo("user connected to call", "call_id", fresh.ID, "user_id", userID, "channel_id", fresh.ChannelID)
}

// HandleWebhookParticipantLeft processes a meeting.participantLeft event from RTK.
func (a *App) HandleWebhookParticipantLeft(meetingID, userID string) {
	if meetingID == "" || userID == "" {
		return
	}

	session, err := a.store.GetCallByMeetingID(meetingID)
	if err != nil {
		a.api.LogError("HandleWebhookParticipantLeft: GetCallByMeetingID failed", "meeting_id", meetingID, "error", err.Error())
		return
	}
	if session == nil || session.EndAt != 0 {
		return // idempotent
	}

	if err := a.LeaveCall(session.ID, userID); err != nil {
		a.api.LogError("HandleWebhookParticipantLeft: LeaveCall failed", "call_id", session.ID, "user_id", userID, "error", err.Error())
	}
}

// HandleWebhookMeetingEnded processes a meeting.ended event from RTK.
func (a *App) HandleWebhookMeetingEnded(meetingID string) {
	if meetingID == "" {
		return
	}

	a.api.LogInfo("HandleWebhookMeetingEnded: received meeting.ended webhook", "meeting_id", meetingID)

	session, err := a.store.GetCallByMeetingID(meetingID)
	if err != nil {
		a.api.LogError("HandleWebhookMeetingEnded: GetCallByMeetingID failed", "meeting_id", meetingID, "error", err.Error())
		return
	}
	if session == nil || session.EndAt != 0 {
		return // idempotent
	}

	a.callMu.Lock()
	defer a.callMu.Unlock()

	// Re-read inside the lock to prevent TOCTOU: another path (EndCall/LeaveCall)
	// may have ended the call between the check above and acquiring the lock.
	fresh, err := a.store.GetCallByID(session.ID)
	if err != nil {
		a.api.LogError("HandleWebhookMeetingEnded: GetCallByID re-check failed", "call_id", session.ID, "error", err.Error())
		return
	}
	if fresh == nil || fresh.EndAt != 0 {
		return // already ended by another path
	}

	if err := a.endCallInternal(fresh, "rtk_webhook"); err != nil {
		a.api.LogError("HandleWebhookMeetingEnded: endCallInternal failed", "call_id", fresh.ID, "error", err.Error())
	}
}
