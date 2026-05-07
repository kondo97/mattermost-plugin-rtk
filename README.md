# Mattermost RTK Plugin

A Mattermost plugin that integrates [Cloudflare RealtimeKit](https://developers.cloudflare.com/realtime/) to add video and voice calling directly within Mattermost channels.

| | |
|---|---|
| **Plugin ID** | `com.kondo97.mattermost-plugin-rtk` |
| **Min Mattermost** | 10.11.0 |
| **Backend** | Go 1.25 |
| **Frontend** | React 18 + TypeScript |

---

## Features

- **Start a call** from the channel header — one active call per channel
- **Join / leave** from the header button, channel post, or toast bar
- **Floating in-call widget** — drag, minimize, and fullscreen while staying in Mattermost
- **Standalone call page** — opens in a new tab for a full-screen experience
- **Incoming call notification** — ringing alert for DM and GM channels (30-second auto-dismiss)
- **Client-side feature toggles** — Recording, Screen Share, Polls, Transcription, Waiting Room, Video, Chat, Plugins, Participants panel, and Raise Hand are configured via the Cloudflare RTK SDK preset / UI
- **Admin Console integration** — configure credentials in the System Console or via environment variables
- **Japanese UI** — full i18n support including RTK SDK UI strings

---

## Architecture Overview

```
+---------------------------------------------------------------------+
|  Browser                                                            |
|  main.js (Mattermost plugin bundle)     call.js (standalone tab)   |
|  ChannelHeaderButton  CallPost          CallPage                    |
|  ToastBar  FloatingWidget               useRealtimeKitClient()      |
|  IncomingCallNotification               RtkMeeting UI               |
|  Redux (calls_slice) + WebSocket handlers                           |
+----------------------------+------------------+---------------------+
                             | REST / WebSocket  | New tab
                             v                  |
+----------------------------+------------------+
|  Go Plugin                                    |
|  app/         —  CreateCall / JoinCall /      |
|                  LeaveCall / EndCall (calls,  |
|                  webhook, push, channels)     |
|  api/         —  REST endpoints (gorilla/mux) |
|  rtkclient/   —  Cloudflare RTK HTTP client   |
|  store/         —  Store interface + models      |
|  store/sqlstore/ —  PostgreSQL-backed store impl |
+-------------------+---------------------------+
                    |                    ^ Webhook (HMAC-SHA256)
                    v                    |
            Mattermost DB (SQL)   Cloudflare RTK API
            rtk_call_sessions / rtk_call_participants
            rtk_channel_meetings
            rtk_app_config / rtk_webhook_config
            (api.realtime.cloudflare.com/v2)
```

**Key design decisions:**

| Topic | Decision |
|-------|---------|
| Call state | Stored in Mattermost DB (SQL); synced via WebSocket events |
| Concurrency | Single `callMu sync.Mutex` guards all call state mutations |
| Participant cleanup | RTK webhook (`meeting.participantLeft`) triggers `LeaveCall` |
| Call page auth | JWT token passed as URL parameter; tab close fires `fetch + keepalive` |
| CSP workaround | Vite build patches `worker-timers` blob URL → static `/worker.js` endpoint |
| Feature flags | Client-side only (Cloudflare RTK SDK preset / UI configuration); no server-side env vars |

For a full description of every component, data flow, and API, see **[ARCHITECTURE.md](./ARCHITECTURE.md)**.

---

## Quick Start

### Prerequisites

- Mattermost 10.11.0+
- A [Cloudflare RealtimeKit](https://dash.cloudflare.com/) account (Organization ID + API Key)
- Go 1.25, Node 18+, npm 9+

### Build

```bash
# Server binaries (linux-amd64, linux-arm64)
make build

# Frontend — builds both bundles (main.js + call.js)
cd webapp && npm install && npm run build
```

> `npm run build` runs `vite build` twice (once with `VITE_BUILD_TARGET=call`) and produces both `webapp/dist/main.js` (Mattermost plugin bundle) and `webapp/dist/call.js` (standalone call page bundle).

### Configure

Set credentials in **System Console → Plugins → RTK Plugin**, or via environment variables:

```bash
RTK_ACCOUNT_ID=<your-cloudflare-account-id>
RTK_API_TOKEN=<your-cloudflare-api-token>

# Optional: pin a specific Cloudflare RealtimeKit App.
# When set, the plugin verifies the app exists in your account and uses it as
# the active app instead of discover-or-create. Env-only (no System Console field).
RTK_APP_ID=<your-cloudflare-rtk-app-id>
```

Environment variables take strict precedence over the System Console values.

### Deploy (local mode)

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=<your-token>
make deploy
```

---

## Configuration

Cloudflare credentials are configurable from the System Console (`CloudflareAccountID`, `CloudflareAPIToken`). Each can also be overridden by an environment variable. The Cloudflare RealtimeKit App ID is **environment-only**.

| Setting | System Console Key | Env Var | Required | Description |
|---------|-------------------|---------|----------|-------------|
| Cloudflare Account ID | `CloudflareAccountID` | `RTK_ACCOUNT_ID` | Yes | Required to enable the plugin |
| Cloudflare API Token | `CloudflareAPIToken` | `RTK_API_TOKEN` | Yes | Cloudflare API Token with Realtime / Realtime Admin permissions |
| Cloudflare RTK App ID | (env-only) | `RTK_APP_ID` | No | When set, the plugin verifies the app exists and pins it as the active app. When unset, the plugin discovers or creates the app automatically. |

Feature flags (Recording, Screen Share, Polls, Transcription, Waiting Room, Video, Chat, Plugins, Participants Panel, Raise Hand) are managed entirely on the client side via the Cloudflare RTK SDK preset / UI configuration; there are no server-side environment variables for them.

---

## Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Full implementation guide: system diagram, server/frontend architecture, data flows, API reference, WebSocket events, security design |
| [docs/mobile-push-notifications.md](./docs/mobile-push-notifications.md) | Mobile push notification design: prerequisites, payload reference for call-started and call-ended, suppression mechanism, and lifecycle flow |
| [docs/openapi.yaml](./docs/openapi.yaml) | REST API schema definition (OpenAPI 3.0) |
| [docs/asyncapi.yaml](./docs/asyncapi.yaml) | WebSocket event schema definition (AsyncAPI) |

---

## License

See [LICENSE](./LICENSE).
