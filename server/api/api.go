package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kondo97/mattermost-plugin-rtk/server/app"
)

// ConfigStatus describes the current RTK plugin configuration state.
type ConfigStatus struct {
	Enabled      bool
	OrgIDViaEnv  bool
	APIKeyViaEnv bool
	OrgID        string
}

// StaticFiles holds the embedded static assets to serve.
type StaticFiles struct {
	CallHTML []byte
	CallJS   []byte
	WorkerJS []byte
}

// API is the HTTP layer of the plugin, analogous to channels/api4.API.
type API struct {
	app      *app.App
	router   *mux.Router
	static   StaticFiles
	configFn func() ConfigStatus
}

// Init creates a new API instance and configures its routes.
func Init(a *app.App, static StaticFiles, configFn func() ConfigStatus) *API {
	h := &API{app: a, static: static, configFn: configFn}
	h.initRouter()
	return h
}

// ServeHTTP implements http.Handler.
func (h *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

// MattermostAuthorizationRequired is middleware that rejects unauthenticated requests.
func (h *API) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *API) initRouter() {
	router := mux.NewRouter()

	// Static asset routes (no Mattermost auth required)
	router.HandleFunc("/call", h.serveCallHTML).Methods(http.MethodGet)
	router.HandleFunc("/call.js", h.serveCallJS).Methods(http.MethodGet)
	router.HandleFunc("/worker.js", h.serveWorkerJS).Methods(http.MethodGet)

	// RTK webhook route (RTK signature auth, not Mattermost auth)
	router.HandleFunc("/api/v1/webhook/rtk", h.handleRTKWebhook).Methods(http.MethodPost)

	// Authenticated API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(h.MattermostAuthorizationRequired)

	// Call management
	apiRouter.HandleFunc("/calls", h.handleCreateCall).Methods(http.MethodPost)
	apiRouter.HandleFunc("/calls/{id}", h.handleGetCall).Methods(http.MethodGet)
	apiRouter.HandleFunc("/calls/{id}/token", h.handleJoinCall).Methods(http.MethodPost)
	apiRouter.HandleFunc("/calls/{id}/leave", h.handleLeaveCall).Methods(http.MethodPost)
	apiRouter.HandleFunc("/calls/{id}", h.handleEndCall).Methods(http.MethodDelete)

	// Config status
	apiRouter.HandleFunc("/config/status", h.handleConfigStatus).Methods(http.MethodGet)
	apiRouter.HandleFunc("/config/admin-status", h.handleAdminConfigStatus).Methods(http.MethodGet)

	// Mobile
	apiRouter.HandleFunc("/calls/{id}/dismiss", h.handleDismiss).Methods(http.MethodPost)

	h.router = router
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
