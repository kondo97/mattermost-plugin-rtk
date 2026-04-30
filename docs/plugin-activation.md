# Plugin Activation (Startup)

This document describes what happens when the RTK plugin is activated by the Mattermost server, i.e. the body of the `OnActivate` hook in `server/plugin.go`. It covers the order of operations, the side effects performed against the database and the Cloudflare RTK API, and how failures are handled.

The scope is intentionally limited to the **server-side activation sequence**. Configuration changes (`OnConfigurationChange`), deactivation (`OnDeactivate`), and webapp-side initialization are out of scope.

---

## Overview

`OnActivate` is invoked by the Mattermost plugin framework whenever the plugin is enabled or reloaded. If it returns a non-nil error, the plugin is deactivated and the error is surfaced in the server log.

The activation pipeline performs the following, in order:

1. Wrap the plugin API in a typed client (`pluginapi.Client`).
2. Open the master database handle and run schema migrations.
3. Read the active plugin configuration (settings + environment variables).
4. Provision (or recover) the single Cloudflare RTK app for this Mattermost instance — `EnsureApp`.
5. Build the app-scoped RTK client and the business-logic layer (`app.App`).
6. Register (or verify) the RTK webhook that delivers meeting events back to this plugin.
7. Initialize the HTTP API handler with the embedded static assets.

Phases 1 and 2 are **fatal**: any failure aborts activation. Phases 4 and 6 are **best-effort**: failures are logged and activation continues so that the operator can fix credentials or networking later via `OnConfigurationChange`.

---

## Startup Sequence

```mermaid
sequenceDiagram
    participant MM as Mattermost server
    participant P as Plugin (OnActivate)
    participant DB as PostgreSQL (master DB)
    participant CF as Cloudflare RTK API

    MM->>P: OnActivate()
    P->>P: pluginapi.NewClient(API, Driver)
    P->>DB: Store.GetMasterDB()
    P->>DB: sqlstore.NewStore(db)
    P->>DB: Store.RunMigrations() (morph)
    Note over P,DB: Fatal on error — activation aborts.

    P->>P: getConfiguration() (settings + env vars)

    alt credentials present
        P->>CF: AccountClient.ListApps()
        alt app named "mm-<siteURL>" exists
            P->>DB: Store.GetAppID() / GetLatestAppConfigID()
            opt app ID changed
                P->>DB: Store.StoreAppConfig(...)
            end
        else not found
            P->>CF: AccountClient.CreateApp(name)
            P->>DB: Store.StoreAppConfig(...)
        end
    end
    Note over P,CF: EnsureApp errors are logged, not fatal.

    alt accountID + appID + token are all set
        P->>P: rtkclient.NewClient(...)
    end

    P->>P: app.New(store, rtk, account, API)

    opt rtk client is non-nil
        P->>DB: Store.GetWebhookConfig()
        alt stored ID exists
            P->>CF: GetWebhook(id)
            alt 404 (deleted)
                P->>DB: ClearWebhookConfig(...)
                P->>CF: RegisterWebhook(url, events)
            end
        else no stored ID
            P->>CF: RegisterWebhook(url, events)
            opt 409 conflict
                P->>CF: ListWebhooks() / DeleteWebhook(id)
                P->>CF: RegisterWebhook(url, events)
            end
        end
        P->>DB: StoreWebhookConfig(...)
    end
    Note over P,CF: RegisterWebhookIfNeeded errors are logged, not fatal.

    P->>P: rtapi.Init(application, staticFiles, configStatus)
    P-->>MM: nil
```

---

## Phase details

### Phase 1 — Mattermost API client

```go
p.client = pluginapi.NewClient(p.API, p.Driver)
```

A typed wrapper around the raw plugin API and Driver. Used immediately afterward to obtain the master database handle.

### Phase 2 — Store initialization & migrations

Files: `server/store/sqlstore/store.go`, `server/store/sqlstore/migrate.go`.

1. `p.client.Store.GetMasterDB()` — returns a `*sql.DB` against the Mattermost master database (PostgreSQL).
2. `sqlstore.NewStore(db)` — wraps the handle in the plugin's typed store.
3. `store.RunMigrations()` — applies embedded schema migrations using [`morph`](https://github.com/mattermost/morph):
   - Drops the legacy hand-rolled tracking table `rtk_schema_migrations` (`dropLegacyMigrationsTable`).
   - All migrations under `server/store/sqlstore/migrations/postgres/` are idempotent (`IF NOT EXISTS`) and tracked in `rtk_db_migrations`.

Any error at this phase is wrapped and returned, aborting activation.

### Phase 3 — Configuration loading

Files: `server/configuration.go`.

`p.getConfiguration()` returns the configuration cached by `OnConfigurationChange`. The plugin reads two settings:

- `CloudflareAccountID`
- `CloudflareAPIToken`

Each has an environment-variable override that takes **strict precedence** over the stored setting:

| Setting               | Environment variable | Resolver                                         |
|-----------------------|----------------------|--------------------------------------------------|
| `CloudflareAccountID` | `RTK_ACCOUNT_ID`     | `configuration.GetEffectiveAccountID()`          |
| `CloudflareAPIToken`  | `RTK_API_TOKEN`      | `configuration.GetEffectiveAPIToken()`           |

If either resolved value is empty, the RTK-related phases below are skipped and the plugin activates in a "configured but disabled" state. The operator can then save configuration to trigger `OnConfigurationChange`, which will re-attempt the RTK setup.

### Phase 4 — RTK app provisioning (`EnsureApp`)

Files: `server/app/app.go`.

When both an account ID and an API token are available:

1. An account-level client is created: `rtkclient.NewAccountClient(accountID, apiToken)`.
2. A temporary `App` is constructed solely to call `EnsureApp(accountID)`.
3. `EnsureApp` derives a deterministic app name via `rtkAppName()`:
   - Strips the scheme and trailing slash from `ServiceSettings.SiteURL`.
   - Returns `"mm-" + siteURL` (e.g. `mm-mattermost.example.com`).
4. It calls `AccountClient.ListApps()` (Cloudflare RTK has no `GET /apps/{id}` endpoint, so a list-and-match is required) and looks for an app with that name:
   - **Found**: if the app ID matches the one previously stored, return the stored `app_config_id`. Otherwise, persist a new `app_config` row via `Store.StoreAppConfig`.
   - **Not found**: call `AccountClient.CreateApp(name)` and persist the new mapping.
5. The returned `(appID, appConfigID)` are used in the next phases.

`EnsureApp` errors are **logged with `LogWarn` and swallowed** so activation continues. If `EnsureApp` fails, `appID` is empty and Phase 5 below skips RTK client construction.

### Phase 5 — App-scoped RTK client & business layer

Files: `server/rtkclient/client.go`, `server/app/app.go`.

If `accountID`, `appID` and `apiToken` are all non-empty:

```go
rtkClient = rtkclient.NewClient(accountID, appID, apiToken)
```

The business-logic layer is then created with all dependencies (the RTK client may be `nil`):

```go
p.application = app.New(store, rtkClient, accountClient, p.API)
```

`app.IsConfigured()` returns `rtk != nil` and is later surfaced to the API layer via `configStatus`.

### Phase 6 — Webhook registration (`RegisterWebhookIfNeeded`)

Files: `server/app/webhook_manager.go`.

Only runs if `rtkClient != nil`. The callback URL is built from the Mattermost site URL:

```
{SiteURL}/plugins/{manifest.Id}/api/v1/webhook/rtk
```

(see `Plugin.webhookURL`).

The plugin subscribes to these RTK events (`rtkWebhookEvents`):

- `meeting.participantJoined`
- `meeting.participantLeft`
- `meeting.ended`

Flow:

1. Look up the stored webhook ID via `Store.GetWebhookConfig()`.
2. **If an ID is stored**: call `rtk.GetWebhook(id)` to verify it still exists.
   - Success → nothing to do.
   - `ErrWebhookNotFound` (404) → clear the stale ID and fall through to re-registration.
   - Any other error → log and return; do not re-register.
3. **Register** via `rtk.RegisterWebhook(url, events)`.
   - On `ErrWebhookConflict` (409 — same URL already registered upstream): list all webhooks, delete the entry whose `URL` matches, and retry registration.
4. Persist the new ID via `Store.StoreWebhookConfig(appConfigID, id)`.

All errors are logged via `LogWarn`/`LogInfo`; none of them abort activation.

### Phase 7 — HTTP API handler

Files: `server/api/`, `server/embed.go`.

```go
p.apiHandler = rtapi.Init(p.application, rtapi.StaticFiles{
    CallHTML: callHTML,
    CallJS:   callJS,
    WorkerJS: workerJS,
}, p.configStatus)
```

The static call-page assets (`assets/call.html`, `assets/call.js`, `assets/worker.js`) are embedded at compile time via `//go:embed` directives in `server/embed.go`. The `configStatus` callback lets the API layer report current readiness without holding a reference to the configuration struct.

After this step, `OnActivate` returns `nil` and the plugin is live. Incoming HTTP requests are dispatched by `Plugin.ServeHTTP` to `apiHandler`.

---

## Activation result and `configStatus`

The `Plugin.configStatus()` helper exposes the post-activation state to the API layer:

| Field             | Meaning                                                                     |
|-------------------|-----------------------------------------------------------------------------|
| `Enabled`         | `true` only if credentials are present **and** `app.IsConfigured()` is true (i.e. Phases 4-5 succeeded). |
| `AccountIDViaEnv` | `RTK_ACCOUNT_ID` is set; the stored setting is therefore overridden.        |
| `APITokenViaEnv`  | `RTK_API_TOKEN` is set; the stored setting is therefore overridden.         |
| `AccountID`       | The raw `CloudflareAccountID` setting value (not the env-resolved value).   |

When `Enabled == false` the API layer can still serve UI/status endpoints but RTK-dependent operations are unavailable.

---

## Failure semantics

| Phase | Failure mode | Behavior |
|-------|--------------|----------|
| 2. Master DB acquisition | error from `GetMasterDB` | **Fatal** — `OnActivate` returns wrapped error; plugin is deactivated. |
| 2. Store creation        | error from `sqlstore.NewStore` | **Fatal**. |
| 2. Migrations            | error from `RunMigrations` | **Fatal**. |
| 3. Config load           | (handled in `OnConfigurationChange`, not `OnActivate`) | n/a |
| 4. EnsureApp             | network/auth error, missing credentials | **Best-effort** — logged; `appID` stays empty so Phase 5 skips RTK client setup. |
| 5. RTK client construction | skipped if any of accountID/appID/token is empty | Plugin activates with `IsConfigured() == false`. |
| 6. Webhook verify/register | any RTK API error, missing SiteURL | **Best-effort** — logged; activation continues. |
| 7. API handler init        | none expected | n/a |

Recovery path: when the operator saves configuration in the System Console (or restarts the plugin after fixing networking), `OnConfigurationChange` re-runs the RTK provisioning steps that were skipped or failed during activation.

---

## Source map

| File                                              | Responsibility in startup                                |
|---------------------------------------------------|----------------------------------------------------------|
| `server/plugin.go`                                | `OnActivate` orchestration, `webhookURL`, `configStatus` |
| `server/configuration.go`                         | Configuration struct, env-var precedence, `getConfiguration` |
| `server/embed.go`                                 | Embeds static call-page assets                           |
| `server/store/sqlstore/store.go`                  | `NewStore`                                                |
| `server/store/sqlstore/migrate.go`                | `RunMigrations`, legacy table cleanup                    |
| `server/store/sqlstore/migrations/postgres/*.sql` | Embedded schema migrations                               |
| `server/rtkclient/account_client.go`              | `ListApps`, `CreateApp` (account-level RTK API)          |
| `server/rtkclient/client.go`                      | `GetWebhook`, `RegisterWebhook`, `ListWebhooks`, `DeleteWebhook` (app-scoped RTK API) |
| `server/app/app.go`                               | `EnsureApp`, `rtkAppName`, `IsConfigured`                |
| `server/app/webhook_manager.go`                   | `RegisterWebhookIfNeeded`, conflict resolution           |
| `server/api/`                                     | `rtapi.Init` HTTP handler wiring                         |
