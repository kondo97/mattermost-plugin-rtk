package app

import (
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	pluginapi "github.com/mattermost/mattermost/server/public/plugin"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	"github.com/kondo97/mattermost-plugin-rtk/server/store"
)

// App is the business logic layer of the plugin, analogous to channels/app.App.
type App struct {
	store  store.Store
	rtk    rtkclient.RTKClient // may be nil when RTK credentials are not configured
	api    pluginapi.API
	callMu sync.Mutex
}

// New creates a new App instance.
func New(store store.Store, rtk rtkclient.RTKClient, api pluginapi.API) *App {
	return &App{store: store, rtk: rtk, api: api}
}

// UpdateRTKClient replaces the RTK client. Called by plugin.go on configuration change.
func (a *App) UpdateRTKClient(rtk rtkclient.RTKClient) {
	a.rtk = rtk
}

func (a *App) LogError(msg string, keyValuePairs ...any) {
	a.api.LogError(msg, keyValuePairs...)
}

func (a *App) LogWarn(msg string, keyValuePairs ...any) {
	a.api.LogWarn(msg, keyValuePairs...)
}

func (a *App) LogInfo(msg string, keyValuePairs ...any) {
	a.api.LogInfo(msg, keyValuePairs...)
}

func (a *App) LogDebug(msg string, keyValuePairs ...any) {
	a.api.LogDebug(msg, keyValuePairs...)
}

// HasPermissionTo checks if a user has the given permission.
func (a *App) HasPermissionTo(userID string, permission *model.Permission) bool {
	return a.api.HasPermissionTo(userID, permission)
}

// PublishWebSocketEvent publishes a WebSocket event to clients.
func (a *App) PublishWebSocketEvent(event string, payload map[string]any, broadcast *model.WebsocketBroadcast) {
	a.api.PublishWebSocketEvent(event, payload, broadcast)
}
