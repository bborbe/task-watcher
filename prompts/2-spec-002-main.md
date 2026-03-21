---
status: created
spec: ["002"]
created: "2026-03-20T19:30:00Z"
branch: dark-factory/main-wiring
---

<summary>
- `main.go` is replaced with a full implementation that accepts a `--config` flag and starts the task watcher
- The binary logs the vault path and assignee from config on successful startup, confirming the correct config was loaded
- The watcher runs continuously until SIGTERM or SIGINT arrives, at which point the binary cancels its context and waits for the watcher to drain
- If the watcher finishes within 5 seconds of signal receipt, the process exits with code 0
- If `--config` is omitted or the file is missing/unreadable/invalid YAML, the binary exits immediately with code 1 and a human-readable error
- No network calls, no filesystem reads, and no context creation occur inside factory or constructor functions — only in `main`
- `go build .` produces a working `task-watcher` binary
- `make precommit` passes after this prompt
</summary>

<objective>
Replace the placeholder `main.go` with the real entry point: flag parsing, config loading, dependency construction via `pkg/factory/`, signal handling, startup logging, and graceful shutdown. This is the final composition step that makes task-watcher a deployable binary.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-error-wrapping.md` for `bborbe/errors` patterns — errors in main may use `log.Printf` + `os.Exit(1)` instead of returning since main cannot return an error.
Read `/home/node/.claude/docs/go-context-cancellation.md` for context propagation patterns.
Read `/home/node/.claude/docs/go-logging-guide.md` for logging conventions (use `log/slog` for new projects).

Preconditions (previous prompts must have run):
- `pkg/config/config.go` — `Loader` interface, `Config` struct, `NewLoader` constructor
- `pkg/notify/notify.go` — `Notifier` interface, `NewNotifier` constructor
- `pkg/watcher/watcher.go` — `Watcher` interface, `NewWatcher` constructor
- `pkg/factory/factory.go` — `CreateConfigLoader`, `CreateNotifier`, `CreateWatcher`

Before writing any code, read:
- `pkg/config/config.go` — confirm `Config` field names (`VaultPath`, `Assignee`)
- `pkg/factory/factory.go` — confirm all three factory function signatures
- Current `main.go` — understand what exists before replacing it
</context>

<requirements>
1. **Critical first step — read the actual API before writing any code:**
   - Read `main.go`, `pkg/config/config.go`, and `pkg/factory/factory.go`
   - Confirm how config loading works: it may be `factory.CreateConfigLoader(path)` returning a `Loader` with `Load(ctx)`, or it may be a direct function call like `config.LoadConfig(path)`. Adapt main.go to match whatever the factory provides.

2. Replace `main.go` with the following structure. **Adapt to match actual factory/config API found in step 1** — the pseudocode below is illustrative, not prescriptive:

   ```
   package main

   Imports: context, flag, fmt, log/slog, os, os/signal, syscall, time
            github.com/bborbe/task-watcher/pkg/config (for Config type reference in log)
            github.com/bborbe/task-watcher/pkg/factory

   func main():
     a. Define --config string flag (default "", usage: "path to config YAML file")
     b. Call flag.Parse()
     c. If configPath == "", print "config flag required" to stderr and os.Exit(1)
     d. Create root context: ctx, cancel := context.WithCancel(context.Background())
        defer cancel()
     e. Load config — adapt to actual API:
        Option A (if Loader interface exists): loader := factory.CreateConfigLoader(configPath); cfg, err := loader.Load(ctx)
        Option B (if standalone function): cfg, err := config.LoadConfig(configPath)
        On error: log to stderr (include file path in message), os.Exit(1)
     g. Log startup (slog.Info): include vault path and assignee from cfg
        Example: slog.Info("task-watcher starting", "vaultPath", cfg.VaultPath, "assignee", cfg.Assignee)
     h. Create notifier: notifier := factory.CreateNotifier(cfg)
     i. Create watcher: w := factory.CreateWatcher(cfg, notifier)
     j. Start signal listener:
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
     k. Run watcher in a goroutine, send error to errCh (buffered channel of size 1)
     l. Select on sigCh or errCh:
        - On signal: slog.Info("shutting down", "signal", sig); cancel()
          Wait for watcher goroutine to finish with a 5-second timeout:
          - If watcher exits within 5s: os.Exit(0)
          - If timeout exceeded: slog.Error("shutdown timed out"); os.Exit(1)
        - On errCh: if err != nil and err != context.Canceled: slog.Error("watcher error", "error", err); os.Exit(1)
          Otherwise (nil or context.Canceled): os.Exit(0)
   ```

3. Ensure `context.Background()` is called exactly once — at step (d) above. No other file in the project should call `context.Background()` (factories and package constructors must not create their own contexts).

4. Verify `go build .` produces a `task-watcher` binary in the workspace root.

5. Verify failure cases by inspection (no integration tests required for main):
   - `--config` omitted → exits with code 1, message "config flag required"
   - `--config /nonexistent` → exits with code 1, message includes the file path
   - Invalid YAML → exits with code 1, message describes parse error

6. Run `make test` — all existing tests must still pass.

7. Run `make precommit` — must pass.
</requirements>

<constraints>
- `context.Background()` is created exactly once, in `main`, and threaded through — no `context.Background()` calls in any other file
- Factory functions are pure composition: main calls them after loading config, never before
- No network calls, no filesystem reads occur before `loader.Load(ctx)` is called
- Ginkgo/Gomega for any new test files (no `testing.T` table tests in new files)
- Error handling in `main` uses `slog.Error` + `os.Exit(1)` (main cannot return errors)
- Error handling in packages follows `github.com/bborbe/errors` patterns — `errors.Wrapf(ctx, err, "message")`
- `make precommit` must pass (linting, tests, formatting)
- `pkg/pkg_suite_test.go` must continue to work — do NOT modify it
- No `//nolint` directives without explanation
- Do NOT commit — dark-factory handles git
- vault-cli is imported as a Go library (`github.com/bborbe/vault-cli/pkg/ops`), never executed via shell
- Single assignee + single webhook per process (no multi-tenant config)
- Graceful shutdown timeout: 5 seconds
</constraints>

<verification>
```bash
go build .
make test
make precommit
```
All must exit with code 0.

Manually verify flag handling:
```bash
# Missing --config flag
./task-watcher; echo "exit: $?"
# Expected: exit: 1, message "config flag required"

# Non-existent config file
./task-watcher --config /nonexistent.yaml; echo "exit: $?"
# Expected: exit: 1, message includes "/nonexistent.yaml"
```

Verify no context.Background() outside main:
```bash
grep -r "context\.Background()" --include="*.go" . | grep -v "^./main.go"
# Expected: no matches (only main.go may call context.Background())
```
</verification>
