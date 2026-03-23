# Unit 4: Webapp - Call Page & Post — Code Generation Plan

## Unit Context

**Purpose**: RTK call page (standalone bundle), `custom_cf_call` post renderer, Vite dual-bundle build migration, heartbeat/sendBeacon lifecycle, Unit 3 URL updates.
**Stories**: US-007, US-009, US-010, US-013, US-015, US-016, US-025 (primary); US-006, US-012 (supporting)
**Tech**: Vite + @cloudflare/realtimekit-react-ui + React 18 + TypeScript + Enzyme (tests)

## Code Location

- **Application code**: `webapp/src/`, `webapp/i18n/`, `server/`, `Makefile`
- **Documentation**: `aidlc-docs/construction/unit-4-webapp-call-page-post/code/`

## Dependencies (already implemented)

- Unit 2: Server endpoints (`/calls/{id}/token`, `/calls/{id}/heartbeat`, `/calls/{id}/leave`, `/calls/{id}/dismiss`)
- Unit 3: Redux slice (`calls_slice.ts`), selectors, `pluginFetch`, `SwitchCallModal`, `index.tsx`

## File Plan

| Step | File | Action |
|---|---|---|
| 1 | `webapp/vite.config.ts` | Create — dual-entry Vite config with conditional externals |
| 2 | `webapp/package.json` | Modify — add Vite + RTK SDK deps, remove webpack deps, update scripts |
| 3 | `Makefile` | Modify — add copy-call-js target, reorder dist dependencies (webapp before server) |
| 4 | `webapp/src/utils/call_tab.ts` | Create — `buildCallTabUrl` shared helper |
| 5 | `webapp/src/components/call_post/index.tsx` | Create — CallPost root component |
| 6 | `webapp/src/components/call_post/CallPostActive.tsx` | Create — active state subcomponent |
| 7 | `webapp/src/components/call_post/CallPostEnded.tsx` | Create — ended state subcomponent |
| 8 | `webapp/src/call_page/main.tsx` | Create — standalone call page entry point |
| 9 | `webapp/src/call_page/CallPage.tsx` | Create — RTK SDK init + heartbeat + sendBeacon |
| 10 | `webapp/i18n/en.json` | Modify — add `plugin.rtk.call_post.*` keys |
| 11 | `webapp/i18n/ja.json` | Modify — add Japanese `plugin.rtk.call_post.*` keys |
| 12 | `webapp/src/index.tsx` | Modify — add `registry.registerPostTypeComponent('custom_cf_call', ...)` |
| 13 | `server/api_static.go` | Modify — add `style-src 'unsafe-inline'` to CSP |
| 14 | `webapp/src/components/channel_header_button/index.tsx` | Modify — use `buildCallTabUrl` |
| 15 | `webapp/src/components/toast_bar/index.tsx` | Modify — use `buildCallTabUrl` |
| 16 | `webapp/src/components/floating_widget/index.tsx` | Modify — use `buildCallTabUrl` |
| 17 | `webapp/src/components/incoming_call_notification/index.tsx` | Modify — use `buildCallTabUrl` |
| 18 | `webapp/src/components/call_post/index.test.tsx` | Create — Enzyme tests: active, ended, disabled-join, error modal |
| 19 | `webapp/src/call_page/CallPage.test.tsx` | Create — Jest tests: URL parsing, heartbeat, sendBeacon, error screen |
| 20 | `aidlc-docs/construction/unit-4-webapp-call-page-post/code/code-summary.md` | Create |

---

## Execution Checklist

### Part A: Build System

- [x] Step 1: Create `webapp/vite.config.ts`
  - Two entries: `main` (`src/index.tsx`) and `call` (`src/call_page/main.tsx`)
  - Output: `dist/main.js` and `dist/call.js` (fixed filenames, no hash)
  - Externals applied to `main` entry only (Pattern U4-1)
  - Mattermost globals: React, ReactDOM, Redux, ReactRedux, PropTypes, ReactBootstrap, ReactRouterDom
  - `@vitejs/plugin-react` for JSX/TSX transform

- [x] Step 2: Modify `webapp/package.json`
  - Add devDependencies: `vite`, `@vitejs/plugin-react`
  - Add dependencies: `@cloudflare/realtimekit-react`, `@cloudflare/realtimekit-react-ui`
  - Remove devDependencies: `webpack`, `webpack-cli`, `babel-loader`, `css-loader`, `sass-loader`, `style-loader`, `file-loader`
  - Update scripts: `build` → `vite build`, `debug` → `vite build --mode development`, `build:watch` → `vite build --watch`
  - Retain: `babel.config.js`, `@babel/core`, `babel-plugin-*` (for Jest)

- [x] Step 3: Modify `Makefile`
  - Add `copy-call-js` target: `cp webapp/dist/call.js server/assets/call.js`
  - Reorder `dist` dependencies: `dist: apply webapp copy-call-js server bundle`
  - (Rationale: webapp must build before server embeds call.js via go:embed)

### Part B: Shared Utility

- [x] Step 4: Create `webapp/src/utils/call_tab.ts`
  - `buildCallTabUrl(pluginId, token, callId, channelName)` → URL string
  - `encodeURIComponent` applied to `callId` and `channelName` (BR-U4-019)
  - Token intentionally not URL-encoded (it is a JWT and already URL-safe)
  - `getChannelDisplayName(state, channelId)` helper — reads from `state.entities.channels.channels[channelId].display_name`

### Part C: CallPost Component

