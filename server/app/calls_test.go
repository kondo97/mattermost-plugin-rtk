package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
	rtkmocks "github.com/kondo97/mattermost-plugin-rtk/server/rtkclient/mocks"
	"github.com/kondo97/mattermost-plugin-rtk/server/store"
	storemocks "github.com/kondo97/mattermost-plugin-rtk/server/store/mocks"
)

// newTestApp creates an App with injected mock dependencies for unit testing.
func newTestApp(t *testing.T, rtkClient rtkclient.RTKClient, store store.Store) (*App, *plugintest.API) {
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
	a := New(store, rtkClient, nil, api)
	return a, api
}

// --- CreateCall ---

func TestCreateCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	channelID := "channel1"
	userID := "user1"
	meetingID := "meeting1"
	tokenStr := "jwt-token"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockStore.EXPECT().GetChannelMeeting(channelID).Return("", "", "", nil)
	mockStore.EXPECT().GetActiveAppConfigID().Return("cfg1", nil)
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: meetingID}, nil)
	mockStore.EXPECT().SaveChannelMeeting(channelID, meetingID, "cfg1").Return("cm1", nil)
	mockRTK.EXPECT().GenerateToken(meetingID, gomock.Any(), userID, gomock.Any(), RTKPresetHost).Return(&rtkclient.Token{Token: tokenStr}, nil)
	mockStore.EXPECT().CreateCallSession(gomock.Any()).Return(nil)

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
	// Newly-saved channel meeting id must be linked to the session.
	assert.Equal(t, "cm1", session.ChannelMeetingID)
}

func TestCreateCall_NotChannelMember(t *testing.T) {
	api := &plugintest.API{}
	api.On("GetChannelMember", "ch1", "user1").Return(nil, &model.AppError{Message: "not a member"})
	t.Cleanup(func() { api.AssertExpectations(t) })
	a := New(nil, nil, nil, api)

	_, _, err := a.CreateCall("ch1", "user1")
	assert.ErrorIs(t, err, ErrNotChannelMember)
}

func TestCreateCall_AlreadyActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	existing := &store.CallSession{ID: "call1", ChannelID: "ch1", MeetingID: "mtg1", EndAt: 0}
	mockStore.EXPECT().GetCallByChannel("ch1").Return(existing, nil)
	// Meeting is still alive — return participants without error.
	mockRTK.EXPECT().GetMeeting("mtg1").Return(&rtkclient.Meeting{ID: "mtg1"}, nil)

	_, _, err := a.CreateCall("ch1", "user1")
	assert.ErrorIs(t, err, ErrCallAlreadyActive)
}

func TestCreateCall_AlreadyActive_ZombieCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	existing := &store.CallSession{
		ID: "old-call", ChannelID: "ch1", MeetingID: "old-mtg", CreateAt: 1000,
	}
	mockStore.EXPECT().GetCallByChannel("ch1").Return(existing, nil)
	// RTK returns 404 — the existing call is stale (zombie).
	mockRTK.EXPECT().GetMeeting("old-mtg").Return(nil, rtkclient.ErrMeetingNotFound)

	// endCallInternal path for the stale call.
	mockStore.EXPECT().EndCall("old-call", gomock.Any()).Return(nil)
	api.On("PublishWebSocketEvent", WSEventCallEnded, mock.Anything, mock.Anything).Return()

	// New call creation path.
	newMeetingID := "new-mtg"
	newToken := "new-token"
	mockStore.EXPECT().GetChannelMeeting("ch1").Return("", "", "", nil)
	mockStore.EXPECT().GetActiveAppConfigID().Return("cfg1", nil)
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: newMeetingID}, nil)
	mockStore.EXPECT().SaveChannelMeeting("ch1", newMeetingID, "cfg1").Return("cm-new", nil)
	mockRTK.EXPECT().GenerateToken(newMeetingID, gomock.Any(), "user1", gomock.Any(), RTKPresetHost).Return(&rtkclient.Token{Token: newToken}, nil)
	mockStore.EXPECT().CreateCallSession(gomock.Any()).Return(nil)
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
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, nil, mockStore)

	_, _, err := a.CreateCall("ch1", "user1")
	assert.ErrorIs(t, err, ErrRTKNotConfigured)
}

func TestCreateCall_CreateMeetingFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByChannel("ch1").Return(nil, nil)
	mockStore.EXPECT().GetChannelMeeting("ch1").Return("", "", "", nil)
	mockStore.EXPECT().GetActiveAppConfigID().Return("cfg1", nil)
	mockRTK.EXPECT().CreateMeeting().Return(nil, errors.New("RTK error"))

	_, _, err := a.CreateCall("ch1", "user1")
	require.Error(t, err)
}

