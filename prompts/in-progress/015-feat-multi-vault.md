---
status: executing
container: task-watcher-015-feat-multi-vault
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T15:28:04Z"
queued: "2026-03-21T15:28:04Z"
started: "2026-03-21T15:28:06Z"
---

<summary>
- Config switches from single `vault.path` to a `vaults` map (name → path + tasks_dir)
- Each vault gets its own `ops.WatchTarget`, all watched by one `WatchOperation.Execute` call
- Events routed to the correct vault path using `WatchEvent.Vault` field
- Startup log shows all vault names and paths being watched
- At least one vault required, each must have `path` and `tasks_dir`
- `make precommit` passes after changes
</summary>

<objective>
Replace single-vault config with a multi-vault map so one task-watcher process watches multiple Obsidian vaults simultaneously.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega test conventions.
Read `/home/node/.claude/docs/go-error-wrapping.md` for error patterns.

Before writing any code, read:
- `pkg/config/config.go` — current `Config` struct with single `VaultPath`
- `pkg/watcher/watcher.go` — current `Watch` method building single `WatchTarget`
- `pkg/factory/factory.go` — current `CreateWatcher` signature
- `pkg/cli/cli.go` — startup logging

Reference: vault-cli uses `vaults` map in its config. The `ops.WatchTarget` struct:
```go
type WatchTarget struct {
    VaultPath string
    VaultName string
    WatchDirs []string
}
```
And `WatchEvent` has a `Vault` field (vault name) to identify the source.
</context>

<requirements>
1. **Read all files listed in context** before making changes.

2. **Update `pkg/config/config.go`**:
   - Add `VaultConfig` struct: `Name string`, `Path string`, `TasksDir string`
   - Replace `VaultPath string` in `Config` with `Vaults []VaultConfig`
   - Update `rawConfig` YAML to parse:
     ```yaml
     vaults:
       personal:
         path: ~/Documents/Obsidian/Personal
         tasks_dir: "24 Tasks"
     ```
   - Map keys become vault names, iterate to build `[]VaultConfig`
   - Apply `~/` expansion to each vault path
   - Validate: at least one vault, each must have non-empty `path` and `tasks_dir`

3. **Update `pkg/watcher/watcher.go`**:
   - Build `[]ops.WatchTarget` from `cfg.Vaults` (one per vault, `WatchDirs` = `[]string{v.TasksDir}`)
   - Build a `map[string]string` from vault name → vault path for event routing
   - Create one `storage.NewStorage` per vault (each with its own `TasksDir`)
   - Build a `map[string]taskReader` from vault name → storage for event routing
   - In `handleEvent`, use `event.Vault` to look up the correct vault path and storage
   - If vault name not found in map, log warning and skip

4. **Update `pkg/factory/factory.go`** if `CreateWatcher` signature changes.

5. **Update `pkg/cli/cli.go`** startup log to show all vaults:
   ```go
   for _, v := range cfg.Vaults {
       slog.Info("watching vault", "name", v.Name, "path", v.Path, "tasksDir", v.TasksDir)
   }
   ```

6. **Update tests**:
   - `pkg/config/config_test.go`: multi-vault YAML parsing, `~/` expansion per vault, validation (empty vaults map, missing path, missing tasks_dir)
   - `pkg/watcher/watcher_test.go`: multi-vault WatchTarget construction, event routing by vault name
   - `pkg/factory/factory_test.go`: CreateWatcher with multi-vault config returns non-nil

7. **Update `CLAUDE.md`**:
   - Architecture section: update config description to mention multi-vault
   - Key Design Decisions: change "single assignee + single webhook per process, no multi-tenant config" to reflect multi-vault support (still single assignee/webhook, but multiple vaults)

8. **Update `CHANGELOG.md`** with entry under `## Unreleased`.

9. Run `make test` — all tests must pass.

10. Run `make precommit` — must pass.
</requirements>

<constraints>
- `Watcher` interface unchanged: `Watch(ctx context.Context) error`
- `Notifier` interface unchanged
- Factory functions remain pure composition
- All vaults share same assignee, statuses, phases, webhook config
- `make precommit` must pass
- Do NOT modify `pkg/pkg_suite_test.go`
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
```bash
make test
make precommit
```
All must exit with code 0.

Example config:
```yaml
vaults:
  personal:
    path: ~/Documents/Obsidian/Personal
    tasks_dir: "24 Tasks"
  octopus:
    path: ~/Documents/Obsidian/Octopus
    tasks_dir: "24 Tasks"
assignee: bborbe
statuses:
  - in_progress
phases:
  - planning
  - in_progress
format: openclaw
webhook: http://localhost:9999/hooks/agent
webhook_token: my-token
```
</verification>
