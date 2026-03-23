# Unit 3: Webapp - Channel UI — Tech Stack Decisions

## Decision Log

---

### TSD-U3-01: Redux State Management — Plain Redux (no RTK)

**Decision**: Use plain `redux` 5.0.1 + `react-redux` 9.2.0 without `@reduxjs/toolkit`.

**Rationale**:
- Neither the Mattermost webapp nor the official Mattermost Calls plugin uses `@reduxjs/toolkit`
- Avoids introducing a new dependency not present in the ecosystem baseline
- The Redux slice for this unit is small enough (7 actions, 1 reducer) that RTK's boilerplate reduction is not necessary
- Plain Redux is fully compatible with `react-redux` 9.2.0 and `redux` 5.0.1

**Pattern**:
```typescript
// Action types as string constants
const SET_PLUGIN_ENABLED = 'rtk-calls/setPluginEnabled';

// Action creators as plain functions
const setPluginEnabled = (enabled: boolean) =>
    ({ type: SET_PLUGIN_ENABLED, payload: enabled } as const);

// Reducer as switch statement
function callsReducer(state = initialState, action: CallsAction): CallsState {
    switch (action.type) {
        case SET_PLUGIN_ENABLED:
            return { ...state, pluginEnabled: action.payload };
        // ...
    }
}
```

---

### TSD-U3-02: Internationalization — Mattermost i18n System

**Decision**: Use Mattermost's i18n infrastructure (`babel-plugin-formatjs`, `useIntl`, `FormattedMessage`).

**Rationale**:
- Product requirement: English and Japanese support
- `babel-plugin-formatjs` and `@mattermost/types` are already in `devDependencies`
- Consistent with how all Mattermost plugins handle localization
- Enables community contributions for additional languages without code changes

**Implementation**:
- All user-visible strings wrapped in `intl.formatMessage({ id: 'plugin.rtk.<key>' })` or `<FormattedMessage id="plugin.rtk.<key>" />`
- `webapp/i18n/en.json` — English strings (created in Unit 3 code generation)
- `webapp/i18n/ja.json` — Japanese strings (created in Unit 3 code generation)
- Translation loading registered via `registry.registerTranslations()`

**Message ID convention**: `plugin.rtk.<component>.<element>`

Examples:
```json
{
  "plugin.rtk.channel_header.start_call": "Start call",
  "plugin.rtk.channel_header.join_call": "Join call",
  "plugin.rtk.channel_header.in_call": "In call",
  "plugin.rtk.channel_header.starting_call": "Starting call...",
  "plugin.rtk.switch_call_modal.title": "You are already in a call",
  "plugin.rtk.switch_call_modal.body": "Do you want to leave your current call and join the new one?",
  "plugin.rtk.switch_call_modal.cancel": "Cancel",
  "plugin.rtk.switch_call_modal.confirm": "Leave and join new call",
  "plugin.rtk.incoming_call.ignore": "Ignore",
  "plugin.rtk.incoming_call.join": "Join",
  "plugin.rtk.toast_bar.join": "Join",
  "plugin.rtk.floating_widget.open_in_tab": "Open in new tab"
}
```

---

### TSD-U3-03: Component Testing — Enzyme (Existing)

**Decision**: Use existing Enzyme 3.11.0 for component tests; no new testing dependencies.

**Rationale**:
- Enzyme is already installed and configured in the project
- Adding `@testing-library/react` would be a new dependency and require setup changes
- The Mattermost Calls plugin also uses Enzyme-based testing patterns
- Enzyme's shallow rendering is sufficient for testing component state and prop rendering

**Test scope for Unit 3**:
- `calls_slice.test.ts` — reducer and action creator unit tests (pure Jest, no DOM)
- `websocket_handlers.test.ts` — WS handler unit tests (pure Jest with mock Redux store)
- `selectors.test.ts` — selector unit tests (pure Jest)
- `channel_header_button.test.tsx` — Enzyme shallow tests for all 5 visual states

---

### TSD-U3-04: CSS / Styling

**Decision**: Use Mattermost CSS utility classes and inline styles where needed; no dedicated CSS module or CSS-in-JS library for Unit 3.

**Rationale**:
- Mattermost provides a comprehensive set of utility classes aligned with its design system
- Emotion (`@emotion/react`) is in devDependencies for advanced styling if needed
- For a plugin UI, adopting Mattermost's own CSS classes ensures visual consistency

**Approach**:
- Standard Mattermost button classes (`btn btn-primary`, `btn-icon`) for buttons
- Mattermost modal component from `mattermost-redux` / shared UI where available
- Inline styles only for layout overrides specific to FloatingWidget positioning

---

### TSD-U3-05: State Selector Library

**Decision**: Use `react-redux` `useSelector` hook with manually written selector functions; no `reselect` memoization library.

**Rationale**:
- The selectors are simple property lookups and do not involve expensive computations
- `reselect` is not in the existing dependencies
- React-Redux's built-in `useSelector` reference equality check is sufficient for the current state shape

**Exception**: If profiling reveals unnecessary re-renders due to derived state (e.g., `selectIsCurrentUserParticipant`), `reselect` may be introduced as a targeted optimization in a later iteration.

---

## Dependency Changes Summary

| Package | Version | Change | Reason |
|---|---|---|---|
| `@reduxjs/toolkit` | — | NOT added | Plain Redux aligns with Mattermost/Calls plugin baseline |
| `@testing-library/react` | — | NOT added | Enzyme already installed and sufficient |
| `reselect` | — | NOT added | Simple selectors do not require memoization |
| No new runtime dependencies | — | — | All needed packages already present |

**Note**: No changes to `package.json` are required for Unit 3. All necessary packages (`redux`, `react-redux`, `mattermost-redux`, `enzyme`, `babel-plugin-formatjs`) are already installed.
