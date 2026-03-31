# Requirements

## Intent Analysis Summary

- **User Request**: Build a Mattermost plugin that integrates Cloudflare RealtimeKit to enable in-channel video/audio calling.
- **Request Type**: New Feature (major implementation on top of starter template)
- **Scope Estimate**: System-wide — Go backend, React/TypeScript frontend, external API integration, admin configuration
- **Complexity Estimate**: Complex — multiple UI surfaces, external service integration, session state management, CSP-aware asset serving

---

## Functional Requirements

### FR-01: Plugin Identity
- The plugin ID shall be `com.kondo97.mattermost-plugin-rtk` (or as configured in `plugin.json`).
- Minimum supported Mattermost server version: 10.11.0.

### FR-02: Start a Call
- A call button shall appear in the channel header for all channel types (public, private, direct message, group direct message) when the plugin is configured.
- Any channel member may start a call (no role restriction).
- Clicking the button shall create a new RealtimeKit meeting session via the Cloudflare RTK API.
- The creator shall be added as a participant with the `group_call_host` preset.
- An auth token (JWT) shall be returned to the creator to join the meeting.
- A custom post of type `custom_cf_call` shall be posted to the channel announcing the call.
- If an active call already exists in the channel (determined via KVStore), starting a new call shall be prevented. The user shall be informed that a call is already in progress.
- Error handling for API failures (e.g., Cloudflare API errors, network failures) shall follow the same patterns as the official Mattermost Calls plugin.
- **Button states** (following Mattermost Calls plugin patterns):
  - No active call: "Start call" (enabled)
  - Active call, user not in it: "Join call" (enabled)
  - User already in this call: "In call" (disabled)
  - Connecting: "Starting call…" with spinner
- A **channel call toast bar** shall appear above the message input when a call is active, showing call start time and participant avatars. The toast shall be dismissable.
- A **sound cue** shall play when the local user joins a call.
- For DM/GM channels, an **incoming call ringing notification** shall be shown to other channel members when a call starts, with "Ignore" and "Join" options. The ring shall last 30 seconds.

### FR-03: Join a Call
- The custom post shall display a Join button when the call is active and the user is not in it.
- When the user is already in the call, the post shall show a "Leave" button instead.
- Clicking Join shall request a new auth token for the existing meeting via the Cloudflare RTK API.
- The joining user shall be added with the `group_call_participant` preset.
- The call window shall open for the joining user.
- If the user is already connected to a **different** call, a **Switch Call Modal** shall be shown: "You're already in a call. Do you want to leave and join this call?" with Cancel and "Leave and join new call" options.
- If the user is already connected to **this** call, the Join button shall be disabled (no modal needed).
- When a call has ended, the post card shall switch to an "ended" state showing end time and duration, with no Join button.

### FR-04: Call Window (Standalone Call Page)
- The plugin shall serve a standalone HTML page at `/plugins/{id}/call`.
- The page shall accept a `token` query parameter as the RTK auth token.
- The page shall initialize the Cloudflare RealtimeKit React SDK using the provided token.
- The page shall open in a **new browser tab** (not a popup window) from the Mattermost webapp.

### FR-05: In-Call Indicator in Mattermost
- No floating widget is required. The call UI is fully contained in the new browser tab (FR-04).
- While the user is participating in a call, Mattermost shall display an indicator that the user is currently in a call. This includes:
  - The channel header call button showing "In call" (disabled) for the active channel.
  - The channel call toast bar remaining visible.
  - The user's avatar appearing in the call post participant list.
- All errors (e.g., API failures) shall be surfaced via modals — never as inline channel messages.

### FR-06: RTK Presets
- Presets are pre-configured in the Cloudflare RealtimeKit dashboard. The plugin shall not create or manage presets.
- The default presets `group_call_host` and `group_call_participant` are assumed to always exist.
- **Future extension**: The plugin may support selecting different presets based on admin configuration (e.g., per-channel or per-role preset assignment). This is out of scope for the initial implementation.

### FR-07: Feature Flags
- The following call features shall be individually togglable via admin settings and environment variables:
  - **Polls** (`polls_enabled`) — in-call polling, default: enabled
  - **Plugins** (`plugins_enabled`) — RTK plugin extensions, default: enabled
  - **Chat** (`chat_enabled`) — side-by-side chat during call, default: enabled
  - **Screen Share** (`screenshare_enabled`) — screensharing, default: enabled
  - **Participants** (`participants_enabled`) — participant list panel, default: enabled
  - **Recording** (`recording_enabled`) — call recording, default: enabled
  - **AI Transcription / Summary** (`ai_transcription_enabled`) — AI-powered transcription and meeting summary, default: enabled
  - **Waiting Room** (`waiting_room_enabled`) — host approval required before participants join, default: disabled
  - **Video** (`video_enabled`) — video capability (disable to enforce audio-only calls), default: enabled
  - **Raise Hand** (`raise_hand_enabled`) — raise hand feature during calls, default: enabled
- Feature flag values shall be returned in the create/join call API responses and applied in the call UI.
- Each feature flag shall support an environment variable override (see FR-09).

