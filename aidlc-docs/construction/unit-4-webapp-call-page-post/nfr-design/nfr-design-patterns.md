# Unit 4: Webapp - Call Page & Post — NFR Design Patterns

## Pattern U4-1: Dual-Entry Vite Build with Conditional Externals (MAINT-U4-03, BR-U4-014/015)

Two Vite entries produce two independent bundles. The `main` entry externalizes Mattermost-provided globals; the `call` entry bundles everything independently.

```typescript
// vite.config.ts
import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// Maps external module IDs to their Mattermost-provided global variable names
const MATTERMOST_EXTERNALS: Record<string, string> = {
    react: 'React',
    'react-dom': 'ReactDOM',
    redux: 'Redux',
    'react-redux': 'ReactRedux',
    'prop-types': 'PropTypes',
    'react-bootstrap': 'ReactBootstrap',
    'react-router-dom': 'ReactRouterDom',
};

export default defineConfig({
    plugins: [react()],
    resolve: {
        alias: {src: path.resolve(__dirname, 'src')},
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
                globals: MATTERMOST_EXTERNALS,
                format: 'iife',  // Mattermost plugin convention
            },
            // external() receives the importer's chunk name via the options context
            // Apply externals only when the import chain originates from the 'main' entry
            external(id, _importer, isResolved) {
                // Resolved IDs are absolute paths — skip
                if (isResolved) { return false; }
                return Object.prototype.hasOwnProperty.call(MATTERMOST_EXTERNALS, id);
            },
        },
    },
});
```

**Note**: The `external` function approach above externalizes these packages globally. A more precise per-entry solution uses a Rollup plugin or build phase; the exact implementation is determined during Code Generation. The design intent is: `main.js` externalizes, `call.js` does not.

---

## Pattern U4-2: Call Page URL Parameter Parser (BR-U4-007/008/019)

URL parameters are parsed once on page load. Missing required params trigger an error screen. `channel_name` is decoded from URL encoding.

```typescript
// webapp/src/call_page/main.tsx
function parseCallPageParams(): {token: string; callId: string; channelName: string} {
    const params = new URLSearchParams(window.location.search);
    return {
        token: params.get('token') ?? '',
        callId: params.get('call_id') ?? '',
        channelName: decodeURIComponent(params.get('channel_name') ?? ''),
    };
}

// Usage:
const {token, callId, channelName} = parseCallPageParams();

// Set tab title (BR-U4-008)
document.title = channelName ? `Call in #${channelName}` : 'RTK Call';

// Validate (BR-U4-007)
if (!token) {
    // Render error screen — no RTK initialization
}
```

**Encoding at source** (in Unit 3 components, updated per BR-U4-018/019):
```typescript
const channelName = encodeURIComponent(
    (store.getState() as any).entities?.channels?.channels?.[channelId]?.display_name ?? '',
);
const url = `/plugins/${manifest.id}/call?token=${token}&call_id=${callId}&channel_name=${channelName}`;
window.open(url, '_blank', 'noopener,noreferrer');
```

---

## Pattern U4-3: Call Page Lifecycle Effects (REL-U4-03/04, BR-U4-010/011)

Two independent `useEffect` hooks manage the call page side effects. Each has its own cleanup to prevent leaks.

```typescript
// In CallPage.tsx

// Effect 1: Heartbeat loop
useEffect(() => {
    if (!callId) { return undefined; }
    const PLUGIN_ID = 'com.mattermost.plugin-rtk';
    const id = setInterval(() => {
        // Fire-and-forget — REL-U4-01
        fetch(`/plugins/${PLUGIN_ID}/api/v1/calls/${callId}/heartbeat`, {method: 'POST'});
    }, 15_000);
    return () => clearInterval(id);  // REL-U4-03
}, [callId]);

// Effect 2: sendBeacon on tab close
useEffect(() => {
    if (!callId) { return undefined; }
    const PLUGIN_ID = 'com.mattermost.plugin-rtk';
    const handler = () => {
        navigator.sendBeacon(`/plugins/${PLUGIN_ID}/api/v1/calls/${callId}/leave`);
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);  // REL-U4-04
}, [callId]);
```

---

## Pattern U4-4: Post Props + Redux Data Merge (BR-U4-005, Q4=C)

CallPost merges post-time snapshot data with live Redux state. Post props provide the initial render; Redux provides real-time updates.

```typescript
// In CallPost component
const CallPost = ({post}: {post: Post}) => {
    const props = post.props as CallPostProps;

    // Live Redux state (may be undefined before first WS event)
    const liveCall = useSelector(selectCallByChannel(props.channel_id));
    const myActiveCall = useSelector(selectMyActiveCall);

    // Merge: Redux wins if available, props as fallback
    const participants = liveCall?.participants ?? props.participants ?? [];
    const isEnded = liveCall
        ? Boolean(liveCall.endAt && liveCall.endAt > 0)
        : props.end_at > 0;
    const startAt = liveCall?.startAt ?? props.start_at;
    const endAt = liveCall?.endAt ?? props.end_at;

    // Join disabled check (BR-U4-002)
    const isAlreadyInCall = myActiveCall?.callId === props.call_id;

    if (isEnded) {
        return <CallPostEnded startAt={startAt} endAt={endAt} />;
    }
    return (
        <CallPostActive
            participants={participants}
            startAt={startAt}
            callId={props.call_id}
            channelId={props.channel_id}
            isAlreadyInCall={isAlreadyInCall}
        />
    );
};
```

---

## Pattern U4-5: RTK SDK Initialization with useDyteClient (REL-U4-07)

RTK SDK is initialized via the `useDyteClient` hook. Errors during initialization are caught and rendered as a user-friendly error screen.

```typescript
// In CallPage.tsx
import {useDyteClient, DyteProvider} from '@cloudflare/realtimekit-react';
import {RtkMeeting} from '@cloudflare/realtimekit-react-ui';

const CallPage = ({token, callId}: {token: string; callId: string}) => {
    const [meeting, initMeeting] = useDyteClient();
    const [initError, setInitError] = useState<string | null>(null);

    useEffect(() => {
        if (!token) { return; }
        initMeeting({
            authToken: token,
            defaults: {audio: false, video: false},
        }).catch((err: Error) => {
            // Token intentionally not logged — SEC-U4-01
            console.error('[rtk-plugin] RTK init error:', err.message);
            setInitError('Failed to connect to the call. Please close this tab and try again.');
        });
    }, [token]); // eslint-disable-line react-hooks/exhaustive-deps

    if (!token) {
        return <div data-testid="call-page-error">Missing call token.</div>;
    }
    if (initError) {
        return <div data-testid="call-page-error">{initError}</div>;
    }
    if (!meeting) {
        return <div data-testid="call-page-loading">Connecting...</div>;
    }
    return (
        <DyteProvider value={meeting} fallback={<div>Loading...</div>}>
            <RtkMeeting
                mode='fill'
                data-testid='call-page-meeting'
            />
        </DyteProvider>
    );
};
```

---

## Pattern U4-6: CSP Update for UI Kit (SEC-U4-02)

The existing CSP in `server/api_static.go` must be extended to allow UI Kit inline styles.

```go
// Before:
w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src *")

// After:
w.Header().Set("Content-Security-Policy",
    "default-src 'self'; connect-src *; style-src 'self' 'unsafe-inline'")
```

This is the minimum change needed. The `connect-src *` already allows RTK WebRTC connections.
