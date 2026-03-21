---
status: completed
summary: Updated dry_run.go Notify method to marshal JSON body and log Content-Type header, and fixed factory_test.go to pass config.Config to CreateDryRunNotifier.
container: task-watcher-013-fix-dry-run-show-request
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T13:14:57Z"
queued: "2026-03-21T13:14:57Z"
started: "2026-03-21T13:15:22Z"
completed: "2026-03-21T13:22:01Z"
---

<summary>
- Dry-run log output now includes the JSON body and Content-Type header
- Log mirrors the exact HTTP request the real notifier would send
- Output is complete enough to reproduce the request with curl
</summary>

<objective>
Add JSON body and Content-Type header to dry-run log output so it shows the complete HTTP request that would be sent.
</objective>

<context>
Read CLAUDE.md for project conventions.

Before writing any code, read:
- `pkg/notify/dry_run.go` — current dry-run Notify method (already logs method + URL)
- `pkg/notify/notify.go` — HTTP notifier for reference (shows json.Marshal + Content-Type header)
</context>

<requirements>
1. **Read `pkg/notify/dry_run.go` and `pkg/notify/notify.go`** before making changes.

2. **Update `dry_run.go` `Notify` method only** — change `_ context.Context` to `ctx context.Context`, add `encoding/json` and `github.com/bborbe/errors` imports, marshal the notification to JSON and add it to the log:
   ```go
   body, err := json.Marshal(n)
   if err != nil {
       return errors.Wrapf(ctx, err, "marshal notification")
   }
   slog.Info("dry-run: would send webhook",
       "method", "POST",
       "url", d.webhookURL,
       "header", "Content-Type: application/json",
       "body", string(body),
   )
   ```

3. Run `make test` — all tests must pass.

4. Run `make precommit` — must pass.
</requirements>

<constraints>
- Only change `pkg/notify/dry_run.go`
- `Notifier` interface must NOT change
- Keep deduplication logic unchanged
- `make precommit` must pass
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
```bash
make precommit
```
Must exit with code 0.
</verification>
