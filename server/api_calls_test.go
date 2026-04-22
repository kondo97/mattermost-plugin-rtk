package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
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
	assert.NotNil(t, resp["feature_flags"], "feature_flags must be present in create call response")
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

	existing := &kvstore.CallSession{ID: "existing", ChannelID: "chan1"}
	mockStore.EXPECT().GetCallByChannel("chan1").Return(existing, nil)

	body, _ := json.Marshal(map[string]string{"channel_id": "chan1"})
	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls", "user1", body)

	assert.Equal(t, http.StatusConflict, w.Code)
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
	mockRTK.EXPECT().GenerateToken("mtg1", "user1", gomock.Any(), rtkPresetParticipant).Return(&rtkclient.Token{Token: tokenStr}, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, gomock.Any()).Return(nil)
	api.On("PublishWebSocketEvent", wsEventUserJoined, mock.Anything, mock.Anything).Return()

	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls/"+callID+"/token", "user1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, tokenStr, resp["token"])
	assert.NotNil(t, resp["feature_flags"], "feature_flags must be present in join call response")
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
