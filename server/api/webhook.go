package api

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
	Event       string                `json:"event"`
	Meeting     rtkWebhookMeeting     `json:"meeting"`
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
func (h *API) handleRTKWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.app.LogError("handleRTKWebhook: failed to read body", "error", err.Error())
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	secret, err := h.app.GetWebhookSecret()
	if err != nil {
		h.app.LogError("handleRTKWebhook: failed to get webhook secret", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !verifyRTKSignature(r.Header.Get("dyte-signature"), body, secret) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
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
		h.app.HandleWebhookParticipantJoined(event.Meeting.ID, event.Participant.CustomParticipantID)
	case "meeting.participantLeft":
		h.app.HandleWebhookParticipantLeft(event.Meeting.ID, event.Participant.CustomParticipantID)
	case "meeting.ended":
		h.app.HandleWebhookMeetingEnded(event.Meeting.ID)
	default:
		// Unknown events are silently ignored per design.
	}

	w.WriteHeader(http.StatusOK)
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
