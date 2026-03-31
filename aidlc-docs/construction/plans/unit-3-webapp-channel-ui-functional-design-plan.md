# Unit 3: Webapp - Channel UI — Functional Design Plan

## Unit Summary
**Purpose**: All Mattermost-side UI components that react to call state, plus Redux state management.

**Primary Stories**: US-003, US-004, US-006, US-008, US-011, US-012, US-023, US-024
**Supporting Stories**: US-005, US-009, US-010, US-013, US-016, US-020, US-022

---

## Execution Checklist

- [x] Step 1: Analyze unit context and prior artifacts
- [x] Step 2: Answer clarifying questions (user input required)
- [x] Step 3: Generate domain entities document
- [x] Step 4: Generate business logic model document
- [x] Step 5: Generate business rules document
- [x] Step 6: Generate frontend components document
- [x] Step 7: Present completion message and await approval

---

## Clarifying Questions

Please fill in the `[Answer]:` tags below and return the file (or paste your answers).

---

### Q1: Redux State Shape — `incomingCall`

The `incomingCall` state is used for DM/GM ringing (US-024). The server emits `custom_com.kondo97.mattermost-plugin-rtk_call_started` to the entire channel.

How should the incoming call notification be triggered?

A) Always show the IncomingCallNotification component when `custom_com.kondo97.mattermost-plugin-rtk_call_started` arrives and the current user is NOT the creator (regardless of channel type)
B) Only show for DM and GM channels (channel type `D` or `G`); suppress for public/private team channels
C) Show for all channels, but only when the user is not already viewing that channel
X) Other (describe after [Answer]:)

[Answer]:

---

### Q2: Channel Header Button — Credential Check

US-003 says the button should be visible only when the plugin is configured (Cloudflare credentials set). The `/api/v1/config/status` endpoint returns `{"enabled": bool}`.

How should the component fetch and cache this status?

A) Fetch on plugin initialization (`initialize()`) and store in Redux; re-fetch on WebSocket reconnect
B) Fetch lazily when the ChannelHeaderButton first renders; cache in Redux; no refresh until page reload
C) Fetch on every channel switch (do not cache)
X) Other (describe after [Answer]:)

[Answer]:

---

### Q3: Channel Header Button — 4 States

The unit definition says the button has 4 states. Please confirm the mapping:

Proposed 4 states:
1. **Hidden** — plugin not enabled (`enabled: false` from config/status)
2. **Start Call** — no active call in channel; user not in any call
3. **Join Call** — active call in channel; user not a participant
4. **In Call** — user is already a participant of this channel's call

Is this mapping correct?

A) Yes — this is the correct 4-state mapping
B) Partially correct — please describe what's different after [Answer]:
X) Other (describe after [Answer]:)

[Answer]:

---

### Q4: Floating Widget — Persistence and Navigation

The floating widget (US-006) should persist while browsing other channels. It must survive channel switches.

Where should the FloatingWidget be registered?

A) `registerGlobalComponent` — renders at root, survives channel switches and product switches
B) `registerRootComponent` — renders at the channel root level
C) `registerCallButtonAction` with a separate overlay component
X) Other (describe after [Answer]:)

[Answer]:

---

### Q5: Floating Widget — Content and Actions

What should the floating widget display and allow the user to do?

A) Show: channel name + participant count + call duration timer. Actions: "Return to Call" (opens call tab) + "Leave Call" (calls POST /calls/{id}/leave)
B) Show: channel name + participant count. Actions: "Return to Call" only (no leave from widget)
C) Show: participant avatars (up to 3) + duration. Actions: "Return to Call" + "Leave Call"
X) Other (describe after [Answer]:)

[Answer]:

---

### Q6: Toast Bar — Trigger and Content

The channel call toast bar (US-008) appears for non-participants when a call is active in the channel they are currently viewing.

A) Show toast bar when: active call in current channel AND current user is NOT a participant. Hide when: user joins, or call ends, or user navigates away.
B) Show toast bar on `custom_com.kondo97.mattermost-plugin-rtk_call_started` event only (one-time notification per call, user can dismiss permanently)
C) Show toast bar when active call exists in channel, always visible to non-participants (persistent, not dismissable)
X) Other (describe after [Answer]:)