func TestCreateCall_CreateMeetingReturnsEmptyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByChannel("ch1").Return(nil, nil)
	mockStore.EXPECT().GetChannelMeeting("ch1").Return("", "", "", nil)
	mockStore.EXPECT().GetActiveAppConfigID().Return("cfg1", nil)
	// Simulate the Cloudflare API returning a meeting with empty ID (e.g. wrong JSON key).
	// SaveChannelMeeting must NOT be called with an empty meeting ID.
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: ""}, nil)

	_, _, err := a.CreateCall("ch1", "user1")
	require.Error(t, err)
}

// CreatePost failure must abort CreateCall before persisting any call session row.
// Otherwise a row with post_id='' would later collide with the UNIQUE(post_id)
// constraint on subsequent CreateCall attempts whose CreatePost also fails.
func TestCreateCall_CreatePostFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	channelID := "ch1"
	userID := "user1"
	meetingID := "mtg1"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockStore.EXPECT().GetChannelMeeting(channelID).Return("", "", "", nil)
	mockStore.EXPECT().GetActiveAppConfigID().Return("cfg1", nil)
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: meetingID}, nil)
	mockStore.EXPECT().SaveChannelMeeting(channelID, meetingID, "cfg1").Return("cm1", nil)
	mockRTK.EXPECT().GenerateToken(meetingID, gomock.Any(), userID, gomock.Any(), RTKPresetHost).Return(&rtkclient.Token{Token: "tok"}, nil)
	// CreateCallSession must NOT be invoked when CreatePost fails.
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Message: "post create error"})

	_, _, err := a.CreateCall(channelID, userID)
	require.Error(t, err)
}

// If CreateCallSession fails after the post was created, the orphaned post must
// be cleaned up via DeletePost (best effort) and the error returned.
func TestCreateCall_CreateCallSessionFailsAfterPost(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	channelID := "ch1"
	userID := "user1"
	meetingID := "mtg1"
	postID := "post-orphan"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockStore.EXPECT().GetChannelMeeting(channelID).Return("", "", "", nil)
	mockStore.EXPECT().GetActiveAppConfigID().Return("cfg1", nil)
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: meetingID}, nil)
	mockStore.EXPECT().SaveChannelMeeting(channelID, meetingID, "cfg1").Return("cm1", nil)
	mockRTK.EXPECT().GenerateToken(meetingID, gomock.Any(), userID, gomock.Any(), RTKPresetHost).Return(&rtkclient.Token{Token: "tok"}, nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: postID}, nil)
	mockStore.EXPECT().CreateCallSession(gomock.Any()).Return(errors.New("db error"))
	api.On("DeletePost", postID).Return((*model.AppError)(nil))

	_, _, err := a.CreateCall(channelID, userID)
	require.Error(t, err)
}

// --- JoinCall ---

func TestJoinCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	userID := "user2"
	session := &store.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1"}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GetMeeting("mtg1").Return(&rtkclient.Meeting{ID: "mtg1"}, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", callID, userID, gomock.Any(), RTKPresetParticipant).Return(&rtkclient.Token{Token: "tok"}, nil)
	mockStore.EXPECT().AddCallParticipant(callID, userID).Return([]string{"user1", userID}, true, true, nil)
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
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	userID := "user2"
	session := &store.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1"}, EndAt: 0, PostID: "post1",
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GetMeeting("mtg1").Return(&rtkclient.Meeting{ID: "mtg1"}, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", callID, userID, gomock.Any(), RTKPresetParticipant).Return(&rtkclient.Token{Token: "tok"}, nil)
	mockStore.EXPECT().AddCallParticipant(callID, userID).Return([]string{"user1", userID}, true, true, nil)

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
	mockStore := storemocks.NewMockStore(ctrl)
	api := &plugintest.API{}
	api.On("GetChannelMember", "ch1", "user2").Return(nil, &model.AppError{Message: "not a member"})
	t.Cleanup(func() { api.AssertExpectations(t) })
	a := New(mockStore, mockRTK, nil, api)

	session := &store.CallSession{ID: "call1", ChannelID: "ch1", MeetingID: "mtg1", Participants: []string{"user1"}}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)
	// GetChannelMember check happens before RTK liveness — no GetMeeting call expected.

	_, _, err := a.JoinCall("call1", "user2")
	assert.ErrorIs(t, err, ErrNotChannelMember)
}

