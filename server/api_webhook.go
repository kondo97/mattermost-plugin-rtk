package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
)

// rtkWebhookEvent is the top-level RTK webhook payload.
type rtkWebhookEvent struct {
	Event   string             `json:"event"`
	Meeting rtkWebhookMeeting  `json:"meeting"`
	Participant rtkWebhookParticipant `json:"participant"`
}

type rtkWebhookMeeting struct {
	ID string `json:"id"`
}

type rtkWebhookParticipant struct {
	CustomParticipantID string `json:"customParticipantId"`
}

// handleRTKWebhook handles POST /api/v1/webhook/rtk.
// Signature verification uses HMAC-SHA256 over the raw request body.
func (p *Plugin) handleRTKWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.API.LogError("handleRTKWebhook: failed to read body", "error", err.Error())
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	secret, err := p.kvStore.GetWebhookSecret()
	if err != nil {
		p.API.LogError("handleRTKWebhook: failed to get webhook secret", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !verifyRTKSignature(r.Header.Get("dyte-signature"), body, secret) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	var event rtkWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		p.API.LogError("handleRTKWebhook: failed to parse event", "error", err.Error())
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	switch event.Event {
	case "meeting.participantLeft":
		p.handleWebhookParticipantLeft(event)
	case "meeting.ended":
		p.handleWebhookMeetingEnded(event)
	default:
		// Unknown events are silently ignored per design.
	}

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) handleWebhookParticipantLeft(event rtkWebhookEvent) {
	meetingID := event.Meeting.ID
	userID := event.Participant.CustomParticipantID
	if meetingID == "" || userID == "" {
		return
	}

	session, err := p.kvStore.GetCallByMeetingID(meetingID)
	if err != nil {
		p.API.LogError("handleWebhookParticipantLeft: GetCallByMeetingID failed", "meeting_id", meetingID, "error", err.Error())
		return
	}
	if session == nil || session.EndAt != 0 {
		return // idempotent
	}

	if err := p.LeaveCall(session.ID, userID); err != nil {
		p.API.LogError("handleWebhookParticipantLeft: LeaveCall failed", "call_id", session.ID, "user_id", userID, "error", err.Error())
	}
}

func (p *Plugin) handleWebhookMeetingEnded(event rtkWebhookEvent) {
	meetingID := event.Meeting.ID
	if meetingID == "" {
		return
	}

	session, err := p.kvStore.GetCallByMeetingID(meetingID)
	if err != nil {
		p.API.LogError("handleWebhookMeetingEnded: GetCallByMeetingID failed", "meeting_id", meetingID, "error", err.Error())
		return
	}
	if session == nil || session.EndAt != 0 {
		return // idempotent
	}

	p.callMu.Lock()
	defer p.callMu.Unlock()

	if err := p.endCallInternal(session); err != nil {
		p.API.LogError("handleWebhookMeetingEnded: endCallInternal failed", "call_id", session.ID, "error", err.Error())
	}
}

// verifyRTKSignature verifies the HMAC-SHA256 signature from RTK.
// The signature is hex-encoded HMAC-SHA256(secret, body).
func verifyRTKSignature(signature string, body []byte, secret string) bool {
	if secret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
