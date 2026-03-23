package push

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"

	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
)

const (
	pushTypeMessage    = "message"
	pushTypeClear      = "clear"
	pushSubTypeCalls   = "calls"
	pushSubTypeEnded   = "calls_ended"
	pushMembersPerPage = 8
)

// Sender is the concrete implementation of PushSender.
type Sender struct {
	api plugin.API
}

// NewSender creates a new Sender with the given Mattermost plugin API.
func NewSender(api plugin.API) *Sender {
	return &Sender{api: api}
}

// SendIncomingCall sends a call-started push notification to DM/GM channel members.
func (s *Sender) SendIncomingCall(session *kvstore.CallSession) error {
	return s.sendToMembers(session, pushTypeMessage, pushSubTypeCalls)
}

// SendCallEnded sends a call-ended clear push notification to DM/GM channel members.
func (s *Sender) SendCallEnded(session *kvstore.CallSession) error {
	return s.sendToMembers(session, pushTypeClear, pushSubTypeEnded)
}

// sendToMembers is the shared implementation for SendIncomingCall and SendCallEnded.
// It fetches up to 8 channel members and dispatches a push notification to each
// member that is not the call creator. Only DM and GM channels are supported.
func (s *Sender) sendToMembers(session *kvstore.CallSession, notifType, subType string) error {
	channel, appErr := s.api.GetChannel(session.ChannelID)
	if appErr != nil {
		return fmt.Errorf("push: GetChannel failed: %w", appErr)
	}

	// Only send push for DM and GM channels (aligns with Mattermost Calls plugin).
	if channel.Type != model.ChannelTypeDirect && channel.Type != model.ChannelTypeGroup {
		return nil
	}

	caller, appErr := s.api.GetUser(session.CreatorID)
	if appErr != nil {
		return fmt.Errorf("push: GetUser failed: %w", appErr)
	}

	channelName := channel.DisplayName
	if channelName == "" {
		channelName = channel.Name
	}

	members, appErr := s.api.GetChannelMembers(session.ChannelID, 0, pushMembersPerPage)
	if appErr != nil {
		return fmt.Errorf("push: GetChannelMembers failed: %w", appErr)
	}

	for _, member := range members {
		if member.UserId == session.CreatorID {
			continue
		}

		n := &model.PushNotification{
			Type:        notifType,
			SubType:     model.PushSubType(subType),
			ChannelId:   session.ChannelID,
			TeamId:      channel.TeamId,
			SenderId:    session.CreatorID,
			SenderName:  caller.Username,
			ChannelName: channelName,
			RootId:      session.PostID,
		}

		if appErr := s.api.SendPushNotification(n, member.UserId); appErr != nil {
			s.api.LogWarn("push: SendPushNotification failed",
				"call_id", session.ID,
				"channel_id", session.ChannelID,
				"user_id", member.UserId,
				"err", appErr.Error())
		}
	}

	return nil
}
