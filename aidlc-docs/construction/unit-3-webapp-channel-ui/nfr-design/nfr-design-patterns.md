# Unit 3: Webapp - Channel UI — NFR Design Patterns

## Pattern U3-1: Plain Redux Reducer (MAINT-U3-01/02/03)

Plain switch-based reducer with action type string constants and action creator functions. No `@reduxjs/toolkit`. Matches the patterns used in the Mattermost webapp and Calls plugin.

```typescript
// Action type constants
const SET_PLUGIN_ENABLED       = 'rtk-calls/setPluginEnabled'    as const;
const UPSERT_CALL              = 'rtk-calls/upsertCall'          as const;
const REMOVE_CALL              = 'rtk-calls/removeCall'          as const;
const SET_MY_ACTIVE_CALL       = 'rtk-calls/setMyActiveCall'     as const;
const CLEAR_MY_ACTIVE_CALL     = 'rtk-calls/clearMyActiveCall'   as const;
const SET_INCOMING_CALL        = 'rtk-calls/setIncomingCall'     as const;
const CLEAR_INCOMING_CALL      = 'rtk-calls/clearIncomingCall'   as const;

// Action creators (typed)
export const setPluginEnabled = (enabled: boolean) =>
    ({ type: SET_PLUGIN_ENABLED, payload: enabled } as const);

export const upsertCall = (call: ActiveCall) =>
    ({ type: UPSERT_CALL, payload: call } as const);

export const removeCall = (channelId: string) =>
    ({ type: REMOVE_CALL, payload: channelId } as const);

export const setMyActiveCall = (call: MyActiveCall) =>
    ({ type: SET_MY_ACTIVE_CALL, payload: call } as const);

export const clearMyActiveCall = () =>
    ({ type: CLEAR_MY_ACTIVE_CALL } as const);

export const setIncomingCall = (call: IncomingCall) =>
    ({ type: SET_INCOMING_CALL, payload: call } as const);

export const clearIncomingCall = () =>
    ({ type: CLEAR_INCOMING_CALL } as const);

// Union type for all actions
type CallsAction =
    | ReturnType<typeof setPluginEnabled>
    | ReturnType<typeof upsertCall>
    | ReturnType<typeof removeCall>
    | ReturnType<typeof setMyActiveCall>
    | ReturnType<typeof clearMyActiveCall>
    | ReturnType<typeof setIncomingCall>
    | ReturnType<typeof clearIncomingCall>;

// Initial state
const initialState: CallsPluginState = {
    callsByChannel: {},
    myActiveCall: null,
    incomingCall: null,
    pluginEnabled: false,
};

// Reducer
export function callsReducer(
    state: CallsPluginState = initialState,
    action: CallsAction,
): CallsPluginState {
    switch (action.type) {
        case SET_PLUGIN_ENABLED:
            return { ...state, pluginEnabled: action.payload };
        case UPSERT_CALL:
            return {
                ...state,
                callsByChannel: {
                    ...state.callsByChannel,
                    [action.payload.channelId]: action.payload,
                },
            };
        case REMOVE_CALL:
            // eslint-disable-next-line no-case-declarations
            const { [action.payload]: _, ...remaining } = state.callsByChannel;
            return { ...state, callsByChannel: remaining };
        case SET_MY_ACTIVE_CALL:
            return { ...state, myActiveCall: action.payload };
        case CLEAR_MY_ACTIVE_CALL:
            return { ...state, myActiveCall: null };
        case SET_INCOMING_CALL:
            return { ...state, incomingCall: action.payload };
        case CLEAR_INCOMING_CALL:
            return { ...state, incomingCall: null };
        default:
            return state;
    }
}
```

---

## Pattern U3-2: TypeScript Type Guard for WS Payloads (SEC-U3-02, SECURITY-05/13)

Each WebSocket event payload is validated with a type guard before any Redux dispatch. Unknown or malformed payloads are discarded with a console error. No `eval` or unsafe deserialization.

```typescript
// Type guard example for custom_com.kondo97.mattermost-plugin-rtk_call_started payload
interface CallStartedPayload {
    call_id: string;
    channel_id: string;
    creator_id: string;
    participants: string[];
    start_at: number;
    post_id: string;
}

function isCallStartedPayload(data: unknown): data is CallStartedPayload {
    if (!data || typeof data !== 'object') { return false; }
    const d = data as Record<string, unknown>;
    return (
        typeof d.call_id === 'string' && d.call_id.length > 0 &&
        typeof d.channel_id === 'string' && d.channel_id.length > 0 &&
        typeof d.creator_id === 'string' && d.creator_id.length > 0 &&
        Array.isArray(d.participants) &&
        typeof d.start_at === 'number' &&
        typeof d.post_id === 'string'
    );
}

// Usage in WS handler
export function handleCallStarted(store: Store, currentUserId: string) {
    return (msg: WebSocketMessage) => {
        const data = msg.data;
        if (!isCallStartedPayload(data)) {
            console.error('[rtk-plugin] invalid custom_com.kondo97.mattermost-plugin-rtk_call_started payload', data);
            return;
        }
        store.dispatch(upsertCall({
            id: data.call_id,
            channelId: data.channel_id,
            creatorId: data.creator_id,
            participants: data.participants,
            startAt: data.start_at,
            postId: data.post_id,
        }));
        // ... incoming call logic
    };
}
```

**Pattern applies to**: all 5 WS event handlers with their respective payload shapes.

---

## Pattern U3-3: Scoped Selector to Prevent Over-Rendering (PERF-U3-05)

