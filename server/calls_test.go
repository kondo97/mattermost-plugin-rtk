package main

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/kondo97/mattermost-plugin-rtk/server/push"
	pushmocks "github.com/kondo97/mattermost-plugin-rtk/server/push/mocks"
	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	rtkmocks "github.com/kondo97/mattermost-plugin-rtk/server/rtkclient/mocks"
	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
	kvmocks "github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore/mocks"
)

// newTestPlugin creates a Plugin with injected mock dependencies for unit testing.
func newTestPlugin(t *testing.T, rtkClient rtkclient.RTKClient, store kvstore.KVStore) (*Plugin, *plugintest.API) {
	t.Helper()
	api := &plugintest.API{}
	// Allow any logging calls without asserting on them (arg counts vary: 1+2k pattern).
	anyArgs := func(n int) []any {
		args := make([]any, n)
		for i := range args {
			args[i] = mock.Anything
		}
		return args
	}
	for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		api.On("LogInfo", anyArgs(n)...).Maybe().Return()
		api.On("LogWarn", anyArgs(n)...).Maybe().Return()
		api.On("LogError", anyArgs(n)...).Maybe().Return()
	}
	t.Cleanup(func() { api.AssertExpectations(t) })
	p := &Plugin{}
	p.SetAPI(api)
	p.rtkClient = rtkClient
	p.kvStore = store
	// pushSender intentionally left nil; inject via p.pushSender in tests that need it.
	return p, api
}

// newTestPluginWithPush creates a Plugin with an injected MockPushSender.
func newTestPluginWithPush(t *testing.T, rtkClient rtkclient.RTKClient, store kvstore.KVStore, pushSender push.PushSender) (*Plugin, *plugintest.API) {
	t.Helper()
	p, api := newTestPlugin(t, rtkClient, store)
	p.pushSender = pushSender
	return p, api
}

// --- CreateCall ---

func TestCreateCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	channelID := "channel1"
	userID := "user1"
	meetingID := "meeting1"
	tokenStr := "jwt-token"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: meetingID}, nil)
	mockRTK.EXPECT().GenerateToken(meetingID, userID, rtkPresetHost).Return(&rtkclient.Token{Token: tokenStr}, nil)
	mockStore.EXPECT().SaveCall(gomock.Any()).Return(nil).Times(2)

	createdPost := &model.Post{Id: "post1"}
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(createdPost, nil)
	api.On("PublishWebSocketEvent", wsEventCallStarted,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	session, tok, err := p.CreateCall(channelID, userID)
	require.NoError(t, err)
	assert.Equal(t, tokenStr, tok)
	assert.Equal(t, channelID, session.ChannelID)
	assert.Equal(t, userID, session.CreatorID)
	assert.Contains(t, session.Participants, userID)
}

func TestCreateCall_AlreadyActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)

	existing := &kvstore.CallSession{ID: "call1", ChannelID: "ch1", EndAt: 0}
	mockStore.EXPECT().GetCallByChannel("ch1").Return(existing, nil)

	_, _, err := p.CreateCall("ch1", "user1")
	assert.ErrorIs(t, err, ErrCallAlreadyActive)
}

func TestCreateCall_RTKNotConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, nil, mockStore)

	_, _, err := p.CreateCall("ch1", "user1")
	assert.ErrorIs(t, err, ErrRTKNotConfigured)
}

func TestCreateCall_CreateMeetingFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByChannel("ch1").Return(nil, nil)
	mockRTK.EXPECT().CreateMeeting().Return(nil, errors.New("RTK error"))

	_, _, err := p.CreateCall("ch1", "user1")
	require.Error(t, err)
}

// --- JoinCall ---

func TestJoinCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	callID := "call1"
	userID := "user2"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1"}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", userID, rtkPresetParticipant).Return(&rtkclient.Token{Token: "tok"}, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{"user1", userID}).Return(nil)
	api.On("PublishWebSocketEvent", wsEventUserJoined,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	sess, tok, err := p.JoinCall(callID, userID)
	require.NoError(t, err)
	assert.Equal(t, "tok", tok)
	assert.Equal(t, callID, sess.ID)
}

