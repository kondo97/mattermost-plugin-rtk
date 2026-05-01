package rtkclient

import "strings"

// customParticipantIDSeparator separates the call_session ID from the Mattermost
// user ID inside the RTK customParticipantId field. Both halves are alphanumeric
// 26-char Mattermost IDs so a single colon is unambiguous.
const customParticipantIDSeparator = ":"

// BuildCustomParticipantID returns the RTK customParticipantId encoding for a
// given (callID, userID) pair. RTK Meetings are permanent and reusable across
// calls within the same channel, so participantId must include the callID to
// disambiguate webhook events that arrive after a call has ended.
func BuildCustomParticipantID(callID, userID string) string {
	return callID + customParticipantIDSeparator + userID
}

// ParseCustomParticipantID parses a customParticipantId previously produced by
// BuildCustomParticipantID. Returns ok=false if the string does not contain the
// separator (e.g. a legacy token issued before the callID binding was added).
func ParseCustomParticipantID(s string) (callID, userID string, ok bool) {
	idx := strings.Index(s, customParticipantIDSeparator)
	if idx <= 0 || idx == len(s)-1 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}
