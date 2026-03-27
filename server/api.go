package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// initRouter initializes the HTTP router for the plugin.
func (p *Plugin) initRouter() *mux.Router {
	router := mux.NewRouter()

	// Static asset routes (no Mattermost auth required)
	router.HandleFunc("/call", p.serveCallHTML).Methods(http.MethodGet)
	router.HandleFunc("/call.js", p.serveCallJS).Methods(http.MethodGet)
	router.HandleFunc("/worker.js", p.serveWorkerJS).Methods(http.MethodGet)

	// RTK webhook route (RTK signature auth, not Mattermost auth)
	router.HandleFunc("/api/v1/webhook/rtk", p.handleRTKWebhook).Methods(http.MethodPost)

	// Authenticated API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(p.MattermostAuthorizationRequired)

	// Call management
	apiRouter.HandleFunc("/calls", p.handleCreateCall).Methods(http.MethodPost)
	apiRouter.HandleFunc("/calls/{id}/token", p.handleJoinCall).Methods(http.MethodPost)
	apiRouter.HandleFunc("/calls/{id}/leave", p.handleLeaveCall).Methods(http.MethodPost)
	apiRouter.HandleFunc("/calls/{id}", p.handleEndCall).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/calls/{id}/force", p.handleForceEndCall).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/channels/{channelId}/calls/force", p.handleForceEndCallByChannel).Methods(http.MethodDelete)

	// Config status
	apiRouter.HandleFunc("/config/status", p.handleConfigStatus).Methods(http.MethodGet)
	apiRouter.HandleFunc("/config/admin-status", p.handleAdminConfigStatus).Methods(http.MethodGet)

	// Mobile
	apiRouter.HandleFunc("/calls/{id}/dismiss", p.handleDismiss).Methods(http.MethodPost)

	return router
}

// ServeHTTP implements the plugin HTTP interface.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

// MattermostAuthorizationRequired is middleware that rejects unauthenticated requests.
func (p *Plugin) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
