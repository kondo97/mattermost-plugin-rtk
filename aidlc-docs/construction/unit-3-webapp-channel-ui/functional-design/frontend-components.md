# Unit 3: Webapp - Channel UI — Frontend Components

## Component Hierarchy

```
Plugin Entry Point (index.tsx)
├── Redux Slice (calls_slice.ts)
│   ├── Actions: setPluginEnabled, upsertCall, removeCall,
│   │            setMyActiveCall, clearMyActiveCall,
│   │            setIncomingCall, clearIncomingCall
│   └── Selectors (selectors.ts)
│
├── WebSocket Handlers (websocket_handlers.ts)
│   ├── handleCallStarted   → custom_cf_call_started
│   ├── handleUserJoined    → custom_cf_user_joined
│   ├── handleUserLeft      → custom_cf_user_left
│   ├── handleCallEnded     → custom_cf_call_ended
│   └── handleNotifDismissed→ custom_cf_notification_dismissed
│
├── ChannelHeaderButton (channel_header_button/)
│   ├── Registered via: registerCallButtonAction
│   ├── Props: channel (Channel from Mattermost)
│   └── Local state: loading: boolean
│
├── ToastBar (toast_bar/)
│   ├── Registered via: registerChannelToastComponent
│   ├── Props: (none — reads channel from Mattermost context)
│   └── Local state: dismissed: boolean
│
├── FloatingWidget (floating_widget/)
│   ├── Registered via: registerGlobalComponent
│   ├── Props: (none — reads from Redux)
│   └── Local state: durationInterval: ReturnType<typeof setInterval> | null
│
├── SwitchCallModal (switch_call_modal/)
│   ├── Rendered inline by ChannelHeaderButton or ToastBar
│   ├── Props: visible, targetCallId, targetChannelId,
│   │         onConfirm, onCancel
│   └── Local state: (none)
│
└── IncomingCallNotification (incoming_call_notification/)
    ├── Registered via: registerGlobalComponent
    ├── Props: (none — reads from Redux)
    └── Local state: dismissTimeout: ReturnType<typeof setTimeout> | null
```

---

## ChannelHeaderButton

**File**: `webapp/src/components/channel_header_button/index.tsx`

**Registration**: `registry.registerCallButtonAction(ChannelHeaderButton, ChannelHeaderButton, handleClick)`

Since `registerCallButtonAction` takes a `ReactResolvable` for `button`, the component itself is passed as the element type and reads Redux state via hooks.

**Props**: none — component reads all state from Redux directly.

**Redux reads**:
- `getCurrentChannelId(state)` — from `mattermost-redux/selectors/entities/channels`
- `selectPluginEnabled()`
- `selectCallByChannel(currentChannelId)`
- `selectMyActiveCall()`
- `selectIsCurrentUserParticipant(currentChannelId)`

**Local state**:
```typescript
const [loading, setLoading] = useState(false);
```

**Rendered states**:

| State | Condition | Label | Disabled |
|---|---|---|---|
| Hidden | `!pluginEnabled` | — | — |
| Starting | `loading === true` | "Starting call..." + spinner | true |
| In Call | user is participant of this channel's call | "In Call" | true |
| Join Call | active call exists, user not participant | "Join Call" | false |
| Start Call | no active call | "Start Call" | false |

**Click handler logic**:
```
if state is "Start Call":
    setLoading(true)
    POST /calls → on success: setMyActiveCall, open tab, play sound, setLoading(false)
                → on error: show error modal, setLoading(false)

if state is "Join Call":
    if myActiveCall exists and different channel:
        show SwitchCallModal
    else:
        execute join flow (POST /calls/{id}/token → setMyActiveCall, open tab)
```

**Sub-components**:
- `SwitchCallModal` — conditionally rendered inside ChannelHeaderButton when switching

---

## ToastBar

**File**: `webapp/src/components/toast_bar/index.tsx`

**Registration**: `registry.registerRootComponent(ToastBar)`

**Redux reads**:
- `getCurrentChannelId(state)` — from `mattermost-redux/selectors/entities/channels` to determine current channel
- `selectCallByChannel(currentChannelId)`
- `selectMyActiveCall()`
- `selectIsCurrentUserParticipant(currentChannelId)`

**Local state**:
```typescript
const [dismissed, setDismissed] = useState(false);
```

**Visibility rule**: Render only when:
```
activeCall exists in current channel
AND currentUser NOT in participants
AND dismissed === false
```

**Content**:
- Call start time (formatted relative time, e.g., "Started 3 minutes ago")
- Participant avatars (up to 3, with overflow count "+N")
- "Join" button
- "×" dismiss button

