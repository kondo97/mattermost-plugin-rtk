# Unit 4: Webapp - Call Page & Post — Code Summary

## Overview

Unit 4 delivers two new rendering surfaces (CallPost and standalone Call Page), migrates the build system from webpack to Vite, and updates all Unit 3 components to pass `call_id` and `channel_name` to the call tab URL.

---

## Files Created

### Build System

| File | Description |
|------|-------------|
| `webapp/vite.config.ts` | Dual-entry Vite config. `main` entry externalizes Mattermost globals; `call` entry bundles React independently. Controlled via `VITE_BUILD_TARGET` env var. |

### Shared Utility

| File | Description |
|------|-------------|
| `webapp/src/utils/call_tab.ts` | `buildCallTabUrl(pluginId, token, callId, channelName)` — shared URL builder with `encodeURIComponent`. `getChannelDisplayName(state, channelId)` — resolves channel name from Redux state. |

### CallPost Component

| File | Description |
|------|-------------|
| `webapp/src/components/call_post/index.tsx` | Root renderer. Merges `post.props` (initial) with Redux live data (Pattern U4-4). Handles join flow with SwitchCallModal and error modal. |
| `webapp/src/components/call_post/CallPostActive.tsx` | Active state: green indicator, start time, participant avatars (up to 3 + overflow), "Join call" button (disabled if already in call). |
| `webapp/src/components/call_post/CallPostEnded.tsx` | Ended state: gray indicator, end time, duration. No join button. |

### Call Page Bundle

| File | Description |
|------|-------------|
| `webapp/src/call_page/main.tsx` | Standalone entry point. Parses URL params, sets `document.title`, mounts `<CallPage />`. No Mattermost framework dependencies. |
| `webapp/src/call_page/CallPage.tsx` | RTK SDK initialization (`useRealtimeKitClient`), `fetch+keepalive` on `beforeunload` for leave detection, `<RtkMeeting mode="fill" />`. Error/loading states. RTK SDK Japanese translations via `useLanguage()`. |

### Tests

| File | Description |
|------|-------------|
| `webapp/src/components/call_post/index.test.tsx` | Enzyme shallow tests: active/ended states, isAlreadyInCall, Redux merge, error modal |
| `webapp/src/call_page/CallPage.test.tsx` | Jest tests: missing token error, loading state, fetch+keepalive leave, SDK init failure |

---

## Files Modified

| File | Change |
|------|--------|
| `webapp/package.json` | Scripts updated (`webpack` → `vite build`). Added: `vite`, `@vitejs/plugin-react`, `@cloudflare/realtimekit-react`, `@cloudflare/realtimekit-react-ui`. Removed: `webpack`, `webpack-cli`, webpack loaders. |
| `Makefile` | Added `copy-call-js` target. Reordered `dist` dependencies: `apply webapp copy-call-js server bundle` (webapp must build before server embeds call.js). |
| `webapp/i18n/en.json` | Added 9 `plugin.rtk.call_post.*` keys |
| `webapp/i18n/ja.json` | Added 9 Japanese `plugin.rtk.call_post.*` keys |
| `webapp/src/index.tsx` | Added `registry.registerPostTypeComponent('custom_cf_call', CallPost)` |
| `server/api_static.go` | Updated CSP: added `; style-src 'self' 'unsafe-inline'` for RTK UI Kit CSS-in-JS (SEC-U4-02) |
| `webapp/src/components/channel_header_button/index.tsx` | `openCallTab()` now uses `buildCallTabUrl` with `channel.display_name` |
| `webapp/src/components/toast_bar/index.tsx` | `joinCall()` now uses `buildCallTabUrl` with channel name from Redux |
| `webapp/src/components/floating_widget/index.tsx` | `handleOpenInNewTab()` now uses `buildCallTabUrl` with channel name from Redux |
| `webapp/src/components/incoming_call_notification/index.tsx` | `joinCall()` now uses `buildCallTabUrl` with channel name from Redux |

---

## Key Design Decisions

### Vite Dual-Bundle (TSD-U4-01)
Two sequential `vite build` invocations controlled by `VITE_BUILD_TARGET` env var:
1. `vite build` (default) → `dist/main.js` with Mattermost externals
2. `VITE_BUILD_TARGET=call vite build` → `dist/call.js` with React bundled

### RTK UI Kit (TSD-U4-02, Q2=A)
`<RtkMeeting mode="fill" />` provides the complete call UI. No custom in-call controls built.

### Post Props + Redux Merge (Pattern U4-4, Q4=C)
`post.props` provides the initial render before any WS events arrive. Redux `selectCallByChannel` provides live updates.

### Call Tab URL Parameters (BR-U4-018/019)
All 4 Unit 3 components + CallPost now include `call_id` and `channel_name` in the call tab URL via `buildCallTabUrl`. This enables leave from the call page and correct tab title.

### Makefile Reordering
`dist` target now runs `webapp` → `copy-call-js` → `server`. Previously `server` ran before `webapp`, which would have embedded a stale placeholder `call.js` into the Go binary.

---

## NFR Compliance

| NFR ID | Status | Notes |
|--------|--------|-------|
| PERF-U4-05 | Compliant | CallPost uses `selectCallByChannel` (scoped selector) |
| REL-U4-01 | N/A | Heartbeat not implemented (deferred) |
| REL-U4-02 | Compliant | `fetch+keepalive` used for tab-close leave (custom CSRF header) |
| REL-U4-03 | N/A | No interval to clear (heartbeat deferred) |
| REL-U4-04 | Compliant | `removeEventListener` in useEffect cleanup |
| REL-U4-05 | Compliant | CallPost falls back to `post.props` when Redux has no data |
| REL-U4-06 | Compliant | Error screen rendered when `token` is absent |
| REL-U4-07 | Compliant | SDK init errors caught and displayed |
| SEC-U4-01 | Compliant | Token not logged anywhere |
| SEC-U4-02 | Compliant | CSP updated with `style-src 'unsafe-inline'` |
| SEC-U4-06 | Compliant | `encodeURIComponent` applied to `callId` and `channelName` in `buildCallTabUrl` |
| USE-U4-01 | Compliant | "Connecting..." loading state |
| USE-U4-03 | Compliant | `data-testid` on all interactive elements |
| USE-U4-04 | Compliant | `document.title = 'Call in #' + channelName` |
