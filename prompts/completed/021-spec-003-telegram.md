---
status: completed
spec: ["003"]
summary: Added Telegram Bot API notifier and log notifier, deleted dry_run notifiers, updated factory and tests
container: task-watcher-021-spec-003-telegram
dark-factory-version: v0.67.3-dirty
created: "2026-03-22T10:00:00Z"
queued: "2026-03-22T11:25:01Z"
started: "2026-03-22T11:33:40Z"
completed: "2026-03-22T11:39:23Z"
branch: dark-factory/multi-watcher-config
---

<summary>
- New `telegram` notifier sends messages via the Telegram Bot API (`sendMessage` endpoint)
- New `log` notifier logs the would-be notification to stdout — replaces dry-run mode
- Both notifiers implement the existing `Notifier` interface with TTL-based dedup (same pattern as openclaw notifier)
- Telegram bot token is never logged (security constraint)
- Both have full Ginkgo/Gomega test coverage including TTL re-send tests
- `pkg/notify/dry_run.go` and its test file are deleted — `log` notifier is the replacement
</summary>

<objective>
Add the two remaining notifier types needed for the new watchers config format: a Telegram Bot API notifier for the `telegram` type, and a log notifier for the `log` type (which replaces the old dry-run mode). Both implement the existing `Notifier` interface unchanged.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega conventions.
Read `/home/node/.claude/docs/go-error-wrapping.md` for `bborbe/errors` patterns.
Read `/home/node/.claude/docs/go-security-linting.md` for gosec rules on file permissions and nosec annotations.

Key files to read before making changes:
- `pkg/notify/notify.go` — `Notifier` interface, `Notification` struct
- `pkg/notify/openclaw.go` — reference implementation with TTL dedup (copy this pattern)
- `pkg/notify/openclaw_test.go` — reference test structure (copy this pattern)
- `pkg/notify/dry_run.go` — to be deleted after writing log.go
- `pkg/notify/dry_run_test.go` — to be deleted after writing log_test.go

This prompt assumes `pkg/config/config.go` has already been updated (prompt 1) and the project compiles.
</context>

<requirements>
### 1. Add `pkg/notify/telegram.go`

The Telegram Bot API `sendMessage` endpoint:
```
POST https://api.telegram.org/bot<token>/sendMessage
Content-Type: application/json
{"chat_id": "<chat_id>", "text": "<text>"}
```

File content:
```go
// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify
```

Implement `NewTelegramNotifier`:
```go
// NewTelegramNotifier returns a Notifier that sends messages via the Telegram Bot API.
// The bot token is never logged.
func NewTelegramNotifier(
    token string,
    chatID string,
    httpClient *http.Client,
    dedupTTL time.Duration,
) Notifier
```

The private struct:
```go
type telegramNotifier struct {
    token      string
    chatID     string
    httpClient *http.Client
    dedupTTL   time.Duration
    seen       map[string]time.Time
    mu         sync.Mutex
}
```

The JSON payload struct:
```go
type telegramPayload struct {
    ChatID string `json:"chat_id"`
    Text   string `json:"text"`
}
```

The `Notify` method:
1. TTL dedup using `key := notification.TaskName + ":" + notification.Phase` (same pattern as openclaw.go)
2. Build payload:
   ```go
   payload := telegramPayload{
       ChatID: t.chatID,
       Text: fmt.Sprintf(
           "Task watcher: %s task changed (task: %s, phase: %s)",
           notification.Assignee,
           notification.TaskName,
           notification.Phase,
       ),
   }
   ```
3. Marshal to JSON.
4. Build request URL: `fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)` — do NOT log the URL (it contains the token).
5. `http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))`
6. Set `Content-Type: application/json`. Do NOT set Authorization header (token is in URL).
7. Log the request but redact the token from the URL in the log:
   ```go
   slog.Debug("sending telegram message",
       "method", http.MethodPost,
       "chat_id", t.chatID,
       "body", string(body),
   )
   ```
