# Unit 3: Webapp - Channel UI — Business Logic Model

## Overview

This document describes the business logic flows for all Unit 3 components. All flows are technology-agnostic; implementation details are addressed in NFR Design and Code Generation.

---

## BL-001: Plugin Initialization

**Trigger**: Mattermost plugin `initialize()` called at page load.

**Flow**:
1. Fetch `GET /plugins/{id}/api/v1/config/status`
2. Dispatch `setPluginEnabled(response.enabled)` to Redux
3. Register Redux reducer with Mattermost store
4. Register all 5 WebSocket event handlers
5. Register all UI components with the plugin registry
6. On WebSocket reconnect: re-fetch config status and dispatch `setPluginEnabled`

**Outcome**: Plugin state is ready; components begin rendering based on Redux state.

---

## BL-002: Channel Header Button State Resolution

**Trigger**: ChannelHeaderButton renders or Redux state changes.

**Input**: `channelId`, `pluginEnabled`, `callsByChannel`, `myActiveCall`

**State Resolution Logic** (evaluated in priority order):

1. `pluginEnabled === false` → render nothing (hidden)
2. `loading === true` (local state) → render "Starting call..." spinner (disabled)
3. `callsByChannel[channelId]` exists AND `currentUser.id` in `participants` → render "In Call" (disabled)
4. `callsByChannel[channelId]` exists AND `currentUser.id` NOT in `participants` → render "Join Call" (enabled)
5. No active call in channel → render "Start Call" (enabled)

**Click Handling**:
- "Start Call" clicked:
  1. Set local `loading = true`
  2. Call `POST /plugins/{id}/api/v1/calls` with `{ channel_id }`
  3. On success: dispatch `setMyActiveCall({ callId, channelId, token })`; open call tab; play sound cue; set `loading = false`
  4. On error: show error modal; set `loading = false`
- "Join Call" clicked:
  1. If `myActiveCall !== null` AND `myActiveCall.channelId !== channelId`: show SwitchCallModal
  2. Else: proceed to join flow (BL-004)
- "In Call" → no-op (button is disabled)

---

## BL-003: Sound Cue on Call Start

**Trigger**: Successful `POST /api/v1/calls` response received by the current user.

**Flow**:
- Trigger Mattermost's desktop notification hook to play a notification sound using the platform's existing notification sound infrastructure.
- This fires on the Mattermost side when the API response is received — not when the call page loads.

---

## BL-004: Join Call Flow

**Trigger**: User clicks "Join Call" (from ChannelHeaderButton, ToastBar, or IncomingCallNotification).

**Pre-condition**: `myActiveCall === null` OR user confirmed switch via SwitchCallModal.

**Flow**:
1. If switching: call `POST /plugins/{id}/api/v1/calls/{myActiveCall.callId}/leave` (fire-and-forget)
2. Call `POST /plugins/{id}/api/v1/calls/{callId}/token`
3. On success: dispatch `setMyActiveCall({ callId, channelId, token })`; open call tab with `?token=<jwt>`
4. On error: show error modal

---

## BL-005: Toast Bar Visibility Logic

**Trigger**: Current channel changes or `callsByChannel` / `myActiveCall` Redux state changes.

**Visibility condition**:
- `callsByChannel[currentChannelId]` exists (active call in current channel)
- AND current user is NOT in `participants`
- AND local `dismissed === false`

**Dismiss**:
- User clicks dismiss → set local `dismissed = true` (not persisted; resets on page reload)
- `custom_cf_call_ended` for this call → clear local `dismissed`, call data removed from Redux

**Join action from toast bar**: Same as BL-004 (checks `myActiveCall` first, shows SwitchCallModal if needed).

---

## BL-006: Floating Widget

**Trigger**: `myActiveCall !== null` in Redux.

**Content**:
- Channel name (looked up via Mattermost store from `myActiveCall.channelId`)
- Participant count (from `callsByChannel[myActiveCall.channelId].participants.length`)
- Participant avatars (up to 3 user IDs, rendered as Mattermost user avatars)
- Call duration timer (computed from `callsByChannel[myActiveCall.channelId].startAt`, updated every second via local `setInterval`)
- "Open in new tab" button

**"Open in new tab" flow**:
1. Call `POST /plugins/{id}/api/v1/calls/{callId}/token` to obtain a fresh JWT
2. Open `window.open('/plugins/{id}/call?token=<jwt>', '_blank')`
3. On error: show error modal

**Note**: No mute/unmute control in the FloatingWidget. Mute is only available in the call tab (Unit 4).

**Hide condition**: `myActiveCall === null` (cleared by `custom_cf_user_left` or `custom_cf_call_ended`)

---

## BL-007: Switch Call Modal

**Trigger**: User attempts to join call B while `myActiveCall` refers to call A (different channel).

**Content**: "You are already in a call. Do you want to leave and join the new call?"

**Actions**:
- "Cancel" → dismiss modal, no action
- "Leave and join new call" → execute BL-004 with leave-first flag

**Context**: Modal is rendered inline by the initiating component (ChannelHeaderButton or ToastBar). Not a global modal registration.

---

## BL-008: Incoming Call Notification (DM/GM Ringing)

**Trigger**: `incomingCall !== null` in Redux.

**Condition for Redux to set `incomingCall`**:
- `custom_cf_call_started` WS event received
- Channel type of `channelId` is `D` or `G`
- `creator_id != currentUser.id`

**Content**: Channel name, caller's display name, "Ignore" and "Join" buttons.

**Auto-dismiss**:
- Start `setTimeout` for 30 seconds when `incomingCall` is set
- On timeout: dispatch `clearIncomingCall()` locally (no server call)
- Cancel the timeout if `incomingCall` is cleared before 30 seconds (cleanup in `useEffect` return)

**"Ignore" flow**:
1. Call `POST /plugins/{id}/api/v1/calls/{callId}/dismiss`
2. Server emits `custom_cf_notification_dismissed` WS event to all user sessions
3. Redux handler clears `incomingCall` (via WS event, not optimistically)

**"Join" flow**: Same as BL-004 (checks `myActiveCall` first, shows SwitchCallModal if needed).

---

## BL-009: Real-Time Participant Updates

**Trigger**: `custom_cf_user_joined` or `custom_cf_user_left` WS events.

**Flow**:
- Redux updates `callsByChannel[channelId].participants` with the new `participants` array from the event
- All components subscribed to this channel's call state re-render automatically:
  - ChannelHeaderButton: re-evaluates state (participant may now be "In Call")
  - ToastBar: re-evaluates visibility (participant may now see FloatingWidget instead)
  - FloatingWidget: updates participant count and avatars
  - Custom post card (Unit 4): updates participant avatars

---

## BL-010: Call Ended Cleanup

**Trigger**: `custom_cf_call_ended` WS event.

**Flow**:
1. Remove `callsByChannel[channelId]` from Redux
2. If `myActiveCall.callId === call_id`: clear `myActiveCall`
3. If `incomingCall.callId === call_id`: clear `incomingCall`
4. FloatingWidget hides (myActiveCall cleared)
5. ToastBar hides (no active call in channel)
6. ChannelHeaderButton reverts to "Start Call"