func TestJoinCall_CallNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	_, _, err := p.JoinCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestJoinCall_AlreadyEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)

	session := &kvstore.CallSession{ID: "call1", EndAt: 1000}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	_, _, err := p.JoinCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestJoinCall_DuplicateParticipantNoDoubleAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	callID := "call1"
	userID := "user1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{userID}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", userID, rtkPresetParticipant).Return(&rtkclient.Token{Token: "tok"}, nil)
	// participants list unchanged since userID already present
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{userID}).Return(nil)
	api.On("PublishWebSocketEvent", wsEventUserJoined,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	_, _, err := p.JoinCall(callID, userID)
	require.NoError(t, err)
}

// --- LeaveCall ---

func TestLeaveCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1", "user2"}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{"user2"}).Return(nil)
	api.On("PublishWebSocketEvent", wsEventUserLeft,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := p.LeaveCall(callID, "user1")
	require.NoError(t, err)
}

func TestLeaveCall_LastParticipantAutoEnds(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1"}, StartAt: 1000, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{}).Return(nil)
	mockStore.EXPECT().EndCall(callID, gomock.Any()).Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", wsEventUserLeft,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()
	api.On("PublishWebSocketEvent", wsEventCallEnded,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := p.LeaveCall(callID, "user1")
	require.NoError(t, err)
}

func TestLeaveCall_Idempotent_CallNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	err := p.LeaveCall("call1", "user1")
	require.NoError(t, err)
}

// --- EndCall ---

func TestEndCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		CreatorID: "user1", Participants: []string{"user1"}, StartAt: 1000, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().EndCall(callID, gomock.Any()).Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", wsEventCallEnded,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := p.EndCall(callID, "user1")
	require.NoError(t, err)
}

func TestEndCall_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)

	session := &kvstore.CallSession{ID: "call1", CreatorID: "user1", EndAt: 0}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	err := p.EndCall("call1", "user2")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestEndCall_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	err := p.EndCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestEndCall_AlreadyEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, _ := newTestPlugin(t, mockRTK, mockStore)

	session := &kvstore.CallSession{ID: "call1", EndAt: 9999}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	err := p.EndCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

// --- Push integration (US-018) ---

func TestCreateCall_InvokesPushSender(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	mockPush := pushmocks.NewMockPushSender(ctrl)
	p, api := newTestPluginWithPush(t, mockRTK, mockStore, mockPush)

	channelID := "channel1"
	userID := "user1"
	meetingID := "meeting1"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: meetingID}, nil)
	mockRTK.EXPECT().GenerateToken(meetingID, userID, rtkPresetHost).Return(&rtkclient.Token{Token: "tok"}, nil)
	mockStore.EXPECT().SaveCall(gomock.Any()).Return(nil).Times(2)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil)
	api.On("PublishWebSocketEvent", wsEventCallStarted,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()
	mockPush.EXPECT().SendIncomingCall(gomock.Any()).Return(nil)

	_, _, err := p.CreateCall(channelID, userID)
	require.NoError(t, err)
}

func TestCreateCall_PushSenderError_CallSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	mockPush := pushmocks.NewMockPushSender(ctrl)
	p, api := newTestPluginWithPush(t, mockRTK, mockStore, mockPush)

	channelID := "channel1"
	userID := "user1"
	meetingID := "meeting1"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: meetingID}, nil)
	mockRTK.EXPECT().GenerateToken(meetingID, userID, rtkPresetHost).Return(&rtkclient.Token{Token: "tok"}, nil)
	mockStore.EXPECT().SaveCall(gomock.Any()).Return(nil).Times(2)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil)
	api.On("PublishWebSocketEvent", wsEventCallStarted,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()
	mockPush.EXPECT().SendIncomingCall(gomock.Any()).Return(errors.New("push failed"))

	// push failure is best-effort — CreateCall must still succeed
	_, _, err := p.CreateCall(channelID, userID)
	require.NoError(t, err)
}

func TestEndCall_InvokesPushSender(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	mockPush := pushmocks.NewMockPushSender(ctrl)
	p, api := newTestPluginWithPush(t, mockRTK, mockStore, mockPush)

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		CreatorID: "user1", Participants: []string{"user1"}, StartAt: 1000, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().EndCall(callID, gomock.Any()).Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", wsEventCallEnded,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()
	mockPush.EXPECT().SendCallEnded(gomock.Any()).Return(nil)

	err := p.EndCall(callID, "user1")
	require.NoError(t, err)
}
