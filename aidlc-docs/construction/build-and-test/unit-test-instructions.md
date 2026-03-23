# Unit Test Execution

## Overview

The plugin has two separate test suites:

| Suite | Framework | Location |
|-------|-----------|----------|
| Backend (Go) | testify + go-mock + gotestsum | `server/` |
| Frontend (TypeScript) | Jest + Enzyme + Testing Library | `webapp/src/` |

---

## Run All Unit Tests

```bash
make test
```

This runs both suites sequentially.

---

## Backend Unit Tests (Go)

### Test Files

| File | Coverage Area |
|------|--------------|
| `server/plugin_test.go` | Plugin lifecycle (OnActivate, OnDeactivate) |
| `server/configuration_test.go` | Configuration load/validate (Unit 1) |
| `server/calls_test.go` | Call state management (Unit 1) |
| `server/api_calls_test.go` | REST API: join/leave/end call (Unit 2) |
| `server/api_webhook_test.go` | Webhook handler (Unit 2) |
| `server/api_config_test.go` | Admin config API (Unit 5) |
| `server/api_mobile_test.go` | Mobile token API (Unit 6) |
| `server/push/push_test.go` | Push notification sender (Unit 6) |
| `server/command/command_test.go` | Slash command handler (Unit 2) |

### Run Backend Tests

```bash
# Run all backend tests with verbose output
cd server && $(go env GOPATH)/bin/gotestsum -- -v ./...

# Or via make (includes gotestsum install)
make test
```

### Run with Coverage Report

```bash
make coverage
# Opens coverage HTML in browser
# Report saved at: server/coverage.txt
```

### Run Specific Package

```bash
cd server && go test -v ./...                    # all packages
cd server && go test -v -run TestJoinCall ./...  # specific test
cd server && go test -v ./push/...               # push package only
```

### CI Mode (JUnit XML output)

```bash
make test-ci
# Output: report.xml (JUnit format)
```

### Expected Results

- All tests pass with 0 failures
- Race condition detector enabled (`-race` flag via `GO_TEST_FLAGS`)
- Test report location: `report.xml` (CI mode)

---

## Frontend Unit Tests (TypeScript/React)

### Test Files

| File | Coverage Area |
|------|--------------|
| `webapp/src/manifest.test.tsx` | Plugin manifest export (Unit 1) |
| `webapp/src/react_fragment.test.tsx` | Plugin registration/components (Unit 1) |
| `webapp/src/redux/calls_slice.test.ts` | Redux slice reducers and actions (Unit 1) |
| `webapp/src/redux/selectors.test.ts` | Redux selectors (Unit 1) |
| `webapp/src/redux/websocket_handlers.test.ts` | WebSocket event handlers (Unit 2) |
| `webapp/src/components/channel_header_button/index.test.tsx` | Channel header button UI (Unit 3) |
| `webapp/src/components/call_post/index.test.tsx` | Call post component UI (Unit 4) |
| `webapp/src/call_page/CallPage.test.tsx` | RTK call page (Unit 4) |

### Run Frontend Tests

```bash
cd webapp && npm run test
# Options:
# --forceExit --detectOpenHandles --verbose (default)
```

### Watch Mode (Development)

```bash
cd webapp && npm run test:watch
```

### CI Mode (Reduced Parallelism)

```bash
cd webapp && npm run test-ci
# Uses --maxWorkers=2 for stability in CI
```

### Expected Results

- 8 test suites (81 tests) pass with 0 failures
- JUnit XML output: `webapp/junit.xml`
- Test environment: jsdom (simulates browser)

---

## Fix Failing Tests

### Backend Failures

1. Review output: `cd server && gotestsum -- -v ./... 2>&1 | grep -A5 FAIL`
2. Common causes:
   - Mock expectations not met → check `EXPECT()` calls in test files
   - KVStore/API plugin mock returning wrong values
3. Fix and rerun: `cd server && go test -v -run <TestName> ./...`

### Frontend Failures

1. Review output in terminal for failing test name
2. Common causes:
   - Enzyme/act wrapping missing for state updates
   - `useSelector` mock not returning expected state shape
3. Fix and rerun: `cd webapp && npx jest --testPathPattern=<filename> --verbose`
