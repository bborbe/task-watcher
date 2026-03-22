---
status: approved
spec: ["003"]
created: "2026-03-22T10:00:00Z"
queued: "2026-03-22T11:25:01Z"
branch: dark-factory/multi-watcher-config
---

<summary>
- Single fsnotify watcher fans out each vault file event to all configured watchers in sequence
- Each watcher applies its own independent filter (assignee, statuses, phases) before dispatching
- Each watcher has its own notifier instance (and therefore its own dedup state)
- One watcher failing logs an error but does not stop other watchers from processing the same event
- Factory builds one notifier per configured watcher based on its action type
- CLI removes the dry-run flag — logging-only mode is now config-driven via `log` type
- Startup log lists each configured watcher (name + type) instead of a single assignee
- `make precommit` passes after all changes
</summary>

<objective>
Wire the new multi-watcher config into the watcher and CLI. The fsnotify file watcher already watches all vaults — this prompt makes it fan out each event to all configured watchers with per-watcher filtering. The factory builds notifiers from `WatcherConfig` entries, and the CLI sheds the now-obsolete `--dry-run` flag.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega conventions.
Read `/home/node/.claude/docs/go-factory-pattern.md` for factory pattern rules.
Read `/home/node/.claude/docs/go-error-wrapping.md` for `bborbe/errors` patterns.
Read `/home/node/.claude/docs/go-composition.md` for DI patterns.

Key files to read before making changes:
- `pkg/config/config.go` — `Config`, `WatcherConfig` structs (updated in prompt 1)
- `pkg/watcher/watcher.go` — current watcher (has stub filter from prompt 1 fix)
- `pkg/watcher/watcher_test.go` — existing watcher tests
- `pkg/factory/factory.go` — current factory functions (has stubs from prompt 1 fix)
- `pkg/factory/factory_test.go` — existing factory tests
- `pkg/cli/cli.go` — current CLI with `buildNotifier` and `--dry-run`
- `pkg/notify/notify.go` — `Notifier` interface
- `pkg/notify/telegram.go` — added in prompt 2
- `pkg/notify/log.go` — added in prompt 2
- `pkg/notify/openclaw.go` — existing OpenClaw notifier

This prompt assumes prompts 1 and 2 are complete and the project compiles with the new config format and new notifier types. Verify actual constructor signatures of `NewTelegramNotifier` and `NewLogNotifier` from prompt 2 output before using them — adapt if they differ from the examples below.
</context>

<requirements>
### 1. Refactor `pkg/watcher/watcher.go`

The watcher needs to fan out each vault event to N per-watcher (filter + notifier) pairs.

**Add** a private `watcherEntry` type:
```go
// watcherEntry pairs per-watcher filter criteria with a notifier.
type watcherEntry struct {
    name     string
    assignee string
    statuses []string
    phases   []string
    notifier notify.Notifier
}
```

**Update** `NewWatcher` signature to accept a slice of notifiers (one per `cfg.Watchers` entry, same order):
```go
// NewWatcher returns a Watcher that watches all configured vaults and fans out
// matching task events to all configured watcher entries.
func NewWatcher(cfg config.Config, notifiers []notify.Notifier) Watcher
```

Inside `NewWatcher`, build `[]watcherEntry` from `cfg.Watchers` and `notifiers`:
```go
entries := make([]watcherEntry, len(cfg.Watchers))
for i, w := range cfg.Watchers {
    entries[i] = watcherEntry{
        name:     w.Name,
        assignee: w.Assignee,
        statuses: w.Statuses,
        phases:   w.Phases,
        notifier: notifiers[i],
    }
}
```

The `watcher` struct becomes:
```go
type watcher struct {
    entries      []watcherEntry
    watchOp      ops.WatchOperation
    vaultPaths   map[string]string
    taskStorages map[string]taskReader
    targets      []ops.WatchTarget
}
```
Remove the old `config config.Config` and `notifier notify.Notifier` fields.

