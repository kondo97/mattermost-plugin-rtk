package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kondo97/mattermost-plugin-rtk/server/app"
)

// channelStateResponse is the response shape for GET /api/v1/channels/{channelID}.
type channelStateResponse struct {
	ChannelID string `json:"channel_id"`
	Enabled   bool   `json:"enabled"`
}

// updateChannelRequest is the request body for PUT /api/v1/channels/{channelID}.
type updateChannelRequest struct {
	Enabled bool `json:"enabled"`
}

// handleGetChannel handles GET /api/v1/channels/{channelID}.
//
// Returns the calls-enabled state for a single channel. If no row exists in
// rtk_calls_channels the channel is treated as enabled by default.
func (h *API) handleGetChannel(w http.ResponseWriter, r *http.Request) {
	channelID := mux.Vars(r)["channelID"]

	ch, err := h.app.GetCallsChannel(channelID)
	if err != nil {
		h.app.LogError("handleGetChannel failed", "channel_id", channelID, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := channelStateResponse{ChannelID: channelID, Enabled: true}
	if ch != nil {
		resp.Enabled = ch.Enabled
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.app.LogError("handleGetChannel: failed to encode response", "error", err.Error())
	}
}

// handleUpdateChannel handles PUT /api/v1/channels/{channelID}.
//
// Sets the calls-enabled flag for the channel. Only channel admins, team
// admins, and system admins may call this endpoint.
func (h *API) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	channelID := mux.Vars(r)["channelID"]

	var req updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.app.UpdateCallsChannelEnabled(channelID, req.Enabled, userID); err != nil {
		if errors.Is(err, app.ErrForbidden) {
			writeError(w, http.StatusForbidden, "permission denied")
			return
		}
		h.app.LogError("handleUpdateChannel failed", "channel_id", channelID, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := channelStateResponse{ChannelID: channelID, Enabled: req.Enabled}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.app.LogError("handleUpdateChannel: failed to encode response", "error", err.Error())
	}
}

// handleGetAllChannels handles GET /api/v1/channels.
//
// Returns one entry per channel that has either a registered rtk_calls_channels
// row or an active call, filtered to channels the requesting user is a member
// of. The response shape mirrors the Calls plugin's GET /channels endpoint so
// existing Calls clients can be repointed at the RTK plugin without changes.
func (h *API) handleGetAllChannels(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")

	channels, err := h.app.GetAllCallChannels(userID)
	if err != nil {
		h.app.LogError("handleGetAllChannels failed", "user_id", userID, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(channels); err != nil {
		h.app.LogError("handleGetAllChannels: failed to encode response", "error", err.Error())
	}
}
