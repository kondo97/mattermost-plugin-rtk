# Unit 1: RTK Integration — NFR Design Plan

## Execution Checklist

- [x] Analyze NFR requirements
- [x] Identify applicable design patterns
- [x] Generate clarifying questions (1 question)
- [x] Collect answers (Q1=B sentinel errors)
- [x] Generate NFR design artifacts:
  - [x] `nfr-design-patterns.md`
  - [x] `logical-components.md`

---

## Patterns to Apply

| NFR | Pattern |
|---|---|
| MAINT-01, MAINT-02 | Interface Abstraction (RTKClient, KVStore) |
| PERF-03 | HTTP Timeout Pattern (10s) |
| REL-01 | Fail-Fast (no retry) |
| REL-02, REL-03 | Best-Effort / Fire-and-Continue |
| SEC-03, SEC-04 | Authorization Guard (participant check, creator check) |
| MAINT-04, SECURITY-03 | Structured Logging |
| SECURITY-15 | Explicit Error Handling on all external calls |
| SECURITY-09 | Generic Error Messages to callers |

---

## Clarifying Questions

### Q1: ドメインエラー型の定義方針

`ErrCallAlreadyActive`、`ErrNotParticipant`、`ErrUnauthorized` などのエラーをどう定義しますか？

A) **`fmt.Errorf` / `errors.New`** — 標準ライブラリのみ。シンプルだが呼び出し側でエラー文字列比較が必要
B) **`sentinel errors`** — `var ErrCallAlreadyActive = errors.New("...")` のようにパッケージレベルで定義。`errors.Is()` で判定可能
C) **`custom error types`** — `type CallError struct { Code string; Msg string }` のような構造体。HTTP ステータスコードとのマッピングが明示的

[Answer]:
