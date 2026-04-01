package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	rtkmocks "github.com/kondo97/mattermost-plugin-rtk/server/rtkclient/mocks"
	kvmocks "github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore/mocks"
)

// --- NotificationWillBePushed ---

func TestNotificationWillBePushed_NonCallPost_PassThrough(t *testing.T) {
	p, _ := newTestPlugin(t, nil, nil)

	notif := &model.PushNotification{PostType: "custom_other", ChannelType: model.ChannelTypeDirect}
	result, reason := p.NotificationWillBePushed(notif, "user1")

	assert.Nil(t, result)
	assert.Empty(t, reason)
}

func TestNotificationWillBePushed_CallPost_DM_Suppressed(t *testing.T) {
	p, api := newTestPlugin(t, nil, nil)
	api.On("GetConfig").Return(&model.Config{
		EmailSettings: model.EmailSettings{
			SendPushNotifications:  model.NewPointer(true),
			PushNotificationServer: model.NewPointer("https://push.mattermost.com"),
		},
	})

	notif := &model.PushNotification{PostType: callPostType, ChannelType: model.ChannelTypeDirect}
	result, reason := p.NotificationWillBePushed(notif, "user1")

	assert.Nil(t, result)
	assert.NotEmpty(t, reason, "should suppress DM call notification with a reason")
}

func TestNotificationWillBePushed_CallPost_GM_Suppressed(t *testing.T) {
	p, api := newTestPlugin(t, nil, nil)
	api.On("GetConfig").Return(&model.Config{
		EmailSettings: model.EmailSettings{
			SendPushNotifications:  model.NewPointer(true),
			PushNotificationServer: model.NewPointer("https://push.mattermost.com"),
		},
	})

	notif := &model.PushNotification{PostType: callPostType, ChannelType: model.ChannelTypeGroup}
	result, reason := p.NotificationWillBePushed(notif, "user1")

	assert.Nil(t, result)
	assert.NotEmpty(t, reason, "should suppress GM call notification with a reason")
}

func TestNotificationWillBePushed_CallPost_DM_PushDisabled_PassThrough(t *testing.T) {
	p, api := newTestPlugin(t, nil, nil)
	api.On("GetConfig").Return(&model.Config{
		EmailSettings: model.EmailSettings{
			SendPushNotifications: model.NewPointer(false),
		},
	})

	notif := &model.PushNotification{PostType: callPostType, ChannelType: model.ChannelTypeDirect}
	result, reason := p.NotificationWillBePushed(notif, "user1")

	assert.Equal(t, notif, result, "should pass through when push is disabled")
	assert.Empty(t, reason)
}

func TestNotificationWillBePushed_CallPost_PublicChannel_PassThrough(t *testing.T) {
	p, _ := newTestPlugin(t, nil, nil)

	notif := &model.PushNotification{PostType: callPostType, ChannelType: model.ChannelTypeOpen}
	result, reason := p.NotificationWillBePushed(notif, "user1")

	assert.Nil(t, result)
	assert.Empty(t, reason, "should not suppress public channel call notification")
}

// --- sendPushNotifications ---

func TestSendPushNotifications_PushDisabled_NoOp(t *testing.T) {
	p, api := newTestPlugin(t, nil, nil)
	api.On("GetConfig").Return(&model.Config{
		EmailSettings: model.EmailSettings{SendPushNotifications: model.NewPointer(false)},
	})

	sender := &model.User{Id: "user1"}
	// No GetChannel/GetUsersInChannel/SendPushNotification calls expected
	p.sendPushNotifications("chan1", "post1", sender)
}

func TestSendPushNotifications_NoServer_NoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{
			SendPushNotifications:    model.NewPointer(true),
			PushNotificationServer:   model.NewPointer(""),
			PushNotificationContents: model.NewPointer(model.FullNotification),
		},
	})

	p.sendPushNotifications("chan1", "post1", &model.User{Id: "user1"})
	// No GetChannel call expected since push server is empty
}

func TestSendPushNotifications_NonDMGM_NoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{
			SendPushNotifications:    model.NewPointer(true),
			PushNotificationServer:   model.NewPointer("https://push.mattermost.com"),
			PushNotificationContents: model.NewPointer(model.FullNotification),
		},
	})
	api.On("GetChannel", "chan1").Return(&model.Channel{
		Id:   "chan1",
		Type: model.ChannelTypeOpen,
	}, nil)

	p.sendPushNotifications("chan1", "post1", &model.User{Id: "user1"})
	// No GetUsersInChannel/SendPushNotification calls expected for public channel
}

