# Personas

## Overview

Three personas are defined for `mattermost-plugin-rtk`. The **Channel Member (Web)** and **Mobile User** share some journeys; mobile-specific stories are owned by the Mobile User persona, while shared channel interactions include mobile variant notes.

---

## Persona 1: Channel Member (Web)

**Name**: Channel Member (Web)
**Role**: Regular Mattermost workspace user accessing via browser or desktop app

### Goals
- Start a video/audio call with channel members without leaving the Mattermost interface.
- Join an ongoing call quickly from the channel header or post card.
- See who is currently in a call and for how long.
- Leave or end a call cleanly without disrupting other participants.

### Context
- Uses Mattermost on a web browser or Electron desktop app.
- May be in public channels, private channels, DMs, or group DMs.
- Expects call UX consistent with the Mattermost Calls plugin they may have used before.
- Opens the call UI in a new browser tab; does not expect a floating overlay.

### Pain Points
- Accidentally starting a second call in a channel that already has an active call.
- Not knowing if they are already in a call when trying to join another.
- Confusion about call state when the post card does not update in real-time.

### Technical Characteristics
- Authenticated via Mattermost session; user ID available in all API calls.
- Has access to the channel header, message input area, and post rendering.

---

## Persona 2: Mobile User

**Name**: Mobile User
**Role**: Mattermost workspace user on a modified Mattermost Mobile app with native call support

### Goals
- Receive an incoming call notification natively when someone starts a call in a DM or channel.
- Join a call from the mobile app without switching to a browser.
- Dismiss an incoming call notification and have the ring stop for all other clients.

### Context
- Uses a Mattermost Mobile app that has been modified to support RTK calls.
- Relies on Mattermost push notification infrastructure for incoming call alerts.
- Native call UI is rendered within the mobile app (not a WebView).
- VoIP device token must be registered with the plugin so push can be delivered.

### Pain Points
- Missing incoming call notifications if the VoIP token is not registered.
- Receiving persistent ring notifications after dismissing the call.
- Feature flags from the server not being reflected in the mobile call UI.

### Technical Characteristics
- Registers a VoIP push token per device via plugin API.
- Joins calls using the same `POST /api/v1/calls/{callId}/token` endpoint as web clients.
- Receives feature flag values in the join response to configure the native call UI.

---

## Persona 3: Mattermost Admin (Minimal)

**Name**: Mattermost Admin
**Role**: Workspace administrator responsible for plugin configuration

### Goals
- Enter Cloudflare RTK credentials (Organization ID and API Key) in the System Console.
- Enable or disable individual call features (screen share, recording, AI transcription, etc.) globally.
- Understand whether credentials are sourced from admin UI or environment variables.

### Context
- Accesses the Mattermost System Console via an admin account.
- May use environment variables to override settings in production (CI/CD or secrets management).
- Is not involved in individual call sessions.

### Pain Points
- Unclear whether a credential field is active or overridden by an environment variable.
- No feedback on whether RTK credentials are valid until a call is actually attempted.

### Technical Characteristics
- Admin role verified server-side on all admin-only API endpoints.
- Environment variable values take precedence over System Console values and render fields as read-only.
