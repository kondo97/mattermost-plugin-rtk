# Execution Plan

## Detailed Analysis Summary

### Transformation Scope
- **Transformation Type**: Architectural — complete replacement of starter template with production Mattermost plugin
- **Primary Changes**:
  - Go backend: RTK API client, call service, session management, WebSocket events, push notifications, HTTP API endpoints
  - React/TypeScript frontend: call button, standalone call page, custom post type, admin settings, in-call indicator
  - Build system: replace webpack with Vite, add dual-bundle configuration, embed call page in Go binary
- **Related Components**: All existing starter template files will be replaced or significantly modified

### Change Impact Assessment
- **User-facing changes**: Yes — new call initiation/joining UX, custom post cards, channel header button, mobile push notifications
- **Structural changes**: Yes — new packages (rtkclient, callservice), new build pipeline (Vite), new file structure
- **Data model changes**: Yes — call session stored in KVStore with 7 fields; WebSocket event payloads defined
- **API changes**: Yes — 8+ new REST endpoints, 5 WebSocket event types, VoIP token registration
- **NFR impact**: Yes — SECURITY-01 through SECURITY-15 enforced, 200 user scale, 1s response time target

### Component Relationships
- **Primary**: server/ (Go plugin core) — owns call lifecycle, session state, RTK API integration
- **Frontend**: webapp/ (React/TS) — consumes server API, renders call UI, registers with Mattermost
- **External**: Cloudflare RTK API — meeting creation, participant tokens
- **Mobile**: Mattermost push infra — VoIP push delivery for iOS/Android
- **Dependent on**: Mattermost Plugin SDK, KVStore, WebSocket pub/sub

### Risk Assessment
- **Risk Level**: High
- **Rollback Complexity**: Moderate (plugin can be disabled via Mattermost admin console)
- **Testing Complexity**: Complex (external API mocking, WebSocket events, mobile push notification testing)

---

## Workflow Visualization

```
INCEPTION PHASE
+-------------------------------+
| [x] Workspace Detection       |
| [x] Reverse Engineering       |
| [x] Requirements Analysis     |
| [~] Workflow Planning         |
| [ ] Application Design EXECUTE|
| [ ] Units Generation  EXECUTE |
| [ ] User Stories      EXECUTE |
+-------------------------------+
             |
             v
CONSTRUCTION PHASE (per unit)
+-------------------------------+
| [ ] Functional Design  EXECUTE|
| [ ] NFR Requirements   EXECUTE|
| [ ] NFR Design         EXECUTE|
| [-] Infrastructure     SKIP   |
| [ ] Code Generation    EXECUTE|
+-------------------------------+
             |
             v
+-------------------------------+
| [ ] Build and Test     EXECUTE|
+-------------------------------+
             |
             v
OPERATIONS PHASE
+-------------------------------+
| [-] Operations    PLACEHOLDER |
+-------------------------------+
```

---

## Phases to Execute

### INCEPTION PHASE
- [x] Workspace Detection — COMPLETED
- [x] Reverse Engineering — COMPLETED
- [x] Requirements Analysis — COMPLETED
- [ ] User Stories — **EXECUTE**
  - **Rationale**: User stories will clarify how this plugin's UX compares to and differs from the Mattermost Calls plugin for each persona and interaction.
- [~] Workflow Planning — IN PROGRESS
- [ ] Application Design — **EXECUTE**
  - **Rationale**: Multiple new components and interfaces needed (RTKClient, CallService, KVStore extensions, 5+ frontend components). Component boundaries and service layer must be defined before code generation.
- [ ] Units Generation — **EXECUTE**
  - **Rationale**: System spans Go backend, React/TS frontend, and mobile API layer. Decomposing into units enables focused implementation and parallel review.

### CONSTRUCTION PHASE

**Units identified (6):**

| Unit | Scope |
|---|---|
| Unit 1: RTK Integration | RTKClient interface, call service (create/join/end), session storage (KVStore) |
| Unit 2: Server API & WebSocket | HTTP endpoints, WebSocket event emission, authentication middleware |
| Unit 3: Webapp - Channel UI | Call button, channel call toast bar, Switch Call Modal, in-call indicator |
| Unit 4: Webapp - Call Page & Post | Standalone call page, custom post type (active/ended states) |
| Unit 5: Admin & Config | Admin console UI, config status API, environment variable override |
| Unit 6: Mobile Support | VoIP token registration, push notification delivery, mobile-compatible API responses |

**Unit dependency order:**
```
Unit 1 (RTK Integration)
  └── Unit 2 (Server API & WebSocket)  ← depends on Unit 1 call service
        ├── Unit 3 (Webapp - Channel UI)    ← depends on Unit 2 API contracts
        ├── Unit 4 (Webapp - Call Page & Post) ← depends on Unit 2 API contracts
        ├── Unit 5 (Admin & Config)         ← depends on Unit 2 config endpoints
        └── Unit 6 (Mobile Support)         ← depends on Unit 1 call service + Unit 2 API
```

For each unit:
- [ ] Functional Design — **EXECUTE**
  - **Rationale**: Complex business logic (call lifecycle, session state transitions, preset selection, WebSocket event emission)
- [ ] NFR Requirements — **EXECUTE**
  - **Rationale**: SECURITY-01–15 enforced, 1s response target, RTKClient interface for testability
- [ ] NFR Design — **EXECUTE**
  - **Rationale**: NFR requirements exist and need patterns incorporated (input validation, structured logging, error handling, interface abstraction)
- [-] Infrastructure Design — **SKIP**
  - **Rationale**: No infrastructure changes. Plugin deployment via Mattermost admin console upload. No new cloud resources, databases, or networking changes required.
- [ ] Code Generation — **EXECUTE** (always)

### BUILD AND TEST
- [ ] Build and Test — **EXECUTE** (always)

### OPERATIONS PHASE
- [-] Operations — PLACEHOLDER

---

## Package Change Sequence

```
Step 1: Unit 1 (RTK Integration)
  - Depends on: Cloudflare RTK API (external), Mattermost Plugin SDK
  - Blocks: Unit 2, Unit 6

Step 2: Unit 2 (Server API & WebSocket)
  - Depends on: Unit 1
  - Blocks: Unit 3, Unit 4, Unit 5, Unit 6

Step 3: Units 3, 4, 5, 6 (parallel — no inter-dependencies)
  - Unit 3: Webapp - Channel UI       (depends on Unit 2)
  - Unit 4: Webapp - Call Page & Post (depends on Unit 2)
  - Unit 5: Admin & Config            (depends on Unit 2)
  - Unit 6: Mobile Support            (depends on Unit 1 + Unit 2)

Step 4: Build and Test
  - Depends on: all units complete
```

---

## Success Criteria
- **Primary Goal**: Functional Mattermost plugin enabling Cloudflare RealtimeKit video/audio calls
- **Key Deliverables**:
  - Working call start/join/end flow in browser (new tab)
  - Custom post card with active/ended states
  - Admin configuration UI
  - Mobile push notification support
  - All SECURITY-01–15 constraints satisfied
  - Unit tests for core business logic
- **Quality Gates**:
  - SECURITY extension rules: all compliant or explicitly N/A
  - RTKClient and KVStore interfaces: 100% mock-tested
  - API response times: under 1s for token generation
  - Build: clean `make` with no errors for linux-amd64 and linux-arm64
