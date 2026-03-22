# Unit of Work Plan

## Execution Checklist

- [x] Analyze context (execution plan, application design, user stories)
- [x] Determine decomposition strategy
- [x] Assess need for clarifying questions (none needed — units defined in execution-plan.md)
- [x] Generate unit artifacts:
  - [x] `aidlc-docs/inception/application-design/unit-of-work.md`
  - [x] `aidlc-docs/inception/application-design/unit-of-work-dependency.md`
  - [x] `aidlc-docs/inception/application-design/unit-of-work-story-map.md`
- [x] Validate unit boundaries and dependencies
- [x] Ensure all stories are assigned to units

---

## Context Analysis Summary

### Decomposition Strategy

This is a **brownfield monolith** (single Mattermost plugin). Units are logical groupings of work within the same deployable plugin binary. No independent deployable services. Implementation order is determined by data and API dependencies.

### Units Defined (from execution-plan.md)

The execution plan already identified 6 units. This plan confirms and details them based on the completed Application Design artifacts.

| Unit | Name | Primary Components |
|---|---|---|
| Unit 1 | RTK Integration | B-04 RTKClient, B-05 KVStore (call extensions), B-01 Plugin Core (call logic), B-07 Background Job |
| Unit 2 | Server API & WebSocket | B-02 API Handler (all HTTP endpoints), WebSocket event emission helpers |
| Unit 3 | Webapp - Channel UI | F-02 ChannelHeaderButton, F-04 ToastBar, F-05 FloatingWidget, F-06 SwitchCallModal, F-07 IncomingCallNotification, F-09 Redux (slice + WS handlers + selectors), F-01 entry (partial) |
| Unit 4 | Webapp - Call Page & Post | F-03 CallPost, F-10 Call Page (separate bundle) |
| Unit 5 | Admin & Config | B-03 Configuration, F-08 AdminSettings |
| Unit 6 | Mobile Support | B-06 Push, mobile API endpoints (dismiss, voip-token) |

### No Clarifying Questions Needed

The decomposition is fully determined by:
- Execution plan unit definitions (already approved by user)
- Application design component inventory (B-01 through B-07, F-01 through F-10)
- Clear dependency chain: Unit 1 → Unit 2 → Units 3/4/5/6

---

## Artifact Generation Plan

### Step A: Generate `unit-of-work.md`

Content to generate:
- Unit definitions with responsibilities, in-scope components, and acceptance scope
- Implementation order and rationale
- Code organization (which directories/files each unit touches)
- Per-unit success criteria

### Step B: Generate `unit-of-work-dependency.md`

Content to generate:
- Dependency matrix (which unit depends on which)
- Parallel execution opportunities (Units 3, 4, 5, 6 can be designed/coded in parallel)
- External dependencies per unit (Cloudflare RTK API, Mattermost Plugin SDK, KVStore)
- Build impact (all units compile into a single plugin binary)

### Step C: Generate `unit-of-work-story-map.md`

Content to generate:
- Story-to-unit assignment table (all US-001 through US-025)
- Rationale for assignments where a story spans multiple units
- Per-unit story summary (total count, journey coverage)
