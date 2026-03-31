# Integration Test Instructions

## Purpose

Verify that the plugin's units interact correctly when deployed as a whole on a real Mattermost server.

> Note: These are manual integration tests. A live Mattermost instance and valid Cloudflare RealtimeKit credentials are required.

---

## Prerequisites

- Mattermost Server 10.11.0+ running locally or in a test environment
- Cloudflare Organization ID and API Key (RealtimeKit account)
- Plugin bundle built: `dist/com.kondo97.mattermost-plugin-rtk-{version}.tar.gz`
- Mattermost admin access
- At least 2 test user accounts

---

## Setup

### 1. Deploy Plugin to Mattermost

```bash
# Build the plugin bundle
make dist

# Deploy via pluginctl (requires MM_SERVICESETTINGS_SITEURL and MM_ADMIN_TOKEN set)
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=<password>
make deploy
```

Or manually: System Console → Plugin Management → Upload Plugin → select `dist/*.tar.gz`

### 2. Configure Plugin

In Mattermost System Console → Plugins → Mattermost RTK Plugin:
- Set **Cloudflare Organization ID**
- Set **Cloudflare API Key**
- Enable desired features (Recording, Screen Share, etc.)
- Click **Save**

### 3. Enable Plugin

```bash
./build/bin/pluginctl enable com.kondo97.mattermost-plugin-rtk
```

---

## Integration Test Scenarios

### Scenario 1: Unit 1 + Unit 2 — Call Lifecycle (Backend)

**Tests**: RTK token generation → KVStore persistence → REST API

**Steps**:
1. As User A, POST to `/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/calls/join`
   with body `{"channel_id": "<channel_id>"}`
2. Verify response contains `rtk_token` (non-empty string)
3. Verify call state persisted: GET `/api/v1/calls/<channel_id>` returns active call
4. As User A, POST `/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/calls/leave`
5. Verify call state cleared

**Expected**: Token issued, call state transitions correctly, post created on join/leave.

---

### Scenario 2: Unit 2 + Unit 3 — WebSocket + Channel Header Button

**Tests**: WebSocket event propagation → UI state update

**Steps**:
1. Open Mattermost in browser as User A
2. Navigate to a channel
3. Verify the Call button appears in the channel header
4. User B joins a call in the same channel via API
5. Verify User A's browser receives `custom_com.kondo97.mattermost-plugin-rtk_call_started` WebSocket event
6. Verify the channel header button shows "active call" state (participant count badge)

**Expected**: WebSocket event delivered within 2 seconds; UI updates without page refresh.

---

### Scenario 3: Unit 3 + Unit 4 — Channel Header → Call Page Navigation

**Tests**: Call button click → call page rendering with RTK session

**Steps**:
1. As User A, click the Call button in the channel header
2. Verify `/call/<channel_id>` route loads (or modal opens)
3. Verify RTK `<RealtimeKitProvider>` renders without error
4. Verify user media permissions are requested (camera/microphone)
5. Verify call controls (mute, camera, leave) are visible and functional

**Expected**: Call page loads, RTK session connects to Cloudflare within 5 seconds.

---

### Scenario 4: Unit 2 + Unit 5 — Config API → Feature Flags in Call

**Tests**: Admin config changes → propagated to call UI

**Steps**:
1. As admin, PUT `/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/config`
   with `{"RecordingEnabled": false}`
2. Verify 200 OK response
3. Start a new call as User A
4. Verify the Recording button is absent from call controls

**Expected**: Config change takes effect for new calls immediately.

---

### ~~Scenario 5: Unit 6 — Mobile Push Notification~~ — REMOVED

> **Updated 2026-03-31**: Push notification subsystem removed. Mobile clients receive call notifications via WebSocket events (`custom_cf_call_started`, `custom_cf_call_ended`). No push-specific integration test is needed.

---

### Scenario 6: End-to-End — Full Call Session (2 Users)

**Steps**:
1. User A starts a call in Channel X (click button or `/call` command)
2. Verify `call_started` post appears in Channel X
3. User B joins the call via the channel header button
4. Verify both users appear in the participants list
5. User A shares screen → verify `screen_share_started` event fires
6. User A stops sharing → verify `screen_share_stopped` event fires
7. User B leaves the call
8. User A ends the call
9. Verify `call_ended` post appears in Channel X with duration

**Expected**: All events processed correctly; call post summary accurate.

---

## Cleanup

```bash
# Disable and reset plugin
./build/bin/pluginctl disable com.kondo97.mattermost-plugin-rtk
./build/bin/pluginctl reset com.kondo97.mattermost-plugin-rtk

# View plugin logs for debugging
./build/bin/pluginctl logs com.kondo97.mattermost-plugin-rtk
```

## Log Locations

- Plugin logs: Mattermost server logs (filter by `plugin_id: com.kondo97.mattermost-plugin-rtk`)
- Browser console: WebSocket events and React errors
- Mattermost System Console → Logs
