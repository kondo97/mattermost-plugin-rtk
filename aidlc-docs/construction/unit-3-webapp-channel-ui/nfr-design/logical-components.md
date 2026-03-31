# Unit 3: Webapp - Channel UI — Logical Components

## Overview

All Unit 3 logical components are client-side (browser). They are organized as logical groupings within the plugin's JavaScript bundle. There are no server-side, infrastructure, or cloud components in this unit.

---

## LC-U3-1: Plugin Redux Slice

**Type**: In-memory client-side state store
**Technology**: `redux` 5.0.1 + `react-redux` 9.2.0
**File**: `webapp/src/redux/calls_slice.ts`

**Responsibilities**:
- Holds the entire Unit 3 state: `callsByChannel`, `myActiveCall`, `incomingCall`, `pluginEnabled`
- Exposes 7 action creators and the reducer function
- Registered with Mattermost's Redux store via `registry.registerReducer(callsReducer)`
- Mounted at: `state['plugins-com.mattermost.plugin-rtk']`

**Pattern applied**: Pattern U3-1 (Plain Redux Reducer)

---

## LC-U3-2: WebSocket Handler Registry

**Type**: Event-driven client-side dispatcher
**Technology**: Mattermost Plugin Registry (`registerWebSocketEventHandler`)
**File**: `webapp/src/redux/websocket_handlers.ts`

**Responsibilities**:
- Registers 5 handlers for custom WS events emitted by the server
- Each handler validates the payload (Pattern U3-2) then dispatches to LC-U3-1
- Handles: `custom_com.kondo97.mattermost-plugin-rtk_call_started`, `custom_com.kondo97.mattermost-plugin-rtk_user_joined`, `custom_com.kondo97.mattermost-plugin-rtk_user_left`, `custom_com.kondo97.mattermost-plugin-rtk_call_ended`, `custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed`

**Error isolation**: Each handler is independently wrapped; a failure in one does not prevent others from executing.

---

## LC-U3-3: Typed Selector Layer

**Type**: Pure functions over Redux state
**Technology**: `react-redux` `useSelector` hook
**File**: `webapp/src/redux/selectors.ts`

**Responsibilities**:
- Provides 5 typed selectors for component use
- Curried channel-scoped selectors prevent cross-channel re-renders (Pattern U3-3)
- No external library (no `reselect`); plain function composition

**Selectors**:
- `selectPluginEnabled` — boolean
- `selectCallByChannel(channelId)` — `ActiveCall | undefined`
- `selectMyActiveCall` — `MyActiveCall | null`
- `selectIncomingCall` — `IncomingCall | null`
- `selectIsCurrentUserParticipant(channelId, userId)` — boolean

---

## LC-U3-4: Plugin Fetch Client

**Type**: HTTP client wrapper (client-side utility)
**Technology**: Browser `fetch` API
**File**: `webapp/src/client.ts` (or inlined in index.tsx)

**Responsibilities**:
- Wraps all plugin API calls with consistent error handling (Pattern U3-6)
- Returns `{ data: T } | { error: string }` — never throws
- Prefixes all paths with `/plugins/{manifest.id}`
- Logs errors to `console.error` without exposing tokens or credentials (SEC-U3-01/03)

**Endpoints used by Unit 3 components**:

| Endpoint | Used by |
|---|---|
| `GET /api/v1/config/status` | Plugin initialization, WS reconnect |
| `POST /api/v1/calls` | ChannelHeaderButton (Start Call) |
| `POST /api/v1/calls/{id}/token` | ChannelHeaderButton/ToastBar (Join) |
| `POST /api/v1/calls/{id}/leave` | SwitchCallModal (leave before join) |
| `POST /api/v1/calls/{id}/dismiss` | IncomingCallNotification (Ignore) |

---

## LC-U3-5: i18n Translation Loader

**Type**: Static resource loader
**Technology**: Mattermost `registerTranslations` API, `babel-plugin-formatjs`
**Files**: `webapp/i18n/en.json`, `webapp/i18n/ja.json`

**Responsibilities**:
- Loads locale-appropriate translation strings at plugin initialization
- Provides English (default) and Japanese translations
- All user-visible strings accessed via `useIntl().formatMessage()` or `<FormattedMessage />`
- Registered once in `initialize()` via `registry.registerTranslations()`

**Locale coverage**:

| Locale | File | Status |
|---|---|---|
| `en` | `webapp/i18n/en.json` | Created in Unit 3 code generation |
| `ja` | `webapp/i18n/ja.json` | Created in Unit 3 code generation |
| All others | — | Fall back to `en` (Mattermost default behavior) |

---

## LC-U3-6: UI Component Tree

**Type**: React component hierarchy (client-side rendering)
**Technology**: React 18.2.0, TypeScript
**Registration**: Mattermost Plugin Registry

| Component | Registry Method | Render Scope |
|---|---|---|
| `ChannelHeaderButton` | `registerCallButtonAction` | Per-channel; renders in channel header call button area |
| `ToastBar` | `registerRootComponent` | Global mount; renders based on `getCurrentChannelId` from Redux |
| `FloatingWidget` | `registerGlobalComponent` | Global; persists across channel and product navigation |
| `IncomingCallNotification` | `registerGlobalComponent` | Global; persists across channel and product navigation |
| `SwitchCallModal` | Inline in parent | Rendered by ChannelHeaderButton or ToastBar as needed |

**Patterns applied**:
- Pattern U3-3 (Scoped Selector) — ChannelHeaderButton, ToastBar
- Pattern U3-4 (Effect Cleanup) — FloatingWidget, IncomingCallNotification
- Pattern U3-5 (i18n) — all components
- Pattern U3-6 (Fetch Error Handling) — ChannelHeaderButton, ToastBar, FloatingWidget, IncomingCallNotification
- Pattern U3-7 (Open-In-Tab Using Saved Token) — FloatingWidget

---

## Component Interaction Diagram

```
[Mattermost WebSocket] ──► [LC-U3-2: WS Handler Registry]
                                       │
                          Type guard   │  dispatch actions
                          (Pattern U3-2)│
                                       ▼
                            [LC-U3-1: Redux Slice]
                                       │
                          useSelector  │  (Pattern U3-3)
                                       ▼
              ┌────────────────────────────────────────────┐
              │           [LC-U3-6: UI Components]         │
              │                                            │
              │  ChannelHeaderButton  ◄──── callsByChannel │
              │  ToastBar             ◄──── callsByChannel │
              │  FloatingWidget       ◄──── myActiveCall   │
              │  IncomingCall         ◄──── incomingCall   │
              └────────────────────────────────────────────┘
                              │  fetch (Pattern U3-6)
                              ▼
                    [LC-U3-4: Plugin Fetch Client]
                              │
                              ▼
                    [Mattermost Plugin HTTP API]
                    (Server: Unit 2 handlers)
```

---

## Non-Components (Excluded from Unit 3)

| Item | Reason |
|---|---|
| Redux DevTools integration | Mattermost's own Redux store; plugin reducer inherits existing devtools config |
| Service Worker / Web Worker | Served by server (Unit 2: `GET /worker.js`) |
| RTK SDK (`@cloudflare/realtimekit`) | Only loaded in call tab (Unit 4) |
| Call page (`/call`) | Unit 4 responsibility |
| Admin settings UI | Unit 5 responsibility |
| Push notification sender | Unit 6 (server-side) |
