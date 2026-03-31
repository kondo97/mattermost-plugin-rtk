# User Stories

## Mattermost Calls Plugin Comparison

The following table maps key features of `mattermost-plugin-rtk` to the equivalent behavior in the official Mattermost Calls plugin. This plugin intentionally aligns with Mattermost Calls UX patterns unless noted as a deliberate divergence.

| Feature | Mattermost Calls Plugin | This Plugin (RTK) | Status |
|---|---|---|---|
| Channel header call button | Yes — Start call / Join call / In call / Connecting | Same button states | Aligned |
| Call button visibility | Shown when plugin is enabled | Shown only when RTK credentials are configured | Diverges (conditional) |
| Custom post type | Yes — active/ended states, participant avatars | Yes — `custom_cf_call`, same active/ended states | Aligned |
| Join button in post | Yes — disabled when already in that call | Yes — disabled when already in that call | Aligned |
| Channel call toast bar | Yes — shows call duration and participant avatars, dismissable | Yes — same | Aligned |
| Switch Call Modal | Yes — prompt when joining different call while in one | Yes — same modal with "Leave and join new call" | Aligned |
| Sound cue on join | Yes | Yes | Aligned |
| DM/GM incoming call ringing | Yes — 30-second ring with Ignore/Join | Yes — same 30-second ring with Ignore/Join | Aligned |
| Error surfaces | Modals (never inline channel messages) | Same — all errors surfaced via modals | Aligned |
| In-call UI location | Floating overlay within Mattermost window | Floating widget within Mattermost; full UI opens in new browser tab on demand | Aligned |
| In-call floating widget | Yes — visible while browsing channels | Yes — minimal widget with option to open full call UI in new tab | Aligned |
| WebRTC / media infrastructure | Mattermost SFU | Cloudflare RealtimeKit | Different backend |
| Admin credentials | Mattermost-specific (TURN/ICE servers) | Cloudflare RTK (org ID + API key) | Different backend |
| Feature flags | Various (recording, screen share, etc.) | 10 RTK-specific flags via admin UI + env vars | Extended |
| Mobile push delivery | Mattermost push notification infrastructure | WebSocket events (push notifications removed) | Diverges |
| Web Worker CSP compliance | Handled by Mattermost server | Plugin serves `/plugins/{id}/worker.js` | Plugin responsibility |

---

## Story Format

```
### US-XXX: [Title]
**Persona**: [Persona]
**As a** [persona], **I want to** [action], **so that** [benefit].

**Acceptance Criteria:**
- [ ] ...
```

---

## Journey 1: Admin Setup

### US-001: Configure Cloudflare Credentials
**Persona**: Mattermost Admin
**As a** Mattermost Admin, **I want to** enter my Cloudflare RTK Organization ID and API Key in the System Console, **so that** the plugin can authenticate with the Cloudflare RTK API and enable calls for workspace members.

**Acceptance Criteria:**
- [ ] The System Console shows an "Organization ID" field and an "API Key" field under the plugin settings.
- [ ] The API Key field is masked (shown as `••••••••`) after saving; the Organization ID field is shown in plain text.
- [ ] Credentials are not validated at save time; if invalid, an error modal is shown when a call is first attempted.
- [ ] Saving credentials makes the call button visible in channel headers for all users (visibility is based on credentials being present, not validated).
- [ ] If a field is overridden by an environment variable (`RTK_ORG_ID` / `RTK_API_KEY`), the field is displayed as read-only with a label indicating it is set via environment variable; the API Key value is masked, the Organization ID value is shown in plain text.

---

### US-002: Toggle Feature Flags
**Persona**: Mattermost Admin
**As a** Mattermost Admin, **I want to** enable or disable individual call features (polls, screen share, recording, AI transcription, waiting room, video, chat, plugins, participants, raise hand), **so that** I can tailor the call experience to my organization's needs.