### FR-08: Admin Configuration
- The plugin shall expose the following settings in the Mattermost System Console:
  - **Organization ID** (`cloudflare_app_id`) — Cloudflare RTK organization ID
  - **API Key** (`cloudflare_api_token`) — Cloudflare RTK API key
  - Feature flag toggles (FR-07)
- Custom admin UI components shall display whether each credential is sourced from an environment variable (field disabled if so).

### FR-09: Environment Variable Override
- The plugin shall support overriding all configuration values via environment variables. Environment variable values take precedence over admin UI settings.
- Cloudflare credentials:
  - `RTK_ORG_ID` — overrides `cloudflare_app_id`
  - `RTK_API_KEY` — overrides `cloudflare_api_token`
- Feature flags:
  - `RTK_POLLS_ENABLED`
  - `RTK_PLUGINS_ENABLED`
  - `RTK_CHAT_ENABLED`
  - `RTK_SCREENSHARE_ENABLED`
  - `RTK_PARTICIPANTS_ENABLED`
  - `RTK_RECORDING_ENABLED`
  - `RTK_AI_TRANSCRIPTION_ENABLED`
  - `RTK_WAITING_ROOM_ENABLED`
  - `RTK_VIDEO_ENABLED`
  - `RTK_RAISE_HAND_ENABLED`

### FR-10: Configuration Status API
- `GET /api/v1/config/status` — returns whether Cloudflare credentials are configured (for all authenticated users).
- `GET /api/v1/config/admin-status` — returns credential status and source (admin only).
- The call button shall only be registered in the channel header when credentials are configured.

### FR-11: Web Worker Endpoint
- The plugin shall serve a Web Worker script at `/plugins/{id}/worker.js` (unauthenticated).
- This satisfies the browser Content Security Policy `worker-src 'self'` restriction required by the RTK SDK's timer worker.

### FR-12: Session Storage
- Call sessions shall be stored in the Mattermost KVStore.
- Storage keys:
  - `call:channel:{channelID}` — look up active call by channel
  - `call:id:{callID}` — look up call by meeting ID
- Stored fields per session:
  - `call_id` — RTK meeting ID
  - `channel_id` — Mattermost channel ID
  - `post_id` — ID of the `custom_cf_call` post (used to update post state on call end)
  - `creator_id` — user ID of the call initiator
  - `start_at` — call start time (Unix timestamp ms)
  - `end_at` — call end time (Unix timestamp ms; 0 = active)
  - `participants` — list of currently active participant user IDs
- A call is considered active when `end_at == 0`.
- **Future extension**: per-participant session records (join time, mute state, video state) may be added in a later iteration.

### FR-13: Custom Post Type
- The plugin shall register a custom post type `custom_cf_call`.
- The post props shall include `call_id` to identify the meeting to join.
- **Active call state**:
  - Green indicator + "Call started"
  - Call start time and participant avatars (up to 3 shown)
  - "Join call" button for users not in the call
  - "Leave" button for users already in the call
- **Ended call state**:
  - Gray indicator + "Call ended"
  - End time and call duration (e.g., "Lasted 12 minutes")
  - No buttons displayed
- Post state shall update in real-time via WebSocket events (FR-15).

### FR-14: End / Leave a Call
- A user may leave a call at any time by closing the call tab or clicking a "Leave" button.
- When a user leaves, their user ID shall be removed from the `participants` list in KVStore.
- A WebSocket event (`custom_com.kondo97.mattermost-plugin-rtk_user_left`) shall be emitted to all channel members.
- When the last participant leaves, the call shall be automatically ended.
- The call creator (host) may explicitly end the call for all participants via a dedicated "End call" action.
- When a call ends:
  - `end_at` in KVStore shall be set to the current timestamp.
  - The `custom_cf_call` post shall be updated to the "ended" state (via `post_id` stored in KVStore).
  - A WebSocket event (`custom_com.kondo97.mattermost-plugin-rtk_call_ended`) shall be emitted to all channel members.
  - The channel call toast bar shall be dismissed for all clients.

### FR-15: WebSocket Events
- The plugin shall emit WebSocket events to all connected clients (browser and mobile) for real-time UI synchronization.
- Events:
  - `custom_com.kondo97.mattermost-plugin-rtk_call_started` — a call has started in a channel (payload: `call_id`, `channel_id`, `creator_id`, `start_at`)
  - `custom_com.kondo97.mattermost-plugin-rtk_call_ended` — a call has ended (payload: `call_id`, `channel_id`, `end_at`, `duration_ms`)
  - `custom_com.kondo97.mattermost-plugin-rtk_user_joined` — a participant joined (payload: `call_id`, `user_id`)
  - `custom_com.kondo97.mattermost-plugin-rtk_user_left` — a participant left (payload: `call_id`, `user_id`)
  - `custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed` — a user dismissed the incoming call notification (payload: `call_id`, `user_id`)

### FR-16: Mobile Client Support

