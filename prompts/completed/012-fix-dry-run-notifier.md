---
status: completed
summary: Added --dry-run flag with logging-only notifier, factory function, and tests
container: task-watcher-012-fix-dry-run-notifier
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T12:40:13Z"
queued: "2026-03-21T12:40:13Z"
started: "2026-03-21T12:40:15Z"
completed: "2026-03-21T12:49:01Z"
---

<summary>
- New `--dry-run` flag logs notifications instead of sending HTTP webhooks
- `notify.Notifier` interface unchanged, two implementations: HTTP notifier and dry-run notifier
- Dry-run notifier logs task name, phase, and assignee via slog at Info level
- Dry-run notifier preserves deduplication behavior (same key only logged once)
- Factory and CLI updated to select notifier based on dry-run flag
- `make precommit` passes after changes
</summary>

<objective>
Add a `--dry-run` flag that replaces the HTTP webhook notifier with a logging-only notifier, allowing safe testing of the watcher without sending real webhook calls.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-factory-pattern.md` for factory conventions.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega test conventions.

Before writing any code, read:
- `pkg/notify/notify.go` — current `Notifier` interface and HTTP implementation
- `pkg/cli/cli.go` — current cobra setup and factory calls
- `pkg/factory/factory.go` — current `CreateNotifier` signature
</context>

<requirements>
1. **Read `pkg/notify/notify.go`, `pkg/cli/cli.go`, `pkg/factory/factory.go`** before making changes.

2. **Create dry-run notifier in `pkg/notify/dry_run.go`**:
   - Implements `Notifier` interface
   - Logs each notification via `slog.Info("dry-run notification", "taskName", n.TaskName, "phase", n.Phase, "assignee", n.Assignee)`
   - Deduplicates using same `taskName:phase` key pattern as the HTTP notifier
   - Constructor: `NewDryRunNotifier() Notifier`

3. **Update `pkg/factory/factory.go`**:
   - Add `CreateDryRunNotifier() notify.Notifier`
   - Keep existing `CreateNotifier(cfg config.Config) notify.Notifier` unchanged

4. **Update `pkg/cli/cli.go`**:
   - Add `--dry-run` bool flag (default false)
   - If dry-run: use `factory.CreateDryRunNotifier()` instead of `factory.CreateNotifier(cfg)`
   - Log which notifier mode is active at startup: `slog.Info("task-watcher starting", "vaultPath", cfg.VaultPath, "assignee", cfg.Assignee, "dryRun", dryRun)`

5. **Create tests in `pkg/notify/dry_run_test.go`**:
   - Calling `Notify` returns nil (no error)
   - Second call with same task+phase is deduplicated (returns nil, no duplicate log)
   - Different task+phase combinations are both logged

6. **Update factory tests** to cover `CreateDryRunNotifier` returns non-nil `Notifier`.

7. Run `make test` — all tests must pass.

8. Run `make precommit` — must pass.
</requirements>

<constraints>
- `Notifier` interface in `pkg/notify/notify.go` must NOT change
- HTTP notifier must NOT be modified
- Dry-run notifier lives in its own file `pkg/notify/dry_run.go`
- Factory functions are pure composition — no conditionals, no I/O
- The if/else for dry-run vs real notifier lives in `cli.go`, not in factory
- `make precommit` must pass
- Do NOT modify `pkg/pkg_suite_test.go`
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
```bash
go build -o /tmp/task-watcher .

# Dry-run mode shows in help
/tmp/task-watcher --help 2>&1 | grep dry-run

make test
make precommit
```
All must exit with code 0.
</verification>