**Update** `handleEvent` to fan out to all entries:
```go
func (w *watcher) handleEvent(ctx context.Context, event ops.WatchEvent) error {
    vaultPath, ok := w.vaultPaths[event.Vault]
    if !ok {
        slog.Warn("unknown vault in event", "vault", event.Vault)
        return nil
    }
    taskStorage, ok := w.taskStorages[event.Vault]
    if !ok {
        slog.Warn("no storage for vault", "vault", event.Vault)
        return nil
    }
    task, err := taskStorage.ReadTask(ctx, vaultPath, domain.TaskID(event.Name))
    if err != nil {
        slog.Debug("skip unreadable task", "name", event.Name, "error", err)
        return nil
    }
    if task.Assignee == "" || task.Status == "" || task.Phase == nil {
        return nil
    }

    notification := notify.Notification{
        TaskName: task.Name,
        Phase:    string(*task.Phase),
        Assignee: task.Assignee,
    }

    for _, entry := range w.entries {
        if entry.assignee != "" && task.Assignee != entry.assignee {
            continue
        }
        if len(entry.statuses) > 0 && !containsString(entry.statuses, string(task.Status)) {
            continue
        }
        if len(entry.phases) > 0 && !containsString(entry.phases, string(*task.Phase)) {
            continue
        }
        if err := entry.notifier.Notify(ctx, notification); err != nil {
            slog.Error("notify failed",
                "watcher", entry.name,
                "task", task.Name,
                "phase", string(*task.Phase),
                "error", err,
            )
            // Continue to next watcher — one failure must not block others
        }
    }
    return nil
}
```

Key behavior: the loop does NOT return early on notifier errors. Other watchers must still run.

### 2. Update `pkg/watcher/watcher_test.go`

Update all `watcher.NewWatcher(cfg, notifier)` calls to `watcher.NewWatcher(cfg, []notify.Notifier{fakeNotifier})` (single-element slice for single-watcher configs).

Add tests for the fan-out behavior:
- Two watcher entries: task matching both → both notifiers called
- Two watcher entries: task matching only first → only first notifier called
- First notifier returns error → second notifier still called, handleEvent returns nil
- Per-watcher assignee filter: task with wrong assignee for one entry → that entry skipped
- Per-watcher phase filter: task in wrong phase for one entry → that entry skipped

Use `mocks.FakeNotifier` (Counterfeiter mock) for all notifier fakes. Do not write manual mocks.

For multi-watcher tests, create a `config.Config` with two `WatcherConfig` entries and pass `[]notify.Notifier{fake1, fake2}` to `NewWatcher`.

### 3. Rewrite `pkg/factory/factory.go`

Replace all existing factory functions with a clean multi-watcher design:

```go
// CreateConfigLoader constructs a config.Loader for the given file path.
// Pure composition: no I/O, no context creation.
func CreateConfigLoader(filePath string) config.Loader {
    return config.NewLoader(filePath)
}

// CreateNotifiers builds one Notifier per WatcherConfig entry in the order they appear.
// Pure composition: no network calls at construction time.
func CreateNotifiers(cfg config.Config) []notify.Notifier {
    notifiers := make([]notify.Notifier, len(cfg.Watchers))
    for i, w := range cfg.Watchers {
        notifiers[i] = createNotifierForWatcher(w)
    }
    return notifiers
}

// CreateWatcher constructs a watcher.Watcher that observes all configured vaults
// and fans out matching task events to all watcher entries.
// Pure composition: no filesystem access at construction time.
func CreateWatcher(cfg config.Config, notifiers []notify.Notifier) watcher.Watcher {
    return watcher.NewWatcher(cfg, notifiers)
}

// createNotifierForWatcher instantiates the correct Notifier implementation based on watcher type.
func createNotifierForWatcher(w config.WatcherConfig) notify.Notifier {
    switch w.Type {
    case "openclaw-wake":
        return notify.NewOpenClawNotifier(w.URL, w.Token, http.DefaultClient, w.DedupTTL)
    case "telegram":
        return notify.NewTelegramNotifier(w.Token, w.ChatID, http.DefaultClient, w.DedupTTL)
    case "log":
        return notify.NewLogNotifier(w.DedupTTL)
    default:
        // Should never reach here — validated in config.Load
        panic("unknown watcher type: " + w.Type)
    }
}
```

**Remove** the following functions (no longer needed):
- `CreateNotifier(cfg config.Config) notify.Notifier`
- `CreateDryRunNotifier(cfg config.Config) notify.Notifier`
- `CreateOpenClawNotifier(cfg config.Config) notify.Notifier`
- `CreateDryRunOpenClawNotifier(cfg config.Config) notify.Notifier`

Update imports: add `"net/http"` if not present.

### 4. Update `pkg/factory/factory_test.go`

