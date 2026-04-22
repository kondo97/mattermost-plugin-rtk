# Feature Flag Management

このドキュメントは、RTKプラグインで管理するfeature flagの一覧と、Cloudflare RTK SDK/APIがサポートする機能との対応状況を管理します。

---

## 1. 概要

### 動作原理

Feature flagは以下の優先順位で評価されます（高い順）：

```
環境変数 (RTK_*_ENABLED)
  ↓ なければ
System Console 設定値 (*bool フィールド)
  ↓ nil（未設定）なら
プログラムデフォルト（基本的に true）
```

実装: `server/configuration.go` の `isFeatureFlagEnabled()`

### 制御フロー

```
plugin.json (settings_schema)
  ↓ OnConfigurationChange
configuration.go (Is*Enabled() メソッド)
  ↓
api_config.go (configFeatureFlags())
  ├→ GET /api/v1/config/status  →  webapp (Redux)  →  RTK SDK initMeeting()
  ├→ POST /api/v1/calls         →  RTK API CreateMeeting()
  └→ POST /api/v1/calls/:id/join
```

---

## 2. 現行の Feature Flags（3件）

> **凡例**: ✅ 公式SDKサポート / ⚠️ ワークアラウンド

| Flag | Default | 実装方式 | ステータス |
|---|---|---|---|
| `screenShare` | `true` | Client: `self.disableScreenShare()` | ⚠️ SDK moduleではなくメソッド呼び出しで代替 |
| `video` | `true` | Client: `defaults.video` | ✅ 公式SDKサポート |
| `participants` | `true` | Client: `modules.participant` | ✅ 公式SDKサポート |

**クライアント実装箇所**: `webapp/src/call_page/CallPage.tsx` の `initMeeting()` 呼び出し

---

## 3. 未対応機能一覧

現在フラグ・パラメータが存在しない機能をまとめます。対象は以下の3カテゴリです。

- **Feature flag**: 過去に検討・実装試みがあったが現在は未対応のフラグ
- **SDKモジュール**: `@cloudflare/realtimekit` の `modules` に指定できるもの
- **APIパラメータ**: `POST /v2/meetings` で指定できるもの（実装箇所: `server/rtkclient/client.go`）

### サポートしない（確定）

| 機能 | カテゴリ | 理由 |
|---|---|---|
| `polls` | Feature flag | このプラグインではサポートしない |
| `waitingRoom` | Feature flag | このプラグインではサポートしない |
| `chat` | Feature flag | このプラグインではサポートしない |
| `plugins` | Feature flag | このプラグインではサポートしない |
| `pip` | SDK module | Mattermostのブラウザ環境での有用性が低い |
| `theme` | SDK module | Mattermostが独自テーマを持つため競合する |
| `stage` | SDK module | ウェビナー用途。チームチャットの用途と異なる |
| `connectedMeetings` | SDK module | クロスミーティング連携。Mattermost統合での用途が不明確 |
| `tracing` | SDK module | 開発者デバッグ用途のみ |
| `internals` | SDK module | SDK内部用フラグ |
| `devTools` | SDK module | 開発環境のみ。本番には不要 |
| `experimentalAudioPlayback` | SDK module | 実験的機能。安定性未保証 |
| `record_on_start` | API parameter | 現行の `recording` flagはUIモジュールのトグル。自動録画開始は別機能として必要になれば追加 |
| `live_stream_on_start` | API parameter | `livestream` モジュール対応時に合わせて再検討（現時点ではサポートしない） |

### 要検討（将来対応候補）

| 機能 | カテゴリ | Default（サポート時） | 理由・備考 |
|---|---|---|---|
| `recording` | Feature flag | `false` | 将来サポート予定。UIモジュールとしての録画UI表示を検討 |
| `transcription` | Feature flag | `false` | 将来サポート予定。`ai_config.transcription.*` APIパラメータとセットで検討 |
| `raiseHand` | Feature flag | `true` | 公式の制御方法はUIKit Addons の `canRaiseHand` / `canManageRaisedHand`。現在サーバー・クライアントとも未実装。UIKit Addons方式での再実装を検討 |
| `livestream` | SDK module | `true` | Cloudflare Live Streams連携が必要。`live_stream_on_start` APIパラメータとセットで検討 |
| `e2ee` | SDK module | `false` | セキュリティ上有益だが、キーマネージャー実装が必要（複雑な追加作業） |
| `persist_chat` | API parameter | — | チャット履歴保持（1週間）は有用だが、`chat` flagとは別軸の設定 |
| `summarize_on_end` | API parameter | — | AI要約は高付加価値だが `ai_config` と依存関係あり。要PoC |
| `ai_config.transcription.*` | API parameter | — | 現行の `transcription_enabled`（未公式）の置き換え候補。公式の正式パラメータ |

