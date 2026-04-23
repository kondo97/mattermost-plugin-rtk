package app

import (
	"errors"
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

// newTestApp creates an App with injected mock dependencies for unit testing.
func newTestApp(t *testing.T, rtkClient rtkclient.RTKClient, store kvstore.KVStore) (*App, *plugintest.API) {
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
	for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12} {
		api.On("LogDebug", anyArgs(n)...).Maybe().Return()
		api.On("LogInfo", anyArgs(n)...).Maybe().Return()
		api.On("LogWarn", anyArgs(n)...).Maybe().Return()
		api.On("LogError", anyArgs(n)...).Maybe().Return()
	}
	// Default GetUser mock for getUserDisplayName and push notification sender lookup
	api.On("GetUser", mock.Anything).Maybe().Return(&model.User{
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
	}, nil)
	// Default GetChannelMember mock: user is a member of any channel
	api.On("GetChannelMember", mock.Anything, mock.Anything).Maybe().Return(&model.ChannelMember{}, nil)
	t.Cleanup(func() { api.AssertExpectations(t) })
	a := New(store, rtkClient, api)
	return a, api
}

// --- CreateCall ---

func TestCreateCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	channelID := "channel1"
	userID := "user1"
	meetingID := "meeting1"
	tokenStr := "jwt-token"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockRTK.EXPECT().CreateMeeting(gomock.Any()).Return(&rtkclient.Meeting{ID: meetingID}, nil)
	mockRTK.EXPECT().GenerateToken(meetingID, userID, gomock.Any(), RTKPresetHost).Return(&rtkclient.Token{Token: tokenStr}, nil)
	mockStore.EXPECT().SaveCall(gomock.Any()).Return(nil).Times(2)
	mockStore.EXPECT().AddActiveCallID(gomock.Any()).Return(nil)

	createdPost := &model.Post{Id: "post1"}
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(createdPost, nil)
	api.On("PublishWebSocketEvent", WSEventCallStarted,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()
	// sendPushNotifications will call GetConfig; return push disabled to keep this test focused
	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{SendPushNotifications: model.NewPointer(false)},
	})

	session, tok, err := a.CreateCall(channelID, userID)
	require.NoError(t, err)
	assert.Equal(t, tokenStr, tok)
	assert.Equal(t, channelID, session.ChannelID)
	assert.Equal(t, userID, session.CreatorID)
	assert.Contains(t, session.Participants, userID)
}

func TestCreateCall_NotChannelMember(t *testing.T) {
	api := &plugintest.API{}
	api.On("GetChannelMember", "ch1", "user1").Return(nil, &model.AppError{Message: "not a member"})
	t.Cleanup(func() { api.AssertExpectations(t) })
	a := New(nil, nil, api)

	_, _, err := a.CreateCall("ch1", "user1")
	assert.ErrorIs(t, err, ErrNotChannelMember)
}

func TestCreateCall_AlreadyActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	existing := &kvstore.CallSession{ID: "call1", ChannelID: "ch1", MeetingID: "mtg1", EndAt: 0}
	mockStore.EXPECT().GetCallByChannel("ch1").Return(existing, nil)
	// Meeting is still alive — return participants without error.
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return([]string{"user0"}, nil)

	_, _, err := a.CreateCall("ch1", "user1")
	assert.ErrorIs(t, err, ErrCallAlreadyActive)
}

