---
status: completed
summary: Added --version flag to CLI via cobra Version field and var version = "dev" in pkg/cli, updated Makefile LDFLAGS to target pkg/cli.version
container: task-watcher-010-spec-002-fix-version-flag
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T11:54:37Z"
queued: "2026-03-21T11:54:37Z"
started: "2026-03-21T11:54:38Z"
completed: "2026-03-21T11:58:47Z"
---

<summary>
- `--version` flag prints the binary version and exits
- Version injected at build time via ldflags into `pkg/cli.version`
- Makefile ldflags target updated from `main.version` to `pkg/cli.version`
- Default version is "dev" when built without ldflags
- `make precommit` passes after changes
</summary>

<objective>
Add `--version` support to the CLI using cobra's built-in version feature, matching the vault-cli pattern.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-cli.md` for the cobra CLI pattern.

Before writing any code, read:
- `pkg/cli/cli.go` — current cobra setup
- `Makefile` — current LDFLAGS definition

Reference: vault-cli pattern:
```go
// pkg/cli/cli.go
var version = "dev"

rootCmd := &cobra.Command{
    Version: version,
    ...
}
```
```makefile
# Makefile
LDFLAGS := -X github.com/bborbe/vault-cli/pkg/cli.version=$(VERSION)
```
</context>

<requirements>
1. **Read `pkg/cli/cli.go` and `Makefile`** before making changes.

2. **Add `var version = "dev"` to `pkg/cli/cli.go`** at package level.

3. **Set `Version: version` on the cobra root command.**

4. **Update Makefile** LDFLAGS from `-X main.version=$(VERSION)` to `-X github.com/bborbe/task-watcher/pkg/cli.version=$(VERSION)`.

5. **Remove any `var version` from `main.go`** if it exists.

6. Run `make test` — all tests must pass.

7. Run `make precommit` — must pass.
</requirements>

<constraints>
- Only change `pkg/cli/cli.go` and `Makefile` (and `main.go` if it has a stale version var)
- Do NOT modify business logic
- `make precommit` must pass
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
```bash
go build -o /tmp/task-watcher .
/tmp/task-watcher --version
# Expected: "task-watcher version dev"

go build -ldflags "-X github.com/bborbe/task-watcher/pkg/cli.version=v1.2.3" -o /tmp/task-watcher .
/tmp/task-watcher --version
# Expected: "task-watcher version v1.2.3"

make test
make precommit
```
All must exit with code 0.
</verification>
