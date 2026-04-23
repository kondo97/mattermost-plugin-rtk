package app

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

const pushSubTypeCallsEnded = "calls_ended"

// NotificationWillBePushed intercepts push notifications before they are sent to mobile devices.
// For call start posts in DM/GM channels where the plugin can send its own push notification,
// we suppress Mattermost's default notification and send one with SubType=calls instead
// (which triggers the native call ringing UI on iOS/Android).
// Falls back to passing the original notification through if push is unavailable.
func (a *App) NotificationWillBePushed(notification *model.PushNotification, userID string) (*model.PushNotification, string) {
	// Only handle call start posts.
	if notification.PostType != CallPostType {
		return nil, ""
	}
	// Only consider overriding notifications for DM/GM channels.
	if notification.ChannelType != model.ChannelTypeDirect && notification.ChannelType != model.ChannelTypeGroup {
		return nil, ""
	}
	// Check whether the plugin is actually able to send push notifications.
	// If not, fall back to letting the server send its default notification.
	cfg := a.api.GetConfig()
	if cfg == nil {
		return notification, ""
	}
	if cfg.EmailSettings.SendPushNotifications == nil || !*cfg.EmailSettings.SendPushNotifications {
		return notification, ""
	}
	if cfg.EmailSettings.PushNotificationServer == nil || *cfg.EmailSettings.PushNotificationServer == "" {
		return notification, ""
	}
	// Plugin can send its own calls-specific push notification; suppress the default one.
	return nil, "rtk plugin will handle this notification"
}

// sendPushNotifications sends a mobile push notification to all eligible members
// of a DM or GM channel when a call starts. Best-effort: errors are logged only.
func (a *App) sendPushNotifications(channelID, postID, threadID string, sender *model.User) {
	cfg := a.api.GetConfig()
	if cfg == nil {
		return
	}
	if cfg.EmailSettings.SendPushNotifications == nil || !*cfg.EmailSettings.SendPushNotifications {
		return
	}
	if cfg.EmailSettings.PushNotificationServer == nil || *cfg.EmailSettings.PushNotificationServer == "" {
		return
	}

	channel, appErr := a.api.GetChannel(channelID)
	if appErr != nil {
		a.api.LogError("sendPushNotifications: GetChannel failed", "channel_id", channelID, "err", appErr.Error())
		return
	}
	if channel.Type != model.ChannelTypeDirect && channel.Type != model.ChannelTypeGroup {
		return
	}

	members, appErr := a.api.GetUsersInChannel(channelID, model.ChannelSortByUsername, 0, 8)
	if appErr != nil {
		a.api.LogError("sendPushNotifications: GetUsersInChannel failed", "channel_id", channelID, "err", appErr.Error())
		return
	}

	pushContents := model.FullNotification
	if cfg.EmailSettings.PushNotificationContents != nil {
		pushContents = *cfg.EmailSettings.PushNotificationContents
	}

	// Compute once before the loop to avoid repeated license API calls.
	idLoadedEnabled := pushContents == model.IdLoadedNotification && a.checkIDLoadedLicense()

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
			RootId:      threadID,
			SenderId:    sender.Id,
			ChannelType: channel.Type,
			Message:     buildGenericPushMessage(),
		}

		if idLoadedEnabled {
			msg.IsIdLoaded = true
		} else {
			nameFormat := a.getNameFormat(member.Id)
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

		if err := a.api.SendPushNotification(msg, member.Id); err != nil {
			a.api.LogError("sendPushNotifications: SendPushNotification failed", "user_id", member.Id, "err", err.Error())
		}
	}
}

// checkIDLoadedLicense reports whether the server license supports ID-loaded push notifications.
func (a *App) checkIDLoadedLicense() bool {
	license := a.api.GetLicense()
	if license == nil || license.Features == nil || license.Features.IDLoadedPushNotifications == nil {
		return false
	}
	return *license.Features.IDLoadedPushNotifications
}

// getNameFormat returns the display name format for a user's push notification.
// It checks the user's preference first, then falls back to the server default.
func (a *App) getNameFormat(userID string) string {
	pref, appErr := a.api.GetPreferenceForUser(userID, model.PreferenceCategoryDisplaySettings, model.PreferenceNameNameFormat)
	if appErr == nil && pref.Value != "" {
		return pref.Value
	}
	cfg := a.api.GetConfig()
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
// The leading zero-width space (\u200b) signals the mobile app to trigger the call ringing UI
// regardless of the PushNotificationContents server setting.
func buildGenericPushMessage() string {
	return "\u200bIncoming call"
}

// sendEndCallPushNotifications sends a mobile push notification (Type=clear, SubType=calls_ended)
// to all eligible members of a DM or GM channel when a call ends.
// This allows the mobile app to dismiss the ringing/incoming call UI.
// Best-effort: errors are logged only.
func (a *App) sendEndCallPushNotifications(channelID, postID, creatorID string) {
	if postID == "" {
		return
	}

	cfg := a.api.GetConfig()
	if cfg == nil {
		return
	}
	if cfg.EmailSettings.SendPushNotifications == nil || !*cfg.EmailSettings.SendPushNotifications {
		return
	}
	if cfg.EmailSettings.PushNotificationServer == nil || *cfg.EmailSettings.PushNotificationServer == "" {
		return
	}

	channel, appErr := a.api.GetChannel(channelID)
	if appErr != nil {
		a.api.LogError("sendEndCallPushNotifications: GetChannel failed", "channel_id", channelID, "err", appErr.Error())
		return
	}
	if channel.Type != model.ChannelTypeDirect && channel.Type != model.ChannelTypeGroup {
		return
	}

	members, appErr := a.api.GetUsersInChannel(channelID, model.ChannelSortByUsername, 0, 8)
	if appErr != nil {
		a.api.LogError("sendEndCallPushNotifications: GetUsersInChannel failed", "channel_id", channelID, "err", appErr.Error())
		return
	}

	msg := &model.PushNotification{
		Version:     model.PushMessageV2,
		Type:        model.PushTypeClear,
		SubType:     pushSubTypeCallsEnded,
		TeamId:      channel.TeamId,
		ChannelId:   channelID,
		PostId:      postID,
		ChannelName: channel.DisplayName,
	}

	for _, member := range members {
		if member.Id == creatorID {
			continue
		}
		if err := a.api.SendPushNotification(msg, member.Id); err != nil {
			a.api.LogError("sendEndCallPushNotifications: SendPushNotification failed", "user_id", member.Id, "err", err.Error())
		}
	}
}
