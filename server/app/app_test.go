package app

import (
	"context"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	storemocks "github.com/kondo97/mattermost-plugin-rtk/server/store/mocks"
)

// fakeAccountClient is a hand-rolled stub for rtkclient.AccountClient so app-layer
// tests don't need a generated mock.
type fakeAccountClient struct {
	apps          []rtkclient.App
	listErr       error
	createCalls   int
	createApp     *rtkclient.App
	createErr     error
	createNameArg string
}

func (f *fakeAccountClient) ListApps() ([]rtkclient.App, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.apps, nil
}

func (f *fakeAccountClient) CreateApp(name string) (*rtkclient.App, error) {
	f.createCalls++
	f.createNameArg = name
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createApp, nil
}

func newResolveTestApp(t *testing.T, account rtkclient.AccountClient) (*App, *plugintest.API, *gomock.Controller) {
	t.Helper()
	api := &plugintest.API{}
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
	api.On("GetConfig").Maybe().Return(&model.Config{
		ServiceSettings: model.ServiceSettings{SiteURL: model.NewPointer("https://mm.example.com")},
	})
	t.Cleanup(func() { api.AssertExpectations(t) })

	ctrl := gomock.NewController(t)
	mockStore := storemocks.NewMockStore(ctrl)
	// WithAppLock invokes fn synchronously in tests; pass through any errors.
	mockStore.EXPECT().WithAppLock(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, fn func() error) error { return fn() },
	).AnyTimes()

	a := New(mockStore, nil, account, api)
	return a, api, ctrl
}

// --- ResolveAppByID ---

func TestResolveAppByID_Found(t *testing.T) {
	account := &fakeAccountClient{apps: []rtkclient.App{
		{ID: "other", Name: "other"},
		{ID: "env-app-id", Name: "mm-mm.example.com"},
	}}
	a, _, ctrl := newResolveTestApp(t, account)
	defer ctrl.Finish()

	mockStore := a.store.(*storemocks.MockStore)
	mockStore.EXPECT().StoreAppConfig("acct1", "env-app-id").Return("cfg-row-1", nil)

	appID, appConfigID, err := a.ResolveAppByID("acct1", "env-app-id")
	require.NoError(t, err)
	assert.Equal(t, "env-app-id", appID)
	assert.Equal(t, "cfg-row-1", appConfigID)
	assert.Equal(t, 0, account.createCalls, "must not auto-create when env-supplied app missing or present")
}

func TestResolveAppByID_NotFound(t *testing.T) {
	account := &fakeAccountClient{apps: []rtkclient.App{
		{ID: "some-other-app", Name: "mm-mm.example.com"},
	}}
	a, _, ctrl := newResolveTestApp(t, account)
	defer ctrl.Finish()

	_, _, err := a.ResolveAppByID("acct1", "missing-app-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-app-id")
	assert.Equal(t, 0, account.createCalls, "must not call CreateApp on miss")
}

func TestResolveAppByID_ListAppsFailure(t *testing.T) {
	account := &fakeAccountClient{listErr: errors.New("boom")}
	a, _, ctrl := newResolveTestApp(t, account)
	defer ctrl.Finish()

	_, _, err := a.ResolveAppByID("acct1", "env-app-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ListApps")
}

func TestResolveAppByID_NoAccountClient(t *testing.T) {
	a, _, ctrl := newResolveTestApp(t, nil)
	defer ctrl.Finish()

	_, _, err := a.ResolveAppByID("acct1", "env-app-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account client is not configured")
}

func TestResolveAppByID_EmptyAppID(t *testing.T) {
	account := &fakeAccountClient{}
	a, _, ctrl := newResolveTestApp(t, account)
	defer ctrl.Finish()

	_, _, err := a.ResolveAppByID("acct1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app ID is empty")
}
