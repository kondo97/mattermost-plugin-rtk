# Mattermost RTK Plugin — アーキテクチャ・実装ガイド

> **対象バージョン**: 現在の `main` ブランチ  
> **プラグイン ID**: `com.kondo97.mattermost-plugin-rtk`  
> **最低 Mattermost バージョン**: 10.11.0

---

## 目次

1. [概要](#1-概要)
2. [システム全体構成](#2-システム全体構成)
3. [サーバーサイドアーキテクチャ](#3-サーバーサイドアーキテクチャ)
   - 3.1 [Plugin ライフサイクル](#31-plugin-ライフサイクル)
   - 3.2 [HTTP ルーティング](#32-http-ルーティング)
   - 3.3 [コールビジネスロジック](#33-コールビジネスロジック)
   - 3.4 [Cloudflare RTK クライアント](#34-cloudflare-rtk-クライアント)
   - 3.5 [KVStore](#35-kvstore)
   - 3.6 [設定管理](#36-設定管理)
   - 3.7 [RTK Webhook](#37-rtk-webhook)
4. [フロントエンドアーキテクチャ](#4-フロントエンドアーキテクチャ)
   - 4.1 [デュアルバンドル構成](#41-デュアルバンドル構成)
   - 4.2 [Redux 状態管理](#42-redux-状態管理)
   - 4.3 [WebSocket イベント処理](#43-websocket-イベント処理)
   - 4.4 [UI コンポーネント](#44-ui-コンポーネント)
   - 4.5 [コールページ (スタンドアロン)](#45-コールページ-スタンドアロン)
5. [データモデル](#5-データモデル)
6. [主要フローの実装](#6-主要フローの実装)
   - 6.1 [コール開始](#61-コール開始)
   - 6.2 [コール参加](#62-コール参加)
   - 6.3 [退出 / 自動終了](#63-退出--自動終了)
   - 6.4 [ホストによる終了](#64-ホストによる終了)
   - 6.5 [RTK Webhook による退出検知](#65-rtk-webhook-による退出検知)
7. [API リファレンス](#7-api-リファレンス)
8. [WebSocket イベント](#8-websocket-イベント)
9. [設定リファレンス](#9-設定リファレンス)
10. [セキュリティ設計](#10-セキュリティ設計)
11. [ビルド構成](#11-ビルド構成)
12. [ディレクトリ構造](#12-ディレクトリ構造)

---

## 1. 概要

本プラグインは Mattermost チャンネルに **Cloudflare RealtimeKit (RTK)** を使ったビデオ・音声通話機能を追加する。

| 項目 | 内容 |
|------|------|
| バックエンド | Go 1.25 |
| フロントエンド | React 18 + TypeScript (Vite ビルド) |
| 通話エンジン | Cloudflare RealtimeKit v2 API |
| セッション永続化 | Mattermost KVStore |
| リアルタイム同期 | Mattermost WebSocket イベント + RTK Webhook |

---

## 2. システム全体構成

```
┌─────────────────────────────────────────────────────────────────────┐
│  ブラウザ (Mattermost Webapp)                                         │
│                                                                       │
│  ┌─────────────────────────────┐  ┌──────────────────────────────┐  │
│  │  メインバンドル (main.js)     │  │  コールページ (call.js)        │  │
│  │  ─────────────────────────  │  │  ──────────────────────────  │  │
│  │  ChannelHeaderButton        │  │  CallPage.tsx                │  │
│  │  CallPost                   │  │  useRealtimeKitClient()      │  │
│  │  ToastBar                   │  │  RtkMeeting UI               │  │
│  │  FloatingWidget  ←──RTK SDK─┼──┼─ /call?token=JWT            │  │
│  │  IncomingCallNotification   │  │  beforeunload→ POST /leave   │  │
│  │  AdminConfig                │  └──────────────────────────────┘  │
│  │  Redux (calls_slice)        │                                     │
│  │  WebSocket handlers         │                                     │
│  └────────────┬────────────────┘                                     │
└───────────────┼─────────────────────────────────────────────────────┘
                │ REST API / WebSocket
                ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Mattermost Server                                                    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Go Plugin (com.kondo97.mattermost-plugin-rtk)                │   │
│  │                                                                │   │
│  │  plugin.go          OnActivate / OnDeactivate                 │   │
│  │  configuration.go   設定管理・Env Var オーバーライド            │   │
│  │  calls.go           CreateCall / JoinCall / LeaveCall / End   │   │
│  │  api.go             gorilla/mux ルーター                       │   │
│  │  api_calls.go       POST/GET /calls, /token, /leave, DELETE   │   │
│  │  api_config.go      GET /config/status, /admin-status         │   │
│  │  api_mobile.go      POST /calls/{id}/dismiss                  │   │
│  │  api_static.go      GET /call, /call.js, /worker.js           │   │
│  │  api_webhook.go     POST /api/v1/webhook/rtk (HMAC検証)       │   │
│  │  rtkclient/         Cloudflare RTK API クライアント             │   │
│  │  store/kvstore/     KVStore 抽象化層                           │   │
│  └──────────────┬───────────────────────────────────┬────────────┘   │
│                 │                                   │                 │
│       Mattermost KVStore                  PublishWebSocketEvent       │
└─────────────────┼───────────────────────────────────┼───────────────┘
                  │                                   │ WebSocket
                  ▼                                   ▼ (全チャンネルメンバー)
      ┌───────────────────┐              ┌────────────────────────┐
      │  KVStore          │              │  ブラウザ WebSocket     │
      │  call:id:{id}     │              │  custom_{pluginID}_    │
      │  call:channel:{c} │              │  call_started 等       │
      │  call:meeting:{m} │              └────────────────────────┘
      │  webhook:id/secret│
      └───────────────────┘
                  ▲
                  │ Webhook (HMAC-SHA256)
      ┌───────────────────┐
      │  Cloudflare RTK   │
      │  api.realtime.    │
      │  cloudflare.com   │
      │  /v2              │
      └───────────────────┘
```

---

## 3. サーバーサイドアーキテクチャ

### 3.1 Plugin ライフサイクル

**エントリポイント**: `server/plugin.go`

```go
type Plugin struct {
    plugin.MattermostPlugin
    kvStore       kvstore.KVStore       // KVStore クライアント
    rtkClient     rtkclient.RTKClient   // Cloudflare RTK API (nil = 未設定)
    client        *pluginapi.Client     // Mattermost pluginapi
    commandClient command.Command       // スラッシュコマンド
    router        *mux.Router           // HTTP ルーター
    callMu        sync.Mutex            // コール状態を守る排他ロック
    stopCleanup   chan struct{}          // クリーンアップ goroutine 停止シグナル
    configurationLock sync.RWMutex     // 設定 Read/Write ロック
    configuration *configuration        // 現在の設定 (immutable pointer)
}
```

| フック | 処理内容 |
|--------|---------|
| `OnActivate()` | KVStore・RTKClient・ルーター初期化、Webhook 登録、クリーンアップ goroutine 起動 |
| `OnDeactivate()` | クリーンアップ goroutine 停止 |
| `OnConfigurationChange()` | 設定再読込、認証情報変更時に RTKClient 再初期化・Webhook 再登録 |
| `ExecuteCommand()` | `/hello` スラッシュコマンド処理 (スターターテンプレート由来) |

**Webhook 登録ロジック** (`registerWebhookIfNeeded`):
- KVStore に `webhook:id` と `webhook:secret` の両方が存在する場合はスキップ
- いずれか欠損時は RTK API に登録し、取得した ID と Secret を KVStore に保存
- 失敗はベストエフォート (WarningLog のみ)

**認証情報変更時** (`reRegisterWebhook`):
1. 既存 Webhook を RTK API で削除
2. KVStore の `webhook:id` と `webhook:secret` をクリア
3. `registerWebhookIfNeeded` を再実行

---

### 3.2 HTTP ルーティング

**ルーター**: `server/api.go` (gorilla/mux)

```
/call                          ← 認証不要 (静的 HTML)
/call.js                       ← 認証不要 (静的 JS)
/worker.js                     ← 認証不要 (静的 JS)
/api/v1/webhook/rtk            ← 認証不要 (HMAC-SHA256 で独自検証)

/api/v1/* ← MattermostAuthorizationRequired ミドルウェア (Mattermost-User-ID ヘッダー必須)
  POST   /calls                    コール開始
  GET    /calls/{id}               コール状態取得
  POST   /calls/{id}/token         コール参加 (トークン発行)
  POST   /calls/{id}/leave         コール退出
  DELETE /calls/{id}               コール終了 (ホストのみ)
  GET    /config/status            設定状態取得 (一般ユーザー)
  GET    /config/admin-status      設定状態取得 (管理者のみ)
  POST   /calls/{id}/dismiss       着信通知を無視
```

**認証ミドルウェア**:
```go
func MattermostAuthorizationRequired(next http.Handler) http.Handler {
    // Mattermost-User-ID ヘッダーが空なら 401 を返す
    // Mattermost Server が認証済みリクエストに付与するヘッダー
}
```

---

### 3.3 コールビジネスロジック

**実装ファイル**: `server/calls.go`

すべてのコール状態変更操作は `callMu sync.Mutex` を取得してから実行する。

#### CreateCall (コール開始)

```
callMu.Lock()
  1. GetCallByChannel → 既存コールがあれば ErrCallAlreadyActive
  2. rtkClient.CreateMeeting() → 失敗すれば即終了 (KVStore 書込みなし)
  3. rtkClient.GenerateToken(meetingID, userID, displayName, "group_call_host")
  4. CallSession 作成 (UUID, creator=participants[0], StartAt=nowMs())
  5. kvStore.SaveCall(session)  +  AddActiveCallID
  6. API.CreatePost(type="custom_cf_call")  [ベストエフォート]
  7. kvStore.SaveCall(session)  PostID を更新  [ベストエフォート]
  8. PublishWebSocketEvent("call_started", ...)  チャンネルにブロードキャスト
callMu.Unlock()
戻り値: (session, token.Token, nil)
```

#### JoinCall (コール参加)

```
callMu.Lock()
  1. GetCallByID → nil or EndAt!=0 なら ErrCallNotFound
  2. rtkClient.GenerateToken(meetingID, userID, displayName, "group_call_participant")
  3. Participants に userID を追加 (dedup)
  4. kvStore.UpdateCallParticipants
  5. updatePostParticipants  [ベストエフォート]
  6. PublishWebSocketEvent("user_joined", ...)  チャンネルにブロードキャスト
callMu.Unlock()
戻り値: (session, token.Token, nil)
```

#### LeaveCall (退出)

```
callMu.Lock()
  1. GetCallByID → nil or EndAt!=0 なら no-op (冪等)
  2. Participants から userID を削除
  3. kvStore.UpdateCallParticipants
  4. updatePostParticipants  [ベストエフォート]
  5. PublishWebSocketEvent("user_left", ...)
  6. Participants が空になったら → endCallInternal()  [自動終了]
callMu.Unlock()
```

#### EndCall (ホスト終了)

```
callMu.Lock()
  1. GetCallByID → nil or EndAt!=0 なら ErrCallNotFound
  2. CreatorID != requestingUserID なら ErrUnauthorized
  3. endCallInternal(session)
callMu.Unlock()
```

#### endCallInternal (共通終了処理)

```
  1. kvStore.EndCall(callID, nowMs())  +  RemoveActiveCallID
  2. rtkClient.EndMeeting(meetingID)  [ベストエフォート]
  3. API.UpdatePost: end_at, duration_ms をポストに書込み  [ベストエフォート]
  4. PublishWebSocketEvent("call_ended", {call_id, channel_id, end_at, duration_ms})
```

**エラー定義** (`server/errors.go`):

| 定数 | HTTP Status | 説明 |
|------|-------------|------|
| `ErrCallAlreadyActive` | 409 | チャンネルに既存コールあり |
| `ErrCallNotFound` | 404 | コール未存在または終了済み |
| `ErrNotParticipant` | 403 | 参加者ではない (現在未使用) |
| `ErrUnauthorized` | 403 | 作成者以外が終了操作 |
| `ErrRTKNotConfigured` | 503 | Cloudflare 認証情報未設定 |

---

### 3.4 Cloudflare RTK クライアント

**インターフェース**: `server/rtkclient/interface.go`  
**実装**: `server/rtkclient/client.go`

```go
type RTKClient interface {
    CreateMeeting() (*Meeting, error)
    GenerateToken(meetingID, userID, displayName, preset string) (*Token, error)
    EndMeeting(meetingID string) error
    RegisterWebhook(url string, events []string) (id, secret string, err error)
    DeleteWebhook(webhookID string) error
    GetMeetingParticipants(meetingID string) ([]string, error)
}
```

**通信仕様**:

| 項目 | 値 |
|------|----|
| ベース URL | `https://api.realtime.cloudflare.com/v2` |
| 認証方式 | HTTP Basic Auth (`orgID:apiKey`) |
| タイムアウト | 10 秒 |
| レスポンス形式 | `{ "success": bool, "data": T }` |

**主要エンドポイントマッピング**:

| メソッド | RTK API | 内容 |
|----------|---------|------|
| `CreateMeeting()` | `POST /meetings` | ミーティング作成 |
| `GenerateToken()` | `POST /meetings/{id}/participants` | 参加者追加 + JWT 発行 |
| `EndMeeting()` | `DELETE /meetings/{id}` | ミーティング終了 |
| `RegisterWebhook()` | `POST /webhooks` | Webhook 登録 |
| `DeleteWebhook()` | `DELETE /webhooks/{id}` | Webhook 削除 |
| `GetMeetingParticipants()` | `GET /meetings/{id}/active-participants` | 参加者一覧取得 |

**RTK プリセット**:

| 役割 | プリセット名 |
|------|-------------|
| コール作成者 (ホスト) | `group_call_host` |
| 参加者 | `group_call_participant` |

---

### 3.5 KVStore

**インターフェース**: `server/store/kvstore/kvstore.go`  
**実装**: `server/store/kvstore/calls.go`

**キースキーマ**:

| キーパターン | 値 | 説明 |
|-------------|-----|------|
| `call:channel:{channelID}` | CallSession (JSON) | チャンネルのアクティブコール |
| `call:id:{callID}` | CallSession (JSON) | ID によるコール検索 |
| `call:meeting:{meetingID}` | CallSession (JSON) | RTK MeetingID による検索 |
| `active_calls` | []string (JSON) | アクティブコール ID 一覧 |
| `voip:{userID}` | string | VoIP デバイストークン (将来用) |
| `webhook:id` | string | 登録済み RTK Webhook ID |
| `webhook:secret` | string | RTK Webhook 署名シークレット |

**同期戦略**: `SaveCall` / `UpdateCallParticipants` / `EndCall` はすべて `call:channel:*`、`call:id:*`、`call:meeting:*` の 3 キーを同時更新する。

---

### 3.6 設定管理

**実装**: `server/configuration.go`

**スレッドセーフ設計**:
- `sync.RWMutex` + `Clone()` パターン
- `getConfiguration()` は RLock で読取り、不変のコピーを返す
- `setConfiguration()` は Lock で書込み

**環境変数オーバーライド** (`os.LookupEnv` による厳密優先):

| 環境変数 | 設定項目 |
|----------|---------|
| `RTK_ORG_ID` | CloudflareOrgID |
| `RTK_API_KEY` | CloudflareAPIKey |
| `RTK_RECORDING_ENABLED` | RecordingEnabled |
| `RTK_SCREEN_SHARE_ENABLED` | ScreenShareEnabled |
| `RTK_POLLS_ENABLED` | PollsEnabled |
| `RTK_TRANSCRIPTION_ENABLED` | TranscriptionEnabled |
| `RTK_WAITING_ROOM_ENABLED` | WaitingRoomEnabled |
| `RTK_VIDEO_ENABLED` | VideoEnabled |
| `RTK_CHAT_ENABLED` | ChatEnabled |
| `RTK_PLUGINS_ENABLED` | PluginsEnabled |
| `RTK_PARTICIPANTS_ENABLED` | ParticipantsEnabled |
| `RTK_RAISE_HAND_ENABLED` | RaiseHandEnabled |

**フィーチャーフラグのデフォルト**: `*bool` フィールドが `nil` の場合は `true` (有効) として扱う。

**`OnConfigurationChange` フロー**:
```
新設定を LoadPluginConfiguration で読込
認証情報が変わった場合:
  ├── 新認証情報が揃っている → RTKClient 再初期化 + Webhook 再登録
  └── 認証情報が消えた      → RTKClient = nil
```

---

### 3.7 RTK Webhook

**実装**: `server/api_webhook.go`

RTK から以下のイベントが `POST /api/v1/webhook/rtk` に届く:

| イベント | 処理 |
|---------|------|
| `meeting.participantLeft` | `GetCallByMeetingID` → `LeaveCall(callID, userID)` |
| `meeting.ended` | `GetCallByMeetingID` → `callMu.Lock()` → 再確認 → `endCallInternal()` |
| その他 | 無視 (200 OK) |

**署名検証**:
```
HMAC-SHA256(secret, rawBody) == hex(dyte-signature ヘッダー)
secret が空の場合は常に拒否 (401)
```

**`meeting.ended` の二重終了防止**:
- ロック取得前に `EndAt != 0` を確認
- ロック取得後に KVStore から再読込して再確認 (TOCTOU 対策)

---

## 4. フロントエンドアーキテクチャ

### 4.1 デュアルバンドル構成

Vite で **2 つの独立したバンドル**をビルドする。

| バンドル | エントリ | 出力 | 用途 |
|---------|---------|------|------|
| `main.js` | `src/index.tsx` | `webapp/dist/main.js` | Mattermost プラグイン本体 |
| `call.js` | `src/call_page/main.tsx` | `webapp/dist/call.js` | スタンドアロンコールページ |

**`main.js` の外部依存 (Mattermost が提供)**:
`React`, `ReactDOM`, `Redux`, `ReactRedux`, `ReactIntl`, `PropTypes`, `ReactBootstrap`, `ReactRouterDom`

**`call.js` は完全自己完結**: React を含めすべてをバンドル。

**CSP 回避パッチ** (`workerTimersCspPatch`):  
`@cloudflare/realtimekit` の依存 `worker-timers` が `blob:` URL で Web Worker を生成しようとするが Mattermost の CSP でブロックされる。Vite ビルド時に Go プラグインが提供する `/plugins/{id}/worker.js` への静的 URL に置換する。

---

### 4.2 Redux 状態管理

**実装**: `src/redux/calls_slice.ts`

```typescript
interface CallsPluginState {
    callsByChannel: Record<string, ActiveCall>; // チャンネル別アクティブコール
    myActiveCall:   MyActiveCall | null;         // 自分が参加中のコール + JWT
    incomingCall:   IncomingCall | null;         // 着信中のコール (DM/GM のみ)
    pluginEnabled:  boolean;                     // /config/status の enabled 値
}
```

**型定義**:

```typescript
interface ActiveCall {
    id: string;           // コール UUID
    channelId: string;
    creatorId: string;
    participants: string[]; // Mattermost userID 配列
    startAt: number;      // Unix ms
    postId: string;
}

interface MyActiveCall {
    callId: string;
    channelId: string;
    token: string;        // RTK JWT — ログ出力禁止
}

interface IncomingCall {
    callId: string;
    channelId: string;
    creatorId: string;
    startAt: number;
}
```

**セレクター** (`src/redux/selectors.ts`):

```typescript
// プラグイン状態を state.plugins-{pluginId} から取得
selectPluginEnabled(state)
selectCallByChannel(channelId)(state)
selectMyActiveCall(state)
selectIncomingCall(state)
selectIsCurrentUserParticipant(channelId, currentUserId)(state)
```

---

### 4.3 WebSocket イベント処理

**実装**: `src/redux/websocket_handlers.ts`

各ハンドラは `index.tsx` の `initialize()` で `registry.registerWebSocketEventHandler` に登録される。

| サーバー公開名 | クライアント受信名 | ハンドラ | 主な処理 |
|--------------|------------------|---------|---------|
| `call_started` | `custom_{pluginID}_call_started` | `handleCallStarted` | `upsertCall` + DM/GM ならば `setIncomingCall` |
| `user_joined` | `custom_{pluginID}_user_joined` | `handleUserJoined` | 既存コールの participants を更新 (`upsertCall`) |
| `user_left` | `custom_{pluginID}_user_left` | `handleUserLeft` | participants 更新、自分なら `clearMyActiveCall` |
| `call_ended` | `custom_{pluginID}_call_ended` | `handleCallEnded` | `removeCall` + `clearMyActiveCall` + `clearIncomingCall` |
| `notification_dismissed` | `custom_{pluginID}_notification_dismissed` | `handleNotifDismissed` | 自分宛なら `clearIncomingCall` |

> **プレフィックスの仕組み**: Go 側は `p.API.PublishWebSocketEvent("call_started", ...)` とシンプルな名前で publish する。Mattermost サーバーが自動的に `custom_{pluginID}_` を付与してブラウザに配信する。

**ペイロード型ガード**: すべてのハンドラはランタイム型チェック関数 (`isCallStartedPayload` 等) でペイロードを検証し、不正なデータは `console.error` してスキップする。

---

### 4.4 UI コンポーネント

#### ChannelHeaderButton (`src/components/channel_header_button/`)

チャンネルヘッダーに表示されるコールボタン。

| 状態 | 表示 |
|------|------|
| コールなし | 「通話を開始」(電話アイコン) |
| コールあり・未参加 | 「参加する」+ 緑ドット |
| 自分が参加中 | 「通話中」(無効化) + 緑ドット |
| 処理中 | スピナーアイコン |

- コール開始: `POST /api/v1/calls`
- コール参加: `POST /api/v1/calls/{id}/token`
- 別コール参加中の場合: `SwitchCallModal` を表示してから退出 → 参加

#### CallPost (`src/components/call_post/`)

カスタムポストタイプ `custom_cf_call` のレンダラー。

- マウント時に `GET /api/v1/calls/{id}` で最新状態を取得 (ページリロード後の再同期)
- `EndAt == 0`: `CallPostActive` (参加ボタン、参加者数、経過時間)
- `EndAt > 0`: `CallPostEnded` (終了時刻、通話時間)
- Redux ストアのライブ状態は `liveCall.id === post.call_id` の一致確認後のみ使用

#### ToastBar (`src/components/toast_bar/`)

メッセージ入力欄上部に表示されるバナー。

- 表示条件: 現在チャンネルにアクティブコールあり かつ 自分が未参加 かつ 非 dismissed
- `dismissed` はコンポーネントローカルの state (タブ内のみ有効)
- 参加ボタン押下で `POST /api/v1/calls/{id}/token`

#### FloatingWidget (`src/components/floating_widget/`)

Mattermost 内に浮かぶインコール UI ウィジェット。

- `myActiveCall` が Redux にある場合に表示
- `@cloudflare/realtimekit-react` の `useRealtimeKitClient()` + `RtkMeeting` で RTK SDK を埋め込み
- 日本語ロケール時は `rtk_lang_ja.ts` の辞書を `useLanguage()` に渡す
- 最小化・最大化 (フルスクリーン) 対応
- ドラッグで位置変更可能 (`position: fixed`, `right`/`bottom` 更新)
- `beforeunload` イベントで `fetch + keepalive` により `POST /leave` を送信
- 終了時: `meeting.leaveRoom()` → `POST /leave` → `clearMyActiveCall`
- 接続失敗時: 最大 3 回 (2 秒間隔) リトライ、それでも失敗ならエラーメッセージを表示

#### IncomingCallNotification (`src/components/incoming_call_notification/`)

画面右上に表示される着信通知。

- `incomingCall` が Redux にある場合に表示 (DM/GM チャンネルのみ)
- 30 秒後に自動消去
- 「無視」ボタン: `POST /api/v1/calls/{id}/dismiss` → WebSocket 経由で全セッションに伝播
- 「参加」ボタン: `POST /api/v1/calls/{id}/token` → `setMyActiveCall`

#### SwitchCallModal (`src/components/switch_call_modal/`)

別のコールに参加しようとした際の確認ダイアログ。  
ChannelHeaderButton / CallPost / ToastBar / IncomingCallNotification の各コンポーネントが共有利用する。

#### EnvVarCredentialSetting (`src/components/admin_config/`)

Admin Console の `CloudflareOrgID` / `CloudflareAPIKey` 設定欄の代替レンダラー。

- マウント時に `GET /api/v1/config/admin-status` を呼び、`org_id_via_env` / `api_key_via_env` を確認
- 環境変数が設定されている場合: 読み取り専用テキスト + 環境変数名の案内を表示 (入力欄を無効化)
- 通常時: `type="text"` または `type="password"` の入力欄を表示

---

### 4.5 コールページ (スタンドアロン)

**エントリ**: `src/call_page/main.tsx`  
**コンポーネント**: `src/call_page/CallPage.tsx`

Mattermost とは独立したスタンドアロン SPA。URL パラメータを解析してコールを初期化する。

**URL フォーマット**:
```
/plugins/com.kondo97.mattermost-plugin-rtk/call
  ?token={RTK JWT}
  &call_id={callID}
  &channel_name={チャンネル名}
  [&embedded=1]
  [&locale=ja]
```

**RTK SDK 初期化シーケンス**:
1. `useRealtimeKitClient()` で `[meeting, initMeeting]` を取得
2. `initMeeting({ authToken: token, defaults: {audio: true, video: true} })` を呼ぶ
3. 失敗時は最大 3 回 (2 秒間隔) リトライ
4. `meeting` が解決したら `RtkMeeting` コンポーネントで UI をレンダリング

**タブクローズ時の退出**:
- `beforeunload` で `fetch + keepalive` (カスタムヘッダーが必要なため `sendBeacon` は使用不可)
- `embedded=1` のとき (FloatingWidget の iframe 内) はスキップ

**ページタイトル**: `channel_name` パラメータから `Call in #channel-name` を設定。

---

## 5. データモデル

### CallSession

```go
type CallSession struct {
    ID           string   `json:"id"`            // UUID (コール識別子)
    ChannelID    string   `json:"channel_id"`    // Mattermost チャンネル ID
    CreatorID    string   `json:"creator_id"`    // ホストの Mattermost userID
    MeetingID    string   `json:"meeting_id"`    // Cloudflare RTK ミーティング ID
    Participants []string `json:"participants"`  // 現在の参加者 userID 配列 (dedup)
    StartAt      int64    `json:"start_at"`      // 開始 Unix タイムスタンプ (ms)
    EndAt        int64    `json:"end_at"`        // 終了 Unix タイムスタンプ (ms); 0=アクティブ
    PostID       string   `json:"post_id"`       // custom_cf_call ポスト ID
}
```

**状態判定**:
- `EndAt == 0`: アクティブ
- `EndAt > 0`: 終了済み

### カスタムポスト Props (type: `custom_cf_call`)

```json
{
  "call_id":      "uuid",
  "channel_id":   "string",
  "creator_id":   "string",
  "participants": ["userID"],
  "start_at":     1234567890000,
  "end_at":       0,
  "duration_ms":  720000
}
```

---

## 6. 主要フローの実装

### 6.1 コール開始

```
ユーザー           ChannelHeaderButton        Go Plugin              Cloudflare RTK
   │                     │                       │                        │
   │──クリック───────────▶│                       │                        │
   │                     │──POST /api/v1/calls──▶│                        │
   │                     │   {channel_id}        │──POST /meetings───────▶│
   │                     │                       │◀──{meetingID}──────────│
   │                     │                       │──POST /meetings/{id}/──▶│
   │                     │                       │    participants        │
   │                     │                       │  (host preset)        │
   │                     │                       │◀──{JWT token}──────────│
   │                     │                       │                        │
   │                     │                       │──SaveCall──▶ KVStore
   │                     │                       │──CreatePost (custom_cf_call)
   │                     │                       │──PublishWebSocketEvent("call_started")
   │                     │◀──201 {call, token}───│
   │                     │                       │
   │                     │ dispatch(upsertCall)   │
   │                     │ dispatch(setMyActiveCall{token})
   │                     │                       │
   │◀──FloatingWidget 表示│                       │
   │  (RTK SDK 初期化)    │                       │
```

### 6.2 コール参加

```
他ユーザー         ChannelHeaderButton/ToastBar   Go Plugin         Cloudflare RTK
    │                     │                          │                    │
    │ (call_started WS)   │                          │                    │
    │◀──────────────────── Redux: upsertCall          │                    │
    │                     │                          │                    │
    │──クリック───────────▶│                          │                    │
    │                     │──POST /calls/{id}/token─▶│                    │
    │                     │                          │──POST /participants▶│
    │                     │                          │  (participant preset)│
    │                     │                          │◀──{JWT token}───────│
    │                     │                          │──UpdateCallParticipants
    │                     │                          │──PublishWebSocketEvent("user_joined")
    │                     │◀──200 {call, token}──────│
    │                     │                          │
    │◀──FloatingWidget 表示│                          │
```

### 6.3 退出 / 自動終了

```
FloatingWidget (×ボタン or beforeunload)
  │
  │──meeting.leaveRoom() ──→ RTK SDK: roomLeft イベント
  │──POST /api/v1/calls/{id}/leave
  │     ↓
  │  LeaveCall():
  │    Participants から削除
  │    UpdateCallParticipants
  │    PublishWebSocketEvent("user_left")
  │    └── Participants が空 → endCallInternal()
  │          EndCall in KVStore
  │          rtkClient.EndMeeting (ベストエフォート)
  │          UpdatePost (end_at, duration_ms)
  │          PublishWebSocketEvent("call_ended")
```

### 6.4 ホストによる終了

```
ChannelHeaderButton (未実装) / CallPost の終了ボタン
  │
  │──DELETE /api/v1/calls/{id}
  │     ↓
  │  EndCall():
  │    CreatorID 確認 (≠ requestingUserID → 403)
  │    endCallInternal()
```

### 6.5 RTK Webhook による退出検知

```
Cloudflare RTK
    │
    │──POST /api/v1/webhook/rtk
    │   dyte-signature: HMAC-SHA256(secret, body)
    │
    ├── meeting.participantLeft:
    │     GetCallByMeetingID
    │     LeaveCall(session.ID, participant.customParticipantId)
    │       └── (LeaveCall が callMu を内部で取得)
    │
    └── meeting.ended:
          GetCallByMeetingID (ロック前確認)
          callMu.Lock()
          GetCallByID (TOCTOU 対策の再確認)
          endCallInternal()
          callMu.Unlock()
```

---

## 7. API リファレンス

### POST /api/v1/calls — コール開始

**リクエスト**:
```json
{ "channel_id": "string" }
```

**レスポンス** (201 Created):
```json
{
  "call": {
    "id": "uuid",
    "channel_id": "string",
    "creator_id": "string",
    "meeting_id": "string",
    "participants": ["userID"],
    "start_at": 1234567890000,
    "end_at": 0,
    "post_id": "string"
  },
  "token": "RTK JWT"
}
```

**エラー**: 400 (channel_id なし) / 409 (コール既存) / 503 (RTK 未設定)

---

### POST /api/v1/calls/{id}/token — コール参加

**レスポンス** (200 OK):
```json
{ "call": { ...CallSession... }, "token": "RTK JWT" }
```

**エラー**: 404 (コール未存在・終了済み) / 503 (RTK 未設定)

---

### GET /api/v1/calls/{id} — コール状態取得

**レスポンス** (200 OK): `CallSession` オブジェクト直接

---

### POST /api/v1/calls/{id}/leave — 退出

**レスポンス**: 200 OK (冪等)

---

### DELETE /api/v1/calls/{id} — コール終了 (ホストのみ)

**レスポンス**: 200 OK  
**エラー**: 403 (ホスト以外) / 404 (コール未存在)

---

### GET /api/v1/config/status — 設定状態 (一般ユーザー)

**レスポンス**:
```json
{
  "enabled": true,
  "feature_flags": {
    "recording": true, "screenShare": true, "polls": true,
    "transcription": true, "waitingRoom": false, "video": true,
    "chat": true, "plugins": true, "participants": true, "raiseHand": true
  }
}
```

---

### GET /api/v1/config/admin-status — 設定状態 (管理者のみ)

**レスポンス**:
```json
{
  "enabled": true,
  "org_id_via_env": false,
  "api_key_via_env": true,
  "cloudflare_org_id": "abc123",
  "feature_flags": { ...同上... }
}
```

**エラー**: 403 (管理者権限なし)

---

### POST /api/v1/calls/{id}/dismiss — 着信通知を無視

RTK JWT 不要。`notification_dismissed` WebSocket イベントを発行者のみに送信。  
**レスポンス**: 200 OK (冪等)

---

## 8. WebSocket イベント

> サーバーは短縮名で `PublishWebSocketEvent` を呼ぶ。Mattermost が `custom_{pluginID}_` を付与してクライアントに配信する。  
> クライアント受信名 = `custom_com.kondo97.mattermost-plugin-rtk_{短縮名}`

### call_started

```json
{
  "call_id": "string",
  "channel_id": "string",
  "creator_id": "string",
  "participants": ["userID"],
  "start_at": 1234567890000,
  "post_id": "string"
}
```
ブロードキャスト範囲: チャンネル全メンバー

### user_joined

```json
{
  "call_id": "string",
  "channel_id": "string",
  "user_id": "string",
  "participants": ["userID"]
}
```
ブロードキャスト範囲: チャンネル全メンバー  
**注意**: 参加済みユーザーの再 Join でも emit される (Participants リスト は dedup 済み)

### user_left

```json
{
  "call_id": "string",
  "channel_id": "string",
  "user_id": "string",
  "participants": ["userID"]
}
```
ブロードキャスト範囲: チャンネル全メンバー

### call_ended

```json
{
  "call_id": "string",
  "channel_id": "string",
  "end_at": 1234567890000,
  "duration_ms": 720000
}
```
ブロードキャスト範囲: チャンネル全メンバー

### notification_dismissed

```json
{ "call_id": "string", "user_id": "string" }
```
ブロードキャスト範囲: 発行者のみ (全セッション)

---

## 9. 設定リファレンス

`plugin.json` の `settings_schema` で定義。管理者は System Console から設定できる。

| キー | 型 | 説明 |
|------|----|------|
| `CloudflareOrgID` | text | Cloudflare Organization ID |
| `CloudflareAPIKey` | text (secret) | Cloudflare API Key |
| `RecordingEnabled` | bool (default: true) | 録画機能 |
| `ScreenShareEnabled` | bool (default: true) | 画面共有 |
| `PollsEnabled` | bool (default: true) | 投票機能 |
| `TranscriptionEnabled` | bool (default: true) | リアルタイム文字起こし |
| `WaitingRoomEnabled` | bool (default: true) | 待合室 |
| `VideoEnabled` | bool (default: true) | カメラ映像 |
| `ChatEnabled` | bool (default: true) | インコールチャット |
| `PluginsEnabled` | bool (default: true) | サードパーティプラグイン |
| `ParticipantsEnabled` | bool (default: true) | 参加者パネル |
| `RaiseHandEnabled` | bool (default: true) | 挙手機能 |

環境変数は System Console 設定に**厳密優先**する (`os.LookupEnv` で空文字も優先)。

---

## 10. セキュリティ設計

### 認証・認可

| レイヤー | 仕組み |
|---------|--------|
| 一般 API | `Mattermost-User-ID` ヘッダー (Mattermost が認証済みリクエストに付与) |
| 管理者 API | `model.PermissionManageSystem` 確認 |
| RTK Webhook | HMAC-SHA256 署名検証 (`dyte-signature` ヘッダー) |
| 静的ファイル | 認証不要 (call.html, call.js, worker.js) |

### 機密情報保護

- Cloudflare API Key はフロントエンドに一切返さない
- RTK JWT トークンはログ出力禁止 (`token` フィールドをログで `len()` のみ記録)
- `GetEffectiveAPIKey()` の戻り値はどこにもログ出力しない

### コールページ CSP

```
default-src 'self';
script-src 'self' 'unsafe-eval' 'wasm-unsafe-eval';  ← RTK WASM
connect-src *;                                         ← WebRTC/WS
style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;
font-src 'self' https://fonts.gstatic.com;
img-src 'self' blob: data:;
worker-src 'self' blob:;                              ← Web Worker
media-src *;                                           ← 音声・映像
```

### CSRF 対策

`X-Requested-With: XMLHttpRequest` ヘッダーをすべての API リクエストに付与 (`client.ts`)。

### ホスト認可

コール終了 (`DELETE /calls/{id}`) は `CreatorID == requestingUserID` を厳密チェック。

---

## 11. ビルド構成

### サーバービルド

```bash
make build
```

| OS/Arch | 出力 |
|---------|------|
| linux/amd64 | `server/dist/plugin-linux-amd64` |
| linux/arm64 | `server/dist/plugin-linux-arm64` |
| darwin/amd64 | `server/dist/plugin-darwin-amd64` |
| darwin/arm64 | `server/dist/plugin-darwin-arm64` |
| windows/amd64 | `server/dist/plugin-windows-amd64.exe` |

### フロントエンドビルド

```bash
cd webapp

# メインバンドル (Mattermost プラグイン本体)
npm run build

# コールページバンドル (スタンドアロン)
VITE_BUILD_TARGET=call npm run build
```

### 静的ファイルの埋め込み

`api_static.go` で Go の `//go:embed` ディレクティブを使用:

```go
//go:embed assets/call.html
var callHTML []byte

//go:embed assets/call.js
var callJS []byte

//go:embed assets/worker.js
var workerJS []byte
```

`assets/call.js` はビルド後の `webapp/dist/call.js` をコピーしたもの。

---

## 12. ディレクトリ構造

```
mattermost-plugin-rtk/
├── plugin.json                    # プラグインマニフェスト・設定スキーマ
├── go.mod / go.sum
├── Makefile
│
├── server/
│   ├── main.go                    # エントリポイント
│   ├── manifest.go                # 自動生成マニフェストローダー
│   ├── plugin.go                  # Plugin 構造体・OnActivate・OnDeactivate
│   ├── configuration.go           # 設定管理・Env Var オーバーライド
│   ├── calls.go                   # コールビジネスロジック
│   ├── errors.go                  # センチネルエラー定義
│   ├── cleanup.go                 # スタブ (将来のクリーンアップ用)
│   ├── api.go                     # gorilla/mux ルーター初期化・認証ミドルウェア
│   ├── api_calls.go               # コール API ハンドラ
│   ├── api_config.go              # 設定状態 API ハンドラ
│   ├── api_mobile.go              # dismiss API ハンドラ
│   ├── api_static.go              # 静的ファイル配信 (go:embed)
│   ├── api_webhook.go             # RTK Webhook ハンドラ
│   ├── assets/
│   │   ├── call.html              # コールページ HTML
│   │   ├── call.js                # コールページ JS (webapp/dist/call.js のコピー)
│   │   └── worker.js              # Web Worker JS
│   ├── rtkclient/
│   │   ├── interface.go           # RTKClient インターフェース・型定義
│   │   ├── client.go              # HTTP クライアント実装
│   │   └── mocks/mock_rtkclient.go
│   ├── store/kvstore/
│   │   ├── kvstore.go             # KVStore インターフェース
│   │   ├── models.go              # CallSession 型定義
│   │   ├── calls.go               # KVStore 操作実装
│   │   ├── startertemplate.go     # スターターテンプレート由来のメソッド
│   │   └── mocks/mock_kvstore.go
│   └── command/
│       ├── command.go             # /hello スラッシュコマンド
│       └── mocks/mock_commands.go
│
├── webapp/
│   ├── package.json
│   ├── vite.config.ts             # デュアルバンドル設定・CSP パッチ
│   ├── src/
│   │   ├── index.tsx              # プラグインエントリ・初期化・WS/UI 登録
│   │   ├── manifest.ts            # plugin.json を参照
│   │   ├── client.ts              # pluginFetch ユーティリティ
│   │   ├── call_page/
│   │   │   ├── main.tsx           # スタンドアロンコールページエントリ
│   │   │   └── CallPage.tsx       # RTK SDK 初期化・RtkMeeting UI
│   │   ├── components/
│   │   │   ├── channel_header_button/  # コールボタン (開始/参加/通話中)
│   │   │   ├── call_post/              # custom_cf_call ポストレンダラー
│   │   │   ├── toast_bar/              # チャンネルトーストバー
│   │   │   ├── floating_widget/        # インコール浮動ウィジェット (RTK UI)
│   │   │   ├── incoming_call_notification/  # 着信通知
│   │   │   ├── switch_call_modal/      # 通話切替確認モーダル
│   │   │   └── admin_config/           # 管理者設定 (Env Var 表示)
│   │   ├── redux/
│   │   │   ├── calls_slice.ts     # Redux レデューサー・アクション・型定義
│   │   │   ├── websocket_handlers.ts  # WS イベントハンドラ
│   │   │   └── selectors.ts       # Redux セレクター
│   │   └── utils/
│   │       ├── call_tab.ts        # コールページ URL ビルダー
│   │       └── rtk_lang_ja.ts     # RTK SDK 日本語辞書
│   └── i18n/
│       ├── en.json                # 英語翻訳
│       └── ja.json                # 日本語翻訳
│
└── aidlc-docs/                    # AI-DLC 方法論による設計ドキュメント
    ├── aidlc-state.md
    ├── audit.md
    ├── inception/
    └── construction/
```
