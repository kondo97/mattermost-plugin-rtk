# Unit 3: Webapp - Channel UI — Code Generation Plan

## Unit Context

**Purpose**: All Mattermost-side UI components for call state + Redux state management.
**Stories**: US-003, US-004, US-005(partial), US-006, US-008, US-009(partial), US-010(partial), US-011, US-012, US-016(partial), US-020(partial), US-022(partial), US-023, US-024
**Tech**: React 18 + TypeScript + plain Redux 5 + react-redux 9 + Enzyme (tests)
**i18n**: English (`en.json`) + Japanese (`ja.json`), all strings via `useIntl`/`FormattedMessage`

## Code Location

- **Application code**: `webapp/src/` (never aidlc-docs/)
- **i18n**: `webapp/i18n/`
- **Documentation**: `aidlc-docs/construction/unit-3-webapp-channel-ui/code/`

## Dependencies (already implemented)

- Unit 1: `CallSession` domain model, KVStore, call lifecycle methods
- Unit 2: All HTTP endpoints + WS event emission

## File Plan

| Step | File | Action |
|---|---|---|
| 1 | `webapp/src/redux/calls_slice.ts` | Create — types + action creators + reducer |
| 2 | `webapp/src/redux/selectors.ts` | Create — 5 typed selectors |
| 3 | `webapp/src/redux/websocket_handlers.ts` | Create — 5 WS event handlers |
| 4 | `webapp/src/client.ts` | Create — pluginFetch helper |
| 5 | `webapp/i18n/en.json` | Modify — add plugin.rtk.* keys |
| 6 | `webapp/i18n/ja.json` | Create — Japanese translations |
| 7 | `webapp/src/components/switch_call_modal/index.tsx` | Create |
| 8 | `webapp/src/components/channel_header_button/index.tsx` | Create |
| 9 | `webapp/src/components/toast_bar/index.tsx` | Create |
| 10 | `webapp/src/components/floating_widget/index.tsx` | Create |
| 11 | `webapp/src/components/incoming_call_notification/index.tsx` | Create |
| 12 | `webapp/src/index.tsx` | Modify — register reducer, WS handlers, components, translations |
| 13 | `webapp/src/redux/calls_slice.test.ts` | Create — reducer + action creator tests |
| 14 | `webapp/src/redux/websocket_handlers.test.ts` | Create — WS handler tests |
| 15 | `webapp/src/redux/selectors.test.ts` | Create — selector tests |
| 16 | `webapp/src/components/channel_header_button/index.test.tsx` | Create — Enzyme state tests |
| 17 | `aidlc-docs/construction/unit-3-webapp-channel-ui/code/code-summary.md` | Create — documentation |

---

## Execution Checklist

### Part A: State Layer

- [x] Step 1: Create `webapp/src/redux/calls_slice.ts`
  - Type definitions: `ActiveCall`, `MyActiveCall`, `IncomingCall`, `CallsPluginState`, `CallsAction`
  - 7 action type constants + action creators (Pattern U3-1)
  - `callsReducer` plain switch function
  - `initialState`
  - Story: US-004, US-011, US-012

- [x] Step 2: Create `webapp/src/redux/selectors.ts`
  - Plugin state path constant: `'plugins-com.mattermost.plugin-rtk'`
  - `selectPluginEnabled`, `selectCallByChannel(channelId)`, `selectMyActiveCall`, `selectIncomingCall`, `selectIsCurrentUserParticipant(channelId, userId)`
  - Curried selectors for channel-scoped state (Pattern U3-3)
  - Story: US-003, US-004, US-008, US-011

- [x] Step 3: Create `webapp/src/redux/websocket_handlers.ts`
  - TypeScript type guards for each WS payload (Pattern U3-2)
  - `handleCallStarted` — upsertCall + setIncomingCall (DM/GM only, non-creator)
  - `handleUserJoined` — upsertCall with updated participants
  - `handleUserLeft` — upsertCall with updated participants + clearMyActiveCall if self
  - `handleCallEnded` — removeCall + clearMyActiveCall/clearIncomingCall if matching
  - `handleNotifDismissed` — clearIncomingCall if self
  - Story: US-004, US-008, US-012, US-016, US-020, US-024

- [x] Step 4: Create `webapp/src/client.ts`
  - `pluginFetch<T>(path, options)` returning `{ data: T } | { error: string }` (Pattern U3-6)
  - No JWT logging (SEC-U3-01)
  - Generic error messages to user (SEC-U3-03)

### Part B: i18n

- [x] Step 5: Modify `webapp/i18n/en.json`
  - Add all `plugin.rtk.*` message keys (English strings)
  - Keys: channel_header, toast_bar, floating_widget, switch_call_modal, incoming_call, error.*

- [x] Step 6: Create `webapp/i18n/ja.json`
  - All `plugin.rtk.*` keys with Japanese translations (identical key set to en.json)

### Part C: UI Components

- [x] Step 7: Create `webapp/src/components/switch_call_modal/index.tsx`
  - Props: `visible`, `targetCallId`, `targetChannelId`, `onConfirm`, `onCancel`
  - Two buttons: Cancel + "Leave and join new call"
  - All strings via `useIntl` (Pattern U3-5)
  - `data-testid` on all interactive elements

