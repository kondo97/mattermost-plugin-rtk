# Component Dependency Diagram

## Backend Dependency Graph

```
+------------------+
|  Mattermost      |
|  Plugin SDK      |
+--------+---------+
         |
         v
+--------+---------+        +------------------+
|  Plugin          |------->|  RTKClient       |
|  (plugin.go)     |        |  (rtkclient/)    |
|                  |        +------------------+
|  - CreateCall    |
|  - JoinCall      |        +------------------+
|  - LeaveCall     |------->|  KVStore         |
|  - EndCall       |        |  (store/kvstore) |
|  - Heartbeat     |        +------------------+
|  - Cleanup       |
|                  |        +------------------+
|                  |------->|  Push Sender     |
+--------+---------+        |  (push/)         |
         |                  +------------------+
         |
         v
+--------+---------+        +------------------+
|  API Handler     |------->|  Plugin          |
|  (api/)          |        |  (via PluginAPI  |
|                  |        |   interface)     |
|  - calls.go      |        +------------------+
|  - heartbeat.go  |
|  - config.go     |
|  - mobile.go     |
|  - static.go     |
+--------+---------+
         |
         v
+--------+---------+
|  Background Job  |------->|  Plugin.Cleanup  |
|  (job.go)        |        |  (every 30s)     |
+------------------+
```

## Frontend Dependency Graph (Main Bundle)

```
+------------------+
|  Plugin Entry    |
|  (index.tsx)     |
+--------+---------+
         |
         +----------> registerReducer(callsReducer)
         |
         +----------> registerChannelHeaderButtonAction
         |                    |
         |                    v
         |            +-------+--------+
         |            | ChannelHeader  |
         |            | Button (F-02)  |
         |            +-------+--------+
         |                    |
         |                    +---> reads: callsByChannel[currentChannelId]
         |                    +---> reads: myActiveCall
         |                    +---> opens: SwitchCallModal (F-06)
         |
         +----------> registerPostTypeComponent('custom_cf_call')
         |                    |
         |                    v
         |            +-------+--------+
         |            | CallPost (F-03)|
         |            +-------+--------+
         |                    |
         |                    +---> reads: callsByChannel[channelId]
         |                    +---> reads: myActiveCall
         |
         +----------> registerWebSocketEventHandler x5
         |                    |
         |                    v
         |            +-------+--------+
         |            | WS Handlers    |
         |            | (redux/)       |
         |            +-------+--------+
         |                    |
         |                    +---> dispatches: slice actions
         |
         +----------> renders: ToastBar (F-04)
         |                    +---> reads: callsByChannel[currentChannelId]
         |                    +---> reads: myActiveCall (to show/hide)
         |
         +----------> renders: FloatingWidget (F-05)
         |                    +---> reads: myActiveCall
         |                    +---> links to: /plugins/{id}/call?token=<jwt>
         |
         +----------> renders: IncomingCallNotification (F-07)
         |                    +---> reads: incomingCall
         |
         +----------> registerAdminConsoleCustomSetting
                             |
                             v
                     +-------+--------+
                     | AdminSettings  |
                     | (F-08)         |
                     +----------------+

All components read from / dispatch to:
+------------------+
|  Calls Redux     |
|  (redux/)        |
|  callsByChannel  |
|  myActiveCall    |
|  incomingCall    |
+------------------+
```

## Frontend Dependency Graph (Call Bundle — Standalone)

```
+------------------+
|  call/index.tsx  |
+--------+---------+
         |
         v
+--------+---------+
|  CallPage.tsx    |
|                  |
|  reads: ?token   |------> Cloudflare RTK SDK (DyteProvider)
|                  |        (media handled entirely by Cloudflare)
|  heartbeat loop  |------> POST /api/v1/calls/{id}/heartbeat
|                  |        (Mattermost session cookie automatic)
|  beforeunload    |------> navigator.sendBeacon
|                  |        POST /api/v1/calls/{id}/leave
+------------------+
```

## Cross-Bundle Communication

```
Main Bundle (Mattermost SPA)          Standalone Call Bundle (new tab)
+---------------------------+         +---------------------------+
| FloatingWidget            |         | CallPage                  |
| - "Open in new tab" btn   +-------> | - reads ?token from URL   |
| - passes token in URL     |  HTTP   | - Cloudflare RTK SDK      |
+---------------------------+         +---------------------------+
         ^                                       |
         |                                       | sendBeacon / heartbeat
         |   WebSocket events                    v
+--------+-----------+              +-----------+---------+
| Calls Redux        |<-------------+ Plugin API          |
| - userLeft action  |  WS event    | (server/api/)       |
| - callEnded action |              +---------------------+
+--------------------+
```

## Key Design Constraints

1. **No circular dependencies**: API handlers depend on Plugin via interface, not concrete type
2. **Testability boundary**: RTKClient and KVStore are interfaces; Plugin methods can be tested with mocks
3. **Bundle isolation**: Call page bundle does NOT import Mattermost Redux or webapp internals; it is fully self-contained
4. **State authority**: Redux store is the single source of truth for UI call state; it is updated exclusively via WebSocket events from the server
