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
	// errInvalidUser is returned when an API call is made without a user ID.
	errInvalidUser = errors.New("user ID is required")
	// ErrForbidden is returned when the user lacks the required permission.
	ErrForbidden = errors.New("you do not have permission to perform this action")
	// ErrCallsDisabled is returned when calls are explicitly disabled in the channel.
	ErrCallsDisabled = errors.New("calls are disabled in this channel")
)
