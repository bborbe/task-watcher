---
status: completed
summary: Added default config path ~/.task-watcher/config.yaml in config loader and made --config flag optional in CLI
container: task-watcher-011-fix-default-config-path
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T12:14:40Z"
queued: "2026-03-21T12:14:40Z"
started: "2026-03-21T12:14:42Z"
completed: "2026-03-21T12:25:27Z"
---

<summary>
- Config path defaults to `~/.task-watcher/config.yaml` when `--config` is not provided
- `--config` flag overrides the default path
- Config loading handles missing default file with a clear error message
- `MarkFlagRequired("config")` removed from cobra setup
- `make precommit` passes after changes
</summary>

<objective>
Add a default config path so `task-watcher` works without `--config`, matching the vault-cli pattern where `--config` is optional with a sensible default.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-error-wrapping.md` for error handling patterns.

Before writing any code, read:
- `pkg/cli/cli.go` — current cobra setup with `MarkFlagRequired("config")`
- `pkg/config/config.go` — current `Loader` implementation

Reference: vault-cli handles default config in the loader (pkg/config/config.go):
```go
func (c *configLoader) Load(ctx context.Context) (*Config, error) {
    configPath := c.configPath
    if configPath == "" {
        homeDir, _ := os.UserHomeDir()
        configPath = filepath.Join(homeDir, ".vault-cli", "config.yaml")
    }
    // ...
}
```
</context>

<requirements>
1. **Read `pkg/cli/cli.go` and `pkg/config/config.go`** before making changes.

2. **Update `pkg/config/config.go` `Load` method** to resolve a default path when `filePath` is empty:
   - If `filePath` is empty, default to `~/.task-watcher/config.yaml`
   - Use `os.UserHomeDir()` + `filepath.Join` (not string concatenation)
   - If the resolved file doesn't exist, return a clear error including the path

3. **Update `pkg/cli/cli.go`**:
   - Remove `MarkFlagRequired("config")` and its error handling
   - Keep `--config` flag with default `""` (empty string triggers default path in loader)

4. **Update tests in `pkg/config/config_test.go`** to cover:
   - Empty `filePath` resolves to `~/.task-watcher/config.yaml`
   - Explicit `filePath` is used as-is (existing tests should already cover this)

5. Run `make test` — all tests must pass.

6. Run `make precommit` — must pass.
</requirements>

<constraints>
- Default path logic lives in the config loader, not in CLI code
- `--config` flag remains optional, overrides default when provided
- No new packages or dependencies
- `make precommit` must pass
- Do NOT modify `pkg/pkg_suite_test.go`
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
```bash
go build -o /tmp/task-watcher .

# Without --config (should try default path)
/tmp/task-watcher 2>&1; echo "exit: $?"
# Expected: error mentions ~/.task-watcher/config.yaml, exit 1

# With --config (should use provided path)
/tmp/task-watcher --config /nonexistent.yaml 2>&1; echo "exit: $?"
# Expected: error mentions /nonexistent.yaml, exit 1

make test
make precommit
```
All verification commands must behave as described.
</verification>
