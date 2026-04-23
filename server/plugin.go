package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	rtapi "github.com/kondo97/mattermost-plugin-rtk/server/api"
	"github.com/kondo97/mattermost-plugin-rtk/server/app"
	"github.com/kondo97/mattermost-plugin-rtk/server/command"
	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	"github.com/kondo97/mattermost-plugin-rtk/server/store/sqlstore"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// client is the Mattermost server API client.
	client *pluginapi.Client

	// commandClient is the client used to register and execute slash commands.
	commandClient command.Command

	// application is the business logic layer.
	application *app.App

	// apiHandler is the HTTP API layer.
	apiHandler *rtapi.API

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	db, err := p.client.Store.GetMasterDB()
	if err != nil {
		return fmt.Errorf("failed to get master DB: %w", err)
	}
	store, err := sqlstore.NewStore(db)
	if err != nil {
		return fmt.Errorf("failed to create SQL store: %w", err)
	}
	if err := store.RunMigrations(); err != nil {
		return fmt.Errorf("failed to run SQL migrations: %w", err)
	}

	p.commandClient = command.NewCommandHandler(p.client)

	cfg := p.getConfiguration()
	var rtkClient rtkclient.RTKClient
	if cfg.GetEffectiveOrgID() != "" && cfg.GetEffectiveAPIKey() != "" {
		rtkClient = rtkclient.NewClient(cfg.GetEffectiveOrgID(), cfg.GetEffectiveAPIKey())
	}

	p.application = app.New(store, rtkClient, p.API)

	if rtkClient != nil {
		p.application.RegisterWebhookIfNeeded(p.webhookURL())
	}

	p.apiHandler = rtapi.Init(p.application, rtapi.StaticFiles{
		CallHTML: callHTML,
		CallJS:   callJS,
		WorkerJS: workerJS,
	}, p.configStatus)

	return nil
}

// webhookURL returns the full RTK webhook callback URL for this plugin instance.
func (p *Plugin) webhookURL() string {
	siteURL := ""
	if cfg := p.API.GetConfig(); cfg != nil && cfg.ServiceSettings.SiteURL != nil {
		siteURL = *cfg.ServiceSettings.SiteURL
	}
	return fmt.Sprintf("%s/plugins/%s/api/v1/webhook/rtk", siteURL, manifest.Id)
}

// configStatus returns the current plugin configuration state for the API layer.
func (p *Plugin) configStatus() rtapi.ConfigStatus {
	cfg := p.getConfiguration()
	return rtapi.ConfigStatus{
		Enabled:      cfg.GetEffectiveOrgID() != "" && cfg.GetEffectiveAPIKey() != "",
		OrgIDViaEnv:  cfg.OrgIDFromEnv(),
		APIKeyViaEnv: cfg.APIKeyFromEnv(),
		OrgID:        cfg.CloudflareOrgID,
	}
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	return nil
}

// ServeHTTP implements the plugin HTTP interface.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.apiHandler.ServeHTTP(w, r)
}

// ExecuteCommand hook calls this method to execute the commands that were registered in the NewCommandHandler function.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	response, err := p.commandClient.Handle(args)
	if err != nil {
		return nil, model.NewAppError("ExecuteCommand", "plugin.command.execute_command.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	return response, nil
}

// NotificationWillBePushed delegates push notification handling to the app layer.
func (p *Plugin) NotificationWillBePushed(notification *model.PushNotification, userID string) (*model.PushNotification, string) {
	return p.application.NotificationWillBePushed(notification, userID)
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