Selectors that take `channelId` as a parameter use curried factory functions so each component subscribes only to the state slice it needs. Changing call state in channel A does not trigger re-render of the button in channel B.

```typescript
// Curried selector — memoization via stable reference equality in react-redux
export const selectCallByChannel = (channelId: string) =>
    (state: GlobalState): ActiveCall | undefined =>
        selectPluginState(state).callsByChannel[channelId];

export const selectIsCurrentUserParticipant =
    (channelId: string, currentUserId: string) =>
    (state: GlobalState): boolean => {
        const call = selectPluginState(state).callsByChannel[channelId];
        return call?.participants.includes(currentUserId) ?? false;
    };

// In ChannelHeaderButton:
const call = useSelector(selectCallByChannel(channel.id));
const isParticipant = useSelector(selectIsCurrentUserParticipant(channel.id, currentUserId));
// Only re-renders when this channel's call state changes
```

---

## Pattern U3-4: Effect Cleanup for Timers (REL-U3-05/06)

All `setInterval` and `setTimeout` handles are cleared in the `useEffect` cleanup function to prevent timer leaks across component unmounts and state changes.

```typescript
// FloatingWidget — duration timer
useEffect(() => {
    if (!myActiveCall) { return undefined; }
    const startAt = callsByChannel[myActiveCall.channelId]?.startAt ?? Date.now();
    const interval = setInterval(() => {
        setElapsedSeconds(Math.floor((Date.now() - startAt) / 1000));
    }, 1000);
    return () => clearInterval(interval);   // cleanup on unmount or myActiveCall change
}, [myActiveCall?.callId]);                 // re-run only when the active call changes

// IncomingCallNotification — 30s auto-dismiss
useEffect(() => {
    if (!incomingCall) { return undefined; }
    const timeout = setTimeout(() => {
        dispatch(clearIncomingCall());
    }, 30_000);
    return () => clearTimeout(timeout);     // cleanup if incomingCall cleared earlier
}, [incomingCall?.callId]);                 // re-run only when the incoming call changes
```

---

## Pattern U3-5: i18n String Pattern (USE-U3-01/02/03)

All user-visible strings are wrapped using `useIntl().formatMessage()` or `<FormattedMessage />`. No hardcoded English strings in JSX or event handlers. Translation IDs follow the `plugin.rtk.<component>.<element>` convention.

```typescript
// Hook usage (recommended for non-JSX contexts)
import {useIntl} from 'react-intl';

const MyComponent = () => {
    const intl = useIntl();
    const label = intl.formatMessage({id: 'plugin.rtk.channel_header.start_call'});
    return <button aria-label={label}>{label}</button>;
};

// JSX usage
import {FormattedMessage} from 'react-intl';

const ModalBody = () => (
    <p>
        <FormattedMessage id='plugin.rtk.switch_call_modal.body' />
    </p>
);
```

Translation files registered in `initialize()`:
```typescript
registry.registerTranslations((locale: string) => {
    switch (locale) {
        case 'ja': return jaTranslations;  // import from 'i18n/ja.json'
        default:   return enTranslations;  // import from 'i18n/en.json'
    }
});
```

---

## Pattern U3-6: Fetch with Explicit Error Handling (REL-U3-01/02, SEC-U3-03, SECURITY-15)

All `fetch()` calls use a helper that:
1. Always resolves (never throws to the caller)
2. Returns a typed result union `{ data: T } | { error: string }`
3. Logs unexpected errors to `console.error` without exposing internal details to the user
4. Shows a generic error modal for user-facing failures

```typescript
type FetchResult<T> = { data: T } | { error: string };

async function pluginFetch<T>(
    path: string,
    options?: RequestInit,
): Promise<FetchResult<T>> {
    try {
        const resp = await fetch(
            `/plugins/${manifest.id}${path}`,
            {
                headers: { 'Content-Type': 'application/json' },
                ...options,
            },
        );
        if (!resp.ok) {
            // Generic user-facing message; raw server message logged only
            console.error(`[rtk-plugin] API error ${resp.status} on ${path}`);
            return { error: 'An error occurred. Please try again.' };
        }
        const data = await resp.json() as T;
        return { data };
    } catch (err) {
        console.error('[rtk-plugin] network error on', path, err);
        return { error: 'A network error occurred. Please try again.' };
    }
}

// Usage in ChannelHeaderButton click handler:
const result = await pluginFetch<CallResponse>('/api/v1/calls', {
    method: 'POST',
    body: JSON.stringify({ channel_id: channel.id }),
});
if ('error' in result) {
    showErrorModal(result.error);  // generic message — SEC-U3-03
    setLoading(false);
    return;
}
const { data } = result;
dispatch(setMyActiveCall({ callId: data.call.id, channelId: data.call.channel_id, token: data.token }));
```

**Note**: JWT `data.token` is stored in Redux but MUST NOT be logged — SEC-U3-01.

---

## Pattern U3-7: Open-In-Tab Using Saved Token (BR-011)

"Open in new tab" in the FloatingWidget reuses the token stored in `myActiveCall.token` (obtained at join time). No additional API call is made.

```typescript
const handleOpenInNewTab = () => {
    if (!myActiveCall?.token) { return; }
    // Token intentionally not logged — SEC-U3-01
    window.open(
        `/plugins/${manifest.id}/call?token=${encodeURIComponent(myActiveCall.token)}`,
        '_blank',
        'noopener,noreferrer',
    );
};
```

**`noopener,noreferrer`**: Prevents the opened call tab from accessing the opener window, reducing XSS attack surface.

**`encodeURIComponent`**: Ensures the token value is safely URL-encoded (SEC-U3-04).
