package app

import (
	"context"
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	pluginapi "github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	"github.com/kondo97/mattermost-plugin-rtk/server/store"
)

// App is the business logic layer of the plugin, analogous to channels/app.App.
type App struct {
	store   store.Store
	rtk     rtkclient.RTKClient     // may be nil when RTK credentials are not configured
	account rtkclient.AccountClient // may be nil when RTK credentials are not configured
	api     pluginapi.API
	callMu  sync.Mutex
}

// New creates a new App instance.
func New(store store.Store, rtk rtkclient.RTKClient, account rtkclient.AccountClient, api pluginapi.API) *App {
	return &App{store: store, rtk: rtk, account: account, api: api}
}

// UpdateRTKClient replaces the RTK client. Called by plugin.go on configuration change.
func (a *App) UpdateRTKClient(rtk rtkclient.RTKClient) {
	a.rtk = rtk
}

// UpdateAccountClient replaces the account-level RTK client. Called by plugin.go on configuration change.
func (a *App) UpdateAccountClient(account rtkclient.AccountClient) {
	a.account = account
}

// IsConfigured reports whether the RTK client has been successfully initialized.
func (a *App) IsConfigured() bool {
	return a.rtk != nil
}

// EnsureApp creates or recovers the single RTK app for this Mattermost instance.
// It stores the app ID in the database to survive plugin restarts.
// Returns the app ID and the ID of the active app config row on success.
//
// The whole operation is serialized across cluster nodes via a PostgreSQL
// advisory lock keyed on the deterministic app name. This prevents two nodes
// from racing on Cloudflare's CreateApp endpoint and creating duplicate apps.
func (a *App) EnsureApp(accountID string) (string, string, error) {
	if a.account == nil {
		return "", "", errors.New("EnsureApp: account client is not configured")
	}

	appName := a.rtkAppName()

	var (
		appID       string
		appConfigID string
		innerErr    error
	)
	if err := a.store.WithAppLock(context.Background(), appName, func() error {
		appID, appConfigID, innerErr = a.ensureAppLocked(accountID, appName)
		return innerErr
	}); err != nil {
		return appID, appConfigID, err
	}
	return appID, appConfigID, nil
}

// ensureAppLocked runs the ListApps → (CreateApp) → StoreAppConfig sequence under
// the caller-held advisory lock.
func (a *App) ensureAppLocked(accountID, appName string) (string, string, error) {
	// The Cloudflare RTK API does not provide a GET /apps/{id} endpoint.
	// Always list all apps and find by deterministic name.
	apps, err := a.account.ListApps()
	if err != nil {
		return "", "", errors.Wrap(err, "EnsureApp: ListApps failed")
	}
	for _, app := range apps {
		if app.Name == appName {
			appConfigID, err := a.store.StoreAppConfig(accountID, app.ID)
			if err != nil {
				a.api.LogWarn("EnsureApp: failed to store app config", "error", err.Error())
			}
			return app.ID, appConfigID, nil
		}
	}

	// No existing app found — create a new one.
	app, err := a.account.CreateApp(appName)
	if err != nil {
		return "", "", errors.Wrap(err, "EnsureApp: CreateApp failed")
	}
	appConfigID, err := a.store.StoreAppConfig(accountID, app.ID)
	if err != nil {
		a.api.LogWarn("EnsureApp: failed to store app config", "error", err.Error())
	}
	return app.ID, appConfigID, nil
}

// rtkAppName returns the deterministic RTK app name derived from the Mattermost site URL.
// Using the site URL ensures the name is unique per Mattermost deployment and human-readable
// in the Cloudflare dashboard.
func (a *App) rtkAppName() string {
	siteURL := ""
	if cfg := a.api.GetConfig(); cfg != nil && cfg.ServiceSettings.SiteURL != nil {
		siteURL = *cfg.ServiceSettings.SiteURL
	}
	siteURL = strings.TrimPrefix(siteURL, "https://")
	siteURL = strings.TrimPrefix(siteURL, "http://")
	siteURL = strings.TrimRight(siteURL, "/")
	return "mm-" + siteURL
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
