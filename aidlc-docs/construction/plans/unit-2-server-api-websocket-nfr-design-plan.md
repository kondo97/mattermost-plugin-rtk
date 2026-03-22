# Unit 2: Server API & WebSocket — NFR Design Plan

## Execution Checklist

- [x] Step 1: Analyze NFR Requirements
- [x] Step 2: Create NFR Design Plan (this file)
- [x] Step 3: Generate context-appropriate questions (none needed — all clear from NFR requirements)
- [x] Step 4: Store Plan (this file)
- [x] Step 5: Collect answers (N/A)
- [x] Step 6: Generate NFR Design artifacts
- [ ] Step 7: Present completion message
- [ ] Step 8: Wait for explicit approval

## Design Patterns to Apply

- Pattern U2-1: Auth Middleware (SEC-U2-01)
- Pattern U2-2: HTTP Security Headers on /call (SEC-U2-05, SECURITY-04)
- Pattern U2-3: Concurrency Mutex (BR-U2-39, callMu)
- Pattern U2-4: Static File Embedding (go:embed, REL-U2-03)
- Pattern U2-5: Error-to-HTTP-Status Mapping (inherits Pattern 3 from Unit 1)
- Pattern U2-6: Structured Logging in Handlers (inherits Pattern 6 from Unit 1)
