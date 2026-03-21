---
status: completed
summary: Replaced permanent dedup with TTL-based dedup across all four notifier types, added dedup_ttl config field defaulting to 5 minutes, and updated all tests with TTL re-send verification.
container: task-watcher-019-ttl-dedup
dark-factory-version: v0.67.3-dirty
created: "2026-03-21T22:00:00Z"
queued: "2026-03-21T22:14:45Z"
started: "2026-03-21T22:14:50Z"
completed: "2026-03-21T22:20:51Z"
---

<summary>
- Webhook notifications for the same task+phase now re-fire after a configurable cooldown instead of being permanently silenced
- New optional config field `dedup_ttl` (e.g. "5m") defaults to 5 minutes — no config change needed for existing deployments
- All four notifier types (generic, OpenClaw, dry-run generic, dry-run OpenClaw) gain TTL-based dedup
- Existing dedup tests updated; new tests verify re-send after cooldown expires
- Config tests cover valid duration, invalid duration, and missing field (default)
- `make precommit` passes
</summary>

<objective>
Replace the permanent dedup in the notifier with TTL-based dedup. Currently `seen[taskName:phase]` is permanent — once a webhook fires for a task+phase combo, it never fires again until process restart. This means if OpenClaw doesn't act (crash, busy), the task is silently lost. With TTL-based dedup, the webhook re-fires after the TTL expires, giving OpenClaw another chance.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega conventions.
Read `/home/node/.claude/docs/go-error-wrapping.md` for `bborbe/errors` patterns.

Key files to read before making changes:
- `pkg/config/config.go` — `Config` struct, `rawConfig`, `func (l *loader) Load`
- `pkg/config/config_test.go` — existing config tests
- `pkg/notify/notify.go` — `Notifier` interface, `notifier` struct with `seen map[string]struct{}`
- `pkg/notify/notify_test.go` — existing dedup tests
- `pkg/notify/openclaw.go` — `openClawNotifier` struct with `seen map[string]struct{}`
- `pkg/notify/openclaw_test.go` — existing OpenClaw dedup tests
- `pkg/notify/dry_run.go` — `dryRunNotifier` and `dryRunOpenClawNotifier` structs with `seen map[string]struct{}`
- `pkg/notify/dry_run_test.go` — existing dry-run dedup tests
- `pkg/factory/factory.go` — all four `Create*Notifier` functions
</context>

<requirements>
### 1. `pkg/config/config.go` — add DedupTTL field

Add to `Config` struct:
```go
DedupTTL time.Duration
```

Add to `rawConfig` struct:
```go
DedupTTL string `yaml:"dedup_ttl"`
```

In `func (l *loader) Load(ctx context.Context)`, after parsing other fields, parse the TTL (add `"time"` to imports):
```go
dedupTTL := 5 * time.Minute // default
if raw.DedupTTL != "" {
    parsed, err := time.ParseDuration(raw.DedupTTL)
    if err != nil {
        return Config{}, errors.Wrapf(ctx, err, "parse dedup_ttl %q", raw.DedupTTL)
    }
    dedupTTL = parsed
}
```

Set `DedupTTL: dedupTTL` in the returned `Config`.

### 2. `pkg/notify/notify.go` — TTL-based dedup

Change `NewNotifier` signature to accept the TTL:
```go
func NewNotifier(webhookURL string, httpClient *http.Client, dedupTTL time.Duration) Notifier
```

Change the notifier struct:
```go
type notifier struct {
    webhookURL string
    httpClient *http.Client
    dedupTTL   time.Duration
    seen       map[string]time.Time
    mu         sync.Mutex
}
```

In `Notify`, replace the dedup check:
```go
key := notification.TaskName + ":" + notification.Phase

n.mu.Lock()
lastSent, exists := n.seen[key]
if exists && time.Since(lastSent) < n.dedupTTL {
    n.mu.Unlock()
    slog.Debug("webhook skipped (duplicate within TTL)",
        "task", notification.TaskName,
        "phase", notification.Phase,
        "ttl", n.dedupTTL,
        "lastSent", lastSent,
    )
    return nil
}
n.seen[key] = time.Now()
n.mu.Unlock()
```

Note: the `seen[key] = time.Now()` is set **before** sending (optimistic). If the HTTP call fails, the entry is still in the map — this is acceptable because the error is returned to the caller and the next file change will retry after TTL.

### 3. `pkg/notify/openclaw.go` — TTL-based dedup

Apply the same TTL pattern as requirement 2:

Change `NewOpenClawNotifier` signature:
```go
func NewOpenClawNotifier(webhookURL string, token string, httpClient *http.Client, dedupTTL time.Duration) Notifier
```

Change the struct field `seen` from `map[string]struct{}` to `map[string]time.Time`, add `dedupTTL time.Duration`.

Replace the dedup check in `Notify` with the same TTL logic as requirement 2.

### 4. `pkg/notify/dry_run.go` — TTL-based dedup for both dry-run notifiers

Apply the same TTL pattern to both `dryRunNotifier` and `dryRunOpenClawNotifier`:

Change constructors:
```go
func NewDryRunNotifier(webhookURL string, dedupTTL time.Duration) Notifier
func NewDryRunOpenClawNotifier(webhookURL string, token string, dedupTTL time.Duration) Notifier
```

Change both struct `seen` fields from `map[string]struct{}` to `map[string]time.Time`, add `dedupTTL time.Duration`.

Replace dedup checks in both `Notify` methods with TTL logic.

### 5. `pkg/factory/factory.go` — pass DedupTTL to all constructors

Update all four factory functions to pass `cfg.DedupTTL`:
- `CreateNotifier`: `notify.NewNotifier(cfg.Webhook, http.DefaultClient, cfg.DedupTTL)`
- `CreateDryRunNotifier`: `notify.NewDryRunNotifier(cfg.Webhook, cfg.DedupTTL)`
- `CreateOpenClawNotifier`: `notify.NewOpenClawNotifier(cfg.Webhook, cfg.WebhookToken, http.DefaultClient, cfg.DedupTTL)`
- `CreateDryRunOpenClawNotifier`: `notify.NewDryRunOpenClawNotifier(cfg.Webhook, cfg.WebhookToken, cfg.DedupTTL)`

Add `"time"` to factory.go imports if not already present (it is not — `config.Config` carries `time.Duration` but the factory file doesn't need `time` import since it just passes the field through).

### 6. Update all test files

**`pkg/notify/notify_test.go`** — update all `NewNotifier` calls to include TTL parameter:
```go
n := notify.NewNotifier(server.URL, server.Client(), time.Minute)
```

Add a TTL re-send test:
```go
It("re-sends webhook after TTL expires", func() {
    n := notify.NewNotifier(server.URL, server.Client(), 50*time.Millisecond)
    notification := notify.Notification{
        TaskName: "retry-task",
        Phase:    "planning",
        Assignee: "alice",
    }
    Expect(n.Notify(ctx, notification)).To(Succeed())
    Expect(requestCount.Load()).To(Equal(int32(1)))

    // Within TTL — should be deduped
    Expect(n.Notify(ctx, notification)).To(Succeed())
    Expect(requestCount.Load()).To(Equal(int32(1)))

    // Wait for TTL to expire
    time.Sleep(60 * time.Millisecond)

    // After TTL — should re-send
    Expect(n.Notify(ctx, notification)).To(Succeed())
    Expect(requestCount.Load()).To(Equal(int32(2)))
})
```

**`pkg/notify/openclaw_test.go`** — update all `NewOpenClawNotifier` calls to include TTL parameter. Add a similar TTL re-send test.

**`pkg/notify/dry_run_test.go`** — update all `NewDryRunNotifier` and `NewDryRunOpenClawNotifier` calls to include TTL parameter. Add TTL re-send tests for both.

**`pkg/config/config_test.go`** — add tests for `dedup_ttl` parsing:
- Valid duration string (e.g. `"5m"`) → `DedupTTL` is 5 minutes
- Invalid string (e.g. `"banana"`) → returns error
- Missing field → defaults to 5 minutes

### 7. Regenerate mocks if Notifier interface changed

The `Notifier` interface itself is unchanged (`Notify(ctx, notification) error`), so mocks should not need regeneration. But run `go generate ./...` to be safe.

### 8. Verification

```bash
make test
make precommit
```
</requirements>

<constraints>
- Default TTL is 5 minutes — no config change needed for existing deployments
- `Notifier` interface is unchanged — only constructor signatures change
- All four notifier types must get TTL dedup (generic, OpenClaw, dry-run generic, dry-run OpenClaw)
- Error handling follows `github.com/bborbe/errors` patterns
- `make precommit` must pass
- Do NOT commit — dark-factory handles git
- Existing tests must pass after updating constructor calls
</constraints>

<verification>
```bash
make test
make precommit
```
Both must exit with code 0.
</verification>
