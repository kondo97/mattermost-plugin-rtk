package kvstore

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	keyCallChannel   = "call:channel:%s"
	keyCallID        = "call:id:%s"
	keyCallMeeting   = "call:meeting:%s"
	keyVoIPToken     = "voip:%s"
	keyWebhookID     = "webhook:id"
	keyWebhookSecret = "webhook:secret"
)

// GetCallByChannel returns the active call for a channel, or nil if none exists or the call has ended.
func (kv Client) GetCallByChannel(channelID string) (*CallSession, error) {
	var session CallSession
	err := kv.client.KV.Get(fmt.Sprintf(keyCallChannel, channelID), &session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get call by channel")
	}
	if session.ID == "" || session.EndAt != 0 {
		return nil, nil
	}
	return &session, nil
}

// GetCallByID returns the call with the given ID, or nil if not found.
func (kv Client) GetCallByID(callID string) (*CallSession, error) {
	var session CallSession
	err := kv.client.KV.Get(fmt.Sprintf(keyCallID, callID), &session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get call by ID")
	}
	if session.ID == "" {
		return nil, nil
	}
	return &session, nil
}

// GetCallByMeetingID returns the call with the given RTK meeting ID, or nil if not found.
func (kv Client) GetCallByMeetingID(meetingID string) (*CallSession, error) {
	var session CallSession
	err := kv.client.KV.Get(fmt.Sprintf(keyCallMeeting, meetingID), &session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get call by meeting ID")
	}
	if session.ID == "" {
		return nil, nil
	}
	return &session, nil
}

// SaveCall persists a call session under the channel, ID, and meeting ID keys.
func (kv Client) SaveCall(session *CallSession) error {
	if session == nil {
		return errors.New("session must not be nil")
	}
	if session.ID == "" {
		return errors.New("session.ID must not be empty")
	}
	if session.ChannelID == "" {
		return errors.New("session.ChannelID must not be empty")
	}
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallChannel, session.ChannelID), session); err != nil {
		return errors.Wrap(err, "failed to save call by channel")
	}
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallID, session.ID), session); err != nil {
		return errors.Wrap(err, "failed to save call by ID")
	}
	if session.MeetingID != "" {
		if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallMeeting, session.MeetingID), session); err != nil {
			return errors.Wrap(err, "failed to save call by meeting ID")
		}
	}
	return nil
}

// UpdateCallParticipants updates the participants list for a call.
func (kv Client) UpdateCallParticipants(callID string, participants []string) error {
	session, err := kv.GetCallByID(callID)
	if err != nil {
		return errors.Wrap(err, "failed to get call for participant update")
	}
	if session == nil {
		return errors.New("call not found")
	}
	session.Participants = participants
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallChannel, session.ChannelID), session); err != nil {
		return errors.Wrap(err, "failed to update call participants by channel")
	}
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallID, callID), session); err != nil {
		return errors.Wrap(err, "failed to update call participants by ID")
	}
	if session.MeetingID != "" {
		if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallMeeting, session.MeetingID), session); err != nil {
			return errors.Wrap(err, "failed to update call participants by meeting ID")
		}
	}
	return nil
}

// EndCall marks a call as ended with the given timestamp.
func (kv Client) EndCall(callID string, endAt int64) error {
	session, err := kv.GetCallByID(callID)
	if err != nil {
		return errors.Wrap(err, "failed to get call for end")
	}
	if session == nil {
		return errors.New("call not found")
	}
	session.EndAt = endAt
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallChannel, session.ChannelID), session); err != nil {
		return errors.Wrap(err, "failed to end call by channel")
	}
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallID, callID), session); err != nil {
		return errors.Wrap(err, "failed to end call by ID")
	}
	if session.MeetingID != "" {
		if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallMeeting, session.MeetingID), session); err != nil {
			return errors.Wrap(err, "failed to end call by meeting ID")
		}
	}
	return nil
}

// StoreVoIPToken stores a VoIP device token for a user.
func (kv Client) StoreVoIPToken(userID, token string) error {
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyVoIPToken, userID), token); err != nil {
		return errors.Wrap(err, "failed to store VoIP token")
	}
	return nil
}

// GetVoIPToken retrieves the VoIP device token for a user.
func (kv Client) GetVoIPToken(userID string) (string, error) {
	var token string
	err := kv.client.KV.Get(fmt.Sprintf(keyVoIPToken, userID), &token)
	if err != nil {
		return "", errors.Wrap(err, "failed to get VoIP token")
	}
	return token, nil
}

// StoreWebhookID persists the registered RTK webhook ID.
func (kv Client) StoreWebhookID(id string) error {
	if _, err := kv.client.KV.Set(keyWebhookID, id); err != nil {
		return errors.Wrap(err, "failed to store webhook ID")
	}
	return nil
}

// GetWebhookID retrieves the registered RTK webhook ID, or empty string if not set.
func (kv Client) GetWebhookID() (string, error) {
	var id string
	err := kv.client.KV.Get(keyWebhookID, &id)
	if err != nil {
		return "", errors.Wrap(err, "failed to get webhook ID")
	}
	return id, nil
}

// StoreWebhookSecret persists the RTK webhook signing secret.
func (kv Client) StoreWebhookSecret(secret string) error {
	if _, err := kv.client.KV.Set(keyWebhookSecret, secret); err != nil {
		return errors.Wrap(err, "failed to store webhook secret")
	}
	return nil
}

// GetWebhookSecret retrieves the RTK webhook signing secret.
func (kv Client) GetWebhookSecret() (string, error) {
	var secret string
	err := kv.client.KV.Get(keyWebhookSecret, &secret)
	if err != nil {
		return "", errors.Wrap(err, "failed to get webhook secret")
	}
	return secret, nil
}
