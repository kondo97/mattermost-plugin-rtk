# Unit 3: Webapp - Channel UI — Business Rules

## BR-001: Channel Header Button Hidden When Plugin Disabled

The ChannelHeaderButton MUST NOT render if `pluginEnabled === false`. No placeholder or disabled state is shown — the element is absent from the DOM entirely.

**Enforcement**: Selector check on `pluginEnabled` before rendering; `registerCallButtonAction` callback returns null if disabled.

---

## BR-002: Loading State Is Local Component State

The "Starting call..." spinner state (`loading: boolean`) MUST be managed as local component state in ChannelHeaderButton — it MUST NOT be stored in Redux.

**Rationale**: This is a transient UI state specific to the initiating client. Other clients do not need it; they receive the authoritative `custom_cf_call_started` WS event.

---

## BR-003: At Most One Active Call Per Channel

`callsByChannel` stores at most one active call per channel. When `custom_cf_call_started` arrives for a channel that already has an entry, the existing entry is replaced with the new call data.

**Note**: The server enforces this at the API layer (409 on duplicate). The client-side replacement is a defensive guard for out-of-order WS events.

---

## BR-004: IncomingCallNotification Only for DM/GM Channels

`incomingCall` MUST only be set when the channel type of the incoming call's `channelId` is `D` (DM) or `G` (Group Message). Public and private team channels do not trigger the ringing notification.

---

## BR-005: IncomingCallNotification Not Shown to Call Creator

`incomingCall` MUST NOT be set if `creator_id == currentUser.id`. The user who started the call does not receive their own ringing notification.

---

## BR-006: IncomingCallNotification Auto-Dismiss After 30 Seconds

If the user does not interact with the IncomingCallNotification within 30 seconds, it MUST be dismissed automatically by clearing `incomingCall` in Redux. This is a client-side-only operation; no API call is made on timeout expiry.

**Cleanup rule**: The `setTimeout` handle MUST be cancelled if `incomingCall` is cleared before the 30-second timer fires (e.g., user clicks "Ignore" or "Join", or the call ends).

---

## BR-007: "Ignore" Triggers Server Dismiss

When the user clicks "Ignore" on the IncomingCallNotification, a `POST /calls/{id}/dismiss` request MUST be sent to the server. The `incomingCall` state is cleared by the resulting `custom_cf_notification_dismissed` WS event — not optimistically by the client.

---

## BR-008: Toast Bar Dismissed State Is Not Persisted

When a user dismisses the ToastBar, the dismissed state (`dismissed: boolean`) is local to the component instance and is NOT persisted to any store or preference. If the user reloads the page while the call is still active, the ToastBar reappears.

---

## BR-009: Toast Bar Clears on Call End

When `custom_cf_call_ended` is received, the corresponding entry is removed from `callsByChannel`. This causes the ToastBar visibility condition to evaluate to false for all users still viewing that channel, regardless of their local `dismissed` state.

---

## BR-010: FloatingWidget Has No Mute Control

The FloatingWidget MUST NOT include a mute/unmute control. Mute operations require direct access to the RTK SDK media session, which runs only in the call tab (Unit 4). The widget is rendered in the main Mattermost window and cannot communicate with the SDK.

---

## BR-011: "Open in New Tab" Reuses Saved Token

When the user clicks "Open in new tab" in the FloatingWidget, the token stored in `myActiveCall.token` (obtained at join time) MUST be reused. No additional API call is made for this purpose.

---

## BR-012: SwitchCallModal Required When Joining Across Calls

If `myActiveCall !== null` AND the target call's `channelId !== myActiveCall.channelId`, the user MUST be shown the SwitchCallModal before the join proceeds. The join MUST NOT execute without explicit user confirmation.

**Exception**: If the user is already a participant of the target call (`currentUser.id` in `participants`), no modal is shown and the join button is disabled.

---

## BR-013: WS Events Update Full Participant List

`custom_cf_user_joined` and `custom_cf_user_left` events include the full `participants` array from the server. The client MUST replace (not diff) the `participants` field in `callsByChannel[channelId]` with the server-provided array. This prevents client-side participant count drift.

---

## BR-014: Config Status Re-Fetched on WS Reconnect

When the WebSocket connection is re-established after a disconnect, `GET /config/status` MUST be re-fetched and Redux updated. This ensures the ChannelHeaderButton visibility reflects any credential changes made during the disconnection window.

---

## BR-015: Error Display via Modal Only

All API errors encountered during call actions (start, join, dismiss) MUST be surfaced via an error modal. No inline channel messages are posted and no toast/banner errors appear in the channel view. This aligns with the Mattermost Calls plugin UX baseline.

---

## BR-016: Sound Cue Fires on API Response, Not on Call Page Load

The sound cue for a call start MUST play when the `POST /api/v1/calls` response is received in the Mattermost window — not when the call page tab loads. This is the behavior defined in US-005 and aligns with the Mattermost Calls plugin.
