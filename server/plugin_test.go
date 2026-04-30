package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kondo97/mattermost-plugin-rtk/server/app"
)

// allowAnyLogs registers permissive expectations for any Log* calls on the API mock.
func allowAnyLogs(api *plugintest.API) {
	anyArgs := func(n int) []any {
		args := make([]any, n)
		for i := range args {
			args[i] = mock.Anything
		}
		return args
	}
	for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		api.On("LogDebug", anyArgs(n)...).Maybe().Return()
		api.On("LogInfo", anyArgs(n)...).Maybe().Return()
		api.On("LogWarn", anyArgs(n)...).Maybe().Return()
		api.On("LogError", anyArgs(n)...).Maybe().Return()
	}
}

// loadEmptyConfig stubs LoadPluginConfiguration to leave the configuration zero-valued.
func loadEmptyConfig(api *plugintest.API) {
	api.On("LoadPluginConfiguration", mock.Anything).Return(nil)
}

func newPluginWithAPI(api *plugintest.API) *Plugin {
	p := &Plugin{}
	p.SetAPI(api)
	return p
}

// When the plugin has not been activated yet (application == nil), OnConfigurationChange
// must succeed without performing credential validation. OnActivate is the authoritative
// validation point.
func TestOnConfigurationChange_BeforeActivate_NoError(t *testing.T) {
	api := &plugintest.API{}
	allowAnyLogs(api)
	loadEmptyConfig(api)

	p := newPluginWithAPI(api)

	require.NoError(t, p.OnConfigurationChange())
	api.AssertExpectations(t)
}

// After activation, when credentials become empty, OnConfigurationChange must return
// an error so Mattermost rejects the configuration update.
func TestOnConfigurationChange_AfterActivate_MissingCredentials_ReturnsError(t *testing.T) {
	api := &plugintest.API{}
	allowAnyLogs(api)
	loadEmptyConfig(api)

	p := newPluginWithAPI(api)
	// Simulate that activation succeeded earlier with non-empty credentials.
	p.application = app.New(nil, nil, nil, api)
	p.setConfiguration(&configuration{
		CloudflareAccountID: "prev-account",
		CloudflareAPIToken:  "prev-token",
	})

	err := p.OnConfigurationChange()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Account ID and API Token are required")
	api.AssertExpectations(t)
}


