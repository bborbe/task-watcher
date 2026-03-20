---
status: completed
spec: ["001"]
summary: Implemented pkg/notify with Notifier interface, in-memory dedup, httptest-based tests (87.5% coverage), and generated FakeNotifier mock
container: task-watcher-004-spec-001-notify
dark-factory-version: v0.59.5-dirty
created: "2026-03-20T19:15:00Z"
queued: "2026-03-20T22:00:54Z"
started: "2026-03-20T22:07:13Z"
completed: "2026-03-20T22:12:33Z"
branch: dark-factory/core-packages
---

<summary>
- A new `pkg/notify/` package exists that sends HTTP webhook notifications for task events
- Each notification is an HTTP POST with a JSON body containing task name, phase, and assignee
- The notifier returns an error when the webhook responds with a non-2xx HTTP status code
- Duplicate notifications (same task name + phase combination) are silently suppressed — only the first fires
- The deduplication state is in-memory and resets on process restart (by design)
- The `Notifier` interface is mockable via Counterfeiter for use by `pkg/watcher`
- `make precommit` passes after this prompt
</summary>

<objective>
Implement `pkg/notify/` — the outbound HTTP delivery layer. The notifier sends a JSON POST to a configured webhook URL when a task phase matches. It deduplicates calls in-memory so the same task+phase event only triggers one HTTP request per process lifetime.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-testing.md` for Ginkgo/Gomega test conventions and Counterfeiter mock usage.
Read `/home/node/.claude/docs/go-precommit.md` for linter limits (funlen 80, nestif 4, golines 100).
Read `/home/node/.claude/docs/go-error-wrapping.md` for `bborbe/errors` patterns.

Precondition: `pkg/config/` must already exist (produced by the previous prompt). The `Config` type from `pkg/config/config.go` is used here to obtain the webhook URL and assignee.

Current project state:
- `pkg/config/config.go` — defines `Config` struct (VaultPath, Assignee, Statuses, Phases, Webhook)
- `pkg/pkg_suite_test.go` — Ginkgo suite bootstrap, do NOT modify
</context>

<requirements>
1. Create `pkg/notify/notify.go` with the following:
   - A `Notification` struct:
     ```go
     type Notification struct {
         TaskName string `json:"task_name"`
         Phase    string `json:"phase"`
         Assignee string `json:"assignee"`
     }
     ```
   - A `Notifier` interface:
     ```go
     type Notifier interface {
         Notify(ctx context.Context, notification Notification) error
     }
     ```
   - A `//go:generate counterfeiter -o mocks/notifier.go --fake-name FakeNotifier . Notifier` annotation above the `Notifier` interface
   - A private `notifier` struct with fields:
     - `webhookURL string`
     - `httpClient *http.Client` (injected for testability)
     - `seen map[string]struct{}` (deduplication map; key = `taskName+":"+phase`)
     - `mu sync.Mutex` (guards `seen`)
   - A constructor:
     ```go
     func NewNotifier(webhookURL string, httpClient *http.Client) Notifier
     ```
     Initialize `seen` as an empty `map[string]struct{}` in the constructor.
   - The `Notify` implementation must:
     a. Build the dedup key: `notification.TaskName + ":" + notification.Phase`
     b. Lock `mu`, check if key exists in `seen` — if so, unlock and return `nil` (silent no-op)
     c. Add key to `seen`, unlock `mu`
     d. Marshal `notification` to JSON using `encoding/json`
     e. Create an HTTP POST request to `webhookURL` with `Content-Type: application/json`
     f. Execute the request using `httpClient`
     g. Read and discard the response body, then close it (always, even on error)
     h. If HTTP status code is not in the 2xx range (< 200 or >= 300), return an error that includes the status code
     i. Return `nil` on success

2. Run `go generate ./pkg/notify/...` to produce `pkg/notify/mocks/notifier.go`.

3. Create `pkg/notify/suite_test.go` (package `notify_test`):
   ```go
   package notify_test

   import (
       "testing"
       . "github.com/onsi/ginkgo/v2"
       . "github.com/onsi/gomega"
   )

   func TestNotify(t *testing.T) {
       RegisterFailHandler(Fail)
       RunSpecs(t, "Notify Suite")
   }
   ```

4. Create `pkg/notify/notify_test.go` (package `notify_test`) with Ginkgo tests covering:
   - Successful notification: HTTP POST is sent to the correct URL, body contains correct JSON fields (task_name, phase, assignee), response 200 → no error
   - Non-2xx response (e.g. 500): Notify returns an error that includes the status code
   - Non-2xx response (e.g. 404): same error behavior
   - Deduplication — same task+phase called twice: only one HTTP request is made
   - Deduplication — different phase for same task: both HTTP requests are made (not deduplicated)
   - Deduplication — different task name for same phase: both requests are made
   - Use `net/http/httptest.NewServer` to create a test HTTP server; do NOT mock `http.Client` directly — instead inject a real `*http.Client` pointed at the test server
   - Use `DeferCleanup` to close the test server

5. Run `make test` and verify all tests pass.

6. Run `make precommit` — must pass.
</requirements>

<constraints>
- vault-cli is imported as a Go library (`github.com/bborbe/vault-cli`), never executed via shell — this package does NOT use vault-cli
- Factory functions use pure composition: no I/O, no `context.Background()` inside constructors
- Tests use Ginkgo/Gomega (package `notify_test`) with `httptest.NewServer` for HTTP testing — no manual HTTP client mocks
- Error handling follows `github.com/bborbe/errors` patterns — use `errors.Wrapf(ctx, err, "message")`, never `fmt.Errorf`
- `make precommit` must pass (linting, tests, formatting)
- `pkg/pkg_suite_test.go` must continue to work — do NOT modify it
- No `//nolint` directives without explanation
- Do NOT commit — dark-factory handles git
- Existing tests (pkg/config) must still pass
</constraints>

<verification>
```bash
make test
make precommit
```
Both must exit with code 0.

Check coverage for `pkg/notify`:
```bash
go test -coverprofile=/tmp/cover.out -mod=vendor ./pkg/notify/... && go tool cover -func=/tmp/cover.out
```
Statement coverage for `pkg/notify` must be ≥80%.
</verification>
