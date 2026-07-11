---
status: completed
summary: Corrected misleading $XDG_CONFIG_HOME wording in --config flag help, README, and added Long help text to task-watcher CLI
execution_id: task-watcher-help-xdg-exec-024-help-xdg-config-path
dark-factory-version: v0.191.0
created: "2026-07-11T13:18:36Z"
queued: "2026-07-11T13:18:36Z"
started: "2026-07-11T13:19:14Z"
completed: "2026-07-11T13:20:45Z"
---

<summary>
- `task-watcher --help` now documents the config file location accurately
- The `--config` flag help currently claims `$XDG_CONFIG_HOME/task-watcher/config.yaml`, but the tool does NOT read that env var — it hardcodes `~/.config/task-watcher/config.yaml`
- The misleading `$XDG_CONFIG_HOME/...` wording is corrected to the real path in both `--help` and the README
- A short configuration description is added to the command's help body
- No behavior change: config resolution already works XDG-first with legacy fallback; this only makes the help/README text truthful
- The existing `--help` test is extended to assert the corrected path is shown
</summary>

<objective>
Make `task-watcher --help` state the config file location truthfully. The code resolves `~/.config/task-watcher/config.yaml` (XDG) with fallback `~/.task-watcher/config.yaml` (legacy) and does not honor `$XDG_CONFIG_HOME`, yet the help and README claim `$XDG_CONFIG_HOME/...`. Correct the wording to the actual path. No resolution behavior changes.
</objective>

<context>
Read CLAUDE.md for project conventions.
This is a single-command cobra CLI (`github.com/spf13/cobra`); help is cobra-generated from the root command's fields and flag descriptions.
Read `pkg/cli/cli.go` — find `Run`. It builds `rootCmd := &cobra.Command{...}` with `Use: "task-watcher"` and `Short: ...` but NO `Long` field. The `--config` flag is registered with the description string `"path to config YAML file (default: $XDG_CONFIG_HOME/task-watcher/config.yaml, fallback: ~/.task-watcher/config.yaml)"`. That `$XDG_CONFIG_HOME/...` wording is the inaccuracy to fix.
Read `pkg/config/config.go` — `findConfigDir` builds `xdgDir := filepath.Join(homeDir, ".config", "task-watcher")` (it uses `os.UserHomeDir()`, NOT `$XDG_CONFIG_HOME`) with legacy fallback `filepath.Join(homeDir, ".task-watcher")`; `config.yaml` is appended. Use `~/.config/task-watcher/config.yaml` and `~/.task-watcher/config.yaml` as the exact paths.
Read `pkg/cli/cli_test.go` — the existing spec `It("--help output contains --config and --verbose but not alsologtostderr", ...)` captures `os.Stdout`, runs `cli.Run(ctx, []string{"--help"})`, and asserts via `ContainSubstring(...)`. This is the home for the new assertion. Test package is external (`package cli_test`), Ginkgo/Gomega.
Read `README.md` — it also states `$XDG_CONFIG_HOME/task-watcher/config.yaml`; correct it to match.
</context>

<requirements>
1. In `pkg/cli/cli.go`, in `Run`, change the `--config` flag description so the default path reads `~/.config/task-watcher/config.yaml` instead of `$XDG_CONFIG_HOME/task-watcher/config.yaml`. Keep the `fallback: ~/.task-watcher/config.yaml` part unchanged. Final string, e.g.: `path to config YAML file (default: ~/.config/task-watcher/config.yaml, fallback: ~/.task-watcher/config.yaml)`.
2. In the same `rootCmd`, add a `Long` field with a short configuration description that appears in `--help`, naming both paths, e.g.: `Watches vault task files and notifies agents via webhook.\n\nConfiguration: reads ~/.config/task-watcher/config.yaml (XDG), falling back to ~/.task-watcher/config.yaml (legacy). Override with --config.`
3. In `pkg/cli/cli_test.go`, extend the existing `--help` spec (matching its `ContainSubstring` Ginkgo/Gomega style) to assert the captured help output (a) contains `~/.config/task-watcher/config.yaml` AND (b) does NOT contain `$XDG_CONFIG_HOME` (`Expect(out).NotTo(ContainSubstring("$XDG_CONFIG_HOME"))`). The negative assertion is required: the positive path is also produced by the new `Long` field, so without (b) the test would still pass even if the stale `$XDG_CONFIG_HOME` flag wording were left in place.
4. In `README.md`, replace the `$XDG_CONFIG_HOME/task-watcher/config.yaml` occurrence(s) with `~/.config/task-watcher/config.yaml`. Do not otherwise restructure the README.
5. Add a `## Unreleased` section at the top of `CHANGELOG.md` (above the latest released entry) with a single bullet, e.g. `fix(help): correct misleading $XDG_CONFIG_HOME wording to the real ~/.config/task-watcher/config.yaml path in --help and README; add config location to command Long help`.
</requirements>

<constraints>
- Do NOT commit — dark-factory handles git.
- Existing tests must still pass.
- Do NOT add `$XDG_CONFIG_HOME` env-var override support — the fix is to make the text match the current behavior (hardcoded `~/.config`), not to add env-var handling. Env-var support is explicitly out of scope.
- Paths must be exact: `~/.config/task-watcher/config.yaml` (XDG) and `~/.task-watcher/config.yaml` (legacy).
</constraints>

<verification>
Run `make precommit` -- must pass (includes the extended `--help` test).
</verification>
