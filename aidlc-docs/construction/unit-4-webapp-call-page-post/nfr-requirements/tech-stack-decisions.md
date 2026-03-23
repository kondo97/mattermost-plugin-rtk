# Unit 4: Webapp - Call Page & Post — Tech Stack Decisions

## TSD-U4-01: Build System — Vite (replaces webpack)

**Decision**: Migrate from `webpack` + `babel-loader` to `vite` + `@vitejs/plugin-react`.

**Rationale**:
- Webpack supports only a single entry point in the current configuration; adding a second entry (`call.js`) without externals requires significant webpack restructuring.
- Vite natively supports multiple `rollupOptions.input` entries and per-entry output configuration.
- Vite's built-in tree-shaking (via Rollup) produces smaller bundles than webpack 5 defaults.

**Impact on existing code**:
- `webpack.config.js` — removed
- `package.json` scripts — updated (`webpack` → `vite build`, `webpack --watch` → `vite build --watch`)
- `babel.config.js` — retained for Jest only (Jest does not use Vite)
- `tsconfig.json` — may need minor adjustments for Vite compatibility

**New devDependencies**:
```json
"vite": "^5.x",
"@vitejs/plugin-react": "^4.x"
```

**Packages removed from devDependencies**:
```json
"webpack": removed,
"webpack-cli": removed,
"babel-loader": removed,
"css-loader": removed,
"sass-loader": removed,
"style-loader": removed,
"file-loader": removed
```

**Note**: `babel.config.js`, `@babel/core`, `@babel/preset-*`, `babel-plugin-formatjs` are retained for Jest.

---

## TSD-U4-02: RTK SDK — Cloudflare RealtimeKit React UI Kit

**Decision**: Use `@cloudflare/realtimekit-react-ui` for the call page UI, with `@cloudflare/realtimekit-react` as the base SDK.

**Rationale**: Q2=A decision. UI Kit provides `<RtkMeeting mode="fill" />` which renders the complete call UI with all host/participant controls. No custom UI construction is needed for MVP.

**New dependencies** (call.js bundle only — not externalized):
```json
"@cloudflare/realtimekit-react": "^x.x",
"@cloudflare/realtimekit-react-ui": "^x.x"
```

**Note**: These packages are NOT imported from `webapp/src/index.tsx` (main bundle). They are only imported from `webapp/src/call_page/`. This ensures they are only bundled into `call.js`, not `main.js`.

---

## TSD-U4-03: Vite Externals — Conditional Per-Entry Strategy

**Decision**: Apply externals only to the `main` entry using a custom Vite plugin (inline in `vite.config.ts`).

**Implementation**:
```typescript
// In vite.config.ts — custom plugin applies externals to main entry only
function mainExternalsPlugin() {
    const EXTERNALS = {
        react: 'React',
        'react-dom': 'ReactDOM',
        redux: 'Redux',
        'react-redux': 'ReactRedux',
        'prop-types': 'PropTypes',
        'react-bootstrap': 'ReactBootstrap',
        'react-router-dom': 'ReactRouterDom',
    };
    return {
        name: 'main-externals',
        resolveId(id: string, _importer: string | undefined, options: {isEntry?: boolean}) {
            if (options?.isEntry) { /* entry tracking */ }
        },
    };
}
```

**Alternative**: Use Rollup's `external` function in `rollupOptions` with a facet that checks the current entry chunk name.

**Note**: The exact implementation is determined during Code Generation; the design intent is clear from Q8=A.

---

## TSD-U4-04: Test Infrastructure — No Changes

**Decision**: Jest + Enzyme remain unchanged. Vite is not used for tests.

**Rationale**: The existing `babel.config.js` + Jest setup works correctly for unit tests. Vite is only the production build tool. This is the standard Mattermost plugin pattern (Vite for build, Babel/Jest for tests).

**Impact**: None. All existing tests continue to work unchanged.

---

## TSD-U4-05: CallPage — React 18 `createRoot` API

**Decision**: The call page uses `ReactDOM.createRoot(root).render(<CallPage />)` (React 18 API), not the legacy `ReactDOM.render()`.

**Rationale**: The call page bundles React 18 independently (call.js entry). React 18's `createRoot` is the correct API. This is separate from `main.js` where React is provided by Mattermost as a global.

---

## Package.json Changes Summary

| Change | Type | Reason |
|--------|------|--------|
| Add `vite` | devDependency | Build system replacement |
| Add `@vitejs/plugin-react` | devDependency | Vite React plugin |
| Add `@cloudflare/realtimekit-react` | dependency | RTK SDK base |
| Add `@cloudflare/realtimekit-react-ui` | dependency | RTK UI Kit |
| Remove `webpack`, `webpack-cli` | devDependency | Replaced by Vite |
| Remove `babel-loader`, `css-loader`, `sass-loader`, `style-loader`, `file-loader` | devDependency | Webpack loaders no longer needed |
| Update `scripts.build` | — | `webpack --mode=production` → `vite build` |
| Update `scripts.debug` | — | `webpack --mode=none` → `vite build --mode development` |
| Update `scripts.build:watch` | — | `webpack --watch` → `vite build --watch` |
