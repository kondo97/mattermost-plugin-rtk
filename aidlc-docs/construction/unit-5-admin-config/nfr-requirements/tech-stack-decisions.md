# Unit 5: Admin & Config — Tech Stack Decisions

## Backend

| Decision | Choice | Rationale |
|---|---|---|
| Env var access | `os.LookupEnv` (stdlib) | Distinguishes "not set" from "set to empty"; consistent with mattermost-plugin-calls pattern |
| String comparison for feature flags | `strings.EqualFold` (stdlib) | Case-insensitive "true" check; no external dependency |
| Config clone | Shallow copy (`*c`) | Valid for all fields: strings (value type) and `*bool` (pointer to immutable value) |
| Test env var isolation | `t.Setenv` (Go 1.17+) | Automatic cleanup after test; no manual `os.Setenv`/`os.Unsetenv` pairs needed |
| Input validation library | None | No format validation required (VAL-01, VAL-02); stdlib empty check sufficient |

## Frontend / Admin UI

| Decision | Choice | Rationale |
|---|---|---|
| Admin console integration | `plugin.json` `settings_schema` only | No custom React components; Mattermost native UI sufficient |
| API Key masking | `"secret": true` in `plugin.json` | Mattermost encrypts and masks the value without any React code |
| Feature flag toggles | `type: "bool"` with `"default": "true"` | Native Mattermost toggle UI; zero frontend implementation cost |
| Custom component registration | None | Not required; `secret: true` satisfies the masking requirement |

## No New Dependencies

Unit 5 introduces no new Go or npm dependencies. All functionality is implemented using:
- Go stdlib: `os`, `strings`
- Existing plugin SDK: `p.API.LoadPluginConfiguration`
- Existing `plugin.json` schema fields: `secret`, `type: "bool"`, `default`
