package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

// handleCreateCall handles POST /api/v1/calls.
func (p *Plugin) handleCreateCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")

	var req struct {
		ChannelID string `json:"channel_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ChannelID == "" {
		writeError(w, http.StatusBadRequest, "channel_id is required")
		return
	}

	session, token, err := p.CreateCall(req.ChannelID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRTKNotConfigured):
			writeError(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, ErrCallAlreadyActive):
			writeError(w, http.StatusConflict, err.Error())
		default:
			p.API.LogError("handleCreateCall failed", "channel_id", req.ChannelID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, err.Error())
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
func (p *Plugin) handleJoinCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	token, err := p.JoinCall(callID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRTKNotConfigured):
			writeError(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, ErrCallNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			p.API.LogError("handleJoinCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	session, err := p.kvStore.GetCallByID(callID)
	if err != nil {
		p.API.LogError("handleJoinCall: GetCallByID failed", "call_id", callID, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "failed to fetch call")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"call":  session,
		"token": token,
	})
}

// handleLeaveCall handles POST /api/v1/calls/{id}/leave.
func (p *Plugin) handleLeaveCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	if err := p.LeaveCall(callID, userID); err != nil {
		p.API.LogError("handleLeaveCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleEndCall handles DELETE /api/v1/calls/{id}.
func (p *Plugin) handleEndCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	if err := p.EndCall(callID, userID); err != nil {
		switch {
		case errors.Is(err, ErrCallNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrUnauthorized):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			p.API.LogError("handleEndCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