**Acceptance Criteria:**
- [ ] Each of the 10 feature flags has a toggle in the System Console; all flags default to enabled (ON) on fresh install.
- [ ] Feature flag values are returned in create/join call API responses and applied in the call UI.
- [ ] Changing a flag does not affect calls already in progress; the new value applies to subsequently created or joined calls.
- [ ] Each flag can be overridden by an environment variable following the pattern `RTK_<FLAG_NAME>_ENABLED` (e.g., `RTK_RECORDING_ENABLED`, `RTK_SCREEN_SHARE_ENABLED`, `RTK_POLLS_ENABLED`, `RTK_TRANSCRIPTION_ENABLED`, `RTK_WAITING_ROOM_ENABLED`, `RTK_VIDEO_ENABLED`, `RTK_CHAT_ENABLED`, `RTK_PLUGINS_ENABLED`, `RTK_PARTICIPANTS_ENABLED`, `RTK_RAISE_HAND_ENABLED`); when overridden, the toggle is read-only.

---

## Journey 2: Initiating a Call

### US-003: See Call Button in Channel Header
**Persona**: Channel Member (Web)
**As a** channel member, **I want to** see a call button in the channel header, **so that** I know I can start or join a call from any channel.

**Acceptance Criteria:**
- [ ] The call button appears in the channel header for public channels, private channels, DMs, and group DMs.
- [ ] The call button is not rendered if RTK credentials are not configured.
- [ ] The button label and state reflect the current call state (see US-004).

---

### US-004: Call Button States Reflect Call Status
**Persona**: Channel Member (Web)
**As a** channel member, **I want** the call button to show the current call state, **so that** I know at a glance whether a call is active and what action I can take.

**Acceptance Criteria:**
- [ ] No active call: button shows "Start call" (enabled).
- [ ] Active call, user not in it: button shows "Join call" (enabled).
- [ ] User already in this call: button shows "In call" (disabled).
- [ ] Call is being created: button shows "Starting call..." with a spinner (disabled).
- [ ] User is in a call in a *different* channel: button shows a distinct state (e.g., "Join call" remains visible but clicking triggers the Switch Call Modal — US-023).
- [ ] If the WebSocket connection is lost, the button retains its last known state; the server enforces correctness (e.g., rejecting invalid join attempts) when the user takes an action.

---

### US-005: Start a Call
**Persona**: Channel Member (Web)
**As a** channel member, **I want to** click the call button to start a new call, **so that** I can initiate a video/audio call with other channel members.

**Acceptance Criteria:**
- [ ] Clicking "Start call" sends a request to `POST /api/v1/calls` and creates an RTK meeting session.
- [ ] The creator is added as a participant with the `group_call_host` preset.
- [ ] An auth token (JWT) is returned and used to open the standalone call page in a new browser tab; the tab opens automatically and receives focus.
- [ ] A `custom_cf_call` post is posted to the channel announcing the call; the post includes the creator's display name and avatar.
- [ ] A `custom_cf_call_started` WebSocket event is emitted to all connected clients.
- [ ] A sound cue plays on the Mattermost side when the `POST /api/v1/calls` response is received (not on call page load).
- [ ] If the RTK API returns an error, a modal is shown with the error message — no inline channel message is posted.

*Mobile variant*: On mobile, the WebSocket event (`custom_cf_call_started`) serves as the call announcement; the creator does not receive their own push notification.

---

### US-006: Floating Widget Appears When Call Starts or Joined
**Persona**: Channel Member (Web)
**As a** channel member who has started or joined a call, **I want** a floating widget to appear within Mattermost, **so that** I can see the call status and continue browsing channels without losing track of the call.

**Acceptance Criteria:**
- [ ] When a call is started or joined, a floating widget appears within the Mattermost window (same as Mattermost Calls plugin behavior).
- [ ] The widget shows minimal call information: participant count/avatars, call duration, and a mute/unmute control.
- [ ] The widget persists while the user browses other channels.
- [ ] The widget has a "Open in new tab" button that opens the full call UI at `/plugins/{id}/call?token=<jwt>` in a new browser tab (not a popup window).
- [ ] The full call page initializes the Cloudflare RealtimeKit React SDK using the `token` query parameter.
- [ ] The full call page is accessible without a Mattermost session cookie (token-based auth via the URL parameter).
- [ ] The JWT token is short-lived (valid for the duration of the call session, e.g., up to 1 hour).
- [ ] The browser tab title of the full call page displays the channel name (e.g., "Call in #general").
- [ ] If the user closes the full call tab, the floating widget remains and the user is still considered in the call.

