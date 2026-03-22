# Unit 1: RTK Integration — NFR Requirements Plan

## Execution Checklist

- [x] Analyze functional design artifacts
- [x] Identify applicable NFR categories and security rules
- [x] Generate clarifying questions
- [x] Collect answers
- [x] Analyze for ambiguities (none)
- [x] Generate NFR artifacts:
  - [x] `nfr-requirements.md`
  - [x] `tech-stack-decisions.md`

---

## Pre-Analysis: Already-Defined NFRs (from requirements.md)

The following are already specified and require no questions:
- NFR-01: Cloudflare RTK API uses HTTPS + Basic Auth
- NFR-02: No credential exposure; RTKClient interface abstraction
- NFR-03: <1s response for token generation; 200 concurrent user target
- NFR-06: RTKClient + KVStore interfaces for testability; structured logging; unit tests

---

## Clarifying Questions

### Q1: RTK API HTTP Client Timeout

Cloudflare RTK API calls (CreateMeeting, GenerateToken, EndMeeting) go over HTTPS. What timeout should the HTTP client enforce?

A) **5 seconds** — tight timeout; fails fast if Cloudflare is slow; CreateCall would return an error to the user
B) **10 seconds** — moderate; accommodates transient Cloudflare latency
C) **30 seconds** — generous; reduces false failures but degrades user experience on actual outages

[Answer]:

---

### Q2: RTK API Retry Policy

If `CreateMeeting` or `GenerateToken` fails due to a transient network error, should the plugin retry?

A) **No retry** — return error immediately; user can retry by clicking the button again
B) **1 retry with 500ms backoff** — retry once before returning error
C) **Exponential backoff (max 3 attempts)** — retry up to 3 times with increasing delay

[Answer]:

---

### Q3: Concurrent CreateCall — Race Condition Handling

Two users in the same channel click "Start call" simultaneously. The current design uses a KVStore read-then-write sequence (GetCallByChannel → SaveCall). KVStore does not support atomic compare-and-swap.

A) **Accept the race** — duplicate call creation is unlikely in practice; the second call would be orphaned but cleaned up eventually
B) **Use KVStore atomic set** — use `KVStore.SetAtomic` (if available in Mattermost Plugin SDK) to set only if key doesn't exist
C) **Use a distributed lock** — acquire a per-channel lock before the check-and-create sequence

[Answer]:

---

### Q4: Background Job — Partial Failure Handling

If `CleanupStaleParticipants` encounters an error on one call (e.g., KVStore read fails), should the job:

A) **Continue to next call** — log the error and process remaining calls; best effort
B) **Abort the entire run** — stop processing; try again in 30 seconds

[Answer]:

---

### Q5: Credential Validation on Plugin Activation

When the plugin activates (`OnActivate`) and RTK credentials are configured, should it validate them with a test API call?

A) **No validation on activation** — validate only on first call attempt; simpler and avoids unnecessary API calls
B) **Validate on activation** — make a lightweight test call (e.g., list meetings or a no-op) to surface credential errors early in the admin console

[Answer]:
