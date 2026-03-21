---
status: completed
summary: Replaced stdlib flag with cobra in pkg/cli package, eliminating glog flag pollution from --help output
container: task-watcher-009-spec-002-fix-cobra-cli
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T11:36:08Z"
queued: "2026-03-21T11:36:08Z"
started: "2026-03-21T11:36:09Z"
completed: "2026-03-21T11:42:22Z"
---

<summary>
- main.go delegates to a new `pkg/cli` package using cobra for flag parsing
- `--help` output shows only `--config` and `--verbose` flags, no glog pollution
- Signal handling and graceful shutdown preserved
- `make precommit` passes after changes
</summary>

<objective>
Replace stdlib `flag` usage with `cobra` to eliminate glog flag pollution in `--help` output. Follow the same pattern as vault-cli: thin `main.go` calling `cli.Execute()`, with a `pkg/cli/` package containing `Execute()` and `Run(ctx, args)`.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-cli.md` for the cobra CLI pattern.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-context-cancellation.md` for context propagation patterns.
Read `/home/node/.claude/docs/go-logging-guide.md` for logging conventions.

Before writing any code, read:
- `main.go` — current implementation with stdlib `flag`
- `pkg/factory/factory.go` — factory function signatures used by main
- `pkg/config/config.go` — confirm config `Loader` interface

Reference: vault-cli uses this pattern:
```go
// main.go
package main
import "github.com/bborbe/vault-cli/pkg/cli"
func main() { cli.Execute() }

// pkg/cli/cli.go
func Execute() {
    ctx := context.Background()
    if err := Run(ctx, os.Args[1:]); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
func Run(ctx context.Context, args []string) error {
    rootCmd := &cobra.Command{...}
    rootCmd.SetArgs(args)
    return rootCmd.ExecuteContext(ctx)
}
```
</context>

<requirements>
1. **Read the actual code first** — read `main.go`, `pkg/factory/factory.go`, `pkg/config/config.go` to understand current wiring.

2. **Create `pkg/cli/cli.go`** (package `cli`) with:
   - `Execute()` — creates `context.Background()`, sets up signal handler (SIGTERM/SIGINT), calls `Run()`, exits on error
   - `Run(ctx context.Context, args []string) error` — builds cobra root command with:
     - `--config` string flag (required via `MarkFlagRequired`)
     - `--verbose` bool flag (default false)
     - `SilenceUsage: true`
     - `RunE` that: sets up slog level based on verbose, loads config via factory, logs startup info, creates notifier + watcher via factory, runs watcher, handles graceful shutdown with 5s timeout

3. **Simplify `main.go`** to:
   ```go
   package main
   import "github.com/bborbe/task-watcher/pkg/cli"
   func main() { cli.Execute() }
   ```

4. **Signal handling and shutdown** — preserve the current behavior:
   - Signal cancels context
   - Watcher goroutine finishes within 5s → exit 0
   - Timeout → exit 1

5. **Add `github.com/spf13/cobra` dependency:**
   ```bash
   go get github.com/spf13/cobra
   ```

6. **Verify `--help` output** shows only `--config` and `--verbose` (no glog flags like `-alsologtostderr`, `-log_dir`, `-v`).

7. **Create `pkg/cli/suite_test.go`** with standard Ginkgo test suite.

8. **Create `pkg/cli/cli_test.go`** with Ginkgo tests covering:
   - `Run(ctx, []string{})` without `--config` returns error (missing required flag)
   - `Run(ctx, []string{"--config", "/nonexistent.yaml"})` returns error (file not found)
   - `Run(ctx, []string{"--help"})` output contains `--config` and `--verbose` but NOT `alsologtostderr`

9. Run `make test` — all tests must pass.

10. Run `make precommit` — must pass.
</requirements>

<constraints>
- `context.Background()` created exactly once in `Execute()`
- No stdlib `flag` package usage anywhere
- No `github.com/golang/glog` direct imports
- Logging uses `log/slog` only
- Factory functions called from `RunE`, not at module level
- `make precommit` must pass
- Do NOT modify `pkg/pkg_suite_test.go`
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
```bash
# Build and check help output
go build -o /tmp/task-watcher .
/tmp/task-watcher --help 2>&1 | grep -v "alsologtostderr\|log_dir\|log_link\|logbuflevel\|logtostderr\|stderrthreshold\|vmodule"

# Verify no glog flags leak
/tmp/task-watcher --help 2>&1 | grep -c "alsologtostderr" | grep -q "^0$"

# Missing config flag
/tmp/task-watcher 2>&1; echo "exit: $?"
# Expected: error about required config flag, exit 1

# Full checks
make test
make precommit
```
All must exit with code 0.
</verification>
