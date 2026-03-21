---
status: executing
container: task-watcher-017-log-version-on-start
dark-factory-version: v0.63.0
created: "2026-03-21T19:27:20Z"
queued: "2026-03-21T19:27:20Z"
started: "2026-03-21T19:27:22Z"
---

<summary>
- Version is logged on startup alongside existing fields
- Operators can confirm which build is running from the startup log
- No new log lines added — version appended to existing startup line
- Matches dark-factory startup logging pattern
- Existing log consumers are unaffected (additive field only)
</summary>

<objective>
Add version to the startup log line so operators can confirm which build is running without checking --version separately.
</objective>

<context>
- Read `CLAUDE.md` and `docs/dod.md` for project conventions
- `pkg/cli/cli.go` has the startup log at `slog.Info("task-watcher starting", ...)` and `var version = "dev"` at package level
- dark-factory logs version on start: `slog.Info("dark-factory starting", "version", version.Version)` in `main.go`
</context>

<requirements>
1. In `pkg/cli/cli.go`, add `"version", version` to the existing `slog.Info("task-watcher starting", ...)` call. Do not add a separate log line.
</requirements>

<constraints>
- Single line change — add field to existing slog call
- Do NOT add new log lines, imports, or other changes
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
make precommit
</verification>