**Actions**:
- "Join" → same check as ChannelHeaderButton: if `myActiveCall` exists in different channel, show `SwitchCallModal`; else call join flow
- "×" → `setDismissed(true)`
- On `custom_cf_call_ended` (call removed from Redux): visibility condition evaluates to false automatically; `dismissed` state is irrelevant

**Sub-components**:
- `SwitchCallModal` — conditionally rendered inside ToastBar when switching

---

## FloatingWidget

**File**: `webapp/src/components/floating_widget/index.tsx`

**Registration**: `registry.registerGlobalComponent(FloatingWidget)`

**Redux reads**:
- `selectMyActiveCall()`
- `selectCallByChannel(myActiveCall?.channelId)`

**Local state**:
```typescript
const [elapsedSeconds, setElapsedSeconds] = useState(0);
```

**Visibility rule**: Render only when `myActiveCall !== null`.

**Duration timer**:
- On mount (when `myActiveCall` becomes non-null): start `setInterval` every 1 second, compute `elapsedSeconds = Math.floor((Date.now() - startAt) / 1000)`
- On unmount / `myActiveCall` cleared: clear interval

**Content layout**:
```
[Channel Name]                           [×]
[Avatar][Avatar][Avatar] +N    00:03:14
    [Open in new tab]   [Leave Call]
```

**"Open in new tab" action**:
1. Use `myActiveCall.token` (saved at join time)
2. `window.open('/plugins/{id}/call?token=<token>', '_blank')`

**"Leave Call" action**:
1. `POST /plugins/{id}/api/v1/calls/{callId}/leave`
2. On success: `dispatch(clearMyActiveCall())` (FloatingWidget hides automatically)
3. On API error: show error modal

**`[×]` close button**: not present — widget is dismissed only by leaving the call.

**Note**: No mute/unmute control. Mute is only in the call tab (Unit 4).

---

## SwitchCallModal

**File**: `webapp/src/components/switch_call_modal/index.tsx`

**Usage**: Rendered inline by ChannelHeaderButton and ToastBar. Not globally registered.

**Props**:
```typescript
interface Props {
    visible: boolean;
    targetCallId: string;
    targetChannelId: string;
    onConfirm: () => void; // caller handles leave-then-join logic
    onCancel: () => void;
}
```

**Content**:
- Title: "You are already in a call"
- Body: "Do you want to leave your current call and join the new one?"
- "Cancel" button → `onCancel()`
- "Leave and join new call" button → `onConfirm()`

**Note**: Modal is a simple dialog overlay. No Redux state; controlled by parent via `visible` prop.

---

## IncomingCallNotification

**File**: `webapp/src/components/incoming_call_notification/index.tsx`

**Registration**: `registry.registerGlobalComponent(IncomingCallNotification)`

**Redux reads**:
- `selectIncomingCall()`

**Local state**:
```typescript
const [dismissTimeout, setDismissTimeout] =
    useState<ReturnType<typeof setTimeout> | null>(null);
```

**Visibility rule**: Render only when `incomingCall !== null`.

**Auto-dismiss timer logic** (in `useEffect`):
```typescript
useEffect(() => {
    if (!incomingCall) return;
    const t = setTimeout(() => dispatch(clearIncomingCall()), 30_000);
    setDismissTimeout(t);
    return () => clearTimeout(t); // cleanup if incomingCall cleared earlier
}, [incomingCall?.callId]);
```

**Content**:
- Caller's display name and avatar (fetched via Mattermost user API)
- Channel name
- "Ignore" button + "Join" button

**Actions**:
- "Ignore":
  1. `POST /plugins/{id}/api/v1/calls/{callId}/dismiss` (fire-and-forget)
  2. Wait for `custom_cf_notification_dismissed` WS event to clear Redux state
- "Join":
  1. Clear the auto-dismiss timeout
  2. If `myActiveCall` exists: show SwitchCallModal (inline, or via state)
  3. Else: execute join flow (BL-004)

---

## Redux Slice

**File**: `webapp/src/redux/calls_slice.ts`

**Slice name**: `'rtk-calls'`

**Actions**:

| Action | Payload | Effect |
|---|---|---|
| `setPluginEnabled` | `boolean` | Sets `pluginEnabled` |
| `upsertCall` | `ActiveCall` | Adds or replaces entry in `callsByChannel` |
| `removeCall` | `{ callId, channelId }` | Removes entry from `callsByChannel` |
| `setMyActiveCall` | `MyActiveCall` | Sets `myActiveCall` |
| `clearMyActiveCall` | — | Sets `myActiveCall = null` |
| `setIncomingCall` | `IncomingCall` | Sets `incomingCall` |
| `clearIncomingCall` | — | Sets `incomingCall = null` |

---

## WebSocket Handlers

**File**: `webapp/src/redux/websocket_handlers.ts`

