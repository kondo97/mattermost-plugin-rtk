# Unit 3: Webapp Channel UI — Code Summary

## Overview

This unit implements the Redux state layer, WebSocket event handlers, i18n translations, and all 5 UI components for the Mattermost plugin's channel-level call functionality.

## Files Created / Modified

### Redux State Layer

| File | Status | Description |
|------|--------|-------------|
| `webapp/src/redux/calls_slice.ts` | Created | Plain Redux reducer, action creators, and TypeScript types for call state |
| `webapp/src/redux/selectors.ts` | Created | Curried selectors for channel-scoped state access |
| `webapp/src/redux/websocket_handlers.ts` | Created | Type-guarded WS event handler factories for 5 event types |

### HTTP Client

| File | Status | Description |
|------|--------|-------------|
| `webapp/src/client.ts` | Created | `pluginFetch<T>` helper returning `{ data: T } \| { error: string }` |

### i18n Translations

| File | Status | Description |
|------|--------|-------------|
| `webapp/i18n/en.json` | Modified | 22 English message keys under `plugin.rtk.*` prefix |
| `webapp/i18n/ja.json` | Created | Full Japanese translations matching en.json key set |

### UI Components

| File | Status | Description |
|------|--------|-------------|
| `webapp/src/components/switch_call_modal/index.tsx` | Created | Shared confirmation modal for switching between calls |
| `webapp/src/components/channel_header_button/index.tsx` | Created | Channel header button with 5 visual states (hidden/starting/in-call/join/start) |
| `webapp/src/components/toast_bar/index.tsx` | Created | In-channel toast bar for active calls the user hasn't joined |
| `webapp/src/components/floating_widget/index.tsx` | Created | Global floating widget with duration timer and open-in-tab |
| `webapp/src/components/incoming_call_notification/index.tsx` | Created | DM/GM incoming call notification with 30s auto-dismiss |

### Plugin Entry Point

| File | Status | Description |
|------|--------|-------------|
| `webapp/src/index.tsx` | Modified | Full plugin registration: reducer, 5 WS handlers, reconnect handler, UI components, translations |

### Tests

| File | Status | Description |
|------|--------|-------------|
| `webapp/src/redux/calls_slice.test.ts` | Created | Reducer tests for all 7 action types + initialState + REMOVE_CALL spread/delete pattern |
| `webapp/src/redux/websocket_handlers.test.ts` | Created | WS handler tests with mock store; covers type guard rejection, DM/GM-only incoming call, self-join/leave detection |
| `webapp/src/redux/selectors.test.ts` | Created | Selector tests for all 5 selectors; covers channel scoping and participant membership |
| `webapp/src/components/channel_header_button/index.test.tsx` | Created | Enzyme shallow tests for 5 visual states of ChannelHeaderButton |

## Key Design Decisions

### TSD-U3-01: Plain Redux (no @reduxjs/toolkit)
Action type constants + action creator functions + switch reducer — matches Mattermost webapp conventions.

### TSD-U3-02: Mattermost i18n
`babel-plugin-formatjs` + `useIntl` hook. All message IDs use `plugin.rtk.*` prefix.

### Pattern U3-2: Type Guards (SEC-U3-02)
All WS event payloads are validated via TypeScript type guards before dispatch. Invalid payloads are silently dropped.

### Pattern U3-3: Curried Selectors
`selectCallByChannel(channelId)` and `selectIsCurrentUserParticipant(channelId, userId)` return per-render selector functions, preventing unnecessary re-renders.

### Pattern U3-4: Effect Cleanup
`FloatingWidget` clears `setInterval` on unmount/call-change. `IncomingCallNotification` clears `setTimeout` on unmount/incoming-change.

### Pattern U3-7: Fresh Token Before Open Tab
`FloatingWidget` and `IncomingCallNotification` always call `POST /calls/{id}/token` immediately before `window.open(...)`. Tokens are never logged (SEC-U3-01).

### BR-007: Notification Dismiss Flow
`handleIgnore` in `IncomingCallNotification` fires `POST /calls/{id}/dismiss` as fire-and-forget. Redux state is cleared only when the `custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed` WS event arrives (TOCTOU-safe).

### BR-013: DM/GM-Only Incoming Calls
`handleCallStarted` only dispatches `setIncomingCall` when `channelType === 'D' || channelType === 'G'` AND `creatorId !== currentUserId`.

## NFR Compliance

| NFR ID | Status | Notes |
|--------|--------|-------|
| PERF-U3-01 | Compliant | Curried selectors prevent over-rendering |
| PERF-U3-02 | Compliant | `setInterval` in FloatingWidget (1s tick) |
| REL-U3-05 | Compliant | FloatingWidget timer cleaned up via useEffect |
| REL-U3-06 | Compliant | IncomingCallNotification auto-dismiss after 30s |
| SEC-U3-01 | Compliant | Tokens never logged; console.error only on failure |
| SEC-U3-02 | Compliant | Type guards on all WS payloads |
| SEC-U3-03 | Compliant | Generic error messages returned to UI |
| USE-U3-01 | Compliant | `data-testid` attributes on all interactive elements |
| MAINT-U3-01 | Compliant | i18n via `plugin.rtk.*` message IDs |
