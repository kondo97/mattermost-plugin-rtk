package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/kondo97/mattermost-plugin-rtk/server/command"
	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
	"github.com/kondo97/mattermost-plugin-rtk/server/store/sqlstore"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// kvStore is the client used to read/write KV records for this plugin.
	kvStore kvstore.KVStore

	// rtkClient is the Cloudflare RealtimeKit API client.
	// May be nil if credentials are not yet configured.
	rtkClient rtkclient.RTKClient

	// client is the Mattermost server API client.
	client *pluginapi.Client

	// commandClient is the client used to register and execute slash commands.
	commandClient command.Command

	// router is the HTTP router for handling API requests.
	router *mux.Router

	// callMu guards call state mutations (CreateCall, JoinCall, LeaveCall, EndCall).
	callMu sync.Mutex

	// stopCleanup signals the cleanup goroutine to stop.
	stopCleanup chan struct{}

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// rtkWebhookEvents are the RTK webhook events this plugin subscribes to.
var rtkWebhookEvents = []string{"meeting.participantLeft", "meeting.ended"}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	db, err := p.client.Store.GetMasterDB()
	if err != nil {
		return fmt.Errorf("failed to get master DB: %w", err)
	}
	store, err := sqlstore.NewStore(db, p.client.Store.DriverName())
	if err != nil {
		return fmt.Errorf("failed to create SQL store: %w", err)
	}
	if err := store.RunMigrations(); err != nil {
		return fmt.Errorf("failed to run SQL migrations: %w", err)
	}
	p.kvStore = store

	p.commandClient = command.NewCommandHandler(p.client)

	cfg := p.getConfiguration()
	if cfg.GetEffectiveOrgID() != "" && cfg.GetEffectiveAPIKey() != "" {
		p.rtkClient = rtkclient.NewClient(cfg.GetEffectiveOrgID(), cfg.GetEffectiveAPIKey())
		p.registerWebhookIfNeeded()
	}

	p.router = p.initRouter()

	p.stopCleanup = make(chan struct{})
	go p.runCleanupLoop(p.stopCleanup)

	return nil
}

// registerWebhookIfNeeded registers the RTK webhook if one is not already registered.
// Both the webhook ID and the signing secret must be present; if either is missing the
// webhook is (re-)registered so that signature verification can succeed.
// This is best-effort: errors are logged but not returned to avoid blocking activation.
func (p *Plugin) registerWebhookIfNeeded() {
	existingID, err := p.kvStore.GetWebhookID()
	if err != nil {
		p.API.LogWarn("Failed to check existing webhook ID", "error", err.Error())
	}
	existingSecret, err := p.kvStore.GetWebhookSecret()
	if err != nil {
		p.API.LogWarn("Failed to check existing webhook secret", "error", err.Error())
	}
	if existingID != "" && existingSecret != "" {
		return
	}

	siteURL := ""
	if cfg := p.API.GetConfig(); cfg != nil && cfg.ServiceSettings.SiteURL != nil {
		siteURL = *cfg.ServiceSettings.SiteURL
	}
	if siteURL == "" {
		p.API.LogWarn("SiteURL not configured; skipping RTK webhook registration")
		return
	}

	webhookURL := fmt.Sprintf("%s/plugins/%s/api/v1/webhook/rtk", siteURL, manifest.Id)
	id, secret, err := p.rtkClient.RegisterWebhook(webhookURL, rtkWebhookEvents)
	if err != nil {
		p.API.LogWarn("Failed to register RTK webhook", "error", err.Error())
		return
	}

	if err := p.kvStore.StoreWebhookID(id); err != nil {
		p.API.LogWarn("Failed to store RTK webhook ID", "error", err.Error())
	}
	if err := p.kvStore.StoreWebhookSecret(secret); err != nil {
		p.API.LogWarn("Failed to store RTK webhook secret", "error", err.Error())
	}
}

// reRegisterWebhook deletes the existing RTK webhook (if any) and registers a fresh one.
// Called when credentials change. Best-effort: errors are logged but not returned.
func (p *Plugin) reRegisterWebhook() {
	existingID, err := p.kvStore.GetWebhookID()
	if err != nil {
		p.API.LogWarn("Failed to get existing webhook ID for re-registration", "error", err.Error())
	}
	if existingID != "" {
		if err := p.rtkClient.DeleteWebhook(existingID); err != nil {
			p.API.LogWarn("Failed to delete old RTK webhook", "webhookID", existingID, "error", err.Error())
		}
		if err := p.kvStore.StoreWebhookID(""); err != nil {
			p.API.LogWarn("Failed to clear RTK webhook ID", "error", err.Error())
		}
		if err := p.kvStore.StoreWebhookSecret(""); err != nil {
			p.API.LogWarn("Failed to clear RTK webhook secret", "error", err.Error())
		}
	}
	p.registerWebhookIfNeeded()
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	if p.stopCleanup != nil {
		close(p.stopCleanup)
		p.stopCleanup = nil
	}
	return nil
}

// ExecuteCommand hook calls this method to execute the commands that were registered in the NewCommandHandler function.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	response, err := p.commandClient.Handle(args)
	if err != nil {
		return nil, model.NewAppError("ExecuteCommand", "plugin.command.execute_command.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	return response, nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