Each handler is a function `(msg: WebSocketMessage, store: Store, currentUserId: string) => void`.

### handleCallStarted
```
Payload: { call_id, channel_id, creator_id, participants, start_at, post_id }

1. dispatch(upsertCall({ id: call_id, channelId: channel_id, ... }))
2. If channel type is 'D' or 'G' AND creator_id !== currentUserId:
       dispatch(setIncomingCall({ callId, channelId, creatorId, startAt }))
3. If creator_id === currentUserId:
       (myActiveCall already set from API response — no action needed here)
```

### handleUserJoined
```
Payload: { call_id, channel_id, user_id, participants }

1. Update callsByChannel[channel_id].participants with new array
   dispatch(upsertCall updated participants)
2. If user_id === currentUserId AND myActiveCall is null:
       dispatch(setMyActiveCall({ callId: call_id, channelId: channel_id, token: '' }))
       // token empty: this path is for multi-session sync only
```

### handleUserLeft
```
Payload: { call_id, channel_id, user_id, participants }

1. Update callsByChannel[channel_id].participants
   dispatch(upsertCall updated participants)
2. If user_id === currentUserId:
       dispatch(clearMyActiveCall())
```

### handleCallEnded
```
Payload: { call_id, channel_id, end_at, duration_ms }

1. dispatch(removeCall({ callId: call_id, channelId: channel_id }))
2. If myActiveCall?.callId === call_id:
       dispatch(clearMyActiveCall())
3. If incomingCall?.callId === call_id:
       dispatch(clearIncomingCall())
```

### handleNotifDismissed
```
Payload: { call_id, user_id }

1. If user_id === currentUserId AND incomingCall?.callId === call_id:
       dispatch(clearIncomingCall())
```

---

## Selectors

**File**: `webapp/src/redux/selectors.ts`

```typescript
// Plugin root state path
const selectPluginState = (state: GlobalState) =>
    state['plugins-com.mattermost.plugin-rtk'] as CallsPluginState;

export const selectPluginEnabled = (state: GlobalState): boolean =>
    selectPluginState(state).pluginEnabled;

export const selectCallByChannel = (channelId: string) =>
    (state: GlobalState): ActiveCall | undefined =>
        selectPluginState(state).callsByChannel[channelId];

export const selectMyActiveCall = (state: GlobalState): MyActiveCall | null =>
    selectPluginState(state).myActiveCall;

export const selectIncomingCall = (state: GlobalState): IncomingCall | null =>
    selectPluginState(state).incomingCall;

export const selectIsCurrentUserParticipant =
    (channelId: string, currentUserId: string) =>
    (state: GlobalState): boolean => {
        const call = selectCallByChannel(channelId)(state);
        return call?.participants.includes(currentUserId) ?? false;
    };
```

---

## index.tsx Registration

**File**: `webapp/src/index.tsx` (modified)

```typescript
public async initialize(registry: PluginRegistry, store: Store<GlobalState>) {
    // 1. Register Redux reducer
    registry.registerReducer(callsReducer);

    // 2. Fetch initial config status
    const currentUserId = getCurrentUserId(store.getState());
    const resp = await fetch(`/plugins/${manifest.id}/api/v1/config/status`, ...);
    store.dispatch(setPluginEnabled(resp.enabled));

    // 3. Register WebSocket handlers
    registry.registerWebSocketEventHandler(
        `custom_${manifest.id}_call_started`, handleCallStarted(store, currentUserId));
    registry.registerWebSocketEventHandler(
        `custom_${manifest.id}_user_joined`, handleUserJoined(store, currentUserId));
    registry.registerWebSocketEventHandler(
        `custom_${manifest.id}_user_left`, handleUserLeft(store, currentUserId));
    registry.registerWebSocketEventHandler(
        `custom_${manifest.id}_call_ended`, handleCallEnded(store, currentUserId));
    registry.registerWebSocketEventHandler(
        `custom_${manifest.id}_notification_dismissed`, handleNotifDismissed(store, currentUserId));

    // 4. Register WS reconnect handler (re-fetch config)
    registry.registerReconnectHandler(async () => {
        const resp = await fetch(...);
        store.dispatch(setPluginEnabled(resp.enabled));
    });

    // 5. Register UI components
    registry.registerCallButtonAction(ChannelHeaderButton, ChannelHeaderButton, () => {});
    registry.registerRootComponent(ToastBar);
    registry.registerGlobalComponent(FloatingWidget);
    registry.registerGlobalComponent(IncomingCallNotification);
}
```

**Note**: `index.tsx` is shared with Unit 4 (which adds `registerPostTypeComponent` for `custom_cf_call`). Unit 4 registration is added in a separate edit to this file.