---

## Journey 3: Receiving a Call Notification (Web)

### US-007: Custom Post Appears When Call Starts
**Persona**: Channel Member (Web)
**As a** channel member, **I want to** see a call post appear in the channel when a call starts, **so that** I know a call is in progress and can join it.

**Acceptance Criteria:**
- [ ] A `custom_cf_call` post is rendered in the channel immediately when a call starts.
- [ ] The post shows a green indicator and "Call started" label.
- [ ] The post shows the call start time.
- [ ] The post shows participant avatars (up to 3 shown, with overflow count).
- [ ] The post shows a "Join call" button for users who are not in the call.

---

### US-008: Channel Call Toast Bar Appears
**Persona**: Channel Member (Web)
**As a** channel member who is **not** in the call, **I want to** see a toast bar above the message input when a call is active in the channel, **so that** I am reminded of the ongoing call and can join it.

**Note**: Members who are already in the call see the floating widget (US-006) instead of this toast bar.

**Acceptance Criteria:**
- [ ] A toast bar appears above the message input area when the user is viewing the channel where a call is active; it is not shown when the user is browsing a different channel.
- [ ] The toast bar is shown to all channel members regardless of whether they are in the call.
- [ ] The toast bar shows the call start time and participant avatars.
- [ ] The toast bar includes a "Join" button for members who are not in the call.
- [ ] The toast bar has a dismiss button; dismissing it hides it for the local user only (dismiss state is not persisted — the toast bar reappears on page reload if the call is still active).
- [ ] The toast bar disappears for all clients when the call ends (via `custom_cf_call_ended` WebSocket event).
- [ ] Members who join the call transition from seeing the toast bar to seeing the floating widget.

---

## Journey 4: Joining a Call

### US-009: Join a Call from the Custom Post
**Persona**: Channel Member (Web)
**As a** channel member, **I want to** click "Join call" in the call post, **so that** I can enter an ongoing call.

**Acceptance Criteria:**
- [ ] Clicking "Join call" sends a request to `POST /api/v1/calls/{callId}/token`.
- [ ] The joining user is added with the `group_call_participant` preset.
- [ ] An auth token is returned and used to open the standalone call page in a new browser tab.
- [ ] A `custom_cf_user_joined` WebSocket event is emitted to all connected clients.
- [ ] A sound cue plays when the call page loads for the joining user.
- [ ] If the RTK API returns an error, a modal is shown — no inline channel message.

*Mobile variant*: On mobile, the user taps "Join" in the incoming call notification or from the app. The API response includes all feature flag values for the native call UI.

---

### US-010: Custom Post Disables "Join" Button When Already in Call
**Persona**: Channel Member (Web)
**As a** channel member who is already in the call, **I want** the post card to show "Join call" as disabled, **so that** I am not confused about my call state and understand I am already participating.

**Acceptance Criteria:**
- [ ] For users currently in the call, the post card shows the "Join call" button in a disabled state.
- [ ] No "Leave" button is shown in the post card; leaving the call is done from within the call UI (US-013).

---

## Journey 5: In-Call State in Mattermost

### US-011: Channel Header Shows "In Call" While User Is in a Call
**Persona**: Channel Member (Web)
**As a** channel member who is in an active call, **I want** the channel header button to show "In call" (disabled), **so that** my call state is visible while I browse other channels.

**Acceptance Criteria:**
- [ ] The call button in the active call's channel header shows "In call" (disabled) for the local user.
- [ ] The state updates in real-time via `custom_cf_user_joined` / `custom_cf_user_left` WebSocket events.

---

### US-012: Custom Post Updates in Real-Time via WebSocket
**Persona**: Channel Member (Web)
**As a** channel member, **I want** the call post card to update participant avatars and state in real-time, **so that** I can see who is in the call without refreshing the page.

