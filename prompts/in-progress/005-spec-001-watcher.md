---
status: approved
spec: ["001"]
created: "2026-03-20T19:15:00Z"
queued: "2026-03-20T22:00:54Z"
branch: dark-factory/core-packages
---

<summary>
- A new `pkg/watcher/` package exists that watches an Obsidian vault for task file changes
- The watcher detects when task files are created or modified in the vault's tasks directory
- Each changed file is read via vault-cli's storage package to extract task frontmatter (assignee, status, phase)
- Tasks are filtered against the configured assignee, allowed statuses, and allowed phases
- When a task matches all filters, the notifier is called with the task name, phase, and assignee
- Tasks that do not match the configured filters produce no notification
- Task files with missing or incomplete frontmatter are silently skipped
- `make precommit` passes after this prompt
</summary>

<objective>
Implement `pkg/watcher/` — the file-watching and filtering layer that ties config and notify together. The watcher uses vault-cli as a library to watch the vault's tasks directory and read task frontmatter. On each matching file event, it calls the notifier with the task details.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for interface/constructor/struct patterns.
Read `/home/node/.claude/docs/go-testing.go` for Ginkgo/Gomega and Counterfeiter mock conventions.
Read `/home/node/.claude/docs/go-precommit.md` for linter limits.
Read `/home/node/.claude/docs/go-error-wrapping.md` for `bborbe/errors` patterns.

Preconditions (previous prompts must have run):
- `pkg/config/config.go` — defines `Config` struct with `VaultPath`, `Assignee`, `Statuses []string`, `Phases []string`
- `pkg/notify/notify.go` — defines `Notifier` interface and `Notification{TaskName, Phase, Assignee}`
- `pkg/notify/mocks/notifier.go` — Counterfeiter fake `FakeNotifier`

**vault-cli API — already known, do not explore:**

Add vault-cli to go.mod:
```bash
go get github.com/bborbe/vault-cli@v0.48.6
go mod tidy
go mod vendor
```

**WatchOperation** (`github.com/bborbe/vault-cli/pkg/ops`):
```go
type WatchEvent struct {
    Event string // "created", "modified", "deleted", "renamed"
    Name  string // filename without .md extension (= task name)
    Vault string
    Path  string // relative path within vault
}

type WatchOperation interface {
    Execute(ctx context.Context, vaults []WatchTarget, handler func(WatchEvent) error) error
}

type WatchTarget struct {
    VaultPath string
    VaultName string
    WatchDirs []string
}

func NewWatchOperation() WatchOperation
```

**TaskStorage** (`github.com/bborbe/vault-cli/pkg/storage`):
```go
type Config struct {
    TasksDir string // subdirectory name, e.g. "24 Tasks"
}

func DefaultConfig() *Config      // TasksDir = "Tasks"
func NewTaskStorage(cfg *Config) TaskStorage

type TaskStorage interface {
    ReadTask(ctx context.Context, vaultPath string, taskID domain.TaskID) (*domain.Task, error)
}
```

**domain.Task** (`github.com/bborbe/vault-cli/pkg/domain`):
```go
type Task struct {
    Name     string          // filename without .md (set from file, not YAML)
    Status   TaskStatus      // e.g. "in_progress"
    Assignee string
    Phase    *TaskPhase      // pointer, may be nil
}

type TaskID string   // filename without .md extension
type TaskStatus string
type TaskPhase string
```
</context>

<requirements>
1. Add `github.com/bborbe/vault-cli` to `go.mod` and `vendor/`:
   ```bash
   go get github.com/bborbe/vault-cli@v0.48.6
   go mod tidy
   go mod vendor
   ```

2. Create `pkg/watcher/watcher.go` with:
   - A `Watcher` interface:
     ```go
     type Watcher interface {
         Watch(ctx context.Context) error
     }
     ```
   - A `//go:generate counterfeiter -o mocks/watcher.go --fake-name FakeWatcher . Watcher` annotation above the interface
   - A private `watcher` struct with fields:
     - `config config.Config`
     - `notifier notify.Notifier`
     - `watchOp ops.WatchOperation`
     - `taskStorage storage.TaskStorage`
   - A constructor:
     ```go
     func NewWatcher(cfg config.Config, notifier notify.Notifier) Watcher {
         storageConfig := storage.DefaultConfig()
         return &watcher{
             config:       cfg,
             notifier:     notifier,
             watchOp:      ops.NewWatchOperation(),
             taskStorage:  storage.NewTaskStorage(storageConfig),
         }
     }
     ```
   - The `Watch` implementation:
     ```go
     func (w *watcher) Watch(ctx context.Context) error {
         targets := []ops.WatchTarget{{
             VaultPath: w.config.VaultPath,
             VaultName: "vault",
             WatchDirs: []string{"24 Tasks"},
         }}
         return w.watchOp.Execute(ctx, targets, func(event ops.WatchEvent) error {
             if event.Event != "created" && event.Event != "modified" {
                 return nil
             }
             return w.handleEvent(ctx, event)
         })
     }
     ```
   - A private `handleEvent` method:
     ```go
     func (w *watcher) handleEvent(ctx context.Context, event ops.WatchEvent) error {
         task, err := w.taskStorage.ReadTask(ctx, w.config.VaultPath, domain.TaskID(event.Name))
         if err != nil {
             slog.Debug("skip unreadable task", "name", event.Name, "error", err)
             return nil
         }
         if task.Assignee == "" || task.Status == "" || task.Phase == nil {
             return nil
         }
         if string(task.Assignee) != w.config.Assignee {
             return nil
         }
         if !containsString(w.config.Statuses, string(task.Status)) {
             return nil
         }
         if !containsString(w.config.Phases, string(*task.Phase)) {
             return nil
         }
         if err := w.notifier.Notify(ctx, notify.Notification{
             TaskName: task.Name,
             Phase:    string(*task.Phase),
             Assignee: task.Assignee,
         }); err != nil {
             return errors.Wrapf(ctx, err, "notify task %s phase %s", task.Name, string(*task.Phase))
         }
         return nil
     }
     ```
   - A private `containsString(slice []string, s string) bool` helper

