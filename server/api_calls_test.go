package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	rtkmocks "github.com/kondo97/mattermost-plugin-rtk/server/rtkclient/mocks"
	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
	kvmocks "github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore/mocks"
)

// serveWithUser is a test helper that sends a request with Mattermost-User-ID set.
func serveWithUser(t *testing.T, p *Plugin, method, path, userID string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Mattermost-User-ID", userID)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	p.ServeHTTP(nil, w, req)
	return w
}

func TestHandleCreateCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	meetingID := "mtg1"
	tokenStr := "tok1"
	mockStore.EXPECT().GetCallByChannel("chan1").Return(nil, nil)
	mockRTK.EXPECT().CreateMeeting(gomock.Any()).Return(&rtkclient.Meeting{ID: meetingID}, nil)
	mockRTK.EXPECT().GenerateToken(meetingID, "user1", gomock.Any(), rtkPresetHost).Return(&rtkclient.Token{Token: tokenStr}, nil)
	mockStore.EXPECT().SaveCall(gomock.Any()).Return(nil).Times(2)
	mockStore.EXPECT().AddActiveCallID(gomock.Any()).Return(nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "p1"}, nil)
	api.On("PublishWebSocketEvent", wsEventCallStarted, mock.Anything, mock.Anything).Return()
	// sendPushNotifications will call GetConfig; return push disabled to keep this test focused
	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{SendPushNotifications: model.NewPointer(false)},
	})

	body, _ := json.Marshal(map[string]string{"channel_id": "chan1"})
	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls", "user1", body)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, tokenStr, resp["token"])
	assert.NotNil(t, resp["call"])
	assert.Nil(t, resp["feature_flags"], "feature_flags must not be present in create call response")
}

