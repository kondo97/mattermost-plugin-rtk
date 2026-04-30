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
	// CloudflareAccountID is the Cloudflare Account ID for the RealtimeKit integration.
	CloudflareAccountID string `json:"CloudflareAccountID"`
	// CloudflareAPIToken is the Cloudflare API Token for the RealtimeKit integration.
	CloudflareAPIToken string `json:"CloudflareAPIToken"`
}

// AccountIDFromEnv reports whether RTK_ACCOUNT_ID is set as an environment variable.
func (c *configuration) AccountIDFromEnv() bool {
	_, ok := os.LookupEnv("RTK_ACCOUNT_ID")
	return ok
}

// APITokenFromEnv reports whether RTK_API_TOKEN is set as an environment variable.
func (c *configuration) APITokenFromEnv() bool {
	_, ok := os.LookupEnv("RTK_API_TOKEN")
	return ok
}

// GetEffectiveAccountID returns the Cloudflare Account ID.
// Environment variable RTK_ACCOUNT_ID takes strict precedence over the stored config value.
func (c *configuration) GetEffectiveAccountID() string {
	if val, ok := os.LookupEnv("RTK_ACCOUNT_ID"); ok {
		return val
	}
	return c.CloudflareAccountID
}

// GetEffectiveAPIToken returns the Cloudflare API Token.
// Environment variable RTK_API_TOKEN takes strict precedence over the stored config value.
func (c *configuration) GetEffectiveAPIToken() string {
	if val, ok := os.LookupEnv("RTK_API_TOKEN"); ok {
		return val
	}
	return c.CloudflareAPIToken
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

	credentialsChanged := prev.GetEffectiveAccountID() != configuration.GetEffectiveAccountID() ||
		prev.GetEffectiveAPIToken() != configuration.GetEffectiveAPIToken()

	// Also re-initialize when the RTK client was never set up (e.g., EnsureApp failed at startup).
	// This allows recovery by saving configuration without changing credentials.
	rtkNotReady := p.application != nil && !p.application.IsConfigured()

	// p.application is nil during the initial OnConfigurationChange call that fires
	// before OnActivate. Defer credential validation to OnActivate in that case.
	if p.application == nil {
		return nil
	}

	if !credentialsChanged && !rtkNotReady {
		return nil
	}

	if configuration.GetEffectiveAccountID() == "" || configuration.GetEffectiveAPIToken() == "" {
		p.application.UpdateAccountClient(nil)
		p.application.UpdateRTKClient(nil)
		err := errors.New("RTK plugin configuration is incomplete: Cloudflare Account ID and API Token are required")
		p.API.LogError(err.Error())
		return err
	}

	newAccountClient := rtkclient.NewAccountClient(configuration.GetEffectiveAccountID(), configuration.GetEffectiveAPIToken())
	p.application.UpdateAccountClient(newAccountClient)

	appID, appConfigID, err := p.application.EnsureApp(configuration.GetEffectiveAccountID())
	if err != nil {
		p.application.UpdateRTKClient(nil)
		wrapped := errors.Wrap(err, "RTK plugin configuration update failed: EnsureApp failed")
		p.API.LogError(wrapped.Error())
		return wrapped
	}
	if appID == "" {
		p.application.UpdateRTKClient(nil)
		err := errors.New("RTK plugin configuration update failed: EnsureApp returned an empty app ID")
		p.API.LogError(err.Error())
		return err
	}

	newClient := rtkclient.NewClient(configuration.GetEffectiveAccountID(), appID, configuration.GetEffectiveAPIToken())
	p.application.UpdateRTKClient(newClient)
	p.application.ReRegisterWebhook(p.webhookURL(), appConfigID)

	return nil
}
