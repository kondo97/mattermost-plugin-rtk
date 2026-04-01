package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// NotificationWillBePushed intercepts push notifications before they are sent to mobile devices.
// For call start posts in DM/GM channels, we suppress Mattermost's default notification
// because the plugin sends its own notification with SubType=calls to trigger the ringing UI.
func (p *Plugin) NotificationWillBePushed(notification *model.PushNotification, userID string) (*model.PushNotification, string) {
	if notification.PostType != callPostType {
		return nil, ""
	}
	if notification.ChannelType == model.ChannelTypeDirect || notification.ChannelType == model.ChannelTypeGroup {
		return nil, "rtk plugin will handle this notification"
	}
	return nil, ""
}

// sendPushNotifications sends a mobile push notification to all eligible members
// of a DM or GM channel when a call starts. Best-effort: errors are logged only.
func (p *Plugin) sendPushNotifications(channelID, postID string, sender *model.User) {
	cfg := p.API.GetConfig()
	if cfg == nil {
		return
	}
	if cfg.EmailSettings.SendPushNotifications == nil || !*cfg.EmailSettings.SendPushNotifications {
		return
	}
	if cfg.EmailSettings.PushNotificationServer == nil || *cfg.EmailSettings.PushNotificationServer == "" {
		return
	}

	channel, appErr := p.API.GetChannel(channelID)
	if appErr != nil {
		p.API.LogError("sendPushNotifications: GetChannel failed", "channel_id", channelID, "err", appErr.Error())
		return
	}
	if channel.Type != model.ChannelTypeDirect && channel.Type != model.ChannelTypeGroup {
		return
	}

	members, appErr := p.API.GetUsersInChannel(channelID, model.ChannelSortByUsername, 0, 8)
	if appErr != nil {
		p.API.LogError("sendPushNotifications: GetUsersInChannel failed", "channel_id", channelID, "err", appErr.Error())
		return
	}

	pushContents := model.FullNotification
	if cfg.EmailSettings.PushNotificationContents != nil {
		pushContents = *cfg.EmailSettings.PushNotificationContents
	}

	for _, member := range members {
		if member.Id == sender.Id {
			continue
		}

		msg := &model.PushNotification{
			Version:     model.PushMessageV2,
			Type:        model.PushTypeMessage,
			SubType:     model.PushSubTypeCalls,
			TeamId:      channel.TeamId,
			ChannelId:   channelID,
			PostId:      postID,
			SenderId:    sender.Id,
			ChannelType: channel.Type,
			Message:     buildGenericPushMessage(),
		}

		if pushContents == model.IdLoadedNotification && p.checkIDLoadedLicense() {
			msg.IsIdLoaded = true
		} else {
			nameFormat := p.getNameFormat(member.Id)
			senderName := sender.GetDisplayName(nameFormat)
			msg.SenderName = senderName
			msg.ChannelName = getChannelNameForNotification(channel, sender, members, nameFormat, member.Id)
			if pushContents == model.GenericNoChannelNotification && channel.Type != model.ChannelTypeDirect {
				msg.ChannelName = ""
			}
			if pushContents == model.FullNotification {
				msg.Message = buildPushMessage(senderName)
			}
		}

		if err := p.API.SendPushNotification(msg, member.Id); err != nil {
			p.API.LogError("sendPushNotifications: SendPushNotification failed", "user_id", member.Id, "err", err.Error())
		}
	}
}

// checkIDLoadedLicense reports whether the server license supports ID-loaded push notifications.
func (p *Plugin) checkIDLoadedLicense() bool {
	license := p.API.GetLicense()
	if license == nil || license.Features == nil || license.Features.IDLoadedPushNotifications == nil {
		return false
	}
	return *license.Features.IDLoadedPushNotifications
}

// getNameFormat returns the display name format for a user's push notification.
// It checks the user's preference first, then falls back to the server default.
func (p *Plugin) getNameFormat(userID string) string {
	pref, appErr := p.API.GetPreferenceForUser(userID, model.PreferenceCategoryDisplaySettings, model.PreferenceNameNameFormat)
	if appErr == nil && pref.Value != "" {
		return pref.Value
	}
	cfg := p.API.GetConfig()
	if cfg != nil && cfg.TeamSettings.TeammateNameDisplay != nil {
		return *cfg.TeamSettings.TeammateNameDisplay
	}
	return model.ShowUsername
}

// getChannelNameForNotification returns the channel display name used in push notifications.
// For DM channels it returns the sender's name; for GM channels it lists all other members.
func getChannelNameForNotification(channel *model.Channel, sender *model.User, members []*model.User, nameFormat, receiverID string) string {
	if channel.Type == model.ChannelTypeDirect {
		return sender.GetDisplayName(nameFormat)
	}
	names := make([]string, 0, len(members))
	for _, m := range members {
		if m.Id != receiverID {
			names = append(names, m.GetDisplayName(nameFormat))
		}
	}
	if len(names) == 0 {
		return channel.DisplayName
	}
	return strings.Join(names, ", ")
}

// buildPushMessage returns the full push notification message with the sender name.
// The leading zero-width space (\u200b) signals the mobile app to trigger the call ringing UI.
func buildPushMessage(senderName string) string {
	return fmt.Sprintf("\u200b%s is calling you", senderName)
}

// buildGenericPushMessage returns a generic push notification message for incoming calls.
func buildGenericPushMessage() string {
	return "Incoming call"
}
