# Story Generation Plan

## Overview

This plan covers the generation of user stories and personas for the `mattermost-plugin-rtk` project: a Mattermost plugin enabling Cloudflare RealtimeKit video/audio calls.

The plan references `aidlc-docs/inception/requirements/requirements.md` for functional and non-functional requirements, and uses the Mattermost Calls plugin as the UX baseline.

---

## Step 1: Validate User Stories Need

- [x] Assessment complete — see `aidlc-docs/inception/plans/user-stories-assessment.md`
- **Decision**: Execute User Stories (High Priority — new user features, multi-persona system, complex business logic)

---

## Step 2: Answer Story Planning Questions

Before generating stories, the following questions must be answered.

Please fill in the letter choice after each `[Answer]:` tag. If none of the options fit, choose the last option (Other) and describe your preference.

---

### Question 1
What story breakdown approach should be used to organize user stories?

A) **Feature-Based** — stories grouped by system features (Call Lifecycle, Admin Config, Mobile Support, etc.)
B) **User Journey-Based** — stories follow the flow from a user's perspective (start call → join → in-call → leave)
C) **Epic-Based** — high-level epics per persona, with detailed sub-stories under each
D) **Hybrid** — feature-based epics with user journey ordering within each epic
E) Other (please describe after [Answer]: tag below)

[Answer]: B) User Journey-Based

---

### Question 2
What level of story granularity is preferred?

A) **Epic-level** — broad stories that span multiple interactions (e.g., "As a user, I can participate in a video call")
B) **Feature-level** — one story per distinct capability (e.g., separate stories for start, join, leave, end)
C) **Scenario-level** — fine-grained stories covering individual scenarios including edge cases (e.g., "As a user already in a call, I see the Switch Call Modal when trying to join another")
D) Other (please describe after [Answer]: tag below)

[Answer]: C) Scenario-level

---

### Question 3
How should the comparison with the Mattermost Calls plugin be documented in user stories?

A) **Inline notes** — add a "Mattermost Calls comparison" note within each story's acceptance criteria where relevant
B) **Separate section** — include a "Differences from Mattermost Calls plugin" subsection within each story that differs
C) **Summary table** — generate a standalone comparison table at the top of stories.md mapping each story to its Mattermost Calls equivalent
D) **No explicit comparison** — alignment with Mattermost Calls is implicit in the acceptance criteria; no dedicated comparison documentation
E) Other (please describe after [Answer]: tag below)

[Answer]: C) Summary table

---

### Question 4
How should the mobile user be represented as a persona?

A) **Separate persona** — "Mobile User" is a distinct persona with its own stories covering push notification, native call join, and dismiss flow
B) **Variant of Channel Member** — mobile-specific behavior is noted as a variant within Channel Member stories (e.g., "on mobile, the user receives a push notification instead of the in-app ring")
C) **Both** — a distinct Mobile User persona for push/native-call stories, plus mobile variant notes in shared Channel Member stories
D) Other (please describe after [Answer]: tag below)

[Answer]: C) Both

---

### Question 5
What acceptance criteria format should be used?

A) **Given/When/Then (Gherkin-style)** — structured behavioral scenarios (Given the user is in a channel with an active call, When they click Join, Then...)
B) **Bullet checklist** — plain list of testable conditions per story
C) **Hybrid** — bullet checklist for happy path + Given/When/Then for key edge cases
D) Other (please describe after [Answer]: tag below)

[Answer]: B) Bullet checklist

---

### Question 6
Should the Admin user have their own dedicated persona and stories?

A) **Yes — full Admin persona** — include a complete "Mattermost Admin" persona with stories for credential setup, feature flag toggling, and env var configuration
B) **Yes — minimal Admin persona** — include an Admin persona but only story-level coverage (no detailed acceptance criteria for admin UI)
C) **No — admin config is implicit** — admin configuration is a prerequisite, not a user story; skip Admin persona
D) Other (please describe after [Answer]: tag below)

[Answer]: B) Minimal Admin persona

---

### Question 7
Which call lifecycle edge cases should be explicitly covered as separate stories (rather than inline in acceptance criteria)?

A) **All edge cases as separate stories** — Switch Call Modal, DM/GM ringing with Ignore/Join, active call blocking, host-end-for-all, last-participant auto-end
B) **Only high-risk edge cases** — Switch Call Modal and active call blocking (the two most likely to be misimplemented)
C) **No separate edge case stories** — cover all edge cases within acceptance criteria of the main stories
D) Other (please describe after [Answer]: tag below)

[Answer]: A) All as separate stories

---

## Step 3: Mandatory Story Artifacts

Upon answer collection and plan approval, the following will be generated:

- [x] `aidlc-docs/inception/user-stories/personas.md` — user archetypes with goals, characteristics, and pain points
- [x] `aidlc-docs/inception/user-stories/stories.md` — user stories following INVEST criteria with acceptance criteria and persona mapping

---

## Step 4: Story Execution Checklist (to be executed after plan approval)

- [x] **4.1** Load requirements.md and reverse-engineering artifacts for context
- [x] **4.2** Generate personas.md — define all personas with goals, context, and motivations
- [x] **4.3** Generate stories.md — create user stories organized per approved breakdown approach
  - [x] **4.3.1** Call Lifecycle stories (start, join, leave, end, auto-end)
  - [x] **4.3.2** Call UI stories (button states, toast bar, post card states, sound cue)
  - [x] **4.3.3** Edge case stories (Switch Call Modal, DM/GM ringing, active call blocking, host end)
  - [x] **4.3.4** Admin & Config stories (credential setup, feature flag toggle, env var override)
  - [x] **4.3.5** Mobile stories (VoIP token registration, incoming push, dismiss notification, mobile join)
  - [x] **4.3.6** Web Worker & Call Page stories (standalone page, worker.js endpoint)
- [x] **4.4** Verify all stories follow INVEST criteria
- [x] **4.5** Verify all stories have acceptance criteria
- [x] **4.6** Verify persona-to-story mapping is complete
- [x] **4.7** Add Mattermost Calls comparison per approved format (Question 3 answer)
- [x] **4.8** Update aidlc-state.md to mark User Stories - COMPLETE
- [x] **4.9** Append completion entry to audit.md

---

## Approval Required

After answering all questions above, this plan must be reviewed and approved before story generation begins.
