# Unit 1: RTK Integration — Functional Design Plan

## Execution Checklist

- [x] Analyze unit context (unit-of-work.md, story map, application design)
- [x] Generate clarifying questions
- [x] Collect answers
- [x] Analyze for ambiguities (none — all answers clear)
- [x] Generate functional design artifacts:
  - [x] `business-logic-model.md`
  - [x] `business-rules.md`
  - [x] `domain-entities.md`

---

## Unit Scope (from unit-of-work.md)

**Primary stories**: US-005, US-009, US-013, US-015, US-022, US-025
**Components**: RTKClient, KVStore (call extensions), Plugin Core (call lifecycle), Background Job

---

## Clarifying Questions

### Q1: CallSession Data Model — `preset` の保存

`CreateCall` 時にホストの preset (`group_call_host`) を KVStore の CallSession に保存しますか？

A) **保存する** — `CallSession.CreatorPreset` フィールドを持つ（EndCall 時の権限チェックに使用）

B) **保存しない** — ホスト判定は `CallSession.CreatorID == userID` のみで行う（preset は RTK 側で管理）

[Answer]:

---

### Q2: JoinCall — 参加上限

コールへの参加者数に上限を設けますか？

A) **上限なし** — KVStore に参加者がいる限り誰でも参加できる

B) **上限あり** — 設定可能な最大参加者数を設ける（超過時はエラー）

[Answer]: A — 上限なし

---

### Q3: HeartbeatCall — 参加者でないユーザーのハートビート

`HeartbeatCall(callID, userID)` が呼ばれた際、そのユーザーが当該コールの参加者リストに存在しない場合の挙動は？

A) **エラーを返す** — 参加者でないユーザーのハートビートは拒否する（セキュリティ上）

B) **無視する（成功扱い）** — 参加者チェックをせず、ハートビートタイムスタンプを更新する

[Answer]: A — エラーを返す

---

### Q4: CleanupStaleParticipants — コール単位 vs 全コール一括

Background Job の `CleanupStaleParticipants` は：

A) **全アクティブコール一括** — ジョブ実行時にすべてのアクティブコールをスキャンし、タイムアウトした参加者を一括で削除する

B) **コール単位** — 特定の callID を引数に取り、そのコールだけをクリーンアップする（ジョブはすべてのアクティブコールをループして呼び出す）

[Answer]: A — 全アクティブコール一括

---

### Q5: EndCall — RTK API の EndMeeting 呼び出し

`EndCall(callID)` 実行時に `RTKClient.EndMeeting(meetingID)` を呼び出しますか？

A) **呼び出す** — コール終了時に Cloudflare 側のミーティングも必ずクリーンアップする

B) **呼び出さない** — RTK ミーティングは TTL で自然消滅させる（KVStore 側のみ終了処理）

C) **呼び出す（ベストエフォート）** — 失敗してもエラーを無視してコール終了処理を続ける

[Answer]: C — ベストエフォート

---

### Q6: CreateCall — ポスト投稿の責務

`CreateCall` メソッドは Mattermost への `custom_cf_call` ポスト投稿も担いますか？

A) **担う** — `CreateCall` 内でポスト作成 + WebSocket イベント発火を一括で行う

B) **担わない** — ポスト作成は API ハンドラー（Unit 2）が行い、`CreateCall` はセッション作成のみ返す

[Answer]: A — CreateCall が全部担う