8. Execute request, drain body, check for non-2xx status.
9. On success: `slog.Info("telegram message sent", "task", notification.TaskName, "phase", notification.Phase, "status", resp.StatusCode)`

Error wrapping uses `github.com/bborbe/errors` — never `fmt.Errorf`.

### 2. Add `pkg/notify/log.go`

The log notifier simply logs what it would send — no HTTP calls.

```go
// NewLogNotifier returns a Notifier that logs notifications to stdout instead of sending HTTP requests.
// Use this as the action type for debugging or dry-run observation.
func NewLogNotifier(dedupTTL time.Duration) Notifier
```

Private struct:
```go
type logNotifier struct {
    dedupTTL time.Duration
    seen     map[string]time.Time
    mu       sync.Mutex
}
```

`Notify` method:
1. TTL dedup (same pattern).
2. Log at Info level:
   ```go
   slog.Info("log notifier: task event",
       "task", notification.TaskName,
       "phase", notification.Phase,
       "assignee", notification.Assignee,
   )
   ```
3. Return nil.

No HTTP client, no URL, no marshaling needed.

### 3. Delete `pkg/notify/dry_run.go` and `pkg/notify/dry_run_test.go`

These files implement `NewDryRunNotifier` and `NewDryRunOpenClawNotifier` which are replaced by `NewLogNotifier`.

Delete both files. If `pkg/factory/factory.go` references them (e.g., `CreateDryRunNotifier`, `CreateDryRunOpenClawNotifier`), update those functions to return `notify.NewLogNotifier(0)` temporarily — later prompt replaces factory entirely. Ensure the project compiles after deletion.

### 4. Add `pkg/notify/telegram_test.go`

Follow the exact same test structure as `pkg/notify/openclaw_test.go`. Use a `httptest.NewServer` to capture requests.

Tests to write:
- Sends POST to correct Telegram URL path (`/bot<token>/sendMessage`)
- Request body has correct `chat_id` and `text` fields
- `text` is `"Task watcher: alice task changed (task: my-task, phase: planning)"`
- Returns nil on success (2xx response)
- Returns error on non-2xx response
- Dedup: second identical notification within TTL is skipped (request count stays at 1)
- TTL re-send: after TTL expires, notification fires again (use 50ms TTL + 60ms sleep)
- Different task names are not deduped (each fires separately)
- Different phases for same task are not deduped

Use `atomic.Int32` for request counter (same as openclaw_test.go pattern).

The test server should verify the Authorization header is NOT set (token is in URL, not header).

### 5. Add `pkg/notify/log_test.go`

Tests to write:
- `Notify` returns nil
- Dedup: second call within TTL is skipped (test by checking no error, use a counter via a wrapped notifier or just verify behavior)
- TTL re-send after expiry
- Different tasks/phases are independent

Since `logNotifier` has no HTTP side effects, test the dedup behavior by capturing slog output with a custom `slog.Handler` in the test. Set a `slog.New(slog.NewTextHandler(&buf, nil))` as default logger before the test, then count log lines:
1. First call → one "log notifier: task event" line in buffer
2. Second call within TTL → no new line (deduped)
3. After TTL expiry → new line appears

This makes dedup observable without HTTP.

### 6. Run tests

```bash
make test
```

All tests must pass.
</requirements>

<constraints>
- `Notifier` interface is unchanged: `Notify(ctx context.Context, notification Notification) error`
- Telegram bot token must NOT appear in any log output (it appears in the URL — do not log the URL)
- TTL dedup pattern must match existing notifiers (optimistic write before HTTP call)
- Error wrapping uses `github.com/bborbe/errors` — never `fmt.Errorf`
- Ginkgo/Gomega tests, external test packages (`package notify_test`)
- Do NOT commit — dark-factory handles git
- Existing notify tests (notify_test.go, openclaw_test.go) must still pass
</constraints>

<verification>
```bash
make test
```
Must exit with code 0.
</verification>