Remove tests for deleted factory functions. Add/update tests for new functions:

- `CreateConfigLoader("")` → returns non-nil Loader
- `CreateNotifiers(cfg)` with one `openclaw-wake` watcher → returns slice of length 1, non-nil element
- `CreateNotifiers(cfg)` with `telegram` watcher → returns slice of length 1, non-nil element
- `CreateNotifiers(cfg)` with `log` watcher → returns slice of length 1, non-nil element
- `CreateNotifiers(cfg)` with empty watchers → returns empty slice
- `CreateWatcher(cfg, notifiers)` → returns non-nil Watcher

### 5. Rewrite `pkg/cli/cli.go`

**Remove** the `--dry-run` flag and the `buildNotifier` helper function.

**Update** the `RunE` function body:

```go
RunE: func(cmd *cobra.Command, _ []string) error {
    if verbose {
        slog.SetLogLoggerLevel(slog.LevelDebug)
    }

    loader := factory.CreateConfigLoader(configPath)
    cfg, err := loader.Load(ctx)
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }

    slog.Info("task-watcher starting", "version", version)
    for _, v := range cfg.Vaults {
        slog.Info("watching vault", "name", v.Name, "path", v.Path, "tasksDir", v.TasksDir)
    }
    for _, w := range cfg.Watchers {
        slog.Info("configured watcher", "name", w.Name, "type", w.Type, "assignee", w.Assignee)
    }

    notifiers := factory.CreateNotifiers(cfg)
    watcher := factory.CreateWatcher(cfg, notifiers)

    errCh := make(chan error, 1)
    go func() {
        errCh <- watcher.Watch(ctx)
    }()

    select {
    case <-ctx.Done():
        slog.Info("shutting down")
        select {
        case <-errCh:
            return nil
        case <-time.After(5 * time.Second):
            slog.Error("shutdown timed out")
            return fmt.Errorf("shutdown timed out")
        }
    case err := <-errCh:
        if err != nil && err != context.Canceled {
            return fmt.Errorf("watcher error: %w", err)
        }
        return nil
    }
},
```

**Update** variable declarations at the top of `Run`:
```go
var configPath string
var verbose bool
// Remove: var dryRun bool
```

**Remove** the `dryRun` flag registration:
```go
// Remove this line:
rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "log notifications instead of sending HTTP webhooks")
```

Remove `"github.com/bborbe/task-watcher/pkg/notify"` import if it's no longer used.

### 6. Update `CHANGELOG.md`

Append to `## Unreleased` (which was created in prompt 1). Only add entries for THIS prompt's work — telegram and log notifier entries were already added by prompt 2:
```
- feat: Fan out vault events to all configured watchers with per-watcher filter and dedup
- refactor: Remove --dry-run flag (use log action type in config instead)
```

### 7. Run make precommit

```bash
make test
make precommit
```

Both must exit with code 0. Fix any lint or formatting issues before declaring complete.

If `make precommit` fails on a specific target, run only that target to fix:
- `make lint` — fix linting issues
- `make gosec` — fix security issues
- `make errcheck` — fix unchecked errors
</requirements>

<constraints>
- Config is backward-incompatible — old flat format must remain a clear error (handled in prompt 1, do not regress)
- One watcher failure must NOT stop other watchers from processing the same event
- Factory functions remain pure composition — `createNotifierForWatcher` is a private helper, not public API
- `Watcher` interface is unchanged: `Watch(ctx context.Context) error`
- `Notifier` interface is unchanged: `Notify(ctx context.Context, notification Notification) error`
- Telegram bot token must not appear in logs
- Ginkgo/Gomega tests, Counterfeiter mocks (no manual mocks)
- Do NOT commit — dark-factory handles git
- `make precommit` must pass at the end of this prompt
</constraints>

<verification>
```bash
make test
make precommit
```
Both must exit with code 0.

Manual config validation — create `/tmp/test-config.yaml`:
```yaml
vaults:
  test:
    path: /tmp/test-vault
    tasks_dir: tasks

watchers:
  - name: debug
    type: log
    assignee: TestUser
    statuses: [in_progress]
    phases: [planning]
```

Run:
```bash
mkdir -p /tmp/test-vault/tasks
./task-watcher --config /tmp/test-config.yaml --verbose &
sleep 1
# Confirm startup log shows "configured watcher name=debug type=log"
# Ctrl+C to stop
```
</verification>