> **Updated 2026-03-31**: FR-16-2 (push notifications) is no longer implemented. Mobile clients receive call notifications via WebSocket events (`custom_com.kondo97.mattermost-plugin-rtk_call_started`, `custom_com.kondo97.mattermost-plugin-rtk_call_ended`) instead.

The plugin shall provide the server-side support necessary for a modified Mattermost Mobile app to deliver native incoming call notifications and participate in calls.

#### FR-16-1: VoIP Device Token Registration
- The plugin shall expose an API endpoint to register and store mobile VoIP push notification tokens per user.
- iOS VoIP tokens shall be stored with the prefix `apple_voip:{token}`.
- Tokens shall be stored in KVStore keyed by user ID.

#### ~~FR-16-2: Incoming Call Push Notification~~ — REMOVED
- ~~When a call is started, the plugin shall send a push notification to all channel members who are not the call creator, via the Mattermost push notification infrastructure.~~
- ~~The push notification payload shall include the following fields required by the native mobile call UI:~~
  - ~~`channel_id` — Mattermost channel ID~~
  - ~~`sender_id` — caller's user ID~~
  - ~~`sender_name` — caller's display name~~
  - ~~`channel_name` — channel or DM display name~~
  - ~~`ack_id` — unique notification acknowledgment ID~~
  - ~~`uuid` — call identifier~~
  - ~~`sub_type: "calls"` — identifies this as a call notification~~
  - ~~`root_id` — thread ID of the call post (if applicable)~~
  - ~~`server_id` — Mattermost server identifier~~

> **Note**: Push notifications have been replaced by WebSocket events (`custom_com.kondo97.mattermost-plugin-rtk_call_started`, `custom_com.kondo97.mattermost-plugin-rtk_call_ended`) for notifying mobile clients of incoming and ended calls.

#### FR-16-3: Dismiss Notification API
- The plugin shall expose an API endpoint to mark an incoming call notification as dismissed for a specific user.
- Upon dismissal, the server shall emit a `custom_com.kondo97.mattermost-plugin-rtk_notification_dismissed` WebSocket event (FR-15) so other clients stop showing the ringing notification.

#### FR-16-4: Mobile Call Join API
- The existing `POST /api/v1/calls/{callId}/token` endpoint shall support mobile clients.
- The response shall include all feature flags (FR-07) so the mobile app can configure the call UI accordingly.

#### FR-16-5: Call UI on Mobile
- The call UI on mobile shall be rendered natively within the Mattermost Mobile app (not a WebView).
- The plugin is responsible for the server-side API and push notification delivery; the native call UI implementation resides in the mobile app.

---

## Non-Functional Requirements

### NFR-01: External API Integration
- All calls to the Cloudflare RTK API shall use HTTPS.
- Authentication shall use HTTP Basic Auth with `orgID:apiKey`.
- API base URL: `https://api.realtime.cloudflare.com/v2`.

### NFR-02: Security
- All plugin API endpoints (except `/worker.js`, `/call`, `/call.js`) shall require a valid `Mattermost-User-ID` header.
- Admin-only endpoints shall verify admin role server-side.
- Cloudflare API credentials shall not be exposed to end users.
- RTK tokens (JWTs) shall be used as the authorization mechanism for joining calls.
- SECURITY extension rules (SECURITY-01 through SECURITY-15) are enforced as blocking constraints.

### NFR-03: Performance
- Target scale: up to 200 concurrent users.
- Media traffic is handled entirely by Cloudflare infrastructure; plugin only manages signaling and session metadata.
- API response times for token generation should be under 1 second under normal conditions.

### NFR-04: Build System
- Frontend shall use Vite with two separate build configurations:
  - Main plugin bundle: `webapp/dist/main.js` (React/Redux externalized)
  - Standalone call page bundle: `webapp/dist/call.js` (React and RTK libraries bundled)
- The call page JS shall be embedded in the Go binary via `//go:embed`.
- Backend shall produce binaries for `linux-amd64` and `linux-arm64` only.

### NFR-05: Mattermost Compatibility
- Plugin shall target Mattermost server v10.11.0+.
- Frontend shall use Mattermost CSS variables for theming compatibility.
- All Mattermost API interactions shall use the official `pluginapi` client.

### NFR-06: Maintainability
- KVStore access shall be encapsulated behind the `KVStore` interface for testability.
- Cloudflare RTK API calls shall be encapsulated behind a `RTKClient` interface, enabling mock-based unit testing without real API calls.
- Command and API handlers shall be separated into distinct packages/files.
- Configuration shall use a thread-safe struct with `sync.RWMutex` and Clone pattern.
- Structured logging shall be used for all significant events (call start, call end, participant join/leave) and errors, using the Mattermost plugin logger.
- Unit tests shall be provided for core business logic (call creation, join, end flows) using mocks for external dependencies (RTK API, KVStore).

---

## Out of Scope

- Persistent call history
- Rate limiting on token generation
- Per-participant session records (join time, mute state, video state)
- Native call UI implementation on mobile (handled in the Mattermost Mobile app repository)
- iOS CallKit / Android ConnectionService integration (handled in the mobile app)
