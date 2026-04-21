package main

import (
	"os"
	"reflect"
	"strings"

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

	// Feature flags — all default to enabled (nil means not configured, treated as true).
	// Override via environment variables (e.g. RTK_RECORDING_ENABLED=false).
	RecordingEnabled     *bool `json:"RecordingEnabled"`
	ScreenShareEnabled   *bool `json:"ScreenShareEnabled"`
	PollsEnabled         *bool `json:"PollsEnabled"`
	TranscriptionEnabled *bool `json:"TranscriptionEnabled"`
	WaitingRoomEnabled   *bool `json:"WaitingRoomEnabled"`
	VideoEnabled         *bool `json:"VideoEnabled"`
	ChatEnabled          *bool `json:"ChatEnabled"`
	PluginsEnabled       *bool `json:"PluginsEnabled"`
	ParticipantsEnabled  *bool `json:"ParticipantsEnabled"`
	RaiseHandEnabled     *bool `json:"RaiseHandEnabled"`
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

// isFeatureFlagEnabled is the shared logic for all Is*Enabled() methods.
// It checks the env var first, then the *bool field, defaulting to true if nil.
func isFeatureFlagEnabled(envVar string, field *bool) bool {
	if val, ok := os.LookupEnv(envVar); ok {
		return strings.EqualFold(val, "true")
	}
	if field == nil {
		return true // default ON
	}
	return *field
}

// IsRecordingEnabled reports whether the recording feature is enabled.
func (c *configuration) IsRecordingEnabled() bool {
	return isFeatureFlagEnabled("RTK_RECORDING_ENABLED", c.RecordingEnabled)
}

// IsScreenShareEnabled reports whether the screen share feature is enabled.
func (c *configuration) IsScreenShareEnabled() bool {
	return isFeatureFlagEnabled("RTK_SCREEN_SHARE_ENABLED", c.ScreenShareEnabled)
}

// IsPollsEnabled reports whether the polls feature is enabled.
func (c *configuration) IsPollsEnabled() bool {
	return isFeatureFlagEnabled("RTK_POLLS_ENABLED", c.PollsEnabled)
}

// IsTranscriptionEnabled reports whether the transcription feature is enabled.
func (c *configuration) IsTranscriptionEnabled() bool {
	return isFeatureFlagEnabled("RTK_TRANSCRIPTION_ENABLED", c.TranscriptionEnabled)
}

// IsWaitingRoomEnabled reports whether the waiting room feature is enabled.
// Defaults to false (opt-in) unlike other feature flags.
func (c *configuration) IsWaitingRoomEnabled() bool {
	if val, ok := os.LookupEnv("RTK_WAITING_ROOM_ENABLED"); ok {
		return strings.EqualFold(val, "true")
	}
	if c.WaitingRoomEnabled == nil {
		return false // default OFF
	}
	return *c.WaitingRoomEnabled
}

// IsVideoEnabled reports whether the video feature is enabled.
func (c *configuration) IsVideoEnabled() bool {
	return isFeatureFlagEnabled("RTK_VIDEO_ENABLED", c.VideoEnabled)
}

// IsChatEnabled reports whether the in-call chat feature is enabled.
func (c *configuration) IsChatEnabled() bool {
	return isFeatureFlagEnabled("RTK_CHAT_ENABLED", c.ChatEnabled)
}

// IsPluginsEnabled reports whether the plugins feature is enabled.
func (c *configuration) IsPluginsEnabled() bool {
	return isFeatureFlagEnabled("RTK_PLUGINS_ENABLED", c.PluginsEnabled)
}

// IsParticipantsEnabled reports whether the participants panel feature is enabled.
func (c *configuration) IsParticipantsEnabled() bool {
	return isFeatureFlagEnabled("RTK_PARTICIPANTS_ENABLED", c.ParticipantsEnabled)
}

// IsRaiseHandEnabled reports whether the raise hand feature is enabled.
func (c *configuration) IsRaiseHandEnabled() bool {
	return isFeatureFlagEnabled("RTK_RAISE_HAND_ENABLED", c.RaiseHandEnabled)
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

	if credentialsChanged {
		if configuration.GetEffectiveOrgID() != "" && configuration.GetEffectiveAPIKey() != "" {
			p.rtkClient = rtkclient.NewClient(configuration.GetEffectiveOrgID(), configuration.GetEffectiveAPIKey())
			if p.kvStore != nil {
				p.reRegisterWebhook()
			}
		} else {
			p.rtkClient = nil
		}
	}

	return nil
}
