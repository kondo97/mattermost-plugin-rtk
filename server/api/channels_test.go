package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/kondo97/mattermost-plugin-rtk/server/store"
	storemocks "github.com/kondo97/mattermost-plugin-rtk/server/store/mocks"
)

// TestHandleGetAllChannels_Unauthenticated verifies the auth middleware
// rejects requests without Mattermost-User-ID.
func TestHandleGetAllChannels_Unauthenticated(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storemocks.NewMockStore(ctrl)
	h, _ := newTestAPI(t, nil, mockStore)

	req := httptest.NewRequest("GET", "/api/v1/channels", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, 401, w.Code)
}

func TestHandleGetAllChannels_OnlyChannelsUserIsMemberOf(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storemocks.NewMockStore(ctrl)
	h, mmAPI := newTestAPI(t, nil, mockStore)

	// User is member of ch1 only (not ch2). Use ChannelId field name from model.ChannelMember.
	mmAPI.On("GetChannelMembersForUser", "", "user1", 0, 200).Return([]*model.ChannelMember{
		&model.ChannelMember{ChannelId: "ch1"},
	}, nil)

	mockStore.EXPECT().GetAllCallsChannels().Return([]*store.CallsChannel{
		{ChannelID: "ch1", Enabled: true},
		{ChannelID: "ch2", Enabled: false},
	}, nil)
	mockStore.EXPECT().GetAllActiveCalls().Return(nil, nil)

	w := serveWithUser(t, h, "GET", "/api/v1/channels", "user1", nil)
	require.Equal(t, 200, w.Code)

	var resp []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "ch1", resp[0]["channel_id"])
	assert.Equal(t, true, resp[0]["enabled"])
	assert.Nil(t, resp[0]["call"])
}

func TestHandleGetAllChannels_ActiveCallShape(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storemocks.NewMockStore(ctrl)
	h, mmAPI := newTestAPI(t, nil, mockStore)

	mmAPI.On("GetChannelMembersForUser", "", "user1", 0, 200).Return([]*model.ChannelMember{
		&model.ChannelMember{ChannelId: "ch1"},
	}, nil)

	mockStore.EXPECT().GetAllCallsChannels().Return([]*store.CallsChannel{
		{ChannelID: "ch1", Enabled: true},
	}, nil)
	mockStore.EXPECT().GetAllActiveCalls().Return([]*store.CallSession{
		{
			ID:           "call1",
			ChannelID:    "ch1",
			CreatorID:    "user1",
			Participants: []string{"user1", "user2"},
			CreateAt:     12345,
			PostID:       "post1",
		},
	}, nil)

	w := serveWithUser(t, h, "GET", "/api/v1/channels", "user1", nil)
	require.Equal(t, 200, w.Code)

	var resp []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp, 1)

	call, ok := resp[0]["call"].(map[string]any)
	require.True(t, ok, "call must be present")
	assert.Equal(t, "call1", call["id"])
	assert.Equal(t, float64(12345), call["start_at"])
	assert.Equal(t, "post1", call["post_id"])
	assert.Equal(t, "user1", call["owner_id"])
	assert.Equal(t, "user1", call["host_id"])
	assert.Equal(t, "", call["thread_id"])
	assert.Equal(t, "", call["screen_sharing_session_id"])
	assert.Nil(t, call["recording"])
	assert.Nil(t, call["transcription"])
	assert.Nil(t, call["live_captions"])

	sessions, ok := call["sessions"].([]any)
	require.True(t, ok)
	require.Len(t, sessions, 2)
	first := sessions[0].(map[string]any)
	assert.Equal(t, "user1", first["session_id"])
	assert.Equal(t, "user1", first["user_id"])
	assert.Equal(t, false, first["unmuted"])
	assert.Equal(t, float64(0), first["raised_hand"])
	assert.Equal(t, false, first["video"])
}

// TestHandleGetAllChannels_ActiveCallWithoutChannelRow verifies that an active
// call in a channel that has no rtk_calls_channels row is still surfaced (so
// older Calls clients see the call), but with no `enabled` field set.
func TestHandleGetAllChannels_ActiveCallWithoutChannelRow(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storemocks.NewMockStore(ctrl)
	h, mmAPI := newTestAPI(t, nil, mockStore)

	mmAPI.On("GetChannelMembersForUser", "", "user1", 0, 200).Return([]*model.ChannelMember{
		&model.ChannelMember{ChannelId: "ch1"},
	}, nil)

	mockStore.EXPECT().GetAllCallsChannels().Return(nil, nil)
	mockStore.EXPECT().GetAllActiveCalls().Return([]*store.CallSession{
		{
			ID:           "call1",
			ChannelID:    "ch1",
			CreatorID:    "user1",
			Participants: []string{"user1"},
			CreateAt:     1000,
			PostID:       "post1",
		},
	}, nil)

	w := serveWithUser(t, h, "GET", "/api/v1/channels", "user1", nil)
	require.Equal(t, 200, w.Code)

	var resp []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "ch1", resp[0]["channel_id"])
	_, hasEnabled := resp[0]["enabled"]
	assert.False(t, hasEnabled, "enabled field must be omitted when no rtk_calls_channels row exists")
}

// TestHandleUpdateChannel_ReturnsJSONBody verifies PUT /api/v1/channels/{id}
// returns a {channel_id, enabled} JSON body so clients can confirm the new
// state without an extra round-trip. A previous version returned an empty body
// which broke clients that always parse the response as JSON.
func TestHandleUpdateChannel_ReturnsJSONBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := storemocks.NewMockStore(ctrl)
	h, mmAPI := newTestAPI(t, nil, mockStore)

	mmAPI.On("GetChannel", "ch1").Return(&model.Channel{Id: "ch1", Type: model.ChannelTypeOpen}, nil)
	mmAPI.On("HasPermissionToChannel", "user1", "ch1", model.PermissionManagePublicChannelProperties).Return(true)
	mockStore.EXPECT().UpsertCallsChannel(&store.CallsChannel{ChannelID: "ch1", Enabled: false}).Return(nil)

	w := serveWithUser(t, h, "PUT", "/api/v1/channels/ch1", "user1", []byte(`{"enabled":false}`))
	require.Equal(t, 200, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ch1", resp["channel_id"])
	assert.Equal(t, false, resp["enabled"])
}
