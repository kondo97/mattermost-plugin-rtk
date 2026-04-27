package app

import (
	"errors"
	"fmt"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
)

// rtkWebhookEvents are the RTK webhook events this plugin subscribes to.
var rtkWebhookEvents = []string{"meeting.participantLeft", "meeting.ended"}

// RegisterWebhookIfNeeded ensures a valid RTK webhook is registered and its ID stored.
//
// Flow:
//  1. If an ID is stored, verify the webhook still exists on the RTK side via GET /webhooks/{id}.
//     If it does, nothing to do. If it was deleted (404), fall through to re-registration.
//  2. If no ID is stored, attempt RegisterWebhook.
//     On 409 (same URL already registered), resolve the conflict by listing all webhooks,
//     deleting the matching entry, then re-registering.
//
// This is best-effort: errors are logged but not returned to avoid blocking activation.
func (a *App) RegisterWebhookIfNeeded(webhookURL string, appConfigID string) {
	existingID, err := a.store.GetWebhookConfig()
	if err != nil {
		a.api.LogWarn("Failed to check existing webhook config", "error", err.Error())
	}

	// Fast path: ID is stored — verify the webhook still exists on RTK.
	if existingID != "" {
		if _, err := a.rtk.GetWebhook(existingID); err == nil {
			return // webhook is valid; nothing to do
		} else if !errors.Is(err, rtkclient.ErrWebhookNotFound) {
			a.api.LogWarn("Failed to verify existing RTK webhook; skipping re-registration", "webhook_id", existingID, "error", err.Error())
			return
		}
		// Webhook was deleted on the RTK side; clear stale ID and re-register.
		a.api.LogInfo("Stored RTK webhook no longer exists; re-registering", "webhook_id", existingID)
		if err := a.store.ClearWebhookConfig(appConfigID); err != nil {
			a.api.LogWarn("Failed to clear stale RTK webhook config", "error", err.Error())
		}
	}

	if webhookURL == "" {
		a.api.LogWarn("SiteURL not configured; skipping RTK webhook registration")
		return
	}

	id, err := a.rtk.RegisterWebhook(webhookURL, rtkWebhookEvents)
	if errors.Is(err, rtkclient.ErrWebhookConflict) {
		a.api.LogInfo("RTK webhook already exists; resolving conflict by deleting and re-registering", "url", webhookURL)
		if deleteErr := a.deleteWebhookByURL(webhookURL); deleteErr != nil {
			a.api.LogWarn("Failed to resolve webhook conflict", "error", deleteErr.Error())
			return
		}
		id, err = a.rtk.RegisterWebhook(webhookURL, rtkWebhookEvents)
	}
	if err != nil {
		a.api.LogWarn("Failed to register RTK webhook", "error", err.Error())
		return
	}

	if id == "" {
		a.api.LogWarn("RegisterWebhook returned empty id; skipping store")
		return
	}

	if err := a.store.StoreWebhookConfig(appConfigID, id); err != nil {
		a.api.LogWarn("Failed to store RTK webhook config", "error", err.Error())
		return
	}
	a.api.LogInfo("RTK webhook registered and stored", "webhook_id", id)
}

// deleteWebhookByURL lists all registered RTK webhooks and deletes the one matching url.
// Returns an error if listing fails or no matching webhook is found.
func (a *App) deleteWebhookByURL(url string) error {
	webhooks, err := a.rtk.ListWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list RTK webhooks: %w", err)
	}
	a.api.LogInfo("deleteWebhookByURL: listed RTK webhooks", "count", len(webhooks), "target_url", url)
	for _, wh := range webhooks {
		a.api.LogInfo("deleteWebhookByURL: found webhook", "id", wh.ID, "url", wh.URL)
		if wh.URL == url {
			if err := a.rtk.DeleteWebhook(wh.ID); err != nil {
				return fmt.Errorf("failed to delete conflicting RTK webhook %s: %w", wh.ID, err)
			}
			return nil
		}
	}
	return fmt.Errorf("no RTK webhook found with URL %s", url)
}

// ReRegisterWebhook deletes the existing RTK webhook (if any) and registers a fresh one.
// Called when credentials change. Best-effort: errors are logged but not returned.
func (a *App) ReRegisterWebhook(webhookURL string, appConfigID string) {
	existingID, err := a.store.GetWebhookConfig()
	if err != nil {
		a.api.LogWarn("Failed to get existing webhook config for re-registration", "error", err.Error())
	}
	if existingID != "" {
		if err := a.rtk.DeleteWebhook(existingID); err != nil {
			a.api.LogWarn("Failed to delete old RTK webhook", "webhookID", existingID, "error", err.Error())
		}
		if err := a.store.ClearWebhookConfig(appConfigID); err != nil {
			a.api.LogWarn("Failed to clear RTK webhook config", "error", err.Error())
		}
	}
	a.RegisterWebhookIfNeeded(webhookURL, appConfigID)
}