[Answer]:

---

### Q7: Switch Call Modal — Trigger Condition

The Switch Call Modal (US-023) appears when the user tries to join a different call while already in one.

Where is `myActiveCall` checked?

A) Inside the ChannelHeaderButton click handler — if user has an active call in a different channel, show SwitchCallModal before calling `POST /calls/{id}/token`
B) Inside the ToastBar "Join" action
C) Both A and B independently
X) Other (describe after [Answer]:)

[Answer]:

---

### Q8: Incoming Call Notification — Auto-Dismiss

US-024 says the IncomingCallNotification auto-dismisses after 30s.

What triggers the dismiss?

A) A `setTimeout` in the component that dispatches a Redux action to clear `incomingCall` after 30s; also auto-dismisses if the user joins the call or `custom_com.kondo97.mattermost-plugin-rtk_call_ended` arrives
B) Call the server `POST /calls/{id}/dismiss` after 30s, which triggers `custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed` WS event, which clears Redux state
C) Option A for auto-dismiss timer, option B only when user explicitly clicks "Ignore" button
X) Other (describe after [Answer]:)

[Answer]:

---

### Q9: WebSocket Event Payload — `custom_com.kondo97.mattermost-plugin-rtk_call_started`

The server emits this event to the channel. The Redux handler needs to update `callsByChannel` and potentially set `incomingCall`.

Does the `custom_com.kondo97.mattermost-plugin-rtk_call_started` payload include the full call object, or just a subset? Confirm from Unit 1 domain entities:

```json
{
  "call_id": "string",
  "channel_id": "string",
  "creator_id": "string",
  "participants": ["string"],
  "start_at": 1234567890000,
  "post_id": "string"
}
```

A) Yes, confirmed — the Redux slice `ActiveCall` entity should mirror this shape
B) The Redux slice needs additional fields — describe after [Answer]:
X) Other (describe after [Answer]:)

[Answer]:

---

### Q10: `myActiveCall` — How It Is Determined

`myActiveCall` tracks the call the current user is actively participating in. How is this set?

A) Set when `custom_com.kondo97.mattermost-plugin-rtk_user_joined` arrives with `user_id == currentUser.id`; cleared on `custom_com.kondo97.mattermost-plugin-rtk_user_left` with `user_id == currentUser.id` or `custom_com.kondo97.mattermost-plugin-rtk_call_ended`
B) Set when `custom_com.kondo97.mattermost-plugin-rtk_call_started` arrives and `creator_id == currentUser.id` (or user presses Join); derived by scanning `callsByChannel` for any call where currentUser is in `participants`
C) Both A and B: set on `custom_com.kondo97.mattermost-plugin-rtk_call_started` (if creator) and `custom_com.kondo97.mattermost-plugin-rtk_user_joined` (if joining); cleared on `custom_com.kondo97.mattermost-plugin-rtk_user_left` / `custom_com.kondo97.mattermost-plugin-rtk_call_ended`
X) Other (describe after [Answer]:)

[Answer]:

---

### Q11: Mattermost Calls UX Baseline

The project uses Mattermost Calls plugin as the UX baseline. For the channel header button, should we follow Calls exactly (icon style, button shape, color scheme) or is this a fresh design?

A) Follow Mattermost Calls UX baseline — use similar green/active-call color scheme and icon style
B) Fresh design — no specific constraint on colors/icons
X) Other (describe after [Answer]:)

[Answer]:

---

### Q12: `registerCallButtonAction` vs `registerChannelHeaderButtonAction`

The `PluginRegistry` has both `registerCallButtonAction` (Mattermost >= 6.5) and `registerChannelHeaderButtonAction`. Which should we use for the 4-state call button?

A) `registerCallButtonAction` — specifically designed for call plugins, shows in call button area; supports `button` and `dropdownButton` React elements
B) `registerChannelHeaderButtonAction` — more control over rendering; always renders as icon + click handler
C) Custom component via `registerRootComponent` (fully custom placement)
X) Other (describe after [Answer]:)

[Answer]:

