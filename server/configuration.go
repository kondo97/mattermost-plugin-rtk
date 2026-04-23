package main

import (
	"os"
	"reflect"

	"github.com/pkg/errors"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	// CloudflareOrgID is the Cloudflare Organization ID for the RealtimeKit integration.
	CloudflareOrgID string `json:"CloudflareOrgID"`
	// CloudflareAPIKey is the Cloudflare API Key for the RealtimeKit integration.
	CloudflareAPIKey string `json:"CloudflareAPIKey"`
}

// OrgIDFromEnv reports whether RTK_ORG_ID is set as an environment variable.
func (c *configuration) OrgIDFromEnv() bool {
	_, ok := os.LookupEnv("RTK_ORG_ID")
	return ok
}

// APIKeyFromEnv reports whether RTK_API_KEY is set as an environment variable.
func (c *configuration) APIKeyFromEnv() bool {
	_, ok := os.LookupEnv("RTK_API_KEY")
	return ok
}

// GetEffectiveOrgID returns the Cloudflare Organization ID.
// Environment variable RTK_ORG_ID takes strict precedence over the stored config value.
func (c *configuration) GetEffectiveOrgID() string {
	if val, ok := os.LookupEnv("RTK_ORG_ID"); ok {
		return val
	}
	return c.CloudflareOrgID
}

// GetEffectiveAPIKey returns the Cloudflare API Key.
// Environment variable RTK_API_KEY takes strict precedence over the stored config value.
func (c *configuration) GetEffectiveAPIKey() string {
	if val, ok := os.LookupEnv("RTK_API_KEY"); ok {
		return val
	}
	return c.CloudflareAPIKey
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	clone := *c
	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	prev := p.getConfiguration()

	configuration := new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	p.setConfiguration(configuration)

	credentialsChanged := prev.GetEffectiveOrgID() != configuration.GetEffectiveOrgID() ||
		prev.GetEffectiveAPIKey() != configuration.GetEffectiveAPIKey()

	if credentialsChanged && p.application != nil {
		if configuration.GetEffectiveOrgID() != "" && configuration.GetEffectiveAPIKey() != "" {
			newClient := rtkclient.NewClient(configuration.GetEffectiveOrgID(), configuration.GetEffectiveAPIKey())
			p.application.UpdateRTKClient(newClient)
			p.application.ReRegisterWebhook(p.webhookURL())
		} else {
			p.application.UpdateRTKClient(nil)
		}
	}

	return nil
}
