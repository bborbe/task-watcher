---
status: created
spec: ["003"]
created: "2026-03-22T10:00:00Z"
branch: dark-factory/multi-watcher-config
---

<summary>
- Config replaces the single flat filter+action with a `watchers` list — each entry is self-contained
- New `WatcherConfig` struct holds name, type, per-watcher filter (assignee, statuses, phases), TTL, and type-specific fields
- Top-level `Config` struct is now `Vaults []VaultConfig` + `Watchers []WatcherConfig` only
- Three action types recognized: `openclaw-wake`, `telegram`, `log`; unknown type is a config error
- Required fields are validated per type: `openclaw-wake` needs `url`+`token`, `telegram` needs `token`+`chat_id`, `log` needs nothing extra
- Empty `watchers` list is valid — process starts and watches but never notifies
- Old flat top-level fields (`assignee`, `webhook`, `format`, `webhook_token`, `statuses`, `phases`, `dedup_ttl`) trigger a descriptive error with a migration hint
- All config tests updated; old field-validation tests removed; new per-watcher validation tests added
</summary>

<objective>
Replace the current flat `Config` struct with a new `watchers` list format so a single task-watcher process can support N independent watchers, each with its own filter criteria and action type. This is the foundational change — later prompts wire the new config into the watcher and factory.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega conventions.
Read `/home/node/.claude/docs/go-error-wrapping.md` for `bborbe/errors` patterns.

Key files to read before making changes:
- `pkg/config/config.go` — current `Config`, `rawConfig`, `loader.Load` (this is the only file to change)
- `pkg/config/config_test.go` — all existing tests (full rewrite needed)
- `pkg/factory/factory.go` — references `config.Config` fields (will break; fix compilation only, not logic)
- `pkg/watcher/watcher.go` — references `config.Config` fields (will break; fix compilation only, not logic)
- `pkg/cli/cli.go` — references `config.Config` fields (will break; fix compilation only, not logic)

The purpose of reading factory, watcher, and cli is to understand what fields they access so you can update them to compile. Do NOT redesign those files — later prompts handle that. For now, just ensure the project compiles and config tests pass.
</context>

<requirements>
### 1. Rewrite `pkg/config/config.go`

**Remove** the old fields from `Config`:
- `Assignee string`
- `Statuses []string`
- `Phases []string`
- `Webhook string`
- `Format string`
- `WebhookToken string`
- `DedupTTL time.Duration`

**Add** `WatcherConfig` struct:
```go
// WatcherConfig holds the configuration for a single watcher entry.
type WatcherConfig struct {
    Name     string
    Type     string
    Assignee string
    Statuses []string
    Phases   []string
    DedupTTL time.Duration
    // openclaw-wake fields
    URL   string
    Token string
    // telegram fields
    ChatID string
}
```

**Update** `Config` struct:
```go
// Config holds the parsed task-watcher configuration.
type Config struct {
    Vaults   []VaultConfig
    Watchers []WatcherConfig
}
```

**Update** `rawConfig` struct — add old flat fields for backward-compat detection, add new watchers list:
```go
type rawWatcherEntry struct {
    Name     string   `yaml:"name"`
    Type     string   `yaml:"type"`
    Assignee string   `yaml:"assignee"`
    Statuses []string `yaml:"statuses"`
    Phases   []string `yaml:"phases"`
    DedupTTL string   `yaml:"dedup_ttl"`
    URL      string   `yaml:"url"`
    Token    string   `yaml:"token"`
    ChatID   string   `yaml:"chat_id"`
}

type rawConfig struct {
    Vaults   map[string]rawVaultEntry `yaml:"vaults"`
    Watchers []rawWatcherEntry        `yaml:"watchers"`
    // Old flat fields — present only for migration error detection
    Assignee     string `yaml:"assignee"`
    Webhook      string `yaml:"webhook"`
    Format       string `yaml:"format"`
    WebhookToken string `yaml:"webhook_token"`
    Statuses     []string `yaml:"statuses"`
    Phases       []string `yaml:"phases"`
    OldDedupTTL  string `yaml:"dedup_ttl"`
}
```

**Update** `func (l *loader) Load(ctx context.Context) (Config, error)`:

1. Read and unmarshal YAML as before.