3. Run `go generate ./pkg/watcher/...` to produce `pkg/watcher/mocks/watcher.go`.

4. Create `pkg/watcher/suite_test.go` (package `watcher_test`):
   ```go
   package watcher_test

   import (
       "testing"
       . "github.com/onsi/ginkgo/v2"
       . "github.com/onsi/gomega"
   )

   func TestWatcher(t *testing.T) {
       RegisterFailHandler(Fail)
       RunSpecs(t, "Watcher Suite")
   }
   ```

5. Create `pkg/watcher/watcher_test.go` (package `watcher_test`) with integration-style Ginkgo tests.

   Since `WatchOperation.Execute` is callback-based, tests create real temp files and verify the callback triggers notification. Use `os.MkdirTemp` for a temp vault, write `.md` files with YAML frontmatter, and assert on `FakeNotifier`.

   Test setup:
   ```go
   var (
       ctx        context.Context
       cancel     context.CancelFunc
       vaultDir   string
       tasksDir   string
       fakeNotifier *mocknotify.FakeNotifier
       w          watcher.Watcher
       cfg        config.Config
       watchDone  chan error
   )

   BeforeEach(func() {
       ctx, cancel = context.WithCancel(context.Background())
       var err error
       vaultDir, err = os.MkdirTemp("", "vault-*")
       Expect(err).NotTo(HaveOccurred())
       tasksDir = filepath.Join(vaultDir, "24 Tasks")
       Expect(os.MkdirAll(tasksDir, 0750)).To(Succeed())

       cfg = config.Config{
           VaultPath: vaultDir,
           Assignee:  "Alice",
           Statuses:  []string{"in_progress"},
           Phases:    []string{"planning"},
       }
       fakeNotifier = &mocknotify.FakeNotifier{}
       w = watcher.NewWatcher(cfg, fakeNotifier)

       watchDone = make(chan error, 1)
       go func() { watchDone <- w.Watch(ctx) }()
       time.Sleep(100 * time.Millisecond) // let watcher start
   })

   AfterEach(func() {
       cancel()
       <-watchDone
       Expect(os.RemoveAll(vaultDir)).To(Succeed())
   })
   ```

   Required test cases:
   - Matching task (correct assignee, status, phase) → `FakeNotifier.NotifyCallCount()` equals 1, args contain correct TaskName/Phase/Assignee (wait up to 500ms for the call)
   - Wrong assignee → `FakeNotifier` not called after 300ms
   - Wrong status → `FakeNotifier` not called
   - Wrong phase → `FakeNotifier` not called
   - Task with missing frontmatter (empty file) → `FakeNotifier` not called, no error
   - Context cancellation → `Watch` returns nil (or `context.Canceled`)

   Helper to write a task file:
   ```go
   func writeTask(dir, name, assignee, status, phase string) {
       content := fmt.Sprintf("---\nassignee: %s\nstatus: %s\nphase: %s\n---\n", assignee, status, phase)
       Expect(os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0600)).To(Succeed())
   }
   ```

   Use `Eventually(func() int { return fakeNotifier.NotifyCallCount() }, "500ms", "20ms").Should(Equal(1))` for async assertions.

6. Run `make test` and verify all tests pass.

7. Run `make precommit` — must pass.
</requirements>

<constraints>
- vault-cli is imported as a Go library (`github.com/bborbe/vault-cli/pkg/ops`, `pkg/storage`, `pkg/domain`), NEVER executed via shell
- Factory functions use pure composition: no I/O, no `context.Background()` inside constructors
- Tests use Ginkgo/Gomega (package `watcher_test`) with Counterfeiter-generated mocks (`FakeNotifier`)
- Error handling follows `github.com/bborbe/errors` patterns — `errors.Wrapf(ctx, err, "message")`, never `fmt.Errorf`
- `make precommit` must pass (linting, tests, formatting)
- `pkg/pkg_suite_test.go` must continue to work — do NOT modify it
- No `//nolint` directives without explanation
- Do NOT commit — dark-factory handles git
- Existing tests (pkg/config, pkg/notify) must still pass
</constraints>

<verification>
```bash
make test
make precommit
```
Both must exit with code 0.

Check coverage for `pkg/watcher`:
```bash
go test -coverprofile=/tmp/cover.out -mod=vendor ./pkg/watcher/... && go tool cover -func=/tmp/cover.out
```
Statement coverage for `pkg/watcher` must be ≥80%.
</verification>