func TestCreateCall_AlreadyActive_ZombieCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	existing := &kvstore.CallSession{
		ID: "old-call", ChannelID: "ch1", MeetingID: "old-mtg", StartAt: 1000,
	}
	mockStore.EXPECT().GetCallByChannel("ch1").Return(existing, nil)
	// RTK returns 404 — the existing call is stale (zombie).
	mockRTK.EXPECT().GetMeetingParticipants("old-mtg").Return(nil, rtkclient.ErrMeetingNotFound)

	// endCallInternal path for the stale call.
	mockStore.EXPECT().EndCall("old-call", gomock.Any()).Return(nil)
	mockStore.EXPECT().RemoveActiveCallID("old-call").Return(nil)
	mockRTK.EXPECT().EndMeeting("old-mtg").Return(nil)
	api.On("PublishWebSocketEvent", WSEventCallEnded, mock.Anything, mock.Anything).Return()

	// New call creation path.
	newMeetingID := "new-mtg"
	newToken := "new-token"
	mockRTK.EXPECT().CreateMeeting(gomock.Any()).Return(&rtkclient.Meeting{ID: newMeetingID}, nil)
	mockRTK.EXPECT().GenerateToken(newMeetingID, "user1", gomock.Any(), RTKPresetHost).Return(&rtkclient.Token{Token: newToken}, nil)
	mockStore.EXPECT().SaveCall(gomock.Any()).Return(nil).Times(2)
	mockStore.EXPECT().AddActiveCallID(gomock.Any()).Return(nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil)
	api.On("PublishWebSocketEvent", WSEventCallStarted, mock.Anything, mock.Anything).Return()
	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{SendPushNotifications: model.NewPointer(false)},
	})

	session, tok, err := a.CreateCall("ch1", "user1")
	require.NoError(t, err)
	assert.Equal(t, newToken, tok)
	assert.Equal(t, "ch1", session.ChannelID)
}

func TestCreateCall_RTKNotConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, nil, mockStore)

	_, _, err := a.CreateCall("ch1", "user1")
	assert.ErrorIs(t, err, ErrRTKNotConfigured)
}

func TestCreateCall_CreateMeetingFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByChannel("ch1").Return(nil, nil)
	mockRTK.EXPECT().CreateMeeting(gomock.Any()).Return(nil, errors.New("RTK error"))

	_, _, err := a.CreateCall("ch1", "user1")
	require.Error(t, err)
}

// --- JoinCall ---

func TestJoinCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	userID := "user2"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1"}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return([]string{"user1"}, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", userID, gomock.Any(), RTKPresetParticipant).Return(&rtkclient.Token{Token: "tok"}, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{"user1", userID}).Return(nil)
	api.On("PublishWebSocketEvent", WSEventUserJoined,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	sess, tok, err := a.JoinCall(callID, userID)
	require.NoError(t, err)
	assert.Equal(t, "tok", tok)
	assert.Equal(t, callID, sess.ID)
}

// TestJoinCall_UpdatesPostParticipants verifies that JoinCall updates the post and emits the
// user_joined WebSocket event immediately. HandleWebhookParticipantJoined also fires later
// (when the RTK SDK actually connects) and performs an idempotent update.
func TestJoinCall_UpdatesPostParticipants(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	userID := "user2"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1"}, EndAt: 0, PostID: "post1",
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return([]string{"user1"}, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", userID, gomock.Any(), RTKPresetParticipant).Return(&rtkclient.Token{Token: "tok"}, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{"user1", userID}).Return(nil)

	existingPost := &model.Post{Id: "post1", Props: model.StringInterface{"call_id": callID}}
	api.On("GetPost", "post1").Return(existingPost, nil)
	api.On("UpdatePost", mock.MatchedBy(func(post *model.Post) bool {
		participants, ok := post.Props["participants"].([]string)
		return ok && len(participants) == 2 && participants[0] == "user1" && participants[1] == "user2"
	})).Return(existingPost, nil)
	api.On("PublishWebSocketEvent", WSEventUserJoined,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	_, _, err := a.JoinCall(callID, userID)
	require.NoError(t, err)
}

func TestJoinCall_NotChannelMember(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	api := &plugintest.API{}
	api.On("GetChannelMember", "ch1", "user2").Return(nil, &model.AppError{Message: "not a member"})
	t.Cleanup(func() { api.AssertExpectations(t) })
	a := New(mockStore, mockRTK, api)

	session := &kvstore.CallSession{ID: "call1", ChannelID: "ch1", MeetingID: "mtg1", Participants: []string{"user1"}}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)
	// GetChannelMember check happens before RTK liveness — no GetMeetingParticipants call expected.

	_, _, err := a.JoinCall("call1", "user2")
	assert.ErrorIs(t, err, ErrNotChannelMember)
}

func TestJoinCall_CallNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	_, _, err := a.JoinCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestJoinCall_AlreadyEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	session := &kvstore.CallSession{ID: "call1", EndAt: 1000}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	_, _, err := a.JoinCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestJoinCall_ZombieCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	session := &kvstore.CallSession{
		ID: "call1", ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user0"}, StartAt: 1000, EndAt: 0,
	}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)
	// RTK returns 404 — the call is stale (zombie).
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return(nil, rtkclient.ErrMeetingNotFound)

	// endCallInternal path for the stale call.
	mockStore.EXPECT().EndCall("call1", gomock.Any()).Return(nil)
	mockStore.EXPECT().RemoveActiveCallID("call1").Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", WSEventCallEnded, mock.Anything, mock.Anything).Return()

	_, _, err := a.JoinCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestJoinCall_DuplicateParticipantNoDoubleAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	userID := "user1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{userID}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GetMeetingParticipants("mtg1").Return([]string{userID}, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", userID, gomock.Any(), RTKPresetParticipant).Return(&rtkclient.Token{Token: "tok"}, nil)
	// participants list unchanged since userID already present
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{userID}).Return(nil)
	api.On("PublishWebSocketEvent", WSEventUserJoined,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	_, _, err := a.JoinCall(callID, userID)
	require.NoError(t, err)
}

