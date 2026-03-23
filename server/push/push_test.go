package push

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
)

func newTestSender(t *testing.T) (*Sender, *plugintest.API) {
	t.Helper()
	api := &plugintest.API{}
	anyArgs := func(n int) []any {
		args := make([]any, n)
		for i := range args {
			args[i] = mock.Anything
		}
		return args
	}
	for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		api.On("LogWarn", anyArgs(n)...).Maybe().Return()
	}
	t.Cleanup(func() { api.AssertExpectations(t) })
	return NewSender(api), api
}

func dmChannel() *model.Channel {
	return &model.Channel{
		Id:          "channel1",
		Type:        model.ChannelTypeDirect,
		TeamId:      "",
		DisplayName: "",
		Name:        "user1__user2",
	}
}

func testSession() *kvstore.CallSession {
	return &kvstore.CallSession{
		ID:           "call1",
		ChannelID:    "channel1",
		CreatorID:    "creator1",
		MeetingID:    "meeting1",
		PostID:       "post1",
		Participants: []string{"creator1"},
	}
}

// --- SendIncomingCall ---

func TestSendIncomingCall_DMChannel_Success(t *testing.T) {
	s, api := newTestSender(t)
	session := testSession()
	ch := dmChannel()
	caller := &model.User{Id: "creator1", Username: "alice"}
	members := model.ChannelMembers{
		{UserId: "creator1"},
		{UserId: "user2"},
	}

	api.On("GetChannel", session.ChannelID).Return(ch, nil)
	api.On("GetUser", session.CreatorID).Return(caller, nil)
	api.On("GetChannelMembers", session.ChannelID, 0, pushMembersPerPage).Return(members, nil)
	api.On("SendPushNotification", mock.MatchedBy(func(n *model.PushNotification) bool {
		return n.Type == pushTypeMessage &&
			string(n.SubType) == pushSubTypeCalls &&
			n.SenderId == session.CreatorID &&
			n.SenderName == caller.Username &&
			n.RootId == session.PostID
	}), "user2").Return(nil)

	err := s.SendIncomingCall(session)
	require.NoError(t, err)
	api.AssertCalled(t, "SendPushNotification", mock.Anything, "user2")
	api.AssertNotCalled(t, "SendPushNotification", mock.Anything, "creator1")
}

func TestSendIncomingCall_NonDMChannel_Skipped(t *testing.T) {
	s, api := newTestSender(t)
	session := testSession()
	ch := &model.Channel{
		Id:   session.ChannelID,
		Type: model.ChannelTypeOpen,
	}

	api.On("GetChannel", session.ChannelID).Return(ch, nil)

	err := s.SendIncomingCall(session)
	require.NoError(t, err)
	api.AssertNotCalled(t, "GetUser", mock.Anything)
	api.AssertNotCalled(t, "SendPushNotification", mock.Anything, mock.Anything)
}

func TestSendIncomingCall_GetChannelFails(t *testing.T) {
	s, api := newTestSender(t)
	session := testSession()
	appErr := model.NewAppError("GetChannel", "not_found", nil, "", 404)

	api.On("GetChannel", session.ChannelID).Return(nil, appErr)

	err := s.SendIncomingCall(session)
	assert.Error(t, err)
}

func TestSendIncomingCall_SendPushFails_ContinuesLoop(t *testing.T) {
	s, api := newTestSender(t)
	session := testSession()
	ch := dmChannel()
	caller := &model.User{Id: "creator1", Username: "alice"}
	members := model.ChannelMembers{
		{UserId: "user2"},
		{UserId: "user3"},
	}
	appErr := model.NewAppError("SendPush", "push_failed", nil, "", 500)

	api.On("GetChannel", session.ChannelID).Return(ch, nil)
	api.On("GetUser", session.CreatorID).Return(caller, nil)
	api.On("GetChannelMembers", session.ChannelID, 0, pushMembersPerPage).Return(members, nil)
	// user2 fails, user3 succeeds
	api.On("SendPushNotification", mock.Anything, "user2").Return(appErr)
	api.On("SendPushNotification", mock.Anything, "user3").Return(nil)

	err := s.SendIncomingCall(session)
	// method returns nil — per-member errors are best-effort
	require.NoError(t, err)
	api.AssertCalled(t, "SendPushNotification", mock.Anything, "user3")
}

// --- SendCallEnded ---

func TestSendCallEnded_DMChannel_Success(t *testing.T) {
	s, api := newTestSender(t)
	session := testSession()
	ch := dmChannel()
	caller := &model.User{Id: "creator1", Username: "alice"}
	members := model.ChannelMembers{
		{UserId: "creator1"},
		{UserId: "user2"},
	}

	api.On("GetChannel", session.ChannelID).Return(ch, nil)
	api.On("GetUser", session.CreatorID).Return(caller, nil)
	api.On("GetChannelMembers", session.ChannelID, 0, pushMembersPerPage).Return(members, nil)
	api.On("SendPushNotification", mock.MatchedBy(func(n *model.PushNotification) bool {
		return n.Type == pushTypeClear && string(n.SubType) == pushSubTypeEnded
	}), "user2").Return(nil)

	err := s.SendCallEnded(session)
	require.NoError(t, err)
	api.AssertCalled(t, "SendPushNotification", mock.Anything, "user2")
}

func TestSendCallEnded_NonDMChannel_Skipped(t *testing.T) {
	s, api := newTestSender(t)
	session := testSession()
	ch := &model.Channel{Id: session.ChannelID, Type: model.ChannelTypePrivate}

	api.On("GetChannel", session.ChannelID).Return(ch, nil)

	err := s.SendCallEnded(session)
	require.NoError(t, err)
	api.AssertNotCalled(t, "SendPushNotification", mock.Anything, mock.Anything)
}

func TestSendCallEnded_GMChannel_Success(t *testing.T) {
	s, api := newTestSender(t)
	session := testSession()
	ch := &model.Channel{
		Id:          session.ChannelID,
		Type:        model.ChannelTypeGroup,
		DisplayName: "Alice, Bob, Carol",
	}
	caller := &model.User{Id: "creator1", Username: "alice"}
	members := model.ChannelMembers{{UserId: "user2"}}

	api.On("GetChannel", session.ChannelID).Return(ch, nil)
	api.On("GetUser", session.CreatorID).Return(caller, nil)
	api.On("GetChannelMembers", session.ChannelID, 0, pushMembersPerPage).Return(members, nil)
	api.On("SendPushNotification", mock.Anything, "user2").Return(nil)

	err := s.SendCallEnded(session)
	require.NoError(t, err)
}
