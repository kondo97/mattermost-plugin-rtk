package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kondo97/mattermost-plugin-rtk/server/app"
)

// handleCreateCall handles POST /api/v1/calls.
func (h *API) handleCreateCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")

	var req struct {
		ChannelID string `json:"channel_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ChannelID == "" {
		writeError(w, http.StatusBadRequest, "channel_id is required")
		return
	}

	session, token, err := h.app.CreateCall(req.ChannelID, userID)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrNotChannelMember):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, app.ErrRTKNotConfigured):
			writeError(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, app.ErrCallAlreadyActive):
			writeError(w, http.StatusConflict, err.Error())
		default:
			h.app.LogError("handleCreateCall failed", "channel_id", req.ChannelID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"call":  session,
		"token": token,
	})
}

// handleJoinCall handles POST /api/v1/calls/{id}/token.
func (h *API) handleJoinCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	session, token, err := h.app.JoinCall(callID, userID)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrNotChannelMember):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, app.ErrRTKNotConfigured):
			writeError(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, app.ErrCallNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			h.app.LogError("handleJoinCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	h.app.LogDebug("handleJoinCall success",
		"call_id", callID,
		"user_id", userID,
		"meeting_id", session.MeetingID,
		"token_len", fmt.Sprintf("%d", len(token)),
		"participants", fmt.Sprintf("%v", session.Participants),
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"call":  session,
		"token": token,
	})
}

// handleGetCall handles GET /api/v1/calls/{id}.
func (h *API) handleGetCall(w http.ResponseWriter, r *http.Request) {
	callID := mux.Vars(r)["id"]

	session, err := h.app.GetCallByID(callID)
	if err != nil {
		h.app.LogError("handleGetCall failed", "call_id", callID, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "call not found")
		return
	}

	if session.EndAt == 0 {
		// Perform an on-demand RTK reconciliation for active calls so that
		// a stale call is force-ended before the response is returned to the client.
		h.app.ReconcileCallOnDemand(session)
		// Re-fetch to reflect any state change from reconciliation.
		session, err = h.app.GetCallByID(callID)
		if err != nil {
			h.app.LogError("handleGetCall re-fetch failed", "call_id", callID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if session == nil {
			writeError(w, http.StatusNotFound, "call not found")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}

// handleLeaveCall handles POST /api/v1/calls/{id}/leave.
func (h *API) handleLeaveCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	if err := h.app.LeaveCall(callID, userID); err != nil {
		h.app.LogError("handleLeaveCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleEndCall handles DELETE /api/v1/calls/{id}.
func (h *API) handleEndCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	if err := h.app.EndCall(callID, userID); err != nil {
		switch {
		case errors.Is(err, app.ErrCallNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, app.ErrUnauthorized):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			h.app.LogError("handleEndCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
