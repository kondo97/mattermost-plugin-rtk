# Frontend Components — Unit 5: Admin & Config

## Overview

Unit 5 admin UI is implemented entirely via `plugin.json` `settings_schema`. No custom React components are registered.

---

## plugin.json settings_schema Structure

```
Settings Schema
├── CloudflareOrgID       (type: "text")
├── CloudflareAPIKey      (type: "text", secret: true)   ← masked by Mattermost
├── RecordingEnabled      (type: "bool", default: "true")
├── ScreenShareEnabled    (type: "bool", default: "true")
├── PollsEnabled          (type: "bool", default: "true")
├── TranscriptionEnabled  (type: "bool", default: "true")
├── WaitingRoomEnabled    (type: "bool", default: "true")
├── VideoEnabled          (type: "bool", default: "true")
├── ChatEnabled           (type: "bool", default: "true")
├── PluginsEnabled        (type: "bool", default: "true")
├── ParticipantsEnabled   (type: "bool", default: "true")
└── RaiseHandEnabled      (type: "bool", default: "true")
```

---

## API Key Masking — No Custom Component Required

`CloudflareAPIKey` uses `"secret": true` in `plugin.json`. Mattermost's built-in secret field behavior:

- **Storage**: Value is stored encrypted in the Mattermost server config
- **Display**: Field is always shown empty in the System Console (value never sent to browser)
- **Change**: Admin must type the new key to update it (cannot see or copy the existing value)

This satisfies the requirement (Q4: always `********`, re-enter to change) without any React component.

**Note**: The previously considered "Option 2" (minimal `<input type="password">` custom component) is not needed because `"secret": true` already provides equivalent behavior via the Mattermost framework.

---

## Feature Flag Toggle Layout

Each feature flag is rendered as a standard Mattermost System Console toggle row:

```
[Label]           [help_text]                          [toggle ON/OFF]
```

Layout: one flag per row, 10 rows total (Q6: A — individual rows with label and description).

### Feature Flag Display Names and Help Texts

| Key | Display Name | Help Text |
|---|---|---|
| `RecordingEnabled` | Enable Recording | Allow participants to record calls. |
| `ScreenShareEnabled` | Enable Screen Share | Allow participants to share their screen during calls. |
| `PollsEnabled` | Enable Polls | Allow participants to create and respond to polls during calls. |
| `TranscriptionEnabled` | Enable Transcription | Enable real-time transcription for calls. |
| `WaitingRoomEnabled` | Enable Waiting Room | Require host approval before participants can join a call. |
| `VideoEnabled` | Enable Video | Allow participants to enable their camera during calls. |
| `ChatEnabled` | Enable In-Call Chat | Allow participants to send chat messages within a call. |
| `PluginsEnabled` | Enable Plugins | Allow third-party plugins within the call experience. |
| `ParticipantsEnabled` | Enable Participants Panel | Show the participants list panel during calls. |
| `RaiseHandEnabled` | Enable Raise Hand | Allow participants to raise their hand during calls. |

---

## Env Var Indicator

Env var overrides are **not indicated in the System Console UI** for this unit. When a field is overridden by an env var, the System Console continues to show the stored config value (which may differ from the effective value). Administrators should consult the plugin documentation to understand env var override behavior.

This is intentional given the low UI requirement stated by the user.
