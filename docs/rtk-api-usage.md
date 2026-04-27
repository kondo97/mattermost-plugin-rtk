# Cloudflare RealtimeKit (RTK) API Usage Reference

This document is a comprehensive inventory of all Cloudflare RealtimeKit (RTK) APIs used by this plugin.
Its purpose is to provide a single reference point for quickly identifying impacted areas when RTK API specifications change.

---

## 1. Server-side — Cloudflare RTK REST API

Implementation files: `server/rtkclient/client.go`, `server/rtkclient/account_client.go`

Authentication: `Authorization: Bearer {apiToken}` header

### 1-1. App-level API

Base URL: `https://api.cloudflare.com/client/v4/accounts/{accountID}/realtime/kit/{appID}`

#### CreateMeeting

| Field | Value |
|-------|-------|
| Method | `POST` |
| Path | `/meetings` |
| Request body | `{"title": ""}` |
| Response | `{"success": true, "data": {"id": "<meetingID>"}}` |
| Success status | 200 / 201 |
| Used in | `server/rtkclient/client.go` → `CreateMeeting()` |

#### GenerateToken (Add Participant)

| Field | Value |
|-------|-------|
| Method | `POST` |
| Path | `/meetings/{meetingID}/participants` |
| Request body | `{"name": "<displayName>", "preset_name": "<preset>", "custom_participant_id": "<userID>"}` |
| Response | `{"success": true, "data": {"id": "<participantID>", "token": "<authToken>"}}` |
| Success status | 200 / 201 |
| Used in | `server/rtkclient/client.go` → `GenerateToken()` |

#### GetMeeting

| Field | Value |
|-------|-------|
| Method | `GET` |
| Path | `/meetings/{meetingID}` |
| Request body | none |
| Response | `{"success": true, "data": {"id": "<meetingID>", ...}}` |
| Success status | 200 |
| Error status | 404 → `ErrMeetingNotFound` |
| Used in | `server/rtkclient/client.go` → `GetMeeting()` |
| Notes | Used purely as a meeting-existence probe by `app/calls.go` (CreateCall stale-check, channel-meeting reuse, JoinCall, ReconcileCallOnDemand). A 200 means the Meeting resource exists; it does **not** imply that any participant is currently connected. Other non-2xx / network failures are treated as transient by callers. |

#### RegisterWebhook

| Field | Value |
|-------|-------|
| Method | `POST` |
| Path | `/webhooks` |
| Request body | `{"name": "mattermost-plugin-rtk", "url": "<webhookURL>", "events": ["<event1>", ...]}` |
| Response | `{"success": true, "data": {"id": "<webhookID>", "name": "...", "url": "...", "events": [...], "organization_id": "...", "enabled": true, "created_at": "...", "updated_at": "..."}}` |
| Success status | 200 / 201 |
| Error status | 409 → `ErrWebhookConflict` |
| Used in | `server/rtkclient/client.go` → `RegisterWebhook()` |

#### GetWebhook

| Field | Value |
|-------|-------|
| Method | `GET` |
| Path | `/webhooks/{webhookID}` |
| Request body | none |
| Response | `{"success": true, "data": {"id": "<webhookID>", "url": "<url>"}}` |
| Success status | 200 |
| Error status | 404 → `ErrWebhookNotFound` |
| Used in | `server/rtkclient/client.go` → `GetWebhook()` |

#### ListWebhooks

| Field | Value |
|-------|-------|
| Method | `GET` |
| Path | `/webhooks` |
| Request body | none |
| Response | `{"success": true, "data": [{"id": "...", "url": "..."}]}` |
| Success status | 200 |
| Used in | `server/rtkclient/client.go` → `ListWebhooks()` |

#### DeleteWebhook

| Field | Value |
|-------|-------|
| Method | `DELETE` |
| Path | `/webhooks/{webhookID}` |
| Request body | none |
| Response | none (empty body) |
| Success status | 200 / 204 |
| Used in | `server/rtkclient/client.go` → `DeleteWebhook()` |

---

### 1-2. Account-level API

Base URL: `https://api.cloudflare.com/client/v4/accounts/{accountID}/realtime/kit`

#### CreateApp

| Field | Value |
|-------|-------|
| Method | `POST` |
| Path | `/apps` |
| Request body | `{"name": "<appName>"}` |
| Response | `{"success": true, "data": {"app": {"id": "<appID>", "name": "<appName>"}}}` |
| Success status | 200 / 201 |
| Used in | `server/rtkclient/account_client.go` → `CreateApp()` |

#### ListApps

| Field | Value |
|-------|-------|
| Method | `GET` |
| Path | `/apps` |
| Request body | none |
| Response | `{"success": true, "data": [{"id": "...", "name": "..."}]}` |
| Success status | 200 |
| Used in | `server/rtkclient/account_client.go` → `ListApps()` |

---

## 2. Client-side — npm SDK

Implementation files: `webapp/src/call_page/CallPage.tsx`, `webapp/src/components/floating_widget/index.tsx`, `webapp/src/utils/rtk_lang_ja.ts`

---

### 2-1. `@cloudflare/realtimekit-react` (v1.2.5)

#### `useRealtimeKitClient()`

