# Security Test Instructions

> Security extension is **enabled** for this project. These tests are mandatory before release.

## Security Risk Areas

| Area | Risk | Unit |
|------|------|------|
| Cloudflare API Key storage | Secret exposure in logs/responses | Unit 1, 5 |
| RTK token generation | Token leakage or reuse across sessions | Unit 1 |
| REST API authorization | Unauthenticated access to join/leave/config endpoints | Unit 2, 5 |
| Mobile token endpoint | Unauthorized device registration | Unit 6 |
| Webhook handler | HMAC signature bypass | Unit 2 |
| Call post content | XSS via injected channel/user names | Unit 4 |

---

## 1. Dependency Vulnerability Scan

### Go Dependencies

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan Go dependencies
cd server && govulncheck ./...
```

**Expected**: 0 HIGH or CRITICAL vulnerabilities.

### Frontend Dependencies

```bash
cd webapp && npm audit --audit-level=high
```

**Expected**: 0 HIGH or CRITICAL vulnerabilities.
If found: `npm audit fix` or manually update affected packages.

---

## 2. Static Analysis (Already in CI)

```bash
# Go static analysis (run as part of check-style)
make check-style

# Covers:
# - go vet: suspicious constructs
# - golangci-lint: security linters (gosec, errcheck, staticcheck)
```

**Verify these gosec rules pass**:
- G101: Hardcoded credentials
- G201/G202: SQL injection (N/A for this plugin)
- G401/G501: Weak cryptography

---

## 3. API Authorization Tests

### Test: Unauthenticated Join Call

```bash
# Must return 401 Unauthorized (no MM auth cookie/token)
curl -X POST http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/calls/join \
  -H "Content-Type: application/json" \
  -d '{"channel_id": "testchannel"}'
```

**Expected**: `401 Unauthorized`

### Test: Non-Member Join Call

```bash
# User with valid auth but NOT a member of the channel
curl -X POST http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/calls/join \
  -H "Authorization: Bearer <non-member-token>" \
  -H "Content-Type: application/json" \
  -d '{"channel_id": "<private-channel-id>"}'
```

**Expected**: `403 Forbidden`

### Test: Unauthenticated Config Update

```bash
curl -X PUT http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/config \
  -H "Content-Type: application/json" \
  -d '{"RecordingEnabled": false}'
```

**Expected**: `401 Unauthorized`

### Test: Non-Admin Config Update

```bash
curl -X PUT http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/config \
  -H "Authorization: Bearer <regular-user-token>" \
  -H "Content-Type: application/json" \
  -d '{"RecordingEnabled": false}'
```

**Expected**: `403 Forbidden`

### Test: Unauthenticated Mobile Token Registration

```bash
curl -X POST http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/mobile/token \
  -H "Content-Type: application/json" \
  -d '{"device_token": "test", "platform": "ios"}'
```

**Expected**: `401 Unauthorized`

---

## 4. Secret Exposure Tests

### Test: API Key Not Returned in Config Response

```bash
curl -X GET http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/config \
  -H "Authorization: Bearer <admin-token>"
```

**Expected**: Response does NOT contain `CloudflareAPIKey` value (must be masked or omitted).

### Test: API Key Not Logged

1. Set Mattermost log level to DEBUG
2. Trigger a call join (which calls Cloudflare API)
3. Search server logs for the actual API key value

```bash
./build/bin/pluginctl logs com.kondo97.mattermost-plugin-rtk | grep -i "apikey\|api_key\|CloudflareAPIKey"
```

**Expected**: Logs contain no plaintext API key values.

---

## 5. Webhook HMAC Validation

### Test: Webhook with Invalid Signature

```bash
curl -X POST http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/webhook \
  -H "Content-Type: application/json" \
  -H "X-RTK-Signature: invalidsignature" \
  -d '{"event": "participant-joined", "sessionId": "test"}'
```

**Expected**: `401 Unauthorized` or `400 Bad Request`

### Test: Webhook without Signature

```bash
curl -X POST http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/webhook \
  -H "Content-Type: application/json" \
  -d '{"event": "participant-joined", "sessionId": "test"}'
```

**Expected**: `401 Unauthorized` or `400 Bad Request`

---

## 6. Input Validation Tests

### Test: Channel ID Injection

```bash
curl -X POST http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/calls/join \
  -H "Authorization: Bearer <user-token>" \
  -H "Content-Type: application/json" \
  -d '{"channel_id": "../../etc/passwd"}'
```

**Expected**: `400 Bad Request` (invalid channel ID format)

### Test: Oversized Payload

```bash
# Send 10MB payload
python3 -c "import sys; sys.stdout.write('{\"channel_id\": \"' + 'A'*10000000 + '\"}')" | \
  curl -X POST http://localhost:8065/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/calls/join \
  -H "Authorization: Bearer <user-token>" \
  -H "Content-Type: application/json" \
  --data-binary @-
```

**Expected**: `413 Request Entity Too Large` or `400 Bad Request`

---

## 7. RTK Token Scope Test

### Test: Token Bound to Session

1. User A joins a call → receives `rtk_token`
2. User B attempts to use User A's `rtk_token` in a different session
3. Cloudflare RTK must reject User B's connection

**Expected**: Cloudflare rejects the token (handled by Cloudflare, verify by testing manually with the RTK SDK).

---

## Security Sign-Off Checklist

- [ ] 0 HIGH/CRITICAL Go dependency vulnerabilities
- [ ] 0 HIGH/CRITICAL npm dependency vulnerabilities
- [ ] All unauthenticated API calls return 401
- [ ] Non-member/non-admin calls return 403
- [ ] CloudflareAPIKey never appears in API responses or logs
- [ ] Invalid webhook signatures rejected
- [ ] Input validation rejects malformed channel IDs
- [ ] golangci-lint passes with no security linter findings
