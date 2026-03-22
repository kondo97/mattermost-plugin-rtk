package kvstore

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

const (
	keyCallChannel        = "call:channel:%s"
	keyCallID             = "call:id:%s"
	keyHeartbeat          = "heartbeat:%s:%s"
	keyVoIPToken          = "voip:%s"
	keyActiveChannelIndex = "calls:index:active_channels"
)

// GetCallByChannel returns the active call for a channel, or nil if none exists.
func (kv Client) GetCallByChannel(channelID string) (*CallSession, error) {
	var session CallSession
	err := kv.client.KV.Get(fmt.Sprintf(keyCallChannel, channelID), &session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get call by channel")
	}
	if session.ID == "" {
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

// GetAllActiveCalls returns all currently active calls (EndAt == 0).
// Uses an index key to track active channel IDs.
func (kv Client) GetAllActiveCalls() ([]*CallSession, error) {
	channelIDs, err := kv.getActiveChannelIndex()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get active channel index")
	}

	sessions := make([]*CallSession, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		session, err := kv.GetCallByChannel(channelID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get call for channel %s", channelID)
		}
		if session != nil && session.EndAt == 0 {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// SaveCall persists a call session under both the channel and ID keys, and updates the active index.
func (kv Client) SaveCall(session *CallSession) error {
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallChannel, session.ChannelID), session); err != nil {
		return errors.Wrap(err, "failed to save call by channel")
	}
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyCallID, session.ID), session); err != nil {
		return errors.Wrap(err, "failed to save call by ID")
	}
	if err := kv.addToActiveChannelIndex(session.ChannelID); err != nil {
		return errors.Wrap(err, "failed to update active channel index")
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
	if err := kv.removeFromActiveChannelIndex(session.ChannelID); err != nil {
		return errors.Wrap(err, "failed to remove from active channel index")
	}
	return nil
}

// SetHeartbeat records a heartbeat timestamp for a participant in a call.
func (kv Client) SetHeartbeat(callID, userID string, ts int64) error {
	if _, err := kv.client.KV.Set(fmt.Sprintf(keyHeartbeat, callID, userID), ts); err != nil {
		return errors.Wrap(err, "failed to set heartbeat")
	}
	return nil
}

// GetHeartbeat returns the last heartbeat timestamp for a participant, or 0 if not found.
func (kv Client) GetHeartbeat(callID, userID string) (int64, error) {
	var ts int64
	err := kv.client.KV.Get(fmt.Sprintf(keyHeartbeat, callID, userID), &ts)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get heartbeat")
	}
	return ts, nil
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

// --- Active Channel Index Helpers ---

func (kv Client) getActiveChannelIndex() ([]string, error) {
	var raw []byte
	err := kv.client.KV.Get(keyActiveChannelIndex, &raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read active channel index")
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var channelIDs []string
	if err := json.Unmarshal(raw, &channelIDs); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal active channel index")
	}
	return channelIDs, nil
}

func (kv Client) addToActiveChannelIndex(channelID string) error {
	channelIDs, err := kv.getActiveChannelIndex()
	if err != nil {
		return err
	}
	for _, id := range channelIDs {
		if id == channelID {
			return nil // already in index
		}
	}
	channelIDs = append(channelIDs, channelID)
	if _, err := kv.client.KV.Set(keyActiveChannelIndex, channelIDs); err != nil {
		return errors.Wrap(err, "failed to write active channel index")
	}
	return nil
}

func (kv Client) removeFromActiveChannelIndex(channelID string) error {
	channelIDs, err := kv.getActiveChannelIndex()
	if err != nil {
		return err
	}
	updated := make([]string, 0, len(channelIDs))
	for _, id := range channelIDs {
		if id != channelID {
			updated = append(updated, id)
		}
	}
	if _, err := kv.client.KV.Set(keyActiveChannelIndex, updated); err != nil {
		return errors.Wrap(err, "failed to write active channel index")
	}
	return nil
}
