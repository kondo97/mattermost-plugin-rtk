# Unit 3: Webapp - Channel UI — NFR Requirements

## Performance

| ID | Requirement | Target |
|---|---|---|
| PERF-U3-01 | Redux state update latency (WS event received → component re-render) | Synchronous; < 5ms (Redux dispatch is synchronous) |
| PERF-U3-02 | Duration timer CPU impact | Negligible; single `setInterval` at 1-second granularity per active call |
| PERF-U3-03 | Config status fetch on initialization | Best-effort; no blocking UI spinner; plugin initialization completes regardless of response |
| PERF-U3-04 | "Open in new tab" action | No API call; uses saved `myActiveCall.token`; synchronous `window.open` |
| PERF-U3-05 | Component re-render scope | Selectors must be scoped (per-channel); changing one channel's call state MUST NOT cause unrelated channel header buttons to re-render |

---

## Reliability

| ID | Requirement | Decision |
|---|---|---|
| REL-U3-01 | WS event handler errors | Catch all errors in each handler; log to browser console; MUST NOT crash the plugin or Mattermost UI |
| REL-U3-02 | API call error handling | All `fetch()` calls MUST have explicit `.catch()` handlers; unhandled promise rejections are prohibited |
| REL-U3-03 | Config fetch failure | If `GET /config/status` fails on initialization, `pluginEnabled` defaults to `false`; button is hidden; no error surfaced to user |
| REL-U3-04 | Dismiss API fire-and-forget | `POST /calls/{id}/dismiss` is fire-and-forget; UI waits for the WS event to clear state, not the HTTP response |
| REL-U3-05 | Duration timer cleanup | `setInterval` MUST be cleared in `useEffect` cleanup when `myActiveCall` is cleared; no leaked intervals |
| REL-U3-06 | Auto-dismiss timeout cleanup | `setTimeout` for 30s auto-dismiss MUST be cleared in `useEffect` cleanup when `incomingCall` changes; no double-dismiss |

---

## Security

| ID | Requirement | Source |
|---|---|---|
| SEC-U3-01 | JWT tokens MUST NOT be logged to browser console at any log level | SECURITY-03, SECURITY-09 |
| SEC-U3-02 | WS event payloads MUST be validated with TypeScript type guards before Redux dispatch | SECURITY-05, SECURITY-13 |
| SEC-U3-03 | API error messages shown to users MUST be generic; raw server error strings MUST NOT be displayed directly | SECURITY-09, SECURITY-15 |
| SEC-U3-04 | All `fetch()` calls include the `Mattermost-User-ID` header (injected by Mattermost's plugin client) or use the Mattermost plugin API client | SECURITY-08 |
| SEC-U3-05 | No credentials (orgID, apiKey) are present in or referenced by frontend code | SECURITY-09, SECURITY-12 |

---

## Availability

| ID | Requirement |
|---|---|
| AVA-U3-01 | Frontend availability follows Mattermost webapp availability; no independent HA requirement |
| AVA-U3-02 | WebSocket reconnect triggers config re-fetch; plugin recovers state from subsequent WS events |

---

## Usability

| ID | Requirement | Source |
|---|---|---|
| USE-U3-01 | All user-visible strings MUST be internationalized using the Mattermost i18n system (`useIntl`/`FormattedMessage`) | User requirement: English + Japanese |
| USE-U3-02 | Translation files: `webapp/i18n/en.json` (English) and `webapp/i18n/ja.json` (Japanese) — both MUST be created with identical key sets | User requirement |
| USE-U3-03 | All i18n message IDs MUST be prefixed with `plugin.rtk.` to avoid collision with Mattermost core or other plugins | Namespace isolation |
| USE-U3-04 | Channel header button tooltip text MUST be internationalized | USE-U3-01 |
| USE-U3-05 | Modal text, toast bar text, widget text, and notification text MUST all be internationalized | USE-U3-01 |

---

## Maintainability

| ID | Requirement | Source |
|---|---|---|
| MAINT-U3-01 | Redux state management: plain `redux` + `react-redux` — do NOT add `@reduxjs/toolkit` | Alignment with Mattermost webapp and Calls plugin patterns |
| MAINT-U3-02 | Reducer implemented as a plain function with `switch` statement (no RTK `createSlice`) | MAINT-U3-01 |
| MAINT-U3-03 | Action creators implemented as plain functions returning action objects | MAINT-U3-01 |
| MAINT-U3-04 | Jest unit tests MUST cover: Redux reducer (all 7 actions), all 5 WS event handlers, all 5 selector functions | NFR-06 baseline |
| MAINT-U3-05 | Component tests: Enzyme-based shallow render tests for `ChannelHeaderButton` (all 5 visual states) | Existing test tooling (enzyme 3.11.0) |
| MAINT-U3-06 | No new test dependencies; use existing enzyme, jest, enzyme-to-json | MAINT-U3-01 (alignment principle) |
| MAINT-U3-07 | TypeScript strict mode enforced; no `any` types in Redux state, selectors, or WS handler payloads | Existing tsconfig |
| MAINT-U3-08 | Browser console logging uses `console.error` for unexpected failures only; no `console.log` in production paths | SECURITY-03 |

---

## Security Compliance Summary (SECURITY Extension)

| Rule | Status | Rationale |
|---|---|---|
| SECURITY-01 | N/A | No data persistence stores in frontend; state is in-memory Redux only |
| SECURITY-02 | N/A | No load balancers or network intermediaries owned by this unit |
| SECURITY-03 | Compliant | JWT tokens and credentials MUST NOT appear in console output (SEC-U3-01, MAINT-U3-08) |
| SECURITY-04 | N/A | Unit 3 does not serve HTML; it is bundled into the Mattermost webapp client |
| SECURITY-05 | Compliant | WS event payloads validated with TypeScript type guards before use (SEC-U3-02) |
| SECURITY-06 | N/A | No IAM policies; authentication is handled entirely by Mattermost server |
| SECURITY-07 | N/A | No network configuration owned by this unit |
| SECURITY-08 | Compliant | All API calls use Mattermost plugin client which injects `Mattermost-User-ID`; server enforces auth (SEC-U3-04) |
| SECURITY-09 | Compliant | Generic error messages shown to users; no raw server error strings displayed (SEC-U3-03); no credentials in frontend code (SEC-U3-05) |
| SECURITY-10 | N/A | Dependency lock file (`package-lock.json`) is a project-level concern addressed in Build and Test |
| SECURITY-11 | Compliant | SwitchCallModal prevents accidental call abandonment; config-hidden button prevents unauthorized call starts |
| SECURITY-12 | N/A | No user authentication managed in this unit; Mattermost handles all auth |
| SECURITY-13 | Compliant | WS event data parsed through TypeScript type guards; no `eval` or unsafe deserialization (SEC-U3-02) |
| SECURITY-14 | N/A | No server-side alerting or log retention in frontend unit |
| SECURITY-15 | Compliant | All `fetch()` calls have explicit `.catch()` handlers; no unhandled promise rejections (REL-U3-02); error paths display generic messages (SEC-U3-03) |
