package app

import (
	"github.com/mattermost/mattermost/server/public/model"
)

// HandleWebhookParticipantJoined processes a meeting.participantJoined event from RTK.
// callID and userID are parsed from the RTK customParticipantId by the API layer
// (format "callID:userID"). callID="" indicates a legacy/unparseable token and the
// event is silently dropped.
func (a *App) HandleWebhookParticipantJoined(meetingID, callID, userID, sessionID string) {
	if meetingID == "" || userID == "" {
		return
	}
	if callID == "" {
		// Legacy customParticipantId without callID binding. We cannot tell which call
		// this event belongs to (RTK Meetings are reusable across calls in the same
		// channel), so we drop the event rather than risk misattribution.
		a.api.LogWarn("HandleWebhookParticipantJoined: missing callID, ignoring (likely legacy token)",
			"meeting_id", meetingID, "user_id", userID)
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
	// Verify the callID embedded in the token matches the active call for this
	// meeting. A mismatch means a delayed webhook from a prior call that shared
	// the same permanent meetingID is arriving after a new call has started —
	// applying it to the new call would cross-contaminate participants.
	if session.ID != callID {
		a.api.LogWarn("HandleWebhookParticipantJoined: callID mismatch, ignoring stale event",
			"meeting_id", meetingID, "event_call_id", callID, "active_call_id", session.ID, "user_id", userID)
		return
	}

	// Re-validate channel membership: a user with a cached/old token who lost
	// channel access since join should not be auto-added by an RTK reconnect.
	if _, appErr := a.api.GetChannelMember(session.ChannelID, userID); appErr != nil {
		a.api.LogWarn("HandleWebhookParticipantJoined: user is no longer a channel member, skipping rescue",
			"call_id", session.ID, "user_id", userID)
		return
	}

	// Rescue: idempotently ensure the user is in the DB participants table.
	// JoinCall normally adds them before issuing the token, but if that DB write
	// failed (or if some race left them missing), this brings DB in sync with the
	// RTK truth that the user actually connected to the room.
	participants, active, _, err := a.store.AddCallParticipant(session.ID, userID)
	if err != nil {
		a.api.LogError("HandleWebhookParticipantJoined: AddCallParticipant rescue failed",
			"call_id", session.ID, "user_id", userID, "err", err.Error())
		return
	}
	if !active {
		// Call ended between the lookups above and the AddCallParticipant tx —
		// silently skip rather than emit a join event for an ended call.
		a.api.LogInfo("HandleWebhookParticipantJoined: call ended before rescue committed, skipping",
			"call_id", session.ID, "user_id", userID)
		return
	}

	// Update post participants — best effort
	a.updatePostParticipants(session.PostID, participants)

	// Backfill session ID if not yet set (best-effort).
	if sessionID != "" && session.SessionID == "" {
		if err := a.store.UpdateCallSessionID(session.ID, sessionID); err != nil {
			a.api.LogWarn("HandleWebhookParticipantJoined: UpdateCallSessionID failed",
				"call_id", session.ID, "err", err.Error())
		}
	}

	// Emit WebSocket event now that the participant is actually in the room.
	a.api.PublishWebSocketEvent(WSEventUserJoined, map[string]any{
		"call_id":      session.ID,
		"channel_id":   session.ChannelID,
		"user_id":      userID,
		"participants": participants,
	}, &model.WebsocketBroadcast{ChannelId: session.ChannelID})

	a.api.LogInfo("user connected to call", "call_id", session.ID, "user_id", userID, "channel_id", session.ChannelID)
}

// HandleWebhookParticipantLeft processes a meeting.participantLeft event from RTK.
// callID is parsed from customParticipantId by the API layer; an empty callID
// (legacy token) causes the event to be ignored.
func (a *App) HandleWebhookParticipantLeft(meetingID, callID, userID string) {
	if meetingID == "" || userID == "" {
		return
	}
	if callID == "" {
		a.api.LogWarn("HandleWebhookParticipantLeft: missing callID, ignoring (likely legacy token)",
			"meeting_id", meetingID, "user_id", userID)
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
	if session.ID != callID {
		// Delayed event from a prior call that shared this permanent meeting.
		// Applying LeaveCall to the current call would erase a legitimate participant.
		a.api.LogWarn("HandleWebhookParticipantLeft: callID mismatch, ignoring stale event",
			"meeting_id", meetingID, "event_call_id", callID, "active_call_id", session.ID, "user_id", userID)
		return
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