**Acceptance Criteria:**
- [ ] When a user joins, the `custom_cf_user_joined` event triggers an update to the post's participant avatar list.
- [ ] When a user leaves, the `custom_cf_user_left` event triggers removal of their avatar from the post.
- [ ] When the call ends, the `custom_cf_call_ended` event transitions the post to the "ended" state.

---

## Journey 6: Leaving a Call

### US-013: Leave a Call by Closing the Tab
**Persona**: Channel Member (Web)
**As a** channel member in a call, **I want to** leave by closing the call browser tab, **so that** I can exit the call without needing a dedicated "Leave" button in Mattermost.

**Acceptance Criteria:**
- [ ] When the user closes the call tab, the plugin detects the departure and removes the user from the `participants` list in KVStore.
- [ ] A `custom_cf_user_left` WebSocket event is emitted to all connected clients.
- [ ] The custom post and toast bar participant avatars are updated accordingly.

---

## Journey 7: Ending a Call

### US-015: Host Ends the Call for All Participants
**Persona**: Channel Member (Web) — Host
**As a** call host, **I want to** end the call for all participants even if others are still present, **so that** I can close the session intentionally without waiting for everyone to leave individually.

**Acceptance Criteria:**
- [ ] The host has an "End call" action in the call UI (in addition to the standard "Leave" action).
- [ ] Clicking "End call" immediately ends the call for all participants regardless of how many are present.
- [ ] Clicking "End call" sends a request to the server which sets `end_at` in KVStore.
- [ ] A `custom_cf_call_ended` WebSocket event is emitted with `call_id`, `channel_id`, `end_at`, and `duration_ms`.
- [ ] The custom post switches to the "ended" state for all clients.
- [ ] The toast bar is dismissed for all clients.
- [ ] Non-host participants do not have access to the "End call" action.

---

### US-016: Post Card Switches to "Ended" State When Call Ends
**Persona**: Channel Member (Web)
**As a** channel member, **I want** the call post card to show the call duration and end time when the call ends, **so that** I have a record of the call without any actionable buttons.

**Acceptance Criteria:**
- [ ] The post card shows a gray indicator and "Call ended" label.
- [ ] The post card shows the call end time and duration (e.g., "Lasted 12 minutes").
- [ ] No "Join call" or "Leave" buttons are shown in the ended state.
- [ ] The transition happens in real-time via the `custom_cf_call_ended` WebSocket event.

---

## Journey 8: Mobile — Incoming Call

### US-018: Receive Incoming Call Push Notification

> **Updated 2026-03-31**: This story's push notification approach has been REMOVED. Mobile clients receive call notifications via WebSocket events instead.

**Persona**: Mobile User
**As a** mobile user, **I want to** receive a push notification when someone starts a call in a channel I belong to, **so that** I can decide whether to join the call natively.

**Acceptance Criteria:**
- [ ] When a call starts, the plugin creates a `PushNotification` with `type: "message"` and `sub_type: "calls"` and dispatches it via the Mattermost push notification infrastructure to all channel members who are not the caller.
- [ ] The notification payload includes: `channel_id`, `team_id`, `sender_id`, `sender_name`, `channel_name`, `root_id` (associated post ID).
- [ ] The plugin does NOT handle platform routing or device token selection; the Mattermost backend automatically routes to iOS VoIP (APNs PushKit) or Android FCM based on the registered session device tokens.
- [ ] iOS devices with a registered VoIP token receive the notification via APNs PushKit; Android devices receive it via FCM. Both paths are handled transparently by the Mattermost push proxy.
- [ ] When a call ends, the plugin sends a `type: "clear"` / `sub_type: "calls_ended"` notification to dismiss the incoming call UI on all devices that have not yet responded.

---

### US-019: Join a Call from Push Notification
**Persona**: Mobile User
**As a** mobile user who received an incoming call notification, **I want to** tap "Join" to enter the call natively, **so that** I can participate in the call without switching to a browser.

**Acceptance Criteria:**
- [ ] Tapping "Join" in the push notification calls `POST /api/v1/calls/{callId}/token`.
- [ ] The response includes an auth token and all 10 feature flag values.
- [ ] The native call UI initializes using the token and feature flags.
- [ ] The mobile user is added with the `group_call_participant` preset.