2. **Backward-compat check** — immediately after unmarshal, before any other validation:
   ```go
   if raw.Assignee != "" || raw.Webhook != "" || raw.Format != "" || raw.WebhookToken != "" {
       return Config{}, errors.Errorf(ctx,
           "config uses the old flat format (assignee/webhook/format fields at the top level). "+
           "Please migrate to the new watchers list format. See CHANGELOG for migration instructions.")
   }
   ```

3. **Vaults validation** — same as before (at least one required, each needs path + tasks_dir).

4. **Watchers parsing** — empty list is allowed (process watches but never notifies):
   ```go
   watchers, err := parseWatchers(ctx, raw.Watchers)
   if err != nil {
       return Config{}, err
   }
   ```

5. Return `Config{Vaults: vaults, Watchers: watchers}`.

**Add** `parseWatchers(ctx context.Context, rawList []rawWatcherEntry) ([]WatcherConfig, error)`:

```go
func parseWatchers(ctx context.Context, rawList []rawWatcherEntry) ([]WatcherConfig, error) {
    watchers := make([]WatcherConfig, 0, len(rawList))
    for i, rw := range rawList {
        if rw.Name == "" {
            return nil, errors.Errorf(ctx, "watcher[%d]: missing required field: name", i)
        }
        if rw.Type == "" {
            return nil, errors.Errorf(ctx, "watcher %q: missing required field: type", rw.Name)
        }
        switch rw.Type {
        case "openclaw-wake":
            if rw.URL == "" {
                return nil, errors.Errorf(ctx, "watcher %q (openclaw-wake): missing required field: url", rw.Name)
            }
            if rw.Token == "" {
                return nil, errors.Errorf(ctx, "watcher %q (openclaw-wake): missing required field: token", rw.Name)
            }
        case "telegram":
            if rw.Token == "" {
                return nil, errors.Errorf(ctx, "watcher %q (telegram): missing required field: token", rw.Name)
            }
            if rw.ChatID == "" {
                return nil, errors.Errorf(ctx, "watcher %q (telegram): missing required field: chat_id", rw.Name)
            }
        case "log":
            // no extra fields required
        default:
            return nil, errors.Errorf(ctx, "watcher %q: unknown type %q (must be openclaw-wake, telegram, or log)", rw.Name, rw.Type)
        }

        dedupTTL := 5 * time.Minute
        if rw.DedupTTL != "" {
            parsed, err := time.ParseDuration(rw.DedupTTL)
            if err != nil {
                return nil, errors.Wrapf(ctx, err, "watcher %q: parse dedup_ttl %q", rw.Name, rw.DedupTTL)
            }
            dedupTTL = parsed
        }

        watchers = append(watchers, WatcherConfig{
            Name:     rw.Name,
            Type:     rw.Type,
            Assignee: rw.Assignee,
            Statuses: rw.Statuses,
            Phases:   rw.Phases,
            DedupTTL: dedupTTL,
            URL:      rw.URL,
            Token:    rw.Token,
            ChatID:   rw.ChatID,
        })
    }
    return watchers, nil
}
```

**Remove** the old `parseFormat` function entirely.

**Remove** the old validation for `Assignee`, `Statuses`, `Phases`, `Webhook` (those moved into per-watcher).

### 2. Fix compilation in files that reference removed Config fields

After updating `config.go`, the project will not compile because `factory.go`, `watcher.go`, and `cli.go` reference fields that no longer exist (`cfg.Assignee`, `cfg.Statuses`, `cfg.Phases`, `cfg.Webhook`, `cfg.Format`, `cfg.WebhookToken`, `cfg.DedupTTL`).

Do **minimal** fixes to restore compilation — do NOT redesign these files:

**`pkg/factory/factory.go`**: Remove or stub out factory functions that use removed fields. Replace them with placeholder implementations that compile but panic or return nil for now. Specifically:
- `CreateNotifier(cfg config.Config) notify.Notifier` — can return `nil` for now (later prompt replaces it)
- `CreateDryRunNotifier(cfg config.Config) notify.Notifier` — same
- `CreateOpenClawNotifier(cfg config.Config) notify.Notifier` — same
- `CreateDryRunOpenClawNotifier(cfg config.Config) notify.Notifier` — same
- `CreateWatcher(cfg config.Config, notifier notify.Notifier) watcher.Watcher` — keep as is (signature unchanged)

