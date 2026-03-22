# Cloudflare RealtimeKit - Comprehensive Technical Reference

> Source: https://developers.cloudflare.com/realtime/realtimekit/
> Note: The original URLs at `/calls/realtimekit/` redirect to 404; the product lives at `/realtime/realtimekit/`.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Core Concepts](#2-core-concepts)
3. [SDK Packages](#3-sdk-packages)
4. [Installation](#4-installation)
5. [Backend Setup](#5-backend-setup)
6. [REST API Reference](#6-rest-api-reference)
7. [Client SDK (Core) API Reference](#7-client-sdk-core-api-reference)
8. [UI Kit](#8-ui-kit)
9. [Features](#9-features)
10. [Error Codes](#10-error-codes)
11. [Pricing](#11-pricing)
12. [FAQ and Limitations](#12-faq-and-limitations)

---

## 1. Overview

**RealtimeKit** is Cloudflare's SDK platform for integrating live video and voice communication into web and mobile applications. It is built on WebRTC, layered atop Cloudflare's **Realtime SFU** (Selective Forwarding Unit) infrastructure, which handles media track management, peer management, and global media routing.

### Use Cases

| Use Case | Description |
|----------|-------------|
| **Group Calls** | Team meetings, virtual classrooms, private video chats |
| **Webinars** | Large-scale one-to-many events with stage management, plugins, chat, polling |
| **Audio-Only Calls** | Bandwidth-efficient calls with mute, hand-raise, and role management |

### Architecture

- **Client SDKs** (UI Kit, Core SDK) run in browsers and mobile apps
- **Realtime SFU** routes media across Cloudflare's global network
- **REST APIs** manage meetings, participants, presets, and recordings server-side
- **Webhooks** deliver server-side events
- **Signaling Server** coordinates real-time connection setup

---

## 2. Core Concepts

### 2.1 App

An **App** is a workspace that groups meetings, participants, presets, recordings, and configurations into an isolated namespace.

- Treat each App as an environment-specific container (e.g., separate Apps for staging and production).
- Presets are defined at the App level and apply across all meetings within that App.
- Created via the Cloudflare Dashboard or API.

### 2.2 Meeting

A **Meeting** is a reusable virtual room. It persists indefinitely—it does not have a specific start or end time. Participants can be added before or just-in-time.

Key properties:
- Has a unique Meeting ID
- Can have a title
- Supports configurable features: `chat`, `recording`, `transcriptions`
- Configuration options: `record_on_start`, `persist_chat`, `ai_config`
- Only **one active session** can exist at a time
- Session status becomes `ENDED` shortly after the last participant leaves
- Can be set to `INACTIVE` via PATCH to prevent new joins

### 2.3 Session

A **Session** is the live instantiation of a Meeting.

- Created automatically when the **first participant joins** a meeting
- Ends shortly after the **last participant leaves**
- Has its own set of participants, chat history, and recordings
- Inherits all parent Meeting settings (recording, chat persistence, etc.)
- Billing is per-participant-minute per active session

**Analogy**: A recurring weekly standup = Meeting; each week's actual call = Session.

### 2.4 Participant

A **Participant** is a user who is allowed to join a specific meeting.

- Added via the **Add Participant REST API**, which returns:
  - `id` - unique participant identifier for that meeting
  - `token` (auth token) - used by the client SDK to join
- Auth tokens are **time-bound** and expire; use the Refresh Token endpoint instead of creating a new participant
- **Do not re-use auth tokens** across multiple participants
- One participant can join multiple live sessions of the same meeting over time
- One participant can have multiple peer connections (multiple devices/tabs)
- `custom_participant_id` can map your internal user ID to the RealtimeKit record; use stable opaque IDs (numeric IDs, UUIDs), **not personal data** like emails

### 2.5 Preset

A **Preset** is a reusable configuration object that defines a participant's experience in a meeting.

- Defined at the App level; applies across any meeting in that App
- Multiple participants in the same meeting can use different presets (role-based access)
- Controls:
  - **Meeting type**: Video, Audio, or Webinar
  - **Permissions**: What actions the participant can take
  - **UI appearance**: Colors, themes, icon packs
- Default presets are auto-created when creating an App via the Dashboard
- Can be created/edited via Dashboard Preset Editor or the Create Preset API

### 2.6 Session Lifecycle (Peer States)

| State | Description |
|-------|-------------|
| `init` | Initial connection state |
| `waitlisted` | Awaiting host approval (waiting room enabled) |
| `joined` | Successfully admitted to the meeting |
| `rejected` | Host denied entry from waiting room |
| `left` | Participant departed voluntarily |
| `kicked` | Removed by host |
| `ended` | Meeting was concluded |
| `disconnected` | Connection lost |

The UI Kit automatically handles state transitions and screen rendering. With the Core SDK, listen to `roomJoined`, `waitlisted`, and `roomLeft` events.

---

## 3. SDK Packages

### 3.1 Comparison

| Aspect | Core SDK | UI Kit |
|--------|----------|--------|
| What it is | Headless client SDK with full APIs | Pre-built component library built atop Core SDK |
| Development speed | 5–6 days | Under 2 hours |
| Bundle size | Minimal | Larger (includes UI components) |
| Customization | Unlimited flexibility | High via component library |
| State management | Manual | Automatic |
| Learning curve | Steeper | More accessible |
| Maintenance | More code | Reduced footprint |

**When to choose Core SDK**: Complete control, headless integration, custom design systems.

**When to choose UI Kit**: Accelerated deployment, standard meeting UI with customization, access to Core SDK APIs remains available.

### 3.2 Packages by Platform

#### Web

| Framework | Core SDK Package | UI Kit Package |
|-----------|-----------------|----------------|
| React | `@cloudflare/realtimekit-react` | `@cloudflare/realtimekit-react-ui` |
| Web Components / HTML / Vue / Svelte | `@cloudflare/realtimekit-web` or `@cloudflare/realtimekit` | `@cloudflare/realtimekit-ui` |
| Angular | `@cloudflare/realtimekit-angular` | `@cloudflare/realtimekit-angular-ui` |

CDN (vanilla): `https://cdn.jsdelivr.net/npm/@cloudflare/realtimekit@latest/dist/browser.js`

#### Mobile

| Platform | Core SDK | UI Package |
|----------|----------|-----------|
| Android | `com.cloudflare.realtimekit:core:1.5.5` | `com.cloudflare.realtimekit:ui-android:0.3.0` |
| iOS | Swift Package Manager: `https://github.com/dyte-in/RealtimeKitCoreiOS.git` | Same repo |
| Flutter | `realtimekit_core` (via `flutter pub add`) | `realtimekit_ui` |
| React Native | `@cloudflare/realtimekit-react-native` + `@cloudflare/react-native-webrtc` | Optional UI package |
| Expo | `expo install` with config plugins for native modules | Optional UI package |

---

## 4. Installation

### 4.1 Prerequisites

- Cloudflare account
- API Token with **Realtime** or **Realtime Admin** permissions (created via Cloudflare Dashboard or API)

### 4.2 iOS / Flutter iOS Configuration

Add to `Info.plist`:
```xml
NSCameraUsageDescription
NSMicrophoneUsageDescription
NSPhotoLibraryUsageDescription
UIBackgroundModes: audio, voip, fetch, remote-notification
```

Minimum iOS version: 13.0 (native), 14.0 (React Native)

### 4.3 Android / Flutter Android Configuration

- Compile SDK: 36, minimum SDK: 24
- Kotlin version: 1.9.0
- Add ProGuard rules for WebRTC libraries
- For screen share on API 14+: declare `FOREGROUND_SERVICE_MEDIA_PROJECTION` permission

### 4.4 React Native (Release Builds)

- Gradle: `newArchEnabled=false`
- ProGuard configuration for WebRTC classes

---

## 5. Backend Setup

The typical backend flow before a participant can join:

1. **Create an App** — via Dashboard or API. Returns an App ID.
2. **Create a Preset** — define participant roles/permissions at the App level.
3. **Create a Meeting** — via REST API (POST to meetings endpoint). Returns a Meeting ID.
4. **Add a Participant** — via REST API (POST to participants endpoint with Meeting ID + Preset ID). Returns `id` and `token`.
5. **Deliver `token` to the client** — the frontend uses this auth token to initialize the SDK.

---

## 6. REST API Reference

Base URL: `https://api.cloudflare.com/client/v4/accounts/{account_id}/`

Authentication: Bearer token in `Authorization` header.

> Note: The official OpenAPI/Swagger spec page at `/realtime/realtimekit/rest-api-reference/` did not return parseable content during extraction. The endpoints below are documented from the conceptual and guide pages.

### 6.1 Apps

#### Create App
```
POST /realtime/apps
```
Creates a new App (workspace). Returns an App ID. Default presets are auto-generated.

#### Get App
```
GET /realtime/apps/{app_id}
```

#### Update App
```
PATCH /realtime/apps/{app_id}
```

#### Delete App
```
DELETE /realtime/apps/{app_id}
```

---

### 6.2 Meetings

#### Create Meeting
```
POST /realtime/apps/{app_id}/meetings
```

**Request body** (JSON):
```json
{
  "title": "string",
  "record_on_start": false,
  "persist_chat": false,
  "ai_config": {
    "language": "en-US",
    "summary_type": "team_meeting"
  }
}
```

**Response**: Includes unique `meeting_id`.

#### Get Meeting
```
GET /realtime/apps/{app_id}/meetings/{meeting_id}
```

#### Update Meeting
```
PATCH /realtime/apps/{app_id}/meetings/{meeting_id}
```
Can set `status: "INACTIVE"` to prevent new participants from joining.

#### Delete Meeting
```
DELETE /realtime/apps/{app_id}/meetings/{meeting_id}
```

#### List Meetings
```
GET /realtime/apps/{app_id}/meetings
```

---

### 6.3 Participants

#### Add Participant
```
POST /realtime/apps/{app_id}/meetings/{meeting_id}/participants
```

**Request body** (JSON):
```json
{
  "name": "string",
  "picture": "string (URL)",
  "preset_name": "string",
  "custom_participant_id": "string"
}
```

**Response**:
```json
{
  "id": "string",
  "token": "string (auth token)"
}
```

> Do **not** re-use auth tokens across participants. `custom_participant_id` should be a stable opaque internal identifier, not personal data (no emails/phone numbers).

#### Get Participant
```
GET /realtime/apps/{app_id}/meetings/{meeting_id}/participants/{participant_id}
```

#### Update Participant
```
PATCH /realtime/apps/{app_id}/meetings/{meeting_id}/participants/{participant_id}
```

#### Delete Participant
```
DELETE /realtime/apps/{app_id}/meetings/{meeting_id}/participants/{participant_id}
```
Prevents the user from joining again. Stop issuing new tokens.

#### Refresh Participant Token
```
POST /realtime/apps/{app_id}/meetings/{meeting_id}/participants/{participant_id}/token
```
Issues a new auth token for the existing participant record. The participant `id` and preset are preserved.

#### List Participants
```
GET /realtime/apps/{app_id}/meetings/{meeting_id}/participants
```

---

### 6.4 Presets

#### Create Preset
```
POST /realtime/apps/{app_id}/presets
```

**Request body** (JSON):
```json
{
  "name": "string",
  "meeting_type": "MEETING | AUDIO_ROOM | WEBINAR",
  "permissions": {
    "can_produce_audio": "ALLOWED | NOT_ALLOWED | CAN_REQUEST",
    "can_produce_video": "ALLOWED | NOT_ALLOWED | CAN_REQUEST",
    "can_produce_screenshare": "ALLOWED | NOT_ALLOWED | CAN_REQUEST",
    "stage_enabled": true,
    "stage_access": "ALLOWED | NOT_ALLOWED | CAN_REQUEST",
    "accept_waiting_requests": true,
    "request_produce_video": true,
    "request_produce_audio": true,
    "request_produce_screenshare": true,
    "can_allow_participant_audio": true,
    "can_allow_participant_screensharing": true,
    "can_allow_participant_video": true,
    "can_disable_participant_audio": true,
    "can_disable_participant_video": true,
    "kick_participant": true,
    "pin_participant": true,
    "can_record": true,
    "waiting_room_behaviour": "SKIP | ON_PRIVILEGED_USER_ENTRY | SKIP_ON_ACCEPT",
    "plugins": {
      "can_start": true,
      "can_close": true
    },
    "polls": {
      "can_create": true,
      "can_vote": true,
      "can_view_results": true
    },
    "chat_public": {
      "can_send": true,
      "text": true,
      "files": true
    },
    "chat_private": {
      "can_send": true,
      "text": true,
      "files": true,
      "can_receive": true
    },
    "hidden_participant": false,
    "show_participant_list": true,
    "can_change_participant_permissions": true,
    "can_livestream": true,
    "transcription_enabled": true
  }
}
```

#### Get Preset
```
GET /realtime/apps/{app_id}/presets/{preset_name}
```

#### Update Preset
```
PATCH /realtime/apps/{app_id}/presets/{preset_name}
```

#### Delete Preset
```
DELETE /realtime/apps/{app_id}/presets/{preset_name}
```

#### List Presets
```
GET /realtime/apps/{app_id}/presets
```

---

### 6.5 Recordings

#### Start Recording
```
POST /realtime/apps/{app_id}/meetings/{meeting_id}/active-session/recording/start
```

#### Stop Recording
```
POST /realtime/apps/{app_id}/meetings/{meeting_id}/active-session/recording/stop
```

#### Fetch Recording Details
```
GET /realtime/apps/{app_id}/meetings/{meeting_id}/sessions/{session_id}/recording
```

**Response includes**:
```json
{
  "chat_download_url": "string",
  "chat_download_url_expiry": "string"
}
```

#### Fetch Active Recording
```
GET /realtime/apps/{app_id}/meetings/{meeting_id}/active-session/recording
```

#### List Recordings
```
GET /realtime/apps/{app_id}/meetings/{meeting_id}/recordings
```

---

### 6.6 AI (Transcription / Summaries)

Enabled at meeting creation via `ai_config`. Transcripts and summaries are available for **7 days** from meeting start.

**Response** (via webhook or REST):
- Presigned R2 URLs for transcript/summary data

---

## 7. Client SDK (Core) API Reference

### 7.1 Initialization

#### Static methods on `RealtimeKitClient`

```typescript
// Initialize media before SDK (media-first approach)
RealtimeKitClient.initMedia(
  options?: { video: boolean; audio: boolean; constraints: MediaConstraints },
  skipAwaits?: boolean,
  cachedUserDetails?: CachedUserDetails
): Promise<void>

// Initialize the SDK (returns client instance)
RealtimeKitClient.init(options: {
  authToken: string;      // required - from Add Participant API
  baseURI?: string;
  defaults?: {
    audio?: boolean;
    video?: boolean;
  };
}): Promise<RealtimeKitClient>
```

**React hook**:
```typescript
const meeting = useRealtimeKitClient();
await meeting.init({ authToken });
```

---

### 7.2 `RealtimeKitClient` Instance

#### Instance Properties

| Property | Type | Description |
|----------|------|-------------|
| `self` | `RTKSelf` | Local participant controls |
| `participants` | `RTKParticipants` | All remote participants |
| `meta` | `RTKMeta` | Meeting metadata |
| `chat` | `RTKChat` | Chat messages |
| `polls` | `RTKPolls` | Polls management |
| `plugins` | `RTKPlugins` | Plugin management |
| `ai` | `RTKAi` | AI features (transcription, summary) |
| `connectedMeetings` | `RTKConnectedMeetings` | Breakout rooms / linked meetings |
| `stage` | `RTKStage` | Stage (webinar) management |
| `recording` | `RTKRecording` | Recording controls |

#### Instance Methods

| Method | Description |
|--------|-------------|
| `join(): void` | Joins the meeting; emits `roomJoined` event on success |
| `leave(): void` | Leaves the meeting |

---

### 7.3 `RTKSelf` (Local Participant)

Access via `meeting.self` (web) or `meeting.localUser` (mobile).

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `peerId` | string | Unique peer identifier |
| `roomState` | string | `init`, `joined`, `waitlisted`, `rejected`, `kicked`, `left`, `ended`, `disconnected` |
| `permissions` | RTKPermissionsPreset | Current participant permissions |
| `config` | object | Meeting configuration |
| `roomJoined` | boolean | Whether user has joined |
| `isPinned` | boolean | Whether current user is pinned |
| `audioEnabled` | boolean | Microphone active |
| `videoEnabled` | boolean | Camera active |
| `screenShareEnabled` | boolean | Screen share active |
| `audioTrack` | MediaStreamTrack | Active audio track |
| `videoTrack` | MediaStreamTrack | Active video track |

#### Methods

| Method | Description |
|--------|-------------|
| `setName(name: string)` | Set display name (call before joining for propagation) |
| `enableAudio()` | Unmute microphone |
| `disableAudio()` | Mute microphone |
| `enableVideo()` | Start camera |
| `disableVideo()` | Stop camera |
| `enableScreenShare()` | Start screen sharing |
| `disableScreenShare()` | Stop screen sharing |
| `updateVideoConstraints()` | Apply constraints to video stream |
| `updateScreenshareConstraints()` | Apply constraints to screenshare |
| `getAllDevices()` | Get all accessible media devices |
| `setDevice(device: MediaDeviceInfo)` | Switch active media device |
| `pin()` | Pin self (if permitted) |
| `unpin()` | Unpin self (if permitted) |
| `hide()` | Hide own tile locally |
| `show()` | Show own tile |
| `cleanupEvents()` | Clean up event listeners |
| `setupTracks(options)` | Initialize audio/video tracks |

#### Events

| Event | Description |
|-------|-------------|
| `roomJoined` | User successfully joined |
| `waitlisted` | User placed in waiting room |
| `roomLeft` | User left/was kicked/meeting ended |
| `audioUpdate` | Audio track state changed |
| `videoUpdate` | Video track state changed |
| `screenShareUpdate` | Screen share state changed |
| `deviceUpdate` | Active device changed |
| `deviceListUpdate` | Available device list changed |
| `mediaPermissionError` | OS permission denied |
| `mediaScoreUpdate` | Network quality update |

---

### 7.4 `RTKParticipants` (Remote Participants)

Access via `meeting.participants`.

#### Properties (Maps)

| Property | Description |
|----------|-------------|
| `joined` | All current meeting participants |
| `waitlisted` | Participants awaiting admission |
| `active` | *(Deprecated)* Participants in grid |
| `pinned` | Pinned participants |
| `videoSubscribed` | Participants with active video consumption |
| `audioSubscribed` | Participants with active audio consumption |
| `all` | All added participants regardless of status |

> Use `joined` map to show all participants. Use `active` for video grid.

#### Properties (Metadata)

| Property | Description |
|----------|-------------|
| `count` | Number of joined participants |
| `lastActiveSpeaker` | ID of most recent speaker |
| `viewMode` | `ACTIVE_GRID` or `PAGINATED` |
| `currentPage` | Current page (PAGINATED mode) |
| `pageCount` | Total pages (PAGINATED mode) |
| `maxActiveRTKParticipantsCount` | Max participants shown in active map |

#### Methods

**Waiting Room:**
```typescript
acceptWaitingRoomRequest(id: string)
acceptAllWaitingRoomRequest(userIds: string[])
rejectWaitingRoomRequest(id: string)
```

**View Control:**
```typescript
setViewMode(viewMode: 'ACTIVE_GRID' | 'PAGINATED')
setPage(page: number)
setMaxActiveRTKParticipantsCount(limit: number)
```

**Subscription:**
```typescript
subscribe(peerIds: string[], kinds?: string[])
unsubscribe(peerIds: string[], kinds?: string[])
```

**Media Host Controls:**
```typescript
disableAllAudio(allowUnmute: boolean)
disableAllVideo()
kickAll()
```

**Broadcast:**
```typescript
broadcastMessage(type: string, payload: object, target?: object)
// Rate limit: 5 invocations/second (client-side default)
// 'spotlight' is a reserved type name
```

**Data Retrieval:**
```typescript
getAllJoinedPeers(searchQuery?: string, limit?: number, offset?: number)
getRTKParticipantsInMeetingPreJoin()
```

---

### 7.5 `RTKParticipant` (Single Remote Participant)

Individual participant from maps like `meeting.participants.joined`.

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Peer ID (for map indexing) |
| `userId` | string | Permanent user identifier |
| `name` | string | Display name |
| `picture` | string | Avatar URL |
| `customRTKParticipantId` | string | Developer-supplied custom ID |
| `presetName` | string | Assigned preset |
| `videoEnabled` | boolean | Camera active |
| `audioEnabled` | boolean | Microphone active |
| `screenShareEnabled` | boolean | Screen share active |
| `videoTrack` | MediaStreamTrack | Video stream |
| `audioTrack` | MediaStreamTrack | Audio stream |
| `screenShareTracks` | object | Screen share video and audio tracks |
| `stageStatus` | string | Current stage participation status |
| `isPinned` | boolean | Pinned state |
| `supportsRemoteControl` | boolean | Remote control capability |

**Two ID types**: `id` (session-specific, changes per connection) vs `userId` (permanent across sessions).

#### Methods

```typescript
// Media control (host, requires permissions)
disableAudio()
disableVideo()
kick()

// Pin/unpin (requires permission)
pin()
unpin()
setIsPinned(isPinned: boolean, emitEvent?: boolean)

// Video rendering
registerVideoElement(videoElem: HTMLVideoElement)
deregisterVideoElement(videoElem?: HTMLVideoElement)

// State setters
setVideoEnabled(videoEnabled: boolean, emitEvent?: boolean)
setAudioEnabled(audioEnabled: boolean, emitEvent?: boolean)
setScreenShareEnabled(screenShareEnabled: boolean, emitEvent?: boolean)
```

---

### 7.6 `RTKMeta` (Meeting Metadata)

Access via `meeting.meta`.

#### Properties

| Property | Platform | Description |
|----------|----------|-------------|
| `meetingTitle` | All | Meeting title |
| `meetingStartedTimestamp` | All | Session start timestamp |
| `meetingType` | All | `GROUP_CALL`, `WEBINAR`, `AUDIO_ROOM`, `LIVESTREAM` |
| `meetingId` | Web | Unique meeting identifier |
| `meetingConfig` | Web | Audio/video configuration |
| `meetingState` | Web | Current meeting state |
| `authToken` | Web | User's auth token |
| `mediaConnectionState` | All | WebRTC connection state |
| `socketConnectionState` | All | WebSocket connection state |

#### Events

| Event | Description |
|-------|-------------|
| `mediaConnectionUpdate` | WebRTC state changed; payload: `{ transport: 'consuming' | 'producing', state: 'new' | 'connecting' | 'connected' | 'disconnected' | 'reconnecting' | 'failed' }` |
| `socketConnectionUpdate` | WebSocket state changed |

---

### 7.7 `RTKChat`

Access via `meeting.chat`.

#### Methods

```typescript
// Sending
sendTextMessage(message: string, peerIds?: string[])
sendImageMessage(image: File, peerIds?: string[])
sendFileMessage(file: File, peerIds?: string[])
sendMessage(message: object, participantIds?: string[])
sendCustomMessage(message: object, peerIds?: string[])

// Editing
editTextMessage(messageId: string, message: string)
editImageMessage(messageId: string, image: File)
editFileMessage(messageId: string, file: File)
editMessage(messageId: string, message: object)

// Deleting
deleteMessage(messageId: string)

// Pinning
pin(id: string)
unpin(id: string)

// Retrieval (paginated)
fetchPublicMessages(options?: object)
fetchPrivateMessages(options?: object)
fetchPinnedMessages(options?: object)

// Configuration
setMaxTextLimit(limit: number)
updateRateLimits(num: number, period: number)
```

#### Message Object Structure

Core fields: `type`, `userId`, `displayName`, `time`, `id`, `isEdited`, `read`, `pluginId`, `pinned`, `targetUserIds`

#### Events

| Event | Description |
|-------|-------------|
| `chatUpdate` | New message received |
| `pinMessage` | Message pinned |
| `unpinMessage` | Message unpinned |

---

### 7.8 `RTKPolls`

Access via `meeting.polls`.

#### Properties

| Property | Description |
|----------|-------------|
| `items` | Array of all Poll objects in the meeting |

#### Poll Object

```typescript
{
  id: string,
  question: string,
  options: PollOption[],
  anonymous: boolean,
  hideVotes: boolean,
  createdBy: string,
  createdByUserId: string,
  voted: string[]  // participant IDs who voted
}
```

#### PollOption Object

```typescript
{
  text: string,
  votes: { id: string, name: string }[],
  count: number
}
```

#### Methods

```typescript
create(question: string, options: string[], anonymous: boolean, hideVotes?: boolean)
vote(pollId: string, optionIndex: number)
```

#### Events

| Event | Payload | Description |
|-------|---------|-------------|
| `pollsUpdate` | `{ polls: Poll[], newPoll: boolean }` | Poll created or updated |

---

### 7.9 `RTKRecording`

Access via `meeting.recording`.

#### Methods

```typescript
start()   // Start recording
stop()    // Stop all recordings in RECORDING state
pause()   // Pause all recordings in RECORDING state
resume()  // Resume all recordings in PAUSED state
```

---

### 7.10 `RTKStage`

Access via `meeting.stage`.

#### Properties

| Property | Description |
|----------|-------------|
| `peerId` | Current user's stage peer ID |

#### Stage Status Values

| Status | Description |
|--------|-------------|
| `ON_STAGE` | User actively streams audio/video |
| `OFF_STAGE` | User watches but doesn't stream |
| `REQUESTED_TO_JOIN_STAGE` | Pending approval |
| `ACCEPTED_TO_JOIN_STAGE` | Host approved; user can join |

#### Methods

```typescript
// Host actions (requires permissions.acceptRTKStageRequests)
grantAccess(userIds: string[])
denyAccess(userIds: string[])
kick(userIds: string[])

// Participant actions
requestAccess()
cancelRequestAccess()
getAccessRequests()

// Join/leave stage
join()
leave()
```

#### Events

| Event | Description |
|-------|-------------|
| `stageRequestUpdate` | Pending requests changed |
| `stageRequestAccepted` | Request approved |
| `stageRequestRejected` | Request denied |
| `stageStatusUpdate` | Participant stage status changed |

---

### 7.11 `RTKAi`

Access via `meeting.ai`.

#### Methods

```typescript
// Instance
onTranscript(transcript: TranscriptionData): void

// Static
RTKAi.parseTranscript(transcriptData: string, isPartialTranscript?: boolean): object
RTKAi.parseTranscripts(transcriptData: string): object[]
```

---

### 7.12 `RTKConnectedMeetings` (Breakout Rooms)

Access via `meeting.connectedMeetings`.

#### Methods

```typescript
getRTKConnectedMeetings(): Promise<object>

createMeetings(request: { title: string }[]): Promise<void>

updateMeetings(request: { id: string, title: string }[]): Promise<void>

deleteMeetings(meetingIds: string[]): Promise<void>

moveParticipants(
  sourceMeetingId: string,
  destinationMeetingId: string,
  participantIds: string[]
): Promise<void>

moveParticipantsWithCustomPreset(
  sourceMeetingId: string,
  destinationMeetingId: string,
  participants: { id: string, presetId: string }[]
): Promise<void>
```

#### Events

| Event | Description |
|-------|-------------|
| `changingMeeting` | Transition in progress |
| `meetingChanged` | Transition complete; provides updated meeting reference |

> Audio/video state persists across breakout room transitions.

---

### 7.13 `RTKPermissionsPreset`

Full permission fields available on `meeting.self.permissions`:

| Field | Type | Values / Description |
|-------|------|---------------------|
| `stageEnabled` | boolean | Stage management available |
| `stageAccess` | string | `ALLOWED`, `NOT_ALLOWED`, `CAN_REQUEST` |
| `acceptWaitingRequests` | boolean | Can accept waiting room entries |
| `requestProduceVideo` | boolean | Can request to produce video |
| `requestProduceAudio` | boolean | Can request to produce audio |
| `requestProduceScreenshare` | boolean | Can request to screen share |
| `canAllowParticipantAudio` | boolean | Can enable others' audio |
| `canAllowParticipantVideo` | boolean | Can enable others' video |
| `canAllowParticipantScreensharing` | boolean | Can enable others' screen share |
| `canDisableParticipantAudio` | boolean | Can disable others' audio |
| `canDisableParticipantVideo` | boolean | Can disable others' video |
| `kickParticipant` | boolean | Can remove participants |
| `pinParticipant` | boolean | Can pin participants |
| `canRecord` | boolean | Can record the meeting |
| `waitingRoomBehaviour` | string | `SKIP`, `ON_PRIVILEGED_USER_ENTRY`, `SKIP_ON_ACCEPT` |
| `plugins.canStart` | boolean | Can start plugins |
| `plugins.canClose` | boolean | Can close plugins |
| `polls.canCreate` | boolean | Can create polls |
| `polls.canVote` | boolean | Can vote on polls |
| `polls.canViewResults` | boolean | Can see poll results |
| `canProduceVideo` | string | `ALLOWED`, `NOT_ALLOWED`, `CAN_REQUEST` |
| `canProduceAudio` | string | `ALLOWED`, `NOT_ALLOWED`, `CAN_REQUEST` |
| `canProduceScreenshare` | string | `ALLOWED`, `NOT_ALLOWED`, `CAN_REQUEST` |
| `chatPublic.canSend` | boolean | Can send public messages |
| `chatPublic.text` | boolean | Can send text |
| `chatPublic.files` | boolean | Can send files |
| `chatPrivate.canSend` | boolean | Can send private messages |
| `chatPrivate.text` | boolean | Can send private text |
| `chatPrivate.files` | boolean | Can send private files |
| `chatPrivate.canReceive` | boolean | Can receive private messages |
| `hiddenParticipant` | boolean | Participant is hidden from others |
| `showParticipantList` | boolean | Participant list visible |
| `canChangeParticipantPermissions` | boolean | Can change others' permissions |
| `canLivestream` | boolean | Can livestream |

---

### 7.14 Broadcast Messages API

Send arbitrary custom messages to participants via the signaling channel.

```typescript
meeting.participants.broadcastMessage(
  type: string,      // message identifier; 'spotlight' is reserved
  payload: {         // custom data
    [key: string]: boolean | number | string | Date
  },
  target?: {
    participantIds?: string[],   // specific participants
    presetNames?: string[],      // participants with specific presets
    meetingIds?: string[]        // cross-meeting broadcast
  }
)
```

**Receive**:
```typescript
meeting.self.on('broadcastedMessage', ({ type, payload, timestamp }) => { ... })
```

**Rate limit**: 5 invocations/second (client-side default).

---

### 7.15 Media Acquisition Approaches

Three approaches for acquiring media tracks:

| Approach | Method | When to Use |
|----------|--------|-------------|
| **SDK-First** (recommended) | `meeting.self.audioTrack` / `meeting.self.videoTrack` after `init()` | Default; most applications |
| **Media-First** | `RealtimeKitClient.initMedia()` before `init()` | Prevent double permission prompts; pre-validation (EdTech proctoring) |
| **Self-Managed** (advanced) | Manage via browser APIs, pass tracks into `enableAudio()` / `enableVideo()` | Custom compliance, third-party media APIs; requires deep WebRTC knowledge |

---

## 8. UI Kit

### 8.1 Overview

The UI Kit delivers a complete meeting experience in a single component. Drop `<rtk-meeting>` (web) or `RtkMeeting` (React) into your app.

### 8.2 Core Component Usage

**React**:
```tsx
import { useRealtimeKitClient, useRealtimeKitMeeting, RealtimeKitProvider } from '@cloudflare/realtimekit-react';
import { RtkMeeting } from '@cloudflare/realtimekit-react-ui';

// Initialize
const meeting = useRealtimeKitClient();
await meeting.init({ authToken });

// Render
<RealtimeKitProvider value={meeting}>
  <RtkMeeting mode="fill" />
</RealtimeKitProvider>
```

**Web Components**:
```html
<rtk-meeting></rtk-meeting>
<script>
  const meeting = await RealtimeKitClient.init({ authToken });
  document.querySelector('rtk-meeting').meeting = meeting;
</script>
```

**Angular**:
```typescript
// Import RTKComponentsModule
// Add <rtk-meeting> to template
// Initialize in component lifecycle
```

### 8.3 Session Lifecycle Handling

The UI Kit auto-handles all state transitions and screens:

| Screen | When Shown |
|--------|-----------|
| Setup Screen | Before joining; preview audio/video |
| Waitlist Screen | If waiting room is enabled |
| Meeting Screen (Stage) | Main interaction area |
| Rejected Screen | Denied from waiting room |
| Ended Screen | Kicked, left, or meeting ended |

Use lifecycle hooks to trigger custom actions at state transitions.

### 8.4 Branding / Customization

- **Icon packs**: Pass `iconPackUrl` to replace the default icon set (hosted at `icons.realtime.cloudflare.com`)
- **Default icons**: 47 icons (mic_on, mic_off, video_on, video_off, call_end, chat, send, attach, etc.)
- **Icon pack format**: JSON object mapping icon name strings to SVG strings
- **Custom UI path**: Component Library → Meeting Lifecycle → Meeting Object API → Full custom build

### 8.5 State Management

The UI Kit handles state automatically. For React, use `useRealtimeKitSelector` to monitor specific properties reactively.

---

## 9. Features

### 9.1 Waiting Room

Controlled via preset `waitingRoomBehaviour`:

| Value | Behavior |
|-------|---------|
| `SKIP` | No waiting room; all join immediately |
| `ON_PRIVILEGED_USER_ENTRY` | All admitted once a privileged user joins |
| `SKIP_ON_ACCEPT` | Host manually accepts each participant |

Host methods: `acceptWaitingRoomRequest(id)`, `rejectWaitingRoomRequest(id)`, `acceptAllWaitingRoomRequest(userIds)`.

Listen for `roomJoined` or `waitlisted` events after calling `meeting.join()`.

### 9.2 Stage (Webinar Mode)

A virtual area where only authorized participants stream media. Others can watch, chat, and use polls.

**Flow**: Participant calls `requestAccess()` → Host calls `grantAccess(userIds)` → Participant calls `join()` on stage.

Direct-permission users (`stageAccess: ALLOWED`) skip the request flow.

### 9.3 Chat

- Supports text messages, images, and files
- Private messaging supported
- Message pinning supported
- Chat replay downloadable as CSV (via `chat_download_url`)
- `persist_chat` meeting config controls cross-session persistence
- Rate limiting configurable via `updateRateLimits()`

### 9.4 Polls

- Create anonymous or named polls with optional hidden votes
- Vote by option index
- Real-time `pollsUpdate` events

### 9.5 Recording

- Composite recording (all participants + plugins merged into one file)
- Recorded by anonymous virtual bot users joining the meeting
- Stored in Cloudflare R2 for **7 days** then deleted
- Transfer to AWS, Azure, or DigitalOcean via Developer Portal or API
- Supported codecs: H.264, VP8
- Configuration: watermark, audio codec, custom storage, interactive metadata
- Monitoring: webhook `recording.statusUpdate`, Fetch Active Recording API, Developer Portal

### 9.6 AI Features

Enable via `ai_config` at meeting creation:

```json
{
  "ai_config": {
    "language": "en-US",
    "summary_type": "team_meeting"
  }
}
```

Participants need `transcription_enabled: true` in their preset.

| Feature | Description |
|---------|-------------|
| **Transcription** | Real-time and post-meeting speech-to-text |
| **Summarization** | AI-generated meeting summaries |

Data stored in R2 with presigned URLs; available for **7 days** from meeting start. Uses Cloudflare Workers AI (Standard model pricing applies separately).

Client-side: `meeting.ai.onTranscript()`, `RTKAi.parseTranscript()`, `RTKAi.parseTranscripts()`.

### 9.7 Breakout Rooms

Beta, web-only. Separate independent sessions within a larger gathering. Each breakout room:
- Can be recorded independently
- Supports chat, polls, audio/video
- Participants can switch back to parent meeting

Permissions involved:
- `Switch Connected Meeting` — move between breakout rooms
- `Switch to Parent Meeting` — return to main meeting
- `Full Access` — host-level control

### 9.8 Plugins

Interactive collaborative apps (Whiteboard, Document Sharing, etc.). Controlled by `plugins.canStart` and `plugins.canClose` preset permissions.

### 9.9 Broadcast Messages

Custom signaling messages to all or targeted participants. Rate-limited to 5/second. Reserved type: `spotlight`.

---

## 10. Error Codes

### 10.1 Web SDK Error Codes

| Range | Module |
|-------|--------|
| 0001–0013 | RealtimeKitClient initialization / networking |
| 0100–0102 | Controller |
| 0200 | RoomNodeClient |
| 0300 | HiveNodeClient |
| 0400, 0404 | SocketService |
| 0500–0510 | Chat |
| 0600–0603 | Plugin |
| 0700, 0705 | Polls |
| 0800–0801 | Meta |
| 0900, 0904 | Preset |
| 1000–1005 | Recording |
| 1100–1106 | Self (local participant) |
| 1200–1209 | Participant |
| 1300 | Spotlight |
| 1500 | Webinar |
| 1601–1611 | LocalMediaHandler |
| 1701 | End-to-End Encryption |
| 1800–1801 | AI |
| 1900–1902 | Livestream |
| 2000–2006 | Stage |
| 9900 | Unknown |

#### Key Error Codes (Web)

| Code | Message | Notes |
|------|---------|-------|
| 0004 | Invalid auth token | Expired or malformed JWT |
| 0010 | Browser not supported | WebRTC incompatibility |
| 0011 | HTTP Network Error | Internet or API request failure |
| 0012 | Websocket Network Error | Connection failure |
| 0013 | Rate Limited | Too many API calls |
| 0101 | Permission denied | Insufficient preset permissions |
| 1004 | Invalid auth token | — |
| 1102 | Device not found | Missing mic/camera |
| 1103 | Failed to access media device | — |
| 1105 | Screen share not supported | — |
| 1202 | Participant not found | — |
| 1206 | Max participants limit reached | — |
| 1605 | Device in use | Accessed by another app |
| 1801 | AI feature not enabled | — |
| 2003 | Stage is full | — |
| 2006 | Stage feature not enabled | — |

### 10.2 Mobile SDK Error Codes

| Range | Module |
|-------|--------|
| 1000–1006 | Meeting initialization/join |
| 2100–2102 | Audio |
| 2200–2203 | Video |
| 2300–2304 | Screen Share |
| 3000–3006 | Participant Controls (host) |
| 4000–4203 | Chat |
| 5000–5005 | Polls |
| 6000–6001 | Plugin |
| 7000–7003 | Recording |
| 8000–8004 | Stage |

---

## 11. Pricing

**Current status**: Beta — **no cost** during beta period.

### Post-GA Pricing (per minute)

| Service | Rate |
|---------|------|
| Audio/Video Participant | $0.002/min |
| Audio-Only Participant | $0.0005/min |
| Recording / RTMP / HLS (video) | $0.010/min |
| Recording / RTMP / HLS (audio-only) | $0.003/min |
| Raw RTP export to R2 | $0.0005/min |
| Real-time Transcription | Workers AI Standard model pricing |

**Note**: Audio-only vs. audio/video classification is determined by the **Meeting Type** in the participant's preset.

---

## 12. FAQ and Limitations

### Meetings

- **No built-in scheduling**: Implement in your app by storing Meeting IDs with timestamps; check time before allowing `addParticipant`.
- **Prevent late joins**: Set meeting status to `INACTIVE` via PATCH to the Update Meeting endpoint.
- **One active session at a time** per meeting.

### Participants

- **Same user, multiple devices**: One participant can have multiple peer connections from different devices/tabs.
- **Prevent rejoin**: Delete participant via API and stop issuing tokens.
- **Token reuse**: Never re-use auth tokens across participants; always refresh via the Refresh Token endpoint.
- **New participant per session**: Not required — create once per user/meeting, refresh tokens as needed.
- **`custom_participant_id`**: Use opaque stable IDs (numeric ID, UUID); **not** email or phone.

### Presets

- **Reusable**: One preset can apply to many participants across many meetings.
- **No per-meeting presets**: Presets are App-level.

### Camera / Video

- Max quality: 1080p at 15fps (configurable during initialization; higher quality = more bandwidth).
- Custom frame rate configurable; ≤30 FPS recommended for group calls.

### Screen Share

- Default: 5 FPS; ≤30 FPS recommended.

### Microphone

- Auto-selection prefers Bluetooth devices and devices labeled "bluetooth", "headset", "earphone", or "microphone".
- Pre-plugged devices without these labels may not be auto-selected; users can manually select.

### Chat

- Cannot embed Cloudflare demo app as an iframe — set up your own UI Kit instance.
- To troubleshoot: verify meeting join success, check preset permissions, review custom UI implementations.

### Recordings

- Retained for **7 days** in Cloudflare R2; then deleted automatically.
- Transfer to external storage (AWS, Azure, DigitalOcean) via Developer Portal or API.

### AI / Transcription

- Data available for **7 days** from meeting start.
- Uses Cloudflare Workers AI; separate Workers AI pricing applies.
- Requires `transcription_enabled: true` in participant preset.

### Breakout Rooms

- Beta, **web-only** currently.

### Broadcast Messages

- Rate limit: **5 messages/second** (client-side default, configurable server-side).
- Reserved type: `spotlight`.

---

*Document compiled from: https://developers.cloudflare.com/realtime/realtimekit/ and sub-pages.*
