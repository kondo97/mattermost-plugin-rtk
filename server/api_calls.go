package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
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
func (p *Plugin) handleJoinCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	session, token, err := p.JoinCall(callID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRTKNotConfigured):
			writeError(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, ErrCallNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			p.API.LogError("handleJoinCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	p.API.LogDebug("handleJoinCall success",
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

// handleLeaveCall handles POST /api/v1/calls/{id}/leave.
func (p *Plugin) handleLeaveCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	if err := p.LeaveCall(callID, userID); err != nil {
		p.API.LogError("handleLeaveCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleForceEndCall handles DELETE /api/v1/calls/{id}/force.
// System admin only — forcibly ends a call regardless of creator.
func (p *Plugin) handleForceEndCall(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if !p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
		writeError(w, http.StatusForbidden, "system admin permission required")
		return
	}

	callID := mux.Vars(r)["id"]

	if err := p.ForceEndCall(callID); err != nil {
		switch {
		case errors.Is(err, ErrCallNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			p.API.LogError("handleForceEndCall failed", "call_id", callID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	p.API.LogWarn("call force-ended by admin", "call_id", callID, "admin_user_id", userID)
	w.WriteHeader(http.StatusOK)
}

// handleForceEndCallByChannel handles DELETE /api/v1/channels/{channelId}/calls/force.
// System admin only — forcibly ends the active call in a channel.
func (p *Plugin) handleForceEndCallByChannel(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if !p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
		writeError(w, http.StatusForbidden, "system admin permission required")
		return
	}

	channelID := mux.Vars(r)["channelId"]

	if err := p.ForceEndCallByChannel(channelID); err != nil {
		switch {
		case errors.Is(err, ErrCallNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			p.API.LogError("handleForceEndCallByChannel failed", "channel_id", channelID, "user_id", userID, "error", err.Error())
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	p.API.LogWarn("call force-ended by admin (channel)", "channel_id", channelID, "admin_user_id", userID)
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
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