Remove unused imports if any.

**`pkg/watcher/watcher.go`**: The `watcher` struct has `config config.Config` which references the old fields in `handleEvent`. Replace the filter logic with a temporary stub that calls the notifier unconditionally (or passes through all tasks) — later prompt fixes this properly:
```go
// TODO(spec-003): per-watcher filter will be added in the fanout prompt
if err := w.notifier.Notify(ctx, notify.Notification{
    TaskName: task.Name,
    Phase:    string(*task.Phase),
    Assignee: task.Assignee,
}); err != nil {
    return errors.Wrapf(ctx, err, "notify task %s phase %s", task.Name, string(*task.Phase))
}
```
Remove `w.config.Assignee`, `w.config.Statuses`, `w.config.Phases` filter checks.

**`pkg/cli/cli.go`**:
- Remove `cfg.Assignee` from the startup `slog.Info` call
- Remove `buildNotifier` function body references to `cfg.Format`, replace with: `return factory.CreateNotifier(cfg)` unconditionally (temporary)
- Keep `--dry-run` flag for now but make `buildNotifier` always call `factory.CreateNotifier(cfg)` regardless of dryRun (later prompt removes --dry-run properly)

**`mocks/`**: If any mock is generated from config types (unlikely), check and update.

### 3. Rewrite `pkg/config/config_test.go`

Remove all old tests. Write new tests covering:

**Vault validation** (same as before):
- Valid multi-vault config with new watchers list → parses successfully
- Empty vaults map → error "missing required field: vaults"
- Vault missing path → error
- Vault missing tasks_dir → error
- `~/` expansion in vault path → expanded correctly

**Backward-compat detection**:
- Config with top-level `assignee` field → error containing "old flat format"
- Config with top-level `webhook` field → error containing "old flat format"
- Config with top-level `format` field → error containing "old flat format"

**Watchers parsing**:
- Empty watchers list → valid (no error), `cfg.Watchers` is empty slice
- Watcher missing `name` → error "missing required field: name"
- Watcher missing `type` → error "missing required field: type"
- Unknown watcher type → error "unknown type"
- `openclaw-wake` missing url → error
- `openclaw-wake` missing token → error
- `telegram` missing token → error
- `telegram` missing chat_id → error
- `log` type with no extra fields → valid
- `dedup_ttl: "30m"` → `DedupTTL` is 30 minutes
- Missing `dedup_ttl` → `DedupTTL` defaults to 5 minutes
- Invalid `dedup_ttl: "banana"` → error

**Full valid config** — parse the full YAML from the spec:
```yaml
vaults:
  openclaw:
    path: /tmp/vault
    tasks_dir: tasks
  personal:
    path: /tmp/obsidian
    tasks_dir: "24 Tasks"

watchers:
  - name: wake-tradingclaw
    type: openclaw-wake
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [planning, in_progress, ai_review]
    dedup_ttl: "5m"
    url: http://127.0.0.1:18789/hooks/wake
    token: "secret"

  - name: notify-review
    type: telegram
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [human_review]
    dedup_ttl: "30m"
    token: "bot123:ABC"
    chat_id: "456"

  - name: debug
    type: log
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [planning]
```
→ 2 vaults parsed, 3 watchers parsed, correct field values on each.

### 4. Update `CHANGELOG.md`

Add under `## Unreleased` (create if not present):
```
- feat: Replace flat config with watchers list supporting multiple independent notifiers per process
```

### 5. Run tests

```bash
make test
```

Tests must pass before declaring complete. If factory/watcher/cli tests fail due to stub changes, fix them.
</requirements>

<constraints>
- Config is backward-incompatible — old flat format must produce a clear error with migration hint
- `VaultConfig` struct and vault parsing logic is unchanged (only `Config` and watcher fields change)
- `Loader` interface is unchanged: `Load(ctx context.Context) (Config, error)`
- Factory functions remain pure composition — no I/O at construction time
- Ginkgo/Gomega tests, Counterfeiter mocks
- Do NOT commit — dark-factory handles git
- Existing vault-related tests must still pass
- The watcher and factory will be fully fixed in later prompts — only fix compilation here
</constraints>

<verification>
```bash
make test
```
Must exit with code 0. `make precommit` is NOT required for this prompt — the later fanout prompt runs it as the final validation.
</verification>
