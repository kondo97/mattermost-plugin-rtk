# Unit 4: Webapp - Call Page & Post — Frontend Components

## Component Overview

```
Unit 4 introduces:
  CallPost                     — Mattermost post renderer (custom_cf_call)
  CallPage                     — Standalone call page (new browser tab)
    └─ RtkMeeting              — Cloudflare RTK UI Kit component

Unit 3 updates (URL parameter propagation):
  ChannelHeaderButton          — openCallTab() updated
  ToastBar                     — joinCall() updated
  FloatingWidget               — handleOpenInNewTab() updated
  IncomingCallNotification     — joinCall() updated
```

---

## New Files

### `webapp/src/components/call_post/index.tsx`

**Type**: Mattermost post renderer (`registerPostTypeComponent`)
**Registered in**: `webapp/src/index.tsx`

**Props**:
```typescript
interface Props {
    post: {
        id: string;
        props: CallPostProps;  // call_id, channel_id, creator_id, start_at, end_at, participants
    };
}
```

**State**: none (stateless functional component using Redux selectors)

**Selectors used**:
- `selectCallByChannel(post.props.channel_id)` — live call data
- `selectMyActiveCall` — determine if "Join" is disabled

**Data merge logic**:
```
const liveCall = useSelector(selectCallByChannel(channelId));
const participants = liveCall?.participants ?? post.props.participants;
const endAt = liveCall ? (liveCall.endAt ?? 0) : post.props.end_at;
// if liveCall has endAt field, use it; otherwise fall back to props
```

**Active state render**:
```tsx
<div data-testid="call-post">
  <div data-testid="call-post-status-indicator" className="active" />
  <span data-testid="call-post-label">Call started</span>
  <span data-testid="call-post-start-time">{formattedStartTime}</span>
  <div data-testid="call-post-avatars">
    {participants.slice(0,3).map(userId => <Avatar key={userId} userId={userId} />)}
    {overflowCount > 0 && <span>(+{overflowCount})</span>}
  </div>
  <button
    type="button"
    disabled={myActiveCall?.callId === post.props.call_id}
    onClick={handleJoin}
    data-testid="call-post-join-button"
  >
    {formatMessage({id: 'plugin.rtk.call_post.join'})}
  </button>
</div>
```

**Ended state render**:
```tsx
<div data-testid="call-post">
  <div data-testid="call-post-status-indicator" className="ended" />
  <span data-testid="call-post-label">Call ended</span>
  <span data-testid="call-post-end-time">{formattedEndTime}</span>
  <span data-testid="call-post-duration">{formattedDuration}</span>
  {/* No join button */}
</div>
```

**Helpers**:
- `formatDuration(ms: number): string` — formats ms to "X min" or "Xh Ym"
- `openCallTab(token, callId, channelName)` — `window.open(...)` with URL params
- SwitchCallModal rendered inline (same pattern as Unit 3)

**i18n keys**:
- `plugin.rtk.call_post.join`
- `plugin.rtk.call_post.label_active`
- `plugin.rtk.call_post.label_ended`
- `plugin.rtk.call_post.started_at`
- `plugin.rtk.call_post.ended_at`
- `plugin.rtk.call_post.duration`
- `plugin.rtk.call_post.tooltip_already_in_call`

---

### `webapp/src/call_page/main.tsx`

**Type**: Standalone bundle entry point (NOT a Mattermost plugin component)
**Bundle**: `call.js` — served at `/plugins/{id}/call.js`
**HTML**: `server/assets/call.html` — `<script src="call.js"></script>`

**Responsibilities**:
1. Parse URL params (`token`, `call_id`, `channel_name`)
2. Validate required params
3. Set `document.title`
4. Mount `<CallPage />` into `document.body`

```tsx
const params = new URLSearchParams(window.location.search);
const token = params.get('token') ?? '';
const callId = params.get('call_id') ?? '';
const channelName = params.get('channel_name') ?? '';

document.title = channelName ? `Call in #${channelName}` : 'RTK Call';

