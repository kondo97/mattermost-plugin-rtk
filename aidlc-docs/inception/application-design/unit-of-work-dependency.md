# Unit of Work — Dependency Matrix

## Dependency Matrix

| Unit | Depends On | Blocks |
|---|---|---|
| Unit 1: RTK Integration | (none — external: Cloudflare RTK API, Mattermost Plugin SDK) | Unit 2, Unit 6 |
| Unit 2: Server API & WebSocket | Unit 1 | Unit 3, Unit 4, Unit 5, Unit 6 |
| Unit 3: Webapp - Channel UI | Unit 2 (API contracts, WS event schemas) | — |
| Unit 4: Webapp - Call Page & Post | Unit 2 (API contracts) | — |
| Unit 5: Admin & Config | Unit 2 (config endpoints) | — |
| Unit 6: Mobile Support | Unit 1 (call service), Unit 2 (API) | — |

## Parallel Execution Opportunities

```
Phase 1 (sequential):
  Unit 1 --> Unit 2

Phase 2 (parallel — all depend only on Unit 2):
  Unit 3  |
  Unit 4  |  (can be designed and coded in parallel)
  Unit 5  |
  Unit 6  |

Phase 3:
  Build and Test (depends on all units complete)
```

## External Dependencies

| Unit | External Dependency | Type | Notes |
|---|---|---|---|
| Unit 1 | Cloudflare RTK API | HTTP (HTTPS, Basic Auth) | `CreateMeeting`, `GenerateToken`, `EndMeeting` |
| Unit 1 | Mattermost Plugin SDK | Go library | KVStore, WebSocket pub/sub, PluginAPI |
| Unit 2 | gorilla/mux | Go library | HTTP router |
| Unit 3 | Mattermost Redux store | JS library | `registry.registerReducer()` |
| Unit 4 | `@cloudflare/calls-react` (DyteProvider) | npm package | RTK React SDK |
| Unit 4 | Vite | npm/build tool | Replaces webpack |
| Unit 6 | Mattermost push infrastructure | Internal API | `SendPushNotification()` via PluginAPI |

## Shared File Conflicts

The following files are modified by multiple units. Coordinate edits to avoid conflicts:

| File | Modified By | Resolution Strategy |
|---|---|---|
| `server/plugin.go` | Unit 1, Unit 2, Unit 6 | Unit 1 establishes structure; Unit 2 adds ServeHTTP; Unit 6 adds push init/calls — merge sequentially |
| `webapp/src/index.tsx` | Unit 3, Unit 4 | Unit 3 registers reducer + channel components; Unit 4 registers custom post type — merge sequentially |

## Build Impact

All units compile into a single plugin artifact:
- `dist/mattermost-plugin-rtk-{version}.tar.gz`
- Contains: `plugin.json`, `server/dist/plugin-linux-amd64`, `server/dist/plugin-linux-arm64`, `webapp/dist/main.js`, `webapp/dist/call.js`

No unit produces an independently deployable artifact. Integration testing requires the full plugin to be built and installed.