// --- LeaveCall ---

func TestLeaveCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1", "user2"}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{"user2"}).Return(nil)
	api.On("PublishWebSocketEvent", WSEventUserLeft,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := a.LeaveCall(callID, "user1")
	require.NoError(t, err)
}

func TestLeaveCall_UpdatesPostParticipants(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1", "user2"}, EndAt: 0, PostID: "post1",
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{"user2"}).Return(nil)

	existingPost := &model.Post{Id: "post1", Props: model.StringInterface{"call_id": callID}}
	api.On("GetPost", "post1").Return(existingPost, nil)
	api.On("UpdatePost", mock.MatchedBy(func(post *model.Post) bool {
		participants, ok := post.Props["participants"].([]string)
		return ok && len(participants) == 1 && participants[0] == "user2"
	})).Return(existingPost, nil)
	api.On("PublishWebSocketEvent", WSEventUserLeft,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := a.LeaveCall(callID, "user1")
	require.NoError(t, err)
}

func TestLeaveCall_LastParticipantAutoEnds(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1"}, StartAt: 1000, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().UpdateCallParticipants(callID, []string{}).Return(nil)
	mockStore.EXPECT().EndCall(callID, gomock.Any()).Return(nil)
	mockStore.EXPECT().RemoveActiveCallID(callID).Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", WSEventUserLeft,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()
	api.On("PublishWebSocketEvent", WSEventCallEnded,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := a.LeaveCall(callID, "user1")
	require.NoError(t, err)
}

func TestLeaveCall_Idempotent_CallNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	err := a.LeaveCall("call1", "user1")
	require.NoError(t, err)
}

// --- EndCall ---

func TestEndCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	session := &kvstore.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		CreatorID: "user1", Participants: []string{"user1"}, StartAt: 1000, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().EndCall(callID, gomock.Any()).Return(nil)
	mockStore.EXPECT().RemoveActiveCallID(callID).Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	api.On("PublishWebSocketEvent", WSEventCallEnded,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := a.EndCall(callID, "user1")
	require.NoError(t, err)
}

func TestEndCall_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	session := &kvstore.CallSession{ID: "call1", CreatorID: "user1", EndAt: 0}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	err := a.EndCall("call1", "user2")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestEndCall_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	err := a.EndCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestEndCall_AlreadyEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	session := &kvstore.CallSession{ID: "call1", EndAt: 9999}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	err := a.EndCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}
