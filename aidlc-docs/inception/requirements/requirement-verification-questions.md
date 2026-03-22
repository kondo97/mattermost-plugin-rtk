# Requirements Clarification Questions

Please answer each question by filling in the letter choice after the `[Answer]:` tag.
If none of the options match your needs, choose the last option (Other) and describe your preference.
Let me know when you are done.

---

## Question 1
What is the primary goal of this plugin? What problem does it solve or what value does it provide to Mattermost users?

A) Real-time collaboration tool (e.g., shared whiteboard, live document editing)
B) Notification or alerting system (e.g., integrating external events into Mattermost channels)
C) Task or project management within Mattermost
D) Integration with an external service or API (e.g., Jira, GitHub, Slack)
E) Other (please describe after [Answer]: tag below)

[Answer]: E
Video calling within Mattermost channels powered by Cloudflare RealtimeKit. Users can start and join video/audio calls directly from a channel, with features such as screensharing, chat, polls, and participant management provided by the RTK SDK.

---

## Question 2
Who are the primary users of this plugin?

A) All Mattermost users in the workspace
B) Specific teams or departments only
C) Mattermost administrators only
D) Both end users and administrators (different feature sets for each)
E) Other (please describe after [Answer]: tag below)

[Answer]: D
End users start and join calls from channels. Administrators configure Cloudflare credentials (Org ID and API Key) and toggle feature flags (screenshare, chat, polls, plugins, participants) via the System Console.

---

## Question 3
What is the expected scope of changes to this starter template?

A) Minor customization — rename the plugin and add a few features on top of the existing boilerplate
B) Moderate extension — implement significant new features while keeping the existing structure
C) Major rewrite — replace most of the boilerplate with custom business logic
D) Other (please describe after [Answer]: tag below)

[Answer]: C
Most of the boilerplate will be replaced. The server gains RTK API integration, KVStore session management, a standalone call page, and a Web Worker endpoint. The webapp gains a channel header call button, call modal, standalone call page, custom post type (call invite), and admin settings components.

---

## Question 4
Does the plugin need to store and retrieve persistent data per user or per channel?

A) Yes, per-user data (user preferences, user-specific state)
B) Yes, per-channel data (channel-specific configuration or state)
C) Yes, both per-user and per-channel data
D) No persistent data needed
E) Other (please describe after [Answer]: tag below)

[Answer]: E
Call session data stored per-channel (key: `call:channel:{channelID}`) and per-call-ID (key: `call:id:{callID}`). Fields: call_id, meeting_id, creator_id, created_at. No user-specific persistent data.

---

## Question 5
Does the plugin need a user interface (UI) rendered within Mattermost?

A) Yes — custom UI components in channels or sidebars (webapp required)
B) Yes — configuration/settings UI only (admin panel)
C) No — server-side only (slash commands and API endpoints are sufficient)
D) Both A and B
E) Other (please describe after [Answer]: tag below)

[Answer]: E
Multiple UI surfaces required:
- Channel header button to start a call
- Floating/fullscreen call modal (draggable, minimizable)
- Standalone call page served as a popup window (/plugins/{id}/call)
- Custom post type rendering call invitations with a Join button
- Admin console settings for Cloudflare credentials and feature flags

---

## Question 6
Does the plugin need to integrate with any external services or APIs?

A) Yes — a specific third-party API or service (please describe in [Answer])
B) Yes — internal company systems or databases
C) No — standalone plugin only
D) Other (please describe after [Answer]: tag below)

[Answer]: A
Cloudflare RealtimeKit API (https://api.realtime.cloudflare.com/v2).
- POST /meetings — create a meeting session
- POST /meetings/{id}/participants — add participant and obtain auth token
- POST /presets — create host/participant presets (auto-created if missing)
Authentication: HTTP Basic Auth (orgID:apiKey).

---

## Question 7
What are the performance expectations?

A) Low traffic — used by a small team (< 50 users), no strict performance requirements
B) Medium traffic — used across the organization (50–500 users), reasonable response times
C) High traffic — large-scale deployment (500+ users), performance is critical
D) Other (please describe after [Answer]: tag below)

[Answer]: A
Initial target is small teams. The actual media traffic is handled by Cloudflare infrastructure; the plugin only manages signaling and session metadata.

---

## Question 8
Are there any specific security or compliance requirements?

A) Standard Mattermost security (authentication via Mattermost-User-ID is sufficient)
B) Additional authorization (role-based access control, team/channel restrictions)
C) Compliance requirements (audit logging, data retention policies)
D) No special security requirements beyond the default
E) Other (please describe after [Answer]: tag below)

[Answer]: E
- All API endpoints require Mattermost-User-ID header (standard auth).
- Admin-only endpoint for credential status.
- Cloudflare credentials configurable via environment variables (RTK_ORG_ID, RTK_API_KEY) to avoid storing secrets only in the admin UI.
- RTK token (JWT) acts as the authorization proof for joining a call.
- Web Worker and standalone call page served unauthenticated (RTK token in URL is the real auth mechanism).

---

## Question 9 — Security Extension
Should security extension rules (SECURITY-01 through SECURITY-15) be enforced for this project?
These rules enforce encryption, access control, input validation, logging, and other security best practices as hard constraints.

A) Yes — enforce all SECURITY rules as blocking constraints (recommended for production-grade applications)
B) No — skip all SECURITY rules (suitable for PoCs, prototypes, and experimental projects)
X) Other (please describe after [Answer]: tag below)

[Answer]: A
This is a production-grade application. All SECURITY rules (SECURITY-01 through SECURITY-15) shall be enforced as blocking constraints.
