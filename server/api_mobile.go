package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
)

const wsEventNotificationDismissed = "notification_dismissed"

// handleDismiss handles POST /api/v1/calls/{id}/dismiss.
// Emits a WebSocket event to the requesting user only, then returns 200 OK.
// This is idempotent — no call state is modified.
func (p *Plugin) handleDismiss(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	callID := mux.Vars(r)["id"]

	p.API.PublishWebSocketEvent(wsEventNotificationDismissed, map[string]any{
		"call_id": callID,
		"user_id": userID,
	}, &model.WebsocketBroadcast{UserId: userID})

	w.WriteHeader(http.StatusOK)
}