- [x] Step 8: Create `webapp/src/components/channel_header_button/index.tsx`
  - 5 visual states: Hidden / Starting / In Call / Join Call / Start Call
  - Local state: `loading: boolean` + `showSwitchModal: boolean`
  - `pluginEnabled` check → render null if false (BR-001)
  - Click handler: start call or join call (with SwitchCallModal check)
  - `SwitchCallModal` rendered inline
  - Sound cue via `registerDesktopNotificationHook` on successful start
  - `data-testid` attributes
  - Story: US-003, US-004, US-005, US-022, US-023

- [x] Step 9: Create `webapp/src/components/toast_bar/index.tsx`
  - Reads `currentChannelId` from Mattermost store (`getCurrentChannelId`)
  - Visibility: active call in current channel AND user not participant AND not dismissed
  - Local state: `dismissed: boolean` + `showSwitchModal: boolean`
  - Shows call start time, participant avatars (up to 3), overflow count
  - "Join" button + dismiss "×" button
  - `SwitchCallModal` rendered inline
  - `data-testid` attributes
  - Story: US-008, US-023

- [x] Step 10: Create `webapp/src/components/floating_widget/index.tsx`
  - Renders when `myActiveCall !== null`
  - Shows: channel name, participant count/avatars (up to 3), duration timer
  - Duration timer: `setInterval` every 1s with `useEffect` cleanup (Pattern U3-4)
  - "Open in new tab" → `pluginFetch POST /calls/{id}/token` → `window.open` (Pattern U3-7)
  - `noopener,noreferrer` on window.open
  - No mute control (BR-010)
  - `registerGlobalComponent` (survives channel switch)
  - `data-testid` attributes
  - Story: US-006, US-011

- [x] Step 11: Create `webapp/src/components/incoming_call_notification/index.tsx`
  - Renders when `incomingCall !== null`
  - 30s auto-dismiss via `setTimeout` + `useEffect` cleanup (Pattern U3-4)
  - "Ignore" → `POST /calls/{id}/dismiss` fire-and-forget; waits for WS event to clear state
  - "Join" → join flow (checks `myActiveCall` → shows SwitchCallModal if needed)
  - `registerGlobalComponent`
  - `data-testid` attributes
  - Story: US-024, US-020

### Part D: Plugin Entry Point

- [x] Step 12: Modify `webapp/src/index.tsx`
  - Import all components, slice, handlers, selectors, translations
  - `registry.registerReducer(callsReducer)`
  - Fetch initial config status → `store.dispatch(setPluginEnabled(...))`
  - Register 5 WS event handlers
  - `registry.registerReconnectHandler` → re-fetch config
  - `registry.registerCallButtonAction(ChannelHeaderButton, ChannelHeaderButton, noop)`
  - `registry.registerChannelToastComponent(ToastBar)`
  - `registry.registerGlobalComponent(FloatingWidget)`
  - `registry.registerGlobalComponent(IncomingCallNotification)`
  - `registry.registerTranslations(locale => locale === 'ja' ? ja : en)`
  - Story: US-003, US-004, US-006, US-008, US-024

### Part E: Tests

- [x] Step 13: Create `webapp/src/redux/calls_slice.test.ts`
  - Test all 7 reducer actions with initial and modified states
  - Test `initialState` defaults
  - Test `REMOVE_CALL` (spread/delete pattern)

- [x] Step 14: Create `webapp/src/redux/websocket_handlers.test.ts`
  - Mock Redux store (simple object with `dispatch` jest.fn())
  - Test each handler: valid payload → correct dispatch; invalid payload → no dispatch + console.error
  - Test `handleCallStarted` DM/GM/non-DM-GM channel type logic
  - Test `handleUserLeft` self vs other

- [x] Step 15: Create `webapp/src/redux/selectors.test.ts`
  - Test each selector with mock GlobalState
  - Test `selectIsCurrentUserParticipant`: user in list, not in list, empty list, call not found

- [x] Step 16: Create `webapp/src/components/channel_header_button/index.test.tsx`
  - Enzyme shallow renders for all 5 visual states
  - Test `pluginEnabled = false` → renders null
  - Test loading state → "Starting call..." label
  - Test "In Call" disabled state
  - Test "Join Call" and "Start Call" enabled states

### Part F: Documentation

- [x] Step 17: Create `aidlc-docs/construction/unit-3-webapp-channel-ui/code/code-summary.md`

---

## Story Traceability

| Story | Implemented In |
|---|---|
| US-003: See Call Button in Channel Header | Steps 8, 12 |
| US-004: Call Button States Reflect Call Status | Steps 1, 2, 3, 8, 12 |
| US-005: Start a Call (UI dispatch only) | Steps 4, 8 |
| US-006: Floating Widget | Steps 10, 12 |
| US-008: Channel Call Toast Bar | Steps 9, 12 |
| US-009: Join from Post (selectors support) | Steps 1, 2 |
| US-010: Disable Join if in Call (selector) | Step 2 |
| US-011: In Call state in channel header | Steps 2, 8 |
| US-012: Real-time WS updates | Steps 3, 1 |
| US-016: Post Card ended state (Redux) | Steps 1, 3 |
| US-020: Dismiss notification (WS handler) | Steps 3, 11 |
| US-022: Error modal on duplicate call | Step 8 |
| US-023: Switch Call Modal | Steps 7, 8, 9 |
| US-024: DM/GM Ringing | Steps 1, 3, 11, 12 |