const root = document.getElementById('root') ?? document.body;
ReactDOM.createRoot(root).render(
    <CallPage token={token} callId={callId} />,
);
```

---

### `webapp/src/call_page/CallPage.tsx`

**Type**: React component (standalone, no Mattermost framework)

**Props**:
```typescript
interface Props {
    token: string;
    callId: string;
}
```

**Responsibilities**:
1. Show error screen if `token` is empty
2. Initialize heartbeat loop via `useEffect` (15s interval, cleanup on unmount)
3. Register `beforeunload` handler via `useEffect` (sendBeacon, cleanup on unmount)
4. Render `<DyteProvider client={meeting}>` + `<RtkMeeting mode="fill" />`

**Heartbeat effect**:
```typescript
useEffect(() => {
    if (!callId) return undefined;
    const id = setInterval(() => {
        fetch(`/plugins/${PLUGIN_ID}/api/v1/calls/${callId}/heartbeat`, {method: 'POST'});
    }, 15_000);
    return () => clearInterval(id);
}, [callId]);
```

**sendBeacon effect**:
```typescript
useEffect(() => {
    if (!callId) return undefined;
    const handler = () => {
        navigator.sendBeacon(`/plugins/${PLUGIN_ID}/api/v1/calls/${callId}/leave`);
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
}, [callId]);
```

**RTK UI Kit initialization**:
```typescript
const [meeting, initMeeting] = useDyteClient();  // from @cloudflare/realtimekit-react

useEffect(() => {
    initMeeting({
        authToken: token,
        defaults: {audio: false, video: false},
    });
}, [token]);  // eslint-disable-line react-hooks/exhaustive-deps
```

**Render**:
```tsx
if (!token) {
    return <div data-testid="call-page-error">Missing call token.</div>;
}
if (!meeting) {
    return <div data-testid="call-page-loading">Connecting...</div>;
}
return (
    <DyteProvider value={meeting} fallback={<div>Loading...</div>}>
        <RtkMeeting mode="fill" />
    </DyteProvider>
);
```

**Dependencies** (new packages):
- `@cloudflare/realtimekit-react` — `useDyteClient`, `DyteProvider`
- `@cloudflare/realtimekit-react-ui` — `RtkMeeting`

---

## New Build File: `webapp/vite.config.ts`

Replaces `webpack.config.js`.

**Key configuration**:
```typescript
import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
    plugins: [react()],
    resolve: {
        alias: {'src': path.resolve(__dirname, 'src')},
    },
    build: {
        outDir: 'dist',
        rollupOptions: {
            input: {
                main: path.resolve(__dirname, 'src/index.tsx'),
                call: path.resolve(__dirname, 'src/call_page/main.tsx'),
            },
            output: {
                entryFileNames: '[name].js',
                chunkFileNames: '[name]-[hash].js',
            },
            // Externals apply to main entry only — see vite plugin pattern
        },
    },
});
```

**Externals strategy**: A custom Vite plugin applies externals only to the `main` entry, not the `call` entry.

---

## Unit 3 Updates

### `webapp/src/components/channel_header_button/index.tsx`

Update `openCallTab`:
```typescript
// Before:
const openCallTab = (callId: string, token: string) => {
    window.open(`/plugins/${manifest.id}/call?token=${token}`, '_blank', 'noopener,noreferrer');
};

// After:
const openCallTab = (callId: string, token: string) => {
    const state = store.getState();
    const channel = (state as any).entities?.channels?.channels?.[channel.id];
    const channelName = encodeURIComponent(channel?.display_name ?? '');
    window.open(
        `/plugins/${manifest.id}/call?token=${token}&call_id=${callId}&channel_name=${channelName}`,
        '_blank',
        'noopener,noreferrer',
    );
};
```

Same pattern applies to `ToastBar`, `FloatingWidget`, `IncomingCallNotification`.

---

## `index.tsx` Registration

Add `registerPostTypeComponent`:
```typescript
registry.registerPostTypeComponent(
    'custom_cf_call',
    () => {
        // CallPost reads post from component props injected by Mattermost
        return <CallPost />;
    },
);
```

---

## Story Traceability

| Story | Components |
|-------|-----------|
| US-007: Custom post appears when call starts | CallPost (active state) |
| US-009: Join from post | CallPost join button handler |
| US-010: Join disabled when already in call | CallPost + selectMyActiveCall |
| US-012: Post updates in real-time | CallPost + Redux selectors |
| US-013: Leave by closing tab | CallPage sendBeacon |
| US-015: Host ends call | CallPage RtkMeeting (UI Kit host controls) |
| US-016: Post ended state | CallPost (ended state) |
| US-006: Tab title | CallPage document.title |
| US-025: Last-participant auto-end | CallPage sendBeacon → server handles |