- [x] Step 5: Create `webapp/src/components/call_post/index.tsx`
  - Props: `{post: {id: string; props: CallPostProps}}`
  - Data merge: `post.props` (initial) + `selectCallByChannel` (live) — Pattern U4-4
  - Active/ended branching on `endAt`
  - Join flow: `pluginFetch POST /calls/{id}/token` → `buildCallTabUrl` → `window.open`
  - SwitchCallModal inline (same pattern as Unit 3)
  - Error modal on API failure
  - `data-testid` on all elements

- [x] Step 6: Create `webapp/src/components/call_post/CallPostActive.tsx`
  - Props: `{participants, startAt, callId, channelId, isAlreadyInCall, onJoin}`
  - Green indicator + "Call started" label
  - Start time formatted as locale time string
  - Participant avatars (up to 3) + overflow count
  - "Join call" button: disabled when `isAlreadyInCall`; tooltip when disabled (USE-U4-05)
  - `data-testid` attributes

- [x] Step 7: Create `webapp/src/components/call_post/CallPostEnded.tsx`
  - Props: `{startAt, endAt}`
  - Gray indicator + "Call ended" label
  - End time + duration string (BR-U4-003)
  - `formatDuration(ms)` helper — formats ms to "X min" or "Xh Ym"
  - No join button
  - `data-testid` attributes

### Part D: Call Page Bundle

- [x] Step 8: Create `webapp/src/call_page/main.tsx`
  - Parse URL params via `URLSearchParams` (Pattern U4-2)
  - Set `document.title` (BR-U4-008)
  - Validate `token` — render error screen if missing
  - `ReactDOM.createRoot(...).render(<CallPage token={token} callId={callId} />)`
  - No Mattermost framework dependencies
  - No i18n (standalone bundle, BR-U4-002 / MAINT-U4-02)

- [x] Step 9: Create `webapp/src/call_page/CallPage.tsx`
  - Props: `{token: string; callId: string}`
  - RTK SDK init: `useDyteClient` + `initMeeting({authToken: token})` (Pattern U4-5)
  - Loading state: "Connecting..." (USE-U4-01)
  - Error state: `initError` string → error screen (REL-U4-07)
  - Render: `<DyteProvider value={meeting}><RtkMeeting mode="fill" /></DyteProvider>`
  - Heartbeat loop: `setInterval(15s)` with cleanup (Pattern U4-3, BR-U4-010)
  - sendBeacon: `beforeunload` handler with cleanup (Pattern U4-3, BR-U4-011)
  - Token must not be logged (SEC-U4-01)

### Part E: i18n + Registration + Server

- [x] Step 10: Modify `webapp/i18n/en.json`
  - Add 8 `plugin.rtk.call_post.*` keys (see LC-U4-5 for key list)

- [x] Step 11: Modify `webapp/i18n/ja.json`
  - Add Japanese translations for same 8 keys

- [x] Step 12: Modify `webapp/src/index.tsx`
  - Import `CallPost` from `components/call_post`
  - Add `registry.registerPostTypeComponent('custom_cf_call', CallPost)`

- [x] Step 13: Modify `server/api_static.go`
  - Update `serveCallHTML` CSP: add `; style-src 'self' 'unsafe-inline'` (Pattern U4-6, SEC-U4-02)

### Part F: Unit 3 URL Updates

- [x] Step 14: Modify `webapp/src/components/channel_header_button/index.tsx`
  - Replace `window.open(...)` with `buildCallTabUrl` helper
  - Resolve `channelName` from `store.getState()` at call time

- [x] Step 15: Modify `webapp/src/components/toast_bar/index.tsx`
  - Same `buildCallTabUrl` update

- [x] Step 16: Modify `webapp/src/components/floating_widget/index.tsx`
  - Same `buildCallTabUrl` update (in `handleOpenInNewTab`)

- [x] Step 17: Modify `webapp/src/components/incoming_call_notification/index.tsx`
  - Same `buildCallTabUrl` update

### Part G: Tests

- [x] Step 18: Create `webapp/src/components/call_post/index.test.tsx`
  - Enzyme shallow: active state renders green indicator + join button
  - Enzyme shallow: ended state renders gray indicator + duration, no join button
  - Enzyme shallow: join button disabled when `isAlreadyInCall`
  - Enzyme shallow: SwitchCallModal shown when in different call
  - Error modal renders when API returns error

- [x] Step 19: Create `webapp/src/call_page/CallPage.test.tsx`
  - Mock `@cloudflare/realtimekit-react` and `@cloudflare/realtimekit-react-ui`
  - Test: `token` absent → error screen renders (`data-testid="call-page-error"`)
  - Test: loading state while `meeting` is null
  - Test: heartbeat `setInterval` called with 15000ms
  - Test: `beforeunload` listener registered with `sendBeacon`
  - Test: cleanup clears interval and removes listener

### Part H: Documentation

- [x] Step 20: Create `aidlc-docs/construction/unit-4-webapp-call-page-post/code/code-summary.md`

---

## Story Traceability

| Story | Steps |
|---|---|
| US-007: Custom post appears | Steps 5, 6, 12 |
| US-009: Join from post | Steps 5, 6, 4 |
| US-010: Join disabled when in call | Steps 5, 6 |
| US-012: Post real-time updates | Steps 5 (Redux selectors) |
| US-013: Leave by closing tab | Step 9 (sendBeacon) |
| US-015: Host ends call (UI Kit) | Step 9 (RtkMeeting host controls) |
| US-016: Post ended state | Steps 5, 7 |
| US-006: Tab title | Step 8 |
| US-025: Auto-end (last participant) | Step 9 (sendBeacon triggers server cleanup) |
