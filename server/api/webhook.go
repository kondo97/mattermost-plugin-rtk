package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
)

// rtkWebhookEvent is the top-level RTK webhook payload.
type rtkWebhookEvent struct {
	Event       string                `json:"event"`
	Meeting     rtkWebhookMeeting     `json:"meeting"`
	Participant rtkWebhookParticipant `json:"participant"`
}

type rtkWebhookMeeting struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
}

type rtkWebhookParticipant struct {
	CustomParticipantID string `json:"customParticipantId"`
}

// handleRTKWebhook handles POST /api/v1/webhook/rtk.
func (h *API) handleRTKWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.app.LogError("handleRTKWebhook: failed to read body", "error", err.Error())
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var event rtkWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.app.LogError("handleRTKWebhook: failed to parse event", "error", err.Error())
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	switch event.Event {
	case "meeting.participantJoined":
		callID, userID, ok := rtkclient.ParseCustomParticipantID(event.Participant.CustomParticipantID)
		if !ok {
			h.app.LogWarn("handleRTKWebhook: participantJoined with unparseable customParticipantId, ignoring",
				"meeting_id", event.Meeting.ID, "raw_participant_id", event.Participant.CustomParticipantID)
		} else {
			h.app.HandleWebhookParticipantJoined(event.Meeting.ID, callID, userID, event.Meeting.SessionID)
		}
	case "meeting.participantLeft":
		callID, userID, ok := rtkclient.ParseCustomParticipantID(event.Participant.CustomParticipantID)
		if !ok {
			h.app.LogWarn("handleRTKWebhook: participantLeft with unparseable customParticipantId, ignoring",
				"meeting_id", event.Meeting.ID, "raw_participant_id", event.Participant.CustomParticipantID)
		} else {
			h.app.HandleWebhookParticipantLeft(event.Meeting.ID, callID, userID)
		}
	case "meeting.ended":
		h.app.HandleWebhookMeetingEnded(event.Meeting.ID)
	default:
		// Unknown events are silently ignored per design.
	}

	w.WriteHeader(http.StatusOK)
}