func TestHandleCreateCall_MissingChannelID(t *testing.T) {
	p, _ := newTestPlugin(t, nil, nil)
	p.router = p.initRouter()

	body, _ := json.Marshal(map[string]string{})
	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls", "user1", body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCreateCall_AlreadyActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	existing := &kvstore.CallSession{ID: "existing", ChannelID: "chan1", MeetingID: "mtg1"}
	mockStore.EXPECT().GetCallByChannel("chan1").Return(existing, nil)
	// Meeting is still alive — normal conflict.
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return([]string{"user0"}, nil)

	body, _ := json.Marshal(map[string]string{"channel_id": "chan1"})
	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls", "user1", body)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleCreateCall_NotChannelMember(t *testing.T) {
	api := &plugintest.API{}
	api.On("GetChannelMember", "chan1", "user1").Return(nil, &model.AppError{Message: "not a member"})
	t.Cleanup(func() { api.AssertExpectations(t) })
	p := &Plugin{}
	p.SetAPI(api)
	p.router = p.initRouter()

	body, _ := json.Marshal(map[string]string{"channel_id": "chan1"})
	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls", "user1", body)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleCreateCall_NoAuth(t *testing.T) {
	p, _ := newTestPlugin(t, nil, nil)
	p.router = p.initRouter()

	body, _ := json.Marshal(map[string]string{"channel_id": "chan1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calls", bytes.NewReader(body))
	w := httptest.NewRecorder()
	p.ServeHTTP(nil, w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleJoinCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "chan1", MeetingID: "mtg1",
		Participants: []string{"user0"},
	}
	tokenStr := "tok2"

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return([]string{"user0"}, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", "user1", gomock.Any(), rtkPresetParticipant).Return(&rtkclient.Token{Token: tokenStr}, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, gomock.Any()).Return(nil)
	api.On("PublishWebSocketEvent", wsEventUserJoined, mock.Anything, mock.Anything).Return()

	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls/"+callID+"/token", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, tokenStr, resp["token"])
	assert.Nil(t, resp["feature_flags"], "feature_flags must not be present in join call response")
}

func TestHandleJoinCall_NotChannelMember(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	api := &plugintest.API{}
	api.On("GetChannelMember", "chan1", "user1").Return(nil, &model.AppError{Message: "not a member"})
	t.Cleanup(func() { api.AssertExpectations(t) })
	p := &Plugin{}
	p.SetAPI(api)
	p.rtkClient = mockRTK
	p.kvStore = mockStore
	p.router = p.initRouter()

	session := &kvstore.CallSession{ID: "call1", ChannelID: "chan1", MeetingID: "mtg1", Participants: []string{"user0"}}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls/call1/token", "user1", nil)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleJoinCall_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls/call1/token", "user1", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleLeaveCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	session := &kvstore.CallSession{ID: "call1", ChannelID: "chan1", MeetingID: "mtg1", Participants: []string{"user1"}}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)
	mockStore.EXPECT().UpdateCallParticipants("call1", gomock.Any()).Return(nil)
	// last participant left → auto-end
	mockStore.EXPECT().EndCall("call1", gomock.Any()).Return(nil)
	mockStore.EXPECT().RemoveActiveCallID("call1").Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", wsEventUserLeft, mock.Anything, mock.Anything).Return()
	api.On("PublishWebSocketEvent", wsEventCallEnded, mock.Anything, mock.Anything).Return()
	api.On("GetPost", mock.Anything).Return(nil, &model.AppError{Message: "not found"}).Maybe()

	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls/call1/leave", "user1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleEndCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	session := &kvstore.CallSession{ID: "call1", ChannelID: "chan1", CreatorID: "user1", MeetingID: "mtg1"}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)
	mockStore.EXPECT().EndCall("call1", gomock.Any()).Return(nil)
	mockStore.EXPECT().RemoveActiveCallID("call1").Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", wsEventCallEnded, mock.Anything, mock.Anything).Return()
	api.On("GetPost", mock.Anything).Return(nil, &model.AppError{Message: "not found"}).Maybe()

	w := serveWithUser(t, p, http.MethodDelete, "/api/v1/calls/call1", "user1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleEndCall_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	session := &kvstore.CallSession{ID: "call1", ChannelID: "chan1", CreatorID: "creator"}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	w := serveWithUser(t, p, http.MethodDelete, "/api/v1/calls/call1", "not-creator", nil)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleGetCall_ActiveCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	session := &kvstore.CallSession{ID: "call1", ChannelID: "chan1", MeetingID: "mtg1", EndAt: 0}
	// First fetch (handleGetCall) + second fetch (re-fetch after reconcile).
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil).Times(2)
	// Meeting is alive — no force-end.
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return([]string{"user0"}, nil)

	w := serveWithUser(t, p, http.MethodGet, "/api/v1/calls/call1", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "call1", resp["id"])
}

func TestHandleGetCall_ZombieCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)
	p.router = p.initRouter()

	activeSession := &kvstore.CallSession{
		ID: "call1", ChannelID: "chan1", MeetingID: "mtg1", StartAt: 1000, EndAt: 0,
	}
	endedSession := &kvstore.CallSession{
		ID: "call1", ChannelID: "chan1", MeetingID: "mtg1", StartAt: 1000, EndAt: 2000,
	}
	// Three ordered GetCallByID calls:
	// 1. initial fetch in handleGetCall
	// 2. re-read under lock inside reconcileCallOnDemand
	// 3. re-fetch in handleGetCall after reconcile
	gomock.InOrder(
		mockStore.EXPECT().GetCallByID("call1").Return(activeSession, nil),
		mockStore.EXPECT().GetCallByID("call1").Return(activeSession, nil),
		mockStore.EXPECT().GetCallByID("call1").Return(endedSession, nil),
	)
	// RTK returns 404 — zombie call.
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return(nil, rtkclient.ErrMeetingNotFound)

	// endCallInternal path inside reconcileCallOnDemand.
	mockStore.EXPECT().EndCall("call1", gomock.Any()).Return(nil)
	mockStore.EXPECT().RemoveActiveCallID("call1").Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", wsEventCallEnded, mock.Anything, mock.Anything).Return()

	w := serveWithUser(t, p, http.MethodGet, "/api/v1/calls/call1", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Response should reflect the ended state.
	assert.NotEqual(t, float64(0), resp["end_at"])
}