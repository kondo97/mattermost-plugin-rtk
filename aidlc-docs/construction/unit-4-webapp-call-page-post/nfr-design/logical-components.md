# Unit 4: Webapp - Call Page & Post — Logical Components

## Component Map

```
Unit 4 Logical Components
│
├── [LC-U4-1] Vite Build Configuration
│     vite.config.ts — dual entry, conditional externals
│
├── [LC-U4-2] CallPost Renderer
│     webapp/src/components/call_post/index.tsx
│     webapp/src/components/call_post/CallPostActive.tsx
│     webapp/src/components/call_post/CallPostEnded.tsx
│
├── [LC-U4-3] Call Page Bundle
│     webapp/src/call_page/main.tsx         — entry point
│     webapp/src/call_page/CallPage.tsx     — main component
│
├── [LC-U4-4] Unit 3 URL Updater
│     Updates to 4 existing Unit 3 components
│
├── [LC-U4-5] i18n Extension
│     webapp/i18n/en.json — call_post.* keys added
│     webapp/i18n/ja.json — call_post.* keys added
│
└── [LC-U4-6] Server CSP Update
      server/api_static.go — style-src 'unsafe-inline' added
```

---

## LC-U4-1: Vite Build Configuration

**File**: `webapp/vite.config.ts`

**Responsibilities**:
- Define two build entries: `main` (`src/index.tsx`) and `call` (`src/call_page/main.tsx`)
- Apply Mattermost externals to `main` entry only (Pattern U4-1)
- Output both bundles to `webapp/dist/` with fixed filenames (`main.js`, `call.js`)
- Replace `webpack.config.js` entirely

**Interfaces with**:
- `Makefile` — triggers `npm run build` and copies `dist/call.js` → `server/assets/call.js`
- `package.json` — scripts updated to use `vite build`

---

## LC-U4-2: CallPost Renderer

**Files**:
- `webapp/src/components/call_post/index.tsx` — root component; registered via `registry.registerPostTypeComponent('custom_cf_call', ...)`
- `webapp/src/components/call_post/CallPostActive.tsx` — active state subcomponent
- `webapp/src/components/call_post/CallPostEnded.tsx` — ended state subcomponent

**Responsibilities**:
- Data merge: `post.props` (initial) + `selectCallByChannel` (live) — Pattern U4-4
- State branching: active (`end_at === 0`) vs ended (`end_at > 0`)
- Join flow: `POST /calls/{id}/token` → `window.open` with `call_id` + `channel_name` params
- SwitchCallModal when user is in a different call
- Error modal for API failures

**Dependencies**:
- `selectCallByChannel` (Unit 3 selector)
- `selectMyActiveCall` (Unit 3 selector)
- `setMyActiveCall` (Unit 3 action creator)
- `pluginFetch` (Unit 3 client)
- `SwitchCallModal` (Unit 3 component)

---

## LC-U4-3: Call Page Bundle

**Files**:
- `webapp/src/call_page/main.tsx` — standalone entry; parses URL params, sets `document.title`, mounts `<CallPage />`
- `webapp/src/call_page/CallPage.tsx` — RTK SDK initialization, heartbeat, sendBeacon, `<RtkMeeting />`

**Responsibilities**:
- Parse `token`, `call_id`, `channel_name` from URL (Pattern U4-2)
- Initialize RTK SDK via `useDyteClient` (Pattern U4-5)
- Render `<RtkMeeting mode="fill" />`
- Heartbeat loop: `setInterval(15s)` with cleanup (Pattern U4-3)
- Leave on close: `navigator.sendBeacon` on `beforeunload` (Pattern U4-3)
- Error screen: missing token, SDK init failure (REL-U4-06/07)

**Dependencies** (bundled into `call.js`, not externalized):
- `react`, `react-dom`
- `@cloudflare/realtimekit-react` — `useDyteClient`, `DyteProvider`
- `@cloudflare/realtimekit-react-ui` — `RtkMeeting`

**No dependencies on**:
- Mattermost Redux store
- `react-intl` / i18n
- Any Unit 3 components

---

## LC-U4-4: Unit 3 URL Updater

**Files modified** (not new):
- `webapp/src/components/channel_header_button/index.tsx`
- `webapp/src/components/toast_bar/index.tsx`
- `webapp/src/components/floating_widget/index.tsx`
- `webapp/src/components/incoming_call_notification/index.tsx`

**Change**: Each component's `window.open(...)` call updated to include `call_id` and `channel_name` URL params per BR-U4-018/019.

**Shared helper** (extracted to avoid duplication):
```typescript
// webapp/src/utils/call_tab.ts
export function buildCallTabUrl(
    pluginId: string,
    token: string,
    callId: string,
    channelName: string,
): string {
    return `/plugins/${pluginId}/call?token=${token}&call_id=${encodeURIComponent(callId)}&channel_name=${encodeURIComponent(channelName)}`;
}
```

This helper is imported by all 4 Unit 3 components and CallPost.

---

## LC-U4-5: i18n Extension

**Files modified**:
- `webapp/i18n/en.json` — add `plugin.rtk.call_post.*` keys
- `webapp/i18n/ja.json` — add Japanese translations for same keys

**New keys**:
```json
{
    "plugin.rtk.call_post.join": "Join call",
    "plugin.rtk.call_post.label_active": "Call started",
    "plugin.rtk.call_post.label_ended": "Call ended",
    "plugin.rtk.call_post.started_at": "Started at {time}",
    "plugin.rtk.call_post.ended_at": "Ended at {time}",
    "plugin.rtk.call_post.duration": "Lasted {duration}",
    "plugin.rtk.call_post.tooltip_already_in_call": "You are already in this call",
    "plugin.rtk.call_post.participants": "{count, plural, one {# participant} other {# participants}}"
}
```

---

## LC-U4-6: Server CSP Update

**File modified**: `server/api_static.go`

**Change**: Add `style-src 'self' 'unsafe-inline'` to the call page CSP header (Pattern U4-6, SEC-U4-02).

**Scope**: Only the `serveCallHTML` handler. `serveCallJS` and `serveWorkerJS` are unchanged.

---

## Dependency Graph

```
index.tsx (main entry)
  └── registerPostTypeComponent → CallPost [LC-U4-2]
        ├── selectCallByChannel (Unit 3)
        ├── selectMyActiveCall (Unit 3)
        ├── pluginFetch (Unit 3)
        ├── SwitchCallModal (Unit 3)
        └── buildCallTabUrl [LC-U4-4]

call_page/main.tsx (call entry — standalone)
  └── CallPage [LC-U4-3]
        ├── useDyteClient (@cloudflare/realtimekit-react)
        ├── DyteProvider (@cloudflare/realtimekit-react)
        └── RtkMeeting (@cloudflare/realtimekit-react-ui)

ChannelHeaderButton / ToastBar / FloatingWidget / IncomingCallNotification
  └── buildCallTabUrl [LC-U4-4]  (Unit 3 files updated)

vite.config.ts [LC-U4-1]
  ├── main entry → dist/main.js (externals applied)
  └── call entry → dist/call.js (no externals)
        ↓ (Makefile cp)
      server/assets/call.js → go:embed → binary
```