func TestJoinCall_CallNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	_, _, err := a.JoinCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestJoinCall_AlreadyEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	session := &store.CallSession{ID: "call1", EndAt: 1000}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	_, _, err := a.JoinCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestJoinCall_ZombieCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	session := &store.CallSession{
		ID: "call1", ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user0"}, CreateAt: 1000, EndAt: 0,
	}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)
	// RTK returns 404 — the call is stale (zombie).
	mockRTK.EXPECT().GetMeeting("mtg1").Return(nil, rtkclient.ErrMeetingNotFound)

	// endCallInternal path for the stale call.
	mockStore.EXPECT().EndCall("call1", gomock.Any()).Return(nil)
	api.On("PublishWebSocketEvent", WSEventCallEnded, mock.Anything, mock.Anything).Return()

	_, _, err := a.JoinCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestJoinCall_DuplicateParticipantNoDoubleAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	userID := "user1"
	session := &store.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{userID}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockRTK.EXPECT().GetMeeting("mtg1").Return(&rtkclient.Meeting{ID: "mtg1"}, nil)
	mockRTK.EXPECT().GenerateToken("mtg1", callID, userID, gomock.Any(), RTKPresetParticipant).Return(&rtkclient.Token{Token: "tok"}, nil)
	// AddCallParticipant is the idempotent insert; the duplicate user is a no-op at the
	// row level and the returned list is unchanged.
	mockStore.EXPECT().AddCallParticipant(callID, userID).Return([]string{userID}, true, true, nil)
	api.On("PublishWebSocketEvent", WSEventUserJoined,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	_, _, err := a.JoinCall(callID, userID)
	require.NoError(t, err)
}

// --- LeaveCall ---

func TestLeaveCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	session := &store.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1", "user2"}, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().RemoveCallParticipant(callID, "user1").Return([]string{"user2"}, false, int64(0), nil)
	api.On("PublishWebSocketEvent", WSEventUserLeft,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := a.LeaveCall(callID, "user1")
	require.NoError(t, err)
}

func TestLeaveCall_UpdatesPostParticipants(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	session := &store.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1", "user2"}, EndAt: 0, PostID: "post1",
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().RemoveCallParticipant(callID, "user1").Return([]string{"user2"}, false, int64(0), nil)

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
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	session := &store.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		Participants: []string{"user1"}, CreateAt: 1000, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	// RemoveCallParticipant returns ended=true with a non-zero endAt so the auto-end
	// path (BR-13) only runs the side-effects; the store wrote endat atomically.
	mockStore.EXPECT().RemoveCallParticipant(callID, "user1").Return([]string{}, true, int64(2000), nil)
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
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	err := a.LeaveCall("call1", "user1")
	require.NoError(t, err)
}

// --- EndCall ---

func TestEndCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	callID := "call1"
	session := &store.CallSession{
		ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
		CreatorID: "user1", Participants: []string{"user1"}, CreateAt: 1000, EndAt: 0,
	}

	mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
	mockStore.EXPECT().EndCall(callID, gomock.Any()).Return(nil)
	api.On("PublishWebSocketEvent", WSEventCallEnded,
		mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

	err := a.EndCall(callID, "user1")
	require.NoError(t, err)
}

func TestEndCall_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	session := &store.CallSession{ID: "call1", CreatorID: "user1", EndAt: 0}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	err := a.EndCall("call1", "user2")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestEndCall_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	mockStore.EXPECT().GetCallByID("call1").Return(nil, nil)

	err := a.EndCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestEndCall_AlreadyEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, _ := newTestApp(t, mockRTK, mockStore)

	session := &store.CallSession{ID: "call1", EndAt: 9999}
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	err := a.EndCall("call1", "user1")
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestCreateCall_ReusesChannelMeeting(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	channelID := "ch1"
	userID := "user1"
	existingMeetingID := "existing-mtg"
	tokenStr := "jwt-token"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockStore.EXPECT().GetChannelMeeting(channelID).Return("cm1", existingMeetingID, "", nil)
	mockStore.EXPECT().GetActiveAppConfigID().Return("cfg1", nil)
	// Meeting is alive in Cloudflare — no CreateMeeting call expected.
	mockRTK.EXPECT().GetMeeting(existingMeetingID).Return(&rtkclient.Meeting{ID: existingMeetingID}, nil)
	mockRTK.EXPECT().GenerateToken(existingMeetingID, gomock.Any(), userID, gomock.Any(), RTKPresetHost).Return(&rtkclient.Token{Token: tokenStr}, nil)
	mockStore.EXPECT().CreateCallSession(gomock.Any()).Return(nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil)
	api.On("PublishWebSocketEvent", WSEventCallStarted, mock.Anything, mock.Anything).Return()
	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{SendPushNotifications: model.NewPointer(false)},
	})

	session, tok, err := a.CreateCall(channelID, userID)
	require.NoError(t, err)
	assert.Equal(t, tokenStr, tok)
	assert.Equal(t, existingMeetingID, session.MeetingID)
	// Reusing an existing channel meeting must propagate its id to the session.
	assert.Equal(t, "cm1", session.ChannelMeetingID)
}

