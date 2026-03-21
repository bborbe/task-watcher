---
status: completed
spec: ["002"]
summary: Created pkg/factory with pure composition factory functions CreateConfigLoader, CreateNotifier, and CreateWatcher, plus Ginkgo test suite with 100% coverage.
container: task-watcher-007-spec-002-factory
dark-factory-version: v0.59.5-dirty
created: "2026-03-20T19:30:00Z"
queued: "2026-03-21T10:58:16Z"
started: "2026-03-21T11:03:40Z"
completed: "2026-03-21T11:08:24Z"
branch: dark-factory/main-wiring
---

<summary>
- A new `pkg/factory/` package exists with pure composition factory functions for all core components
- `CreateConfigLoader` wires a config loader from a file path — no I/O at construction time
- `CreateNotifier` wires an HTTP notifier from a validated config — no network calls at construction time
- `CreateWatcher` wires a task watcher from a config and a notifier — no filesystem access at construction time
- All factory functions accept only value types and interfaces as parameters — no side effects, no context creation
- The factory package has a Ginkgo/Gomega test suite verifying that constructed objects are non-nil and satisfy their interfaces
- `make precommit` passes after this prompt
</summary>

<objective>
Implement `pkg/factory/factory.go` — pure composition functions that wire the three core packages (config, notify, watcher) into ready-to-use objects. Main.go will call these factories once to build the dependency graph; no business logic belongs here.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-factory-pattern.md` for the `Create*` prefix, zero-logic factories, and constructor vs factory distinction.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega test conventions.
Read `/home/node/.claude/docs/go-precommit.md` for linter limits.

Preconditions (previous prompts must have run):
- `pkg/config/config.go` — defines `Config` struct and `Loader` interface
- `pkg/notify/notify.go` — defines `Notifier` interface and `Notification` struct; constructor `NewNotifier`
- `pkg/watcher/watcher.go` — defines `Watcher` interface; constructor `NewWatcher`

Before writing any code, read each of these files to confirm the exact interface signatures and constructor parameters.
</context>

<requirements>
1. **Critical first step — read the actual API before writing any code:**
   - `pkg/config/config.go` — find the config loading API. It may be a `Loader` interface with `Load(ctx) (Config, error)`, or a standalone function like `LoadConfig(path string) (Config, error)`. Adapt the factory to match whatever exists.
   - `pkg/notify/notify.go` — confirm `Notifier` interface signature and constructor name/params
   - `pkg/watcher/watcher.go` — confirm `Watcher` interface signature and constructor name/params

2. Create `pkg/factory/factory.go` with factory functions (package `factory`).

   **Adapt signatures to match the actual API found in step 1.** The examples below assume a `Loader` interface — if config uses a standalone function instead, wrap it or adjust the factory accordingly (e.g., return a `func(context.Context) (config.Config, error)` closure, or skip the config factory and have main call the function directly).

   Example signatures (adapt to actual API):
   ```go
   // CreateConfigLoader constructs a config.Loader for the given file path.
   // Pure composition: no I/O, no context creation.
   func CreateConfigLoader(filePath string) config.Loader

   // CreateNotifier constructs a notify.Notifier from a validated config.
   // Pure composition: no network calls at construction time.
   func CreateNotifier(cfg config.Config) notify.Notifier

   // CreateWatcher constructs a watcher.Watcher that observes the vault and
   // forwards matching task events to the notifier.
   // Pure composition: no filesystem access at construction time.
   func CreateWatcher(cfg config.Config, notifier notify.Notifier) watcher.Watcher
   ```

   Each function must:
   - Contain zero business logic (no conditionals, no loops, no error returns)
   - Call only the constructors from the respective packages
   - Return the interface type (not the concrete struct)
   - Adapt constructor parameters as needed — if a constructor requires fields from `cfg`, pass them directly
   - **Correctness over specification fidelity** — if the actual API differs from these examples, match the real API

3. Create `pkg/factory/suite_test.go` (package `factory_test`):
   ```go
   package factory_test

   import (
       "testing"
       . "github.com/onsi/ginkgo/v2"
       . "github.com/onsi/gomega"
   )

   func TestFactory(t *testing.T) {
       RegisterFailHandler(Fail)
       RunSpecs(t, "Factory Suite")
   }
   ```

4. Create `pkg/factory/factory_test.go` (package `factory_test`) with Ginkgo tests covering:
   - `CreateConfigLoader("/some/path")` returns a non-nil `config.Loader`
   - `CreateNotifier(config.Config{Webhook: "http://example.com"})` returns a non-nil `notify.Notifier`
   - `CreateWatcher(cfg, notifier)` returns a non-nil `watcher.Watcher` — use a `FakeNotifier` from project-root `mocks/` package (`github.com/bborbe/task-watcher/mocks`) as the notifier argument

   These tests verify construction succeeds, not runtime behavior (which is tested in each package's own tests).

5. Run `make test` and verify all tests pass before running `make precommit`.

6. Run `make precommit` — must pass.
</requirements>

<constraints>
- Factory functions must be pure composition: no I/O, no `context.Background()` inside factories, no conditionals, no loops, no switch statements
- Factory functions use `Create*` prefix — NOT `New*`
- Return interface types, not concrete structs
- Error handling follows `github.com/bborbe/errors` patterns — though factories should not return errors at all
- `make precommit` must pass (linting, tests, formatting)
- `pkg/pkg_suite_test.go` must continue to work — do NOT modify it
- No `//nolint` directives without explanation
- Do NOT commit — dark-factory handles git
- Existing tests (pkg/config, pkg/notify, pkg/watcher) must still pass
- vault-cli is imported as a Go library, never executed via shell
- Single assignee + single webhook per process — no multi-tenant support
</constraints>

<verification>
```bash
make test
make precommit
```
Both must exit with code 0.

Check coverage for `pkg/factory`:
```bash
go test -coverprofile=/tmp/cover.out -mod=vendor ./pkg/factory/... && go tool cover -func=/tmp/cover.out
```
Statement coverage for `pkg/factory` must be ≥80%.
</verification>