| Field | Value |
|-------|-------|
| Kind | React Hook |
| Returns | `[meeting: DyteClient \| null, initMeeting: (config) => Promise<DyteClient>]` |
| Used in | `CallPage.tsx`, `floating_widget/index.tsx` |

`initMeeting` call signature:
```ts
initMeeting({
  authToken: string,  // JWT obtained from GenerateToken
  defaults: { audio: boolean },
})
```

#### `RealtimeKitProvider`

| Field | Value |
|-------|-------|
| Kind | React Context Provider component |
| Props | `value: DyteClient`, `fallback?: ReactNode` |
| Used in | `CallPage.tsx`, `floating_widget/index.tsx` |

---

### 2-2. `@cloudflare/realtimekit-react-ui` (v1.1.1)

#### `RtkMeeting`

| Field | Value |
|-------|-------|
| Kind | React component |
| Props | See below |
| Used in | `CallPage.tsx`, `floating_widget/index.tsx` |

```ts
<RtkMeeting
  meeting={meeting}           // DyteClient instance
  t={rtkT}                    // Translation function returned by useLanguage()
  mode="fill"                 // Layout mode
  showSetupScreen={boolean}   // Whether to show the pre-join setup screen
  style={CSSProperties}       // Optional inline styles
/>
```

---

### 2-3. `@cloudflare/realtimekit-ui` (v1.1.1)

#### `useLanguage(dict?)`

| Field | Value |
|-------|-------|
| Kind | React Hook |
| Argument | `dict?: Partial<LangDict>` (omit to use the SDK's default English strings) |
| Returns | Translation function `t` (passed to the `t` prop of `RtkMeeting`) |
| Used in | `CallPage.tsx`, `floating_widget/index.tsx` |

#### `LangDict` (type)

| Field | Value |
|-------|-------|
| Kind | TypeScript type |
| Purpose | Type annotation for the Japanese dictionary `jaDict` (`Partial<LangDict>`) |
| Used in | `webapp/src/utils/rtk_lang_ja.ts` |

---

### 2-4. Meeting Object API (`@cloudflare/realtimekit` v1.2.5)

Methods and events called on the `meeting` object (`DyteClient`) returned by `useRealtimeKitClient()`.

#### `meeting.self` events

| Event name | Description | Used in |
|-----------|-------------|---------|
| `roomJoined` | Fired when the local participant has joined the room | `floating_widget/index.tsx` |
| `roomLeft` | Fired when the local participant has left the room | `floating_widget/index.tsx` |

```ts
meeting.self.on('roomJoined', handler)
meeting.self.off('roomJoined', handler)
meeting.self.on('roomLeft', handler)
meeting.self.off('roomLeft', handler)
```

#### `meeting.meta` events

> ⚠️ **Warning**: `meeting.meta` is not part of the official TypeScript type definitions and is accessed via the unsafe cast `(meeting as any).meta`. Type checking will not catch breaking changes here — manual verification is required on every RTK upgrade.

| Event name | Description | Used in |
|-----------|-------------|---------|
| `mediaConnectionUpdate` | Fired when the media connection state changes | `floating_widget/index.tsx` |

```ts
// unsafe cast — not type-safe
(meeting as any).meta?.on('mediaConnectionUpdate', handler)
(meeting as any).meta?.off('mediaConnectionUpdate', handler)
```

#### `meeting.leaveRoom()`

| Field | Value |
|-------|-------|
| Kind | Method |
| Returns | `Promise<void>` |
| Purpose | Leave the current room, triggered by user action or a call-ended event |
| Used in | `floating_widget/index.tsx` |

---

## 3. Version Information

| Package | Version |
|---------|---------|
| `@cloudflare/realtimekit` | `1.2.5` |
| `@cloudflare/realtimekit-react` | `1.2.5` |
| `@cloudflare/realtimekit-react-ui` | `1.1.1` |
| `@cloudflare/realtimekit-ui` | `1.1.1` (peer dependency of realtimekit-react-ui) |

Versions are managed in `webapp/package.json` and `webapp/package-lock.json`.

---

## 4. Checklist for Detecting API Spec Changes

Areas to review when RTK releases a new version or publishes breaking-change notices.

### Server-side
| Type of change | Files to check |
|----------------|---------------|
| Endpoint URL changes | Base URL and paths in `server/rtkclient/client.go`, `server/rtkclient/account_client.go` |
| Request / response JSON field changes | `*Request` and `*Data` structs in each file |
| Authentication scheme changes | `doRequest()` header setup |
| HTTP status code changes | Status code comparisons in each method |

### Client-side
| Type of change | Files to check |
|----------------|---------------|
| `useRealtimeKitClient` return value changes | `CallPage.tsx`, `floating_widget/index.tsx` |
| `initMeeting` argument changes | `CallPage.tsx`, `floating_widget/index.tsx` |
| `RtkMeeting` prop changes | `CallPage.tsx`, `floating_widget/index.tsx` |
| `useLanguage` / `LangDict` changes | `utils/rtk_lang_ja.ts` |
| `meeting.self` event name changes | `floating_widget/index.tsx` |
| `meeting.meta` changes | `floating_widget/index.tsx` (extra care needed — unsafe cast) |
| `meeting.leaveRoom()` signature changes | `floating_widget/index.tsx` |
