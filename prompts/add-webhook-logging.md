---
status: draft
---

<summary>
- Successful webhook calls produce an info-level log line with task name, phase, and HTTP status
- Debug mode shows full request details (method, URL, headers, body) before sending
- Authorization tokens are masked in all log output
- Duplicate-skipped webhooks log at debug level instead of silently returning
- No changes to dry-run notifiers (already log correctly)
- Existing tests still pass
</summary>

<objective>
Add logging to real notifiers so webhook calls are visible. Currently only dry-run notifiers log; real notifiers are silent.
</objective>

<context>
- Read `CLAUDE.md` and `docs/dod.md` for project conventions
- `pkg/notify/dry_run.go` has the logging pattern to follow (slog.Info with method, url, header, body)
- `pkg/notify/notify.go` is the generic notifier — sends JSON, no logging
- `pkg/notify/openclaw.go` is the OpenClaw notifier — sends structured payload with auth header, no logging
- `log/slog` is already imported in dry_run.go — add it to notify.go and openclaw.go
- `--verbose` sets slog level to Debug via `slog.SetLogLoggerLevel(slog.LevelDebug)` in `pkg/cli/cli.go`
</context>

<requirements>
1. In `pkg/notify/notify.go` `Notify()` method: add `slog.Debug` before `httpClient.Do` with method, url, Content-Type header, body. Add `slog.Info("webhook sent", "task", notification.TaskName, "phase", notification.Phase, "status", resp.StatusCode)` after the non-2xx error check, before the final `return nil`. Add `slog.Debug("webhook skipped (duplicate)", "task", ..., "phase", ...)` in dedup-skip path before `return nil`.

2. In `pkg/notify/openclaw.go` `Notify()` method: same pattern as requirement 1. In Debug log, mask auth header as `Authorization: Bearer ***` (never log actual token).

3. Do NOT change `pkg/notify/dry_run.go` — it already logs correctly.

4. Do NOT commit — dark-factory handles git.
</requirements>

<constraints>
- Debug log goes BEFORE the HTTP call (shows what's about to be sent)
- Info log goes AFTER successful response (confirms delivery), after the non-2xx error check
- Do NOT log full Authorization header value — use `Authorization: Bearer ***`
- Use `log/slog` (already used in dry_run.go)
- Dedup-skip currently returns nil silently — add Debug log before return
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
make precommit
</verification>
