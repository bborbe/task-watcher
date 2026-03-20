---
status: completed
spec: ["001"]
summary: Implemented pkg/config with YAML config loader, field validation, ~/  expansion, Ginkgo tests at 91.7% coverage, and Counterfeiter mock generation.
container: task-watcher-003-spec-001-config
dark-factory-version: v0.59.5-dirty
created: "2026-03-20T19:15:00Z"
queued: "2026-03-20T22:00:54Z"
started: "2026-03-20T22:00:56Z"
completed: "2026-03-20T22:07:09Z"
branch: dark-factory/core-packages
---

<summary>
- A new `pkg/config/` package exists that loads and validates YAML task-watcher configuration
- Configuration is loaded from a file path and returns a strongly-typed Go struct or a descriptive error
- All required fields are validated: vault path, assignee, at least one status, at least one phase, webhook URL
- The vault path has `~/` expanded to the user's home directory automatically
- Missing required fields produce errors that name the specific missing field
- The package has a Ginkgo/Gomega test suite with ≥80% statement coverage
- `make precommit` passes after this prompt
</summary>

<objective>
Implement `pkg/config/` — the foundation of task-watcher. This package loads a YAML configuration file, validates that all required fields are present, and expands `~/` in the vault path. All other packages depend on the `Config` type it produces.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega test conventions and Counterfeiter usage.
Read `/home/node/.claude/docs/go-precommit.md` for linter limits (funlen 80, nestif 4, golines 100).
Read `/home/node/.claude/docs/go-error-wrapping.md` for error wrapping patterns (`bborbe/errors`, never `fmt.Errorf`).

Current project state:
- `main.go` — minimal placeholder (just prints "task-watcher")
- `pkg/pkg_suite_test.go` — existing Ginkgo suite bootstrap, do NOT modify
- `go.mod` — does NOT yet import `gopkg.in/yaml.v3` or `github.com/bborbe/errors` — add them if needed via `go get` and `go mod vendor`
</context>

<requirements>
1. Create `pkg/config/config.go` with the following:
   - A `Config` struct with fields:
     - `VaultPath string` (from YAML key `vault.path`)
     - `Assignee string` (from YAML key `assignee`)
     - `Statuses []string` (from YAML key `statuses`)
     - `Phases []string` (from YAML key `phases`)
     - `Webhook string` (from YAML key `webhook`)
   - A `Loader` interface with a single method:
     ```go
     Load(ctx context.Context) (Config, error)
     ```
   - A private `loader` struct with a `filePath string` field
   - A constructor: `func NewLoader(filePath string) Loader`
   - The `Load` implementation must:
     a. Read the file at `filePath` (return descriptive error with path if not found)
     b. Parse YAML into Config using `go.yaml.in/yaml/v3` (already in go.mod as a transitive dep — check `go.mod` vendor; if not available use `gopkg.in/yaml.v3`)
     c. Validate required fields — return an error naming the missing field for each:
        - `vault.path` must be non-empty
        - `assignee` must be non-empty
        - `statuses` must have at least one entry
        - `phases` must have at least one entry
        - `webhook` must be non-empty
     d. Expand `~/` prefix in `VaultPath` using `os.UserHomeDir()` — replace only a leading `~/`
     e. Return the populated `Config` or the first validation error encountered
   - Add a `//go:generate counterfeiter -o mocks/config_loader.go --fake-name FakeConfigLoader . Loader` annotation above the `Loader` interface

2. Add a counterfeiter `//go:generate` directive and ensure `mocks/` directory will be created by running `go generate ./pkg/config/...`

3. Create `pkg/config/config_test.go` with a Ginkgo test suite that covers:
   - Successfully loading a valid YAML file (all fields present)
   - Error when file does not exist (check error message contains the file path)
   - Error when `vault.path` is missing (check error message names the field)
   - Error when `assignee` is missing
   - Error when `statuses` is empty
   - Error when `phases` is empty
   - Error when `webhook` is missing
   - `~/` expansion: given a config with `vault.path: ~/notes`, the returned `VaultPath` starts with the actual home directory and does NOT contain `~`
   - Use `os.CreateTemp` / `os.WriteFile` for temp YAML files in tests; clean up with `DeferCleanup`

4. Create `pkg/config/suite_test.go` registering the Ginkgo suite (package `config_test`):
   ```go
   package config_test

   import (
       "testing"
       . "github.com/onsi/ginkgo/v2"
       . "github.com/onsi/gomega"
   )

   func TestConfig(t *testing.T) {
       RegisterFailHandler(Fail)
       RunSpecs(t, "Config Suite")
   }
   ```

5. Run `go mod tidy && go mod vendor` after adding any new imports to ensure `vendor/` is up to date.

6. Run `go generate ./pkg/config/...` to produce `pkg/config/mocks/config_loader.go` (the Counterfeiter fake). Commit the generated file.

7. Run `make test` and verify all tests pass before proceeding to `make precommit`.

8. Run `make precommit` — must pass.

Example YAML config format (for reference in tests):
```yaml
vault:
  path: ~/notes/vault
assignee: alice
statuses:
  - active
  - in-review
phases:
  - planning
  - execution
webhook: https://hooks.example.com/notify
```
</requirements>

<constraints>
- vault-cli is imported as a Go library (`github.com/bborbe/vault-cli`), never executed via shell — this package does NOT use vault-cli yet, that's for pkg/watcher
- Factory functions use pure composition: no I/O, no `context.Background()` inside constructors
- Tests use Ginkgo/Gomega (package `config_test`) with Counterfeiter-generated mocks for all cross-package interfaces
- Error handling follows `github.com/bborbe/errors` patterns — use `errors.Wrapf(ctx, err, "message")` not `fmt.Errorf`
- `make precommit` must pass (linting, tests, formatting)
- `pkg/pkg_suite_test.go` must continue to work — do NOT modify it
- No `//nolint` directives without explanation
- Do NOT commit — dark-factory handles git
- Existing tests must still pass
</constraints>

<verification>
```bash
make test
make precommit
```
Both must exit with code 0.

Check coverage for `pkg/config`:
```bash
go test -coverprofile=/tmp/cover.out -mod=vendor ./pkg/config/... && go tool cover -func=/tmp/cover.out
```
Statement coverage for `pkg/config` must be ≥80%.
</verification>
