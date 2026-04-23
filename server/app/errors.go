package app

import "errors"

var (
	// ErrCallAlreadyActive is returned when a call already exists in the channel.
	ErrCallAlreadyActive = errors.New("call already active in channel")
	// ErrCallNotFound is returned when a call is not found or has already ended.
	ErrCallNotFound = errors.New("call not found or already ended")
	// ErrNotParticipant is returned when the user is not a participant of the call.
	ErrNotParticipant = errors.New("user is not a participant of this call")
	// ErrUnauthorized is returned when the user is not authorized to perform the action.
	ErrUnauthorized = errors.New("only the call creator can perform this action")
	// ErrRTKNotConfigured is returned when Cloudflare RTK credentials are not configured.
	ErrRTKNotConfigured = errors.New("cloudflare RTK credentials are not configured")
	// ErrNotChannelMember is returned when the user is not a member of the channel.
	ErrNotChannelMember = errors.New("user is not a member of this channel")
)