func TestCreateCall_ChannelMeetingGone(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := storemocks.NewMockStore(ctrl)
	a, api := newTestApp(t, mockRTK, mockStore)

	channelID := "ch1"
	userID := "user1"
	staleMeetingID := "stale-mtg"
	newMeetingID := "new-mtg"
	tokenStr := "jwt-token"

	mockStore.EXPECT().GetCallByChannel(channelID).Return(nil, nil)
	mockStore.EXPECT().GetChannelMeeting(channelID).Return("cm-stale", staleMeetingID, "", nil)
	mockStore.EXPECT().GetActiveAppConfigID().Return("cfg1", nil)
	// Cloudflare returns 404 — stored meeting is gone.
	mockRTK.EXPECT().GetMeeting(staleMeetingID).Return(nil, rtkclient.ErrMeetingNotFound)
	// Creates a fresh meeting and saves it.
	mockRTK.EXPECT().CreateMeeting().Return(&rtkclient.Meeting{ID: newMeetingID}, nil)
	mockStore.EXPECT().SaveChannelMeeting(channelID, newMeetingID, "cfg1").Return("cm-new", nil)
	mockRTK.EXPECT().GenerateToken(newMeetingID, gomock.Any(), userID, gomock.Any(), RTKPresetHost).Return(&rtkclient.Token{Token: tokenStr}, nil)
	mockStore.EXPECT().CreateCallSession(gomock.Any()).Return(nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil)
	api.On("PublishWebSocketEvent", WSEventCallStarted, mock.Anything, mock.Anything).Return()
	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{SendPushNotifications: model.NewPointer(false)},
	})

	session, tok, err := a.CreateCall(channelID, userID)
	require.NoError(t, err)
	assert.Equal(t, tokenStr, tok)
	assert.Equal(t, newMeetingID, session.MeetingID)
	// On stale meeting recovery the session must link to the freshly-saved channel meeting id,
	// not the prior stale one.
	assert.Equal(t, "cm-new", session.ChannelMeetingID)
}

// --- JoinCall token failure compensation ---

// TestJoinCall_TokenFailure_CompensatesAdded verifies that when AddCallParticipant
// succeeds and inserts the row, but rtk.GenerateToken subsequently fails, the
// store is rolled back via RemoveCallParticipant so no ghost participant remains.
func TestJoinCall_TokenFailure_CompensatesAdded(t *testing.T) {
ctrl := gomock.NewController(t)
mockRTK := rtkmocks.NewMockRTKClient(ctrl)
mockStore := storemocks.NewMockStore(ctrl)
a, _ := newTestApp(t, mockRTK, mockStore)

callID := "call1"
userID := "user2"
session := &store.CallSession{
ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
Participants: []string{"user1"}, EndAt: 0,
}

mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
mockRTK.EXPECT().GetMeeting("mtg1").Return(&rtkclient.Meeting{ID: "mtg1"}, nil)
mockStore.EXPECT().AddCallParticipant(callID, userID).Return([]string{"user1", userID}, true, true, nil)
mockRTK.EXPECT().GenerateToken("mtg1", callID, userID, gomock.Any(), RTKPresetParticipant).
Return(nil, fmt.Errorf("rtk down"))
// Compensation: user2 was newly added → remove. Other participants remain so no auto-end.
mockStore.EXPECT().RemoveCallParticipant(callID, userID).Return([]string{"user1"}, false, int64(0), nil)

_, _, err := a.JoinCall(callID, userID)
require.Error(t, err)
}