---

### US-020: Dismiss Incoming Call Notification
**Persona**: Mobile User
**As a** mobile user who received an incoming call notification, **I want to** dismiss it, **so that** the ringing stops on all my devices and other clients.

**Acceptance Criteria:**
- [ ] The mobile app calls `POST /api/v1/calls/{callId}/dismiss`.
- [ ] The server emits a `custom_cf_notification_dismissed` WebSocket event with `call_id` and `user_id`.
- [ ] All other clients receiving the event stop showing the ringing notification for this user.

---

## Journey 9: Infrastructure

### US-021: Web Worker Script is Served by the Plugin
**Persona**: Channel Member (Web)
**As a** channel member using the RTK call UI, **I want** the plugin to serve a Web Worker script at a known URL, **so that** the RTK SDK's timer worker can load without violating the browser Content Security Policy.

**Acceptance Criteria:**
- [ ] The plugin serves a Web Worker script at `GET /plugins/{id}/worker.js`.
- [ ] The endpoint is unauthenticated (no `Mattermost-User-ID` required).
- [ ] The response satisfies the browser `worker-src 'self'` CSP restriction.

---

## Edge Cases (Separate Stories)

### US-022: Active Call Blocking — Cannot Start a Second Call in a Channel
**Persona**: Channel Member (Web)
**As a** channel member, **I want to** be prevented from starting a second call in a channel that already has an active call, **so that** there is never more than one active call session per channel.

**Acceptance Criteria:**
- [ ] If `call:channel:{channelID}` exists in KVStore with `end_at == 0`, starting a new call returns an error.
- [ ] The user is shown a modal informing them that a call is already in progress in this channel.
- [ ] No new RTK meeting session is created.
- [ ] No new `custom_cf_call` post is created.

---

### US-023: Switch Call Modal — Joining a Different Call While Already in One
**Persona**: Channel Member (Web)
**As a** channel member already in a call, **I want to** see a confirmation modal when I try to join a different call, **so that** I don't accidentally abandon my current call without realizing it.

**Acceptance Criteria:**
- [ ] When the user is in call A and attempts to join call B, a modal appears: "You're already in a call. Do you want to leave and join this call?"
- [ ] The modal has two actions: "Cancel" (dismisses modal, no action) and "Leave and join new call" (leaves call A, joins call B).
- [ ] If the user is already in the same call they are trying to join, no modal is shown and the "Join" button is disabled.

---

### US-024: DM/GM Ringing — Incoming Call Notification with Ignore and Join
**Persona**: Channel Member (Web)
**As a** member of a DM or group DM, **I want to** receive an in-app ringing notification with "Ignore" and "Join" options when a call starts, **so that** I can respond to the call in real-time.

**Acceptance Criteria:**
- [ ] When a call starts in a DM or group DM channel, an incoming call ringing notification is shown to all other channel members (not the caller).
- [ ] The notification shows "Ignore" and "Join" options.
- [ ] Clicking "Join" initiates the join flow (US-009).
- [ ] Clicking "Ignore" dismisses the notification for the local user and emits `custom_cf_notification_dismissed`.
- [ ] The ringing notification automatically dismisses after 30 seconds.

*Mobile variant*: On mobile, this ringing is delivered as a push notification (US-018). Dismissing it from mobile emits the same `custom_cf_notification_dismissed` event (US-020), causing other clients to stop ringing.

---

### US-025: Last-Participant Auto-End — Call Ends Automatically When All Leave
**Persona**: Channel Member (Web)
**As a** channel member, **I want** a call to end automatically when the last participant leaves, **so that** a ghost call session is never left open with no participants.

**Acceptance Criteria:**
- [ ] When the last participant leaves (participants list becomes empty), the server automatically sets `end_at` to the current timestamp.
- [ ] The `custom_cf_call_ended` WebSocket event is emitted.
- [ ] The call post switches to the "ended" state.
- [ ] No further join requests are accepted after auto-end.

---
