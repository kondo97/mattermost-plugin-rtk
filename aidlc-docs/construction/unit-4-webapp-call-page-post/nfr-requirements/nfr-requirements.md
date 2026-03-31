# Unit 4: Webapp - Call Page & Post — NFR Requirements

## Performance

| ID | Requirement | Target |
|---|---|---|
| PERF-U4-01 | `call.js` bundle size | Must be kept reasonable; UI Kit + React bundled together. Tree-shaking via Vite/Rollup; no explicit size limit defined for MVP but avoid unnecessary dependencies |
| PERF-U4-02 | `main.js` bundle size impact | Adding CallPost registration adds negligible overhead; React/Redux already externalized |
| PERF-U4-03 | ~~Heartbeat API overhead~~ | **Deferred / not implemented** — heartbeat mechanism replaced by RTK webhook |
| PERF-U4-04 | Call page initialization time | RTK SDK initializes on `authToken` receipt; no additional round-trips required beyond the initial token |
| PERF-U4-05 | CallPost re-render scope | Must use `selectCallByChannel(channelId)` from Unit 3; scoped selector prevents re-render on unrelated channel state changes |

---

## Reliability

| ID | Requirement | Decision |
|---|---|---|
| REL-U4-01 | ~~Heartbeat failure handling~~ | **Deferred / not implemented** — heartbeat mechanism replaced by RTK webhook. |
| REL-U4-02 | fetch+keepalive reliability | **Updated 2026-03-30**: `fetch` with `keepalive: true` is used (not `navigator.sendBeacon`) to allow custom `X-Requested-With` header while surviving tab close. If fetch+keepalive fails, the RTK webhook handles cleanup. |
| REL-U4-03 | ~~heartbeat interval cleanup~~ | **Deferred / not implemented** — no heartbeat interval exists. |
| REL-U4-04 | beforeunload handler cleanup | `window.removeEventListener('beforeunload', handler)` MUST be called in `useEffect` cleanup to prevent duplicate handlers on re-mount |
| REL-U4-05 | CallPost render with missing Redux data | If `selectCallByChannel(channelId)` returns `undefined` (before first WS event), CallPost MUST fall back to `post.props` data; MUST NOT throw or render empty |
| REL-U4-06 | Call page missing token | If `token` is absent from URL params, CallPage renders an error screen ("Missing call token") instead of attempting RTK initialization |
| REL-U4-07 | RTK SDK initialization error | If `initMeeting()` fails, CallPage displays a user-friendly error screen; no crash or unhandled rejection |
| REL-U4-08 | Vite build reproducibility | `call.js` output MUST be deterministic; Makefile copy step MUST always use the freshly built file |

---

## Security

| ID | Requirement | Source |
|---|---|---|
| SEC-U4-01 | Token in URL — no logging | The `token` URL parameter MUST NOT be logged in `console.log`, `console.error`, or any other output on the call page. `call_id` and `channel_name` MAY be logged. | Carry-over SEC-U3-01 |
| SEC-U4-02 | CSP — call page | The existing CSP on `call.html` is `default-src 'self'; connect-src *`. The RTK UI Kit uses CSS-in-JS which requires `style-src 'unsafe-inline'`. The CSP in `api_static.go` MUST be updated to: `default-src 'self'; connect-src *; style-src 'self' 'unsafe-inline'`. |
| SEC-U4-03 | Token URL exposure | The RTK JWT in the URL will appear in browser history and server access logs. This is an accepted risk for MVP (same pattern used by other RTK-based applications). The token is short-lived (1 hour per Unit 2 design). |
| SEC-U4-04 | fetch+keepalive authentication | **Updated 2026-03-30**: Implementation uses `fetch` with `keepalive: true` and custom `X-Requested-With` header (not `sendBeacon`). The `/calls/{id}/leave` endpoint authenticates via Mattermost session cookie (same-origin). |
| SEC-U4-05 | No inline secrets in call.html | `call.html` MUST NOT contain tokens or credentials inline. All secrets pass via URL params to the JavaScript bundle only. |
| SEC-U4-06 | `channel_name` URL encoding | `channel_name` MUST be `encodeURIComponent`-encoded before appending to the URL to prevent URL injection. |

---

## Usability

| ID | Requirement | Source |
|---|---|---|
| USE-U4-01 | Call page loading state | While RTK SDK initializes, display "Connecting..." with a visible loading indicator; never show a blank white screen |
| USE-U4-02 | Call page error state | If `token` is missing, show a clear error message with a "Close tab" affordance; text must be i18n-free (call page has no Mattermost i18n) |
| USE-U4-03 | CallPost `data-testid` attributes | All interactive elements (join button, status indicator, avatar list) MUST have `data-testid` attributes |
| USE-U4-04 | Tab title reflects channel | Browser tab title MUST be `'Call in #${channelName}'` for a good multi-tab experience |
| USE-U4-05 | CallPost join button tooltip | When the join button is disabled (`myActiveCall?.callId === post.call_id`), show a tooltip explaining why |

---

## Maintainability

| ID | Requirement | Decision |
|---|---|---|
| MAINT-U4-01 | CallPost i18n | All user-visible strings in CallPost MUST use `useIntl` with `plugin.rtk.call_post.*` message IDs. Keys added to `en.json` and `ja.json`. |
| MAINT-U4-02 | CallPage — no i18n dependency | The call page (`call.js`) MUST NOT import from `webapp/i18n/` or use `react-intl`. It is a standalone bundle with no Mattermost framework. Error/loading strings are hardcoded in English. |
| MAINT-U4-03 | Vite config documentation | `vite.config.ts` MUST have comments explaining the externals strategy for `main` vs `call` entries |
| MAINT-U4-04 | Separation of call page code | All call page source code MUST live under `webapp/src/call_page/`. No mixing with Mattermost plugin components under `webapp/src/components/`. |
| MAINT-U4-05 | Test coverage — CallPost | CallPost MUST have Enzyme shallow tests for active state, ended state, join-disabled state, and error modal |
| MAINT-U4-06 | Test coverage — CallPage | CallPage logic (fetch+keepalive on unload, URL param parsing, error screen) MUST have Jest unit tests; RTK SDK (`useRealtimeKitClient`) MUST be mocked |

---

## Security Extension Compliance Summary

| Extension Rule | Status | Notes |
|---|---|---|
| No JWT/token logging | Compliant | SEC-U4-01: token param must not be logged |
| Generic error messages | Compliant | No internal error details exposed to call page UI |
| Input validation | Compliant | SEC-U4-06: channel_name is URL-encoded before use |
| CSP compliance | Update required | SEC-U4-02: `style-src 'unsafe-inline'` needed for UI Kit; existing CSP in api_static.go must be updated |