// TestJoinCall_TokenFailure_NoCompensateWhenNotAdded verifies that if the user
// was already a participant (added=false from ON CONFLICT DO NOTHING) and token
// generation fails, we do NOT call RemoveCallParticipant (which would erase a
// legitimate participant).
func TestJoinCall_TokenFailure_NoCompensateWhenNotAdded(t *testing.T) {
ctrl := gomock.NewController(t)
mockRTK := rtkmocks.NewMockRTKClient(ctrl)
mockStore := storemocks.NewMockStore(ctrl)
a, _ := newTestApp(t, mockRTK, mockStore)

callID := "call1"
userID := "user1"
session := &store.CallSession{
ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
Participants: []string{"user1"}, EndAt: 0,
}

mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
mockRTK.EXPECT().GetMeeting("mtg1").Return(&rtkclient.Meeting{ID: "mtg1"}, nil)
// added=false: ON CONFLICT DO NOTHING — user was already in the table.
mockStore.EXPECT().AddCallParticipant(callID, userID).Return([]string{"user1"}, true, false, nil)
mockRTK.EXPECT().GenerateToken("mtg1", callID, userID, gomock.Any(), RTKPresetParticipant).
Return(nil, fmt.Errorf("rtk down"))
// NO RemoveCallParticipant expectation → if it is called, gomock fails the test.

_, _, err := a.JoinCall(callID, userID)
require.Error(t, err)
}

// TestJoinCall_TokenFailure_CompensationAutoEnds verifies that when compensation
// removes the only participant (the one we just added), the call auto-ends and
// the call_ended side-effects are emitted exactly once (endedNow=true path).
func TestJoinCall_TokenFailure_CompensationAutoEnds(t *testing.T) {
ctrl := gomock.NewController(t)
mockRTK := rtkmocks.NewMockRTKClient(ctrl)
mockStore := storemocks.NewMockStore(ctrl)
a, api := newTestApp(t, mockRTK, mockStore)

callID := "call1"
userID := "user1"
// Start with no participants — exotic, but covers the auto-end-on-compensation case.
session := &store.CallSession{
ID: callID, ChannelID: "ch1", MeetingID: "mtg1", CreateAt: 1000,
Participants: []string{}, EndAt: 0,
}

mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
mockRTK.EXPECT().GetMeeting("mtg1").Return(&rtkclient.Meeting{ID: "mtg1"}, nil)
mockStore.EXPECT().AddCallParticipant(callID, userID).Return([]string{userID}, true, true, nil)
mockRTK.EXPECT().GenerateToken("mtg1", callID, userID, gomock.Any(), RTKPresetParticipant).
Return(nil, fmt.Errorf("rtk down"))
// Compensation: this remove transitions the call to ended → endedNow=true.
mockStore.EXPECT().RemoveCallParticipant(callID, userID).Return([]string{}, true, int64(2000), nil)
api.On("PublishWebSocketEvent", WSEventCallEnded,
mock.Anything, mock.AnythingOfType("*model.WebsocketBroadcast")).Return()

_, _, err := a.JoinCall(callID, userID)
require.Error(t, err)
}

// TestJoinCall_TokenFailure_NoEndedDuplicateOnAlreadyEnded verifies that if the
// compensation finds the call has already been ended by another path
// (endedNow=false, endAt!=0), we do NOT re-emit call_ended side effects.
func TestJoinCall_TokenFailure_NoEndedDuplicateOnAlreadyEnded(t *testing.T) {
ctrl := gomock.NewController(t)
mockRTK := rtkmocks.NewMockRTKClient(ctrl)
mockStore := storemocks.NewMockStore(ctrl)
a, _ := newTestApp(t, mockRTK, mockStore)

callID := "call1"
userID := "user2"
session := &store.CallSession{
ID: callID, ChannelID: "ch1", MeetingID: "mtg1",
Participants: []string{"user1"}, EndAt: 0,
}

mockStore.EXPECT().GetCallByID(callID).Return(session, nil)
mockRTK.EXPECT().GetMeeting("mtg1").Return(&rtkclient.Meeting{ID: "mtg1"}, nil)
mockStore.EXPECT().AddCallParticipant(callID, userID).Return([]string{"user1", userID}, true, true, nil)
mockRTK.EXPECT().GenerateToken("mtg1", callID, userID, gomock.Any(), RTKPresetParticipant).
Return(nil, fmt.Errorf("rtk down"))
// Concurrent EndCall already ran → endedNow=false even though endAt!=0.
mockStore.EXPECT().RemoveCallParticipant(callID, userID).Return([]string{}, false, int64(2000), nil)
// NO PublishWebSocketEvent(WSEventCallEnded, ...) expectation → would fail if emitted.

_, _, err := a.JoinCall(callID, userID)
require.Error(t, err)
}