func TestSendPushNotifications_DM_FullNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{
			SendPushNotifications:    model.NewPointer(true),
			PushNotificationServer:   model.NewPointer("https://push.mattermost.com"),
			PushNotificationContents: model.NewPointer(model.FullNotification),
		},
		TeamSettings: model.TeamSettings{
			TeammateNameDisplay: model.NewPointer(model.ShowUsername),
		},
	})
	api.On("GetChannel", "chan1").Return(&model.Channel{
		Id:   "chan1",
		Type: model.ChannelTypeDirect,
	}, nil)
	api.On("GetUsersInChannel", "chan1", model.ChannelSortByUsername, 0, 8).Return([]*model.User{
		{Id: "user1", Username: "sender"},
		{Id: "user2", Username: "receiver"},
	}, nil)
	api.On("GetPreferenceForUser", "user2", model.PreferenceCategoryDisplaySettings, model.PreferenceNameNameFormat).
		Return(model.Preference{}, &model.AppError{Message: "not found"})
	api.On("SendPushNotification", mock.MatchedBy(func(n *model.PushNotification) bool {
		return n.Type == model.PushTypeMessage &&
			n.SubType == model.PushSubTypeCalls &&
			n.ChannelId == "chan1" &&
			n.PostId == "post1" &&
			n.SenderId == "user1" &&
			n.ChannelType == model.ChannelTypeDirect
	}), "user2").Return(nil)

	sender := &model.User{Id: "user1", Username: "sender"}
	p.sendPushNotifications("chan1", "post1", sender)

	api.AssertExpectations(t)
}

func TestSendPushNotifications_SkipsSender(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	p, api := newTestPlugin(t, mockRTK, mockStore)

	api.On("GetConfig").Maybe().Return(&model.Config{
		EmailSettings: model.EmailSettings{
			SendPushNotifications:    model.NewPointer(true),
			PushNotificationServer:   model.NewPointer("https://push.mattermost.com"),
			PushNotificationContents: model.NewPointer(model.FullNotification),
		},
		TeamSettings: model.TeamSettings{
			TeammateNameDisplay: model.NewPointer(model.ShowUsername),
		},
	})
	api.On("GetChannel", "chan1").Return(&model.Channel{
		Id:   "chan1",
		Type: model.ChannelTypeDirect,
	}, nil)
	// Only one member returned (the sender) — nobody else to notify
	api.On("GetUsersInChannel", "chan1", model.ChannelSortByUsername, 0, 8).Return([]*model.User{
		{Id: "user1", Username: "sender"},
	}, nil)
	// SendPushNotification should NOT be called

	p.sendPushNotifications("chan1", "post1", &model.User{Id: "user1", Username: "sender"})
}

// --- helper functions ---

func TestBuildPushMessage(t *testing.T) {
	msg := buildPushMessage("Alice")
	assert.Contains(t, msg, "Alice")
	assert.Contains(t, msg, "\u200b", "zero-width space required to trigger mobile call ringing UI")
}

func TestBuildGenericPushMessage(t *testing.T) {
	msg := buildGenericPushMessage()
	assert.NotEmpty(t, msg)
	assert.Contains(t, msg, "\u200b", "zero-width space required to trigger mobile call ringing UI")
}

func TestGetChannelNameForNotification_DM(t *testing.T) {
	channel := &model.Channel{Type: model.ChannelTypeDirect}
	sender := &model.User{Username: "alice"}
	name := getChannelNameForNotification(channel, sender, nil, model.ShowUsername, "user2")
	assert.Equal(t, "alice", name)
}

func TestGetChannelNameForNotification_GM(t *testing.T) {
	channel := &model.Channel{Type: model.ChannelTypeGroup}
	sender := &model.User{Id: "user1", Username: "alice"}
	members := []*model.User{
		{Id: "user1", Username: "alice"},
		{Id: "user2", Username: "bob"},
		{Id: "user3", Username: "carol"},
	}
	// receiverID=user3, so channel name should be alice, bob
	name := getChannelNameForNotification(channel, sender, members, model.ShowUsername, "user3")
	assert.Contains(t, name, "alice")
	assert.Contains(t, name, "bob")
	assert.NotContains(t, name, "carol")
}

func TestCheckIDLoadedLicense_NilLicense(t *testing.T) {
	p, api := newTestPlugin(t, nil, nil)
	api.On("GetLicense").Return(nil)

	result := p.checkIDLoadedLicense()
	require.False(t, result)
}
