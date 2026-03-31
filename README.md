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
- **10 feature flags** — toggle Recording, Screen Share, Polls, Transcription, Waiting Room, Video, Chat, Plugins, Participants panel, and Raise Hand
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
|  calls.go  —  CreateCall / JoinCall /         |
|               LeaveCall / EndCall             |
|  api_*.go  —  REST endpoints                  |
|  rtkclient/  —  Cloudflare RTK HTTP client    |
|  store/kvstore/  —  KVStore abstraction       |
+-------------------+---------------------------+
                    |                    ^ Webhook (HMAC-SHA256)
                    v                    |
            Mattermost KVStore    Cloudflare RTK API
            call:id / channel /   api.realtime.cloudflare.com/v2
            meeting / active_calls
```

**Key design decisions:**

| Topic | Decision |
|-------|---------|
| Call state | Stored in Mattermost KVStore; synced via WebSocket events |
| Concurrency | Single `callMu sync.Mutex` guards all call state mutations |
| Participant cleanup | RTK webhook (`meeting.participantLeft`) triggers `LeaveCall` |
| Call page auth | JWT token passed as URL parameter; tab close fires `fetch + keepalive` |
| CSP workaround | Vite build patches `worker-timers` blob URL → static `/worker.js` endpoint |
| Feature flags | `*bool` fields default to `true` (nil = enabled); overridable via env vars |

For a full description of every component, data flow, and API, see **[ARCHITECTURE.md](./ARCHITECTURE.md)**.

---

## Quick Start

### Prerequisites

- Mattermost 10.11.0+
- A [Cloudflare RealtimeKit](https://dash.cloudflare.com/) account (Organization ID + API Key)
- Go 1.25, Node 18+, npm 9+

### Build

```bash
# Server binaries (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64)
make build

# Frontend — main bundle
cd webapp && npm install && npm run build

# Frontend — standalone call page bundle
cd webapp && VITE_BUILD_TARGET=call npm run build
```

### Configure

Set credentials in **System Console → Plugins → RTK Plugin**, or via environment variables:

```bash
RTK_ORG_ID=<your-cloudflare-org-id>
RTK_API_KEY=<your-cloudflare-api-key>
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

All settings are configurable from the System Console. Each can also be overridden by an environment variable.

| Setting | Env Var | Default | Description |
|---------|---------|---------|-------------|
| Cloudflare Org ID | `RTK_ORG_ID` | — | Required to enable the plugin |
| Cloudflare API Key | `RTK_API_KEY` | — | Required to enable the plugin |
| Recording | `RTK_RECORDING_ENABLED` | `true` | Allow call recording |
| Screen Share | `RTK_SCREEN_SHARE_ENABLED` | `true` | Allow screen sharing |
| Polls | `RTK_POLLS_ENABLED` | `true` | In-call polls |
| Transcription | `RTK_TRANSCRIPTION_ENABLED` | `true` | Real-time transcription |
| Waiting Room | `RTK_WAITING_ROOM_ENABLED` | `true` | Require host approval to join |
| Video | `RTK_VIDEO_ENABLED` | `true` | Camera video |
| Chat | `RTK_CHAT_ENABLED` | `true` | In-call text chat |
| Plugins | `RTK_PLUGINS_ENABLED` | `true` | Third-party RTK plugins |
| Participants Panel | `RTK_PARTICIPANTS_ENABLED` | `true` | Participant list UI |
| Raise Hand | `RTK_RAISE_HAND_ENABLED` | `true` | Raise hand feature |

---

## Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Full implementation guide: system diagram, server/frontend architecture, data flows, API reference, WebSocket events, security design |
| [aidlc-docs/](./aidlc-docs/) | AI-DLC design artifacts: requirements, user stories, functional design, NFR design, and code generation plans for each unit |

---

## License

See [LICENSE](./LICENSE).
