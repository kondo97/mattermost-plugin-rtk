package app

import (
	"errors"
	"fmt"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
)

// rtkWebhookEvents are the RTK webhook events this plugin subscribes to.
var rtkWebhookEvents = []string{"meeting.participantLeft", "meeting.ended"}

// RegisterWebhookIfNeeded ensures a valid RTK webhook is registered and its credentials stored.
//
// Flow:
//  1. If both ID and Secret are stored, verify the webhook still exists on the RTK side
//     via GET /webhooks/{id}. If it does, nothing to do. If it was deleted (404), fall
//     through to re-registration.
//  2. If ID or Secret is missing, attempt RegisterWebhook.
//     On 409 (same URL already registered), resolve the conflict by listing all webhooks,
//     deleting the matching entry, then re-registering.
//
// This is best-effort: errors are logged but not returned to avoid blocking activation.
func (a *App) RegisterWebhookIfNeeded(webhookURL string) {
	existingID, err := a.store.GetWebhookID()
	if err != nil {
		a.api.LogWarn("Failed to check existing webhook ID", "error", err.Error())
	}
	existingSecret, err := a.store.GetWebhookSecret()
	if err != nil {
		a.api.LogWarn("Failed to check existing webhook secret", "error", err.Error())
	}

	// Fast path: ID and Secret are stored — verify the webhook still exists on RTK.
	if existingID != "" && existingSecret != "" {
		if _, err := a.rtk.GetWebhook(existingID); err == nil {
			return // webhook is valid; nothing to do
		} else if !errors.Is(err, rtkclient.ErrWebhookNotFound) {
			a.api.LogWarn("Failed to verify existing RTK webhook; skipping re-registration", "webhook_id", existingID, "error", err.Error())
			return
		}
		// Webhook was deleted on the RTK side; clear stale credentials and re-register.
		a.api.LogInfo("Stored RTK webhook no longer exists; re-registering", "webhook_id", existingID)
		if err := a.store.StoreWebhookID(""); err != nil {
			a.api.LogWarn("Failed to clear stale RTK webhook ID", "error", err.Error())
		}
		if err := a.store.StoreWebhookSecret(""); err != nil {
			a.api.LogWarn("Failed to clear stale RTK webhook secret", "error", err.Error())
		}
	}

	if webhookURL == "" {
		a.api.LogWarn("SiteURL not configured; skipping RTK webhook registration")
		return
	}

	id, secret, err := a.rtk.RegisterWebhook(webhookURL, rtkWebhookEvents)
	if errors.Is(err, rtkclient.ErrWebhookConflict) {
		a.api.LogInfo("RTK webhook already exists; resolving conflict by deleting and re-registering", "url", webhookURL)
		if deleteErr := a.deleteWebhookByURL(webhookURL); deleteErr != nil {
			a.api.LogWarn("Failed to resolve webhook conflict", "error", deleteErr.Error())
			return
		}
		id, secret, err = a.rtk.RegisterWebhook(webhookURL, rtkWebhookEvents)
	}
	if err != nil {
		a.api.LogWarn("Failed to register RTK webhook", "error", err.Error())
		return
	}

	if err := a.store.StoreWebhookID(id); err != nil {
		a.api.LogWarn("Failed to store RTK webhook ID", "error", err.Error())
	}
	if err := a.store.StoreWebhookSecret(secret); err != nil {
		a.api.LogWarn("Failed to store RTK webhook secret", "error", err.Error())
	}
}

// deleteWebhookByURL lists all registered RTK webhooks and deletes the one matching url.
// Returns an error if listing fails or no matching webhook is found.
func (a *App) deleteWebhookByURL(url string) error {
	webhooks, err := a.rtk.ListWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list RTK webhooks: %w", err)
	}
	for _, wh := range webhooks {
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
func (a *App) ReRegisterWebhook(webhookURL string) {
	existingID, err := a.store.GetWebhookID()
	if err != nil {
		a.api.LogWarn("Failed to get existing webhook ID for re-registration", "error", err.Error())
	}
	if existingID != "" {
		if err := a.rtk.DeleteWebhook(existingID); err != nil {
			a.api.LogWarn("Failed to delete old RTK webhook", "webhookID", existingID, "error", err.Error())
		}
		if err := a.store.StoreWebhookID(""); err != nil {
			a.api.LogWarn("Failed to clear RTK webhook ID", "error", err.Error())
		}
		if err := a.store.StoreWebhookSecret(""); err != nil {
			a.api.LogWarn("Failed to clear RTK webhook secret", "error", err.Error())
		}
	}
	a.RegisterWebhookIfNeeded(webhookURL)
}
