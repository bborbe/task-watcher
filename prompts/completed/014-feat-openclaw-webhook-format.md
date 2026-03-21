---
status: completed
summary: Added OpenClaw webhook format support with format/webhook_token config fields, OpenClaw notifier sending Bearer-authed payloads to /hooks/agent, dry-run variant, factory functions, format selection in cli.go, and full test coverage.
container: task-watcher-014-feat-openclaw-webhook-format
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T15:01:16Z"
queued: "2026-03-21T15:01:16Z"
started: "2026-03-21T15:01:18Z"
completed: "2026-03-21T15:08:39Z"
---

<summary>
- New `format` config field selects webhook payload format (`generic` or `openclaw`)
- New `webhook_token` config field for Authorization Bearer header
- OpenClaw format sends `{name, message, wakeMode, deliver}` payload to `/hooks/agent`
- Generic format unchanged (current `{task_name, phase, assignee}` payload)
- Dry-run notifier updated to show format-specific request details
- Config validation requires `webhook_token` when format is `openclaw`
- `make precommit` passes after changes
</summary>

<objective>
Add OpenClaw webhook format support so task-watcher can send notifications directly to OpenClaw's `/hooks/agent` endpoint with the correct payload shape and auth header.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-factory-pattern.md` for factory conventions.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega test conventions.
Read `/home/node/.claude/docs/go-error-wrapping.md` for error patterns.

Before writing any code, read:
- `pkg/config/config.go` — Config struct and YAML parsing
- `pkg/notify/notify.go` — current HTTP notifier
- `pkg/notify/dry_run.go` — current dry-run notifier
- `pkg/factory/factory.go` — notifier factory functions
- `pkg/cli/cli.go` — where notifiers are created
</context>

<requirements>
1. **Read all files listed in context** before making changes.

2. **Update `pkg/config/config.go`**:
   - Add `Format string` and `WebhookToken string` fields to `Config` struct
   - Add `format` and `webhook_token` to `rawConfig` YAML tags
   - Default `format` to `"generic"` if empty
   - Validate: if `format` is `"openclaw"`, `webhook_token` must not be empty
   - Validate: `format` must be `"generic"` or `"openclaw"` (reject unknown values)

3. **Create `pkg/notify/openclaw.go`** — OpenClaw notifier implementation:
   - Implements `Notifier` interface
   - Constructor: `NewOpenClawNotifier(webhookURL string, token string, httpClient *http.Client) Notifier`
   - Sends POST with `Authorization: Bearer <token>` header
   - Payload format:
     ```json
     {
       "name": "task-watcher",
       "message": "Task update: <TaskName>. Assignee: <Assignee>. Phase: <Phase>.",
       "wakeMode": "now",
       "deliver": false
     }
     ```
   - Same deduplication pattern as generic notifier (taskName:phase key)
   - Same response status check as generic notifier

4. **Add `CreateOpenClawNotifier` to `pkg/factory/factory.go`**:
   - `CreateOpenClawNotifier(cfg config.Config) notify.Notifier` → calls `notify.NewOpenClawNotifier(cfg.Webhook, cfg.WebhookToken, http.DefaultClient)`
   - Keep existing `CreateNotifier` unchanged (generic format)
   - Factory functions remain pure composition — no conditionals

5. **Update `pkg/cli/cli.go`** to select notifier based on format:
   - The if/else for format selection lives in `cli.go` RunE, not in factory
   - `"generic"` → `factory.CreateNotifier(cfg)`
   - `"openclaw"` → `factory.CreateOpenClawNotifier(cfg)`
   - Same pattern for dry-run: `"generic"` → `factory.CreateDryRunNotifier(cfg)`, `"openclaw"` → `factory.CreateDryRunOpenClawNotifier(cfg)`

6. **Update dry-run notifier** (`pkg/notify/dry_run.go`):
   - Create a second dry-run constructor `NewDryRunOpenClawNotifier(webhookURL string, token string) Notifier` that logs the OpenClaw payload shape and auth header
   - Keep existing `NewDryRunNotifier` unchanged for generic format
   - Add matching factory function `CreateDryRunOpenClawNotifier(cfg config.Config) notify.Notifier`

8. **Create tests in `pkg/notify/openclaw_test.go`**:
   - Successful POST with correct payload shape and auth header (use `httptest.NewServer`)
   - Deduplication (same key not sent twice)
   - Non-2xx response returns error

9. **Update config tests** for new fields and validation.

10. **Update `CHANGELOG.md`** — add entry under `## Unreleased` for the new OpenClaw webhook format.

11. Run `make test` — all tests must pass.

12. Run `make precommit` — must pass.
</requirements>

<constraints>
- `Notifier` interface must NOT change
- Generic notifier (`notify.go`) must NOT be modified
- Format selection logic in factory, not in notifier implementations
- Each notifier implementation in its own file
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

Example config for OpenClaw:
```yaml
vault:
  path: ~/Documents/Obsidian/Personal
assignee: bborbe
statuses:
  - in_progress
phases:
  - planning
  - in_progress
format: openclaw
webhook: http://localhost:9999/hooks/agent
webhook_token: my-secret-token
```
</verification>
