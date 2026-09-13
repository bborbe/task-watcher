---
status: completed
spec: [004-retire-bespoke-notifier]
execution_id: task-watcher-exec-027-1b-spec-004-publish-path
dark-factory-version: dev
created: "2026-09-13T20:54:32Z"
queued: "2026-09-13T21:42:20Z"
started: "2026-09-13T21:47:08Z"
completed: "2026-09-13T21:52:29Z"
---

# Retire the bespoke notifier 1b: publish matched task events into the shared notification core

<summary>
- A task event that matches a watcher entry now leaves this process as a single notification published into the shared delivery core that already serves trading and agent escalations
- The notification is typed `agent-escalation` and carries no target, so the deployed routing table keeps owning the channel decision (Discord plus Telegram)
- The message names the task, its status, its phase and its assignee, and links straight back to the note inside the vault
- Source fields ride along as opaque metadata, so the delivery core and its logs can attribute the notification without parsing prose
- Duplicate suppression collapses from three copies into one gate: task plus phase, per watcher entry, that entry's own window, 5 minutes by default
- The window is consumed by the attempt itself, so a failed publish does not cause a retry storm inside the same window
- The window is measured on an injectable clock, so expiry is provable in a test without sleeping
- A failed publish is logged with the watcher, task, phase and error and the watch loop keeps serving later events — a broken broker cannot take the human-review channel down
- Silence stays silent: a non-matching assignee, status or phase, or a task with an empty assignee, empty status or no phase, publishes nothing
- The bespoke notifier is still in the tree but no longer wired into delivery; it is deleted only by the gated prompt, after the operator's live proof is committed
</summary>

<objective>
Turn a matching task event into a publish into the shared notification core through the core's command sender, with producer-side dedup owned by one component and a message the operator can act on. This is what lets this repo stop being a channel implementation: the delivery core already resolves `agent-escalation` to the deployed channels. The bespoke path stays in the tree, unwired, until the operator's live proof is committed — the deletion is a separate gated prompt.
</objective>

<context>
Read `CLAUDE.md` for project conventions; this repo is edited **only** through the dark-factory pipeline.
Read `docs/dod.md` — the repo's own Definition of Done, used as the post-implementation self-review checklist.

Coding plugin docs (in-container paths):
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-cqrs.md` — command sending, topics derived from the SchemaID
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md` — Ginkgo v2 / Gomega conventions
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-mocking-guide.md` — counterfeiter fakes, including fakes for a library interface
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-time-injection.md` — inject `libtime.CurrentDateTimeGetter`, drive tests with `SetNow`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-error-wrapping-guide.md` — `github.com/bborbe/errors`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-factory-pattern.md` — factories stay pure composition

Read before changing anything:
- `pkg/config/config.go` and `pkg/config/destination.go` — the filter-only entry and `config.Destination` from the previous prompt
- `pkg/watcher/watcher.go` and `pkg/watcher/watcher_test.go` — `NewWatcher`, `watcherEntry`, `handleEvent`, the guard and the filters
- `pkg/factory/factory.go` and `pkg/factory/factory_test.go` — `CreateWatcher`, `CreateNotifiers`
- `pkg/cli/cli.go` — `Run`: config load, the INFO destination line, the watcher start and the shutdown path
- `pkg/notify/notify.go`, `pkg/notify/telegram.go`, `pkg/notify/openclaw.go`, `pkg/notify/log.go` — read the dedup block for its **semantics** (three copies collapsing into one); do not reuse the code, and do not import this package from the new path
- `mocks/mocks.go` — `make generate` wipes and regenerates `mocks/` from the `//counterfeiter:generate` directives

Library facts, verified against the release this repo will pin (import paths are exact — the two notification packages are different packages):

| Fact | Value |
|---|---|
| library root package (import path `github.com/bborbe/notification`, package name `core`) | `core.AgentEscalationNotificationType` = `"agent-escalation"` |
| command package (import path `github.com/bborbe/notification/command/notification`) | `notification.NotificationPublishCommand` |
| command fields | `Type core.NotificationType`, `Target *core.NotificationTarget`, `Message core.NotificationMessage`, `Metadata map[string]string` |
| message constructor | `core.NotificationMessagef(format string, a ...any) core.NotificationMessage` |
| sender interface | `notification.NotificationPublishCommandSender` with `SendPublishNotificationCommand(ctx context.Context, command notification.NotificationPublishCommand) error` |
| sender constructor | `notification.NewNotificationPublishCommandSender(commandCreator base.CommandCreator, commandObjectSender cdb.CommandObjectSender, initiator cqrsiam.Initiator) notification.NotificationPublishCommandSender` |
| command-object sender | `cdb.NewCommandObjectSender(syncProducer kafka.SyncProducer, prefix base.TopicPrefix, logSamplerFactory log.SamplerFactory) cdb.CommandObjectSender` |
| command creator | `base.NewCommandCreator(requestIDChan <-chan base.RequestID) base.CommandCreator`, channel from `base.RequestIDChannel(ctx)` |
| producer | `kafka.NewSyncProducerWithName(ctx context.Context, brokers kafka.Brokers, name string, opts ...kafka.SaramaConfigOptions) (kafka.SyncProducer, error)`; `kafka.SyncProducer` has `Close() error` |
| sampler factory | `log.DefaultSamplerFactory` from `github.com/bborbe/log` |
| validation | `command.Validate(ctx)` checks Type (allowlist), Target (nil or valid), Message (non-empty) |
| serialisation boundary | the sender passes the command through `base.ParseEvent(ctx, command)`, i.e. a JSON round-trip — that is the shape the wire sees |

Imports to use (aliases matter): `core "github.com/bborbe/notification"`, `"github.com/bborbe/notification/command/notification"`, `"github.com/bborbe/cqrs/base"`, `"github.com/bborbe/cqrs/cdb"`, `cqrsiam "github.com/bborbe/cqrs/iam"`, `"github.com/bborbe/kafka"` (unaliased — this repo has no `libkafka` alias), `"github.com/bborbe/log"`, `libtime "github.com/bborbe/time"`.

Optional: if the module cache is mounted, the pinned library sources are readable under `/home/node/go/pkg/mod/github.com/bborbe/`.
</context>

<requirements>
1. **Add the dependency in the right order.** Write the importing code (requirements 2–4) **first**; only then run:
   ```bash
   go get github.com/bborbe/notification@latest
   go mod tidy
   ```
   Running `go get`/`tidy` before something imports the module demotes or drops the require — the order above is not optional. Then confirm the resolved release registers the type:
   ```bash
   grep -rn "agent-escalation" "$(go list -m -f '{{.Dir}}' github.com/bborbe/notification)/core_notification-type.go"
   ```
   That grep must return a line. If it returns nothing, pin `go get github.com/bborbe/notification@v0.6.1` (verified to carry `AgentEscalationNotificationType`) and re-run. The library's `go.mod` declares Go 1.27; if `go mod tidy` raises this repo's `go` directive, accept the bump — do not fight it, and report it in your summary. If the toolchain in this container is older than the required version and the build cannot resolve it, stop and report the blocker instead of pinning an older library release.

2. **Create `pkg/publish`, the producer-side seam.** `pkg/publish/event.go` holds the value that travels from the watcher to the publisher:
   ```go
   // TaskEvent is one matched task change, carrying everything the notification
   // message and its metadata need.
   type TaskEvent struct {
   	Vault    string
   	TasksDir string
   	TaskName string
   	Status   string
   	Phase    string
   	Assignee string
   }
   ```
   `TasksDir` is the vault's `tasks_dir` from the config (e.g. `24 Tasks`), not an absolute path.

3. **`pkg/publish/command.go` — build the command.** Contract:
   ```go
   // BuildCommand builds the notification-publish command for one matched task event.
   func BuildCommand(ctx context.Context, event TaskEvent) (notification.NotificationPublishCommand, error)
   ```
   - `Type: core.AgentEscalationNotificationType` — use the library constant, never a hand-written string literal. The type must stay the one the deployed routing tables already resolve.
   - `Target` is left at its zero value (`nil`) on every publish. Never set it: a non-nil target would override the deployed routing table, and the routing table owns the channel decision.
   - `Message` is built with `core.NotificationMessagef(...)`. Wording is yours; the content is not — the message must contain the task name, the status, the phase, the assignee and the Obsidian deeplink (below), and it must stay actionable for a human reading it in a chat.
   - `Metadata` is exactly `map[string]string{"taskName": ..., "phase": ..., "assignee": ..., "vault": ...}` — those four keys, spelled that way, values from the event. Metadata carries no credentials, no host paths and no task body text.
   - Deeplink format: `obsidian://open?vault=<vault>&file=<tasks_dir>/<taskname>.md`, with the vault and the file value percent-encoded so that a space becomes `%20` and never `+` (`24 Tasks` → `24%20Tasks`; a task named `My Task` → `My%20Task.md`). Do not encode the `/` between the tasks dir and the file name. Build it in a small unexported helper and unit-test the encoding — the real vault's tasks dir contains a space, so an unencoded link is broken in practice.
   - `BuildCommand` does not send anything and does not validate; it returns a wrapped error only if it cannot build (e.g. bad input).

4. **`pkg/publish/publisher.go` — the single dedup gate plus the send.** Contract:
   ```go
   //counterfeiter:generate -o ../../mocks/notification_publish_command_sender.go --fake-name NotificationPublishCommandSender github.com/bborbe/notification/command/notification.NotificationPublishCommandSender

   // Publisher publishes one notification per matched task event, suppressing
   // repeats of the same task and phase inside its window.
   type Publisher interface {
   	Publish(ctx context.Context, event TaskEvent) error
   }

   // NewPublisher returns a Publisher that publishes through sender and keeps its
   // own dedup window (one instance per watcher entry, using that entry's TTL).
   func NewPublisher(
   	sender notification.NotificationPublishCommandSender,
   	topicPrefix base.TopicPrefix,
   	dedupTTL time.Duration,
   	currentDateTimeGetter libtime.CurrentDateTimeGetter,
   ) Publisher
   ```
   The counterfeiter directive above generates the fake for the **library** interface (a library interface has no fake of its own in this release) — the specs assert on `SendPublishNotificationCommand`, so the fake has to sit at that seam. Keep the directive attached to a declaration in this file; `make generate` regenerates `mocks/` from it.

   `Publish` behaviour, in this order:
   a. Read the clock **once** per call from `currentDateTimeGetter.Now()`.
   b. Dedup gate, keyed on `event.TaskName + ":" + event.Phase` (the key the three bespoke implementations used), per publisher instance:
      - a stamped key whose age is **less than** `dedupTTL` → log at Debug (`"skipped (duplicate within TTL)"` with task, phase, ttl) and return `nil`;
      - otherwise stamp the key with the value read in (a) **before** attempting the send, and prune entries whose age has reached `dedupTTL` in the same critical section so the map stays proportional to the distinct task+phase pairs seen inside one window, not to lifetime traffic. Guard the map with a mutex, as the deleted implementations did.
      - Both the stamp and the expiry read use the value from (a) — never `time.Since`, never a second clock read. The deleted notifier had exactly that bug (write on the injectable clock, read on the wall clock) and it is fixed there for a reason; do not reintroduce it.
   c. Build the command via `BuildCommand`, then validate it: `if err := cmd.Validate(ctx); err != nil { return errors.Wrapf(ctx, err, "validate publish notification command") }`. Validation happens **before** the send so an unroutable type or an empty message fails locally and visibly instead of being published.
   d. Emit one log line per publish attempt, **before** the send, carrying the effective topic prefix (`"topic_prefix", topicPrefix.String()`) plus the task, the phase and the notification type — this is what makes a wrong-prefix deployment visible while the attempt is still in flight. The level is your choice (INFO or DEBUG); the prefix field must be present.
   e. `if err := sender.SendPublishNotificationCommand(ctx, cmd); err != nil { return errors.Wrapf(ctx, err, "send publish notification command") }`.
   f. Never retry inside the publisher, never call `os.Exit`, never panic, never consume a Kafka topic. The window stays consumed when the attempt fails — that is the contract, not an oversight.
   Do everything synchronously inside `Publish` — spawn no goroutines. The event path is **not** single-goroutine: `ops.WatchOperation.Execute` in vault-cli fires each debounced event from its own `time.AfterFunc` goroutine, so `Publish` must be safe for concurrent calls — the mutex above is load-bearing, not decorative.

5. **`pkg/watcher/watcher.go` — wire the publish call into the matched-event path.**
   - `func NewWatcher(cfg config.Config, publishers []publish.Publisher) Watcher` — one publisher per `cfg.Watchers` entry, indexed exactly like the notifier slice was.
   - `watcherEntry.notifier notify.Notifier` becomes `watcherEntry.publisher publish.Publisher`; drop the `pkg/notify` import from this file. The `Watcher` interface (`Watch(ctx) error`), the `ops.NewWatchOperation()` trigger, the `created`/`modified` filter, the storage read and the empty-assignee / empty-status / nil-phase guard all stay exactly as they are.
   - Resolve each event's `tasks_dir` alongside the existing per-vault maps (`vaultPaths`, `taskStorages`) so the event carries `TasksDir` for the deeplink.
   - In `handleEvent`, after the guard and after **all** entry filters (assignee, statuses, phases) pass, build the `publish.TaskEvent` (`Vault: event.Vault`, `TasksDir`, `TaskName: task.Name`, `Status: task.Status().String()`, `Phase: task.Phase().String()`, `Assignee: task.Assignee()`) and call `entry.publisher.Publish(ctx, event)`.
   - On error, log at ERROR with `watcher`, `task`, `phase`, `error` (keep the existing message shape) and **continue** to the next entry. Never return the error from `handleEvent`: a failed publish must not stop other entries and must not stop the watch loop — the process must not exit when a broker is down.

6. **`pkg/factory/factory.go` — construct the producer side, pure composition.**
   - ```go
     // notificationInitiator names this producer in the command's traceability
     // field. It is not an authorization: publishing is not IAM-gated.
     const notificationInitiator cqrsiam.Initiator = "task-watcher"
     ```
   - ```go
     // CreateNotificationSender constructs the shared-core notification command sender
     // backed by the given Kafka sync producer; topicPrefix selects the command topic.
     // ctx feeds the request-ID channel so downstream cancellation propagates.
     func CreateNotificationSender(
     	ctx context.Context,
     	syncProducer kafka.SyncProducer,
     	topicPrefix base.TopicPrefix,
     ) notification.NotificationPublishCommandSender {
     	return notification.NewNotificationPublishCommandSender(
     		base.NewCommandCreator(base.RequestIDChannel(ctx)),
     		cdb.NewCommandObjectSender(syncProducer, topicPrefix, log.DefaultSamplerFactory),
     		notificationInitiator,
     	)
     }
     ```
   - ```go
     // CreatePublishers builds one Publisher per watcher entry, in config order,
     // each with that entry's dedup window.
     func CreatePublishers(
     	cfg config.Config,
     	sender notification.NotificationPublishCommandSender,
     	topicPrefix base.TopicPrefix,
     	currentDateTimeGetter libtime.CurrentDateTimeGetter,
     ) []publish.Publisher
     ```
   - `CreateWatcher(cfg config.Config, publishers []publish.Publisher) watcher.Watcher`.
   - Keep the interim `CreateNotifiers` from the previous prompt in place, untouched: nothing calls it any more, and it is deleted by the gated prompt. Do **not** wire it: publishing through the shared core and sending through the bespoke notifier at the same time would deliver every push twice.
   - No I/O, no conditionals, no `context.Background()` inside these functions.

7. **`pkg/cli/cli.go` — build the producer once and close it.**
   - Keep the config load and the INFO `publish destination` line from the previous prompt exactly as they are (same level, same `topic_prefix` field, still logged before anything that can block).
   - Then:
     ```go
     syncProducer, err := kafka.NewSyncProducerWithName(ctx, destination.KafkaBrokers, "task-watcher")
     if err != nil {
     	return errors.Wrapf(ctx, err, "create sync producer")
     }
     defer func() {
     	if closeErr := syncProducer.Close(); closeErr != nil {
     		slog.Warn("close sync producer", "error", closeErr)
     	}
     }()

     currentDateTime := libtime.NewCurrentDateTime()
     sender := factory.CreateNotificationSender(ctx, syncProducer, destination.TopicPrefix)
     publishers := factory.CreatePublishers(cfg, sender, destination.TopicPrefix, currentDateTime)
     w := factory.CreateWatcher(cfg, publishers)
     ```
     The clock is created **once** in `Run` and passed down — never inside a factory (see the time-injection guide). Stop calling `factory.CreateNotifiers` here.
     This line needs the `github.com/bborbe/errors` import the previous prompt added to this file — if it is not there, add it. Do not use `fmt.Errorf`; the three pre-existing calls in this file are out of scope.
     `Run` sits at roughly 71 of the 80 lines `funlen` allows in `.golangci.yml`, and the previous prompt already added to it, so these additions will push it over and `make precommit` will fail on the linter rather than on a test. If that happens, extract the producer construction into an unexported helper returning `(kafka.SyncProducer, error)` (e.g. `newSyncProducer(ctx, destination)`) and keep the deferred `Close` in `Run`; if that alone is not enough, move the sender and publisher construction into a helper too — do **not** raise the `funlen` limit, and do not delete or merge existing statements to make room.
   - The watcher start, the `errCh` select and the 5-second shutdown timeout stay as they are.

8. **Specs: `pkg/publish` (new package, `publish_test`, plus a `suite_test.go` with the package's `//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.12.2 -generate` line, matching the other suites).** Use `mocks.NotificationPublishCommandSender` and a real `publish.NewPublisher`; drive time with `libtime.NewCurrentDateTime()` + `SetNow` — never `time.Sleep`, never a fake clock that is a different clock from the one the publisher reads. Cover:
   - repeated identical task+phase inside the window → exactly one `SendPublishNotificationCommand` call;
   - after advancing the injected clock past the window → a second call (this is the expiry proof, without sleeping);
   - the sender returns an error → `Publish` returns that error wrapped, and a second identical event inside the same window does not attempt again (call count stays 1);
   - a different phase for the same task → two calls; the same phase for a different task → two calls;
   - two publishers with different windows on the same task+phase → one call each (the window is per entry, not global);
   - the built command: `string(cmd.Type)` equals the literal `"agent-escalation"`, `cmd.Target` is nil, the message is non-empty and contains the task name, the status, the phase, the assignee and the deeplink, and `Metadata` carries the four keys with the event's values;
   - **boundary assertions on the value that actually leaves:** `cmd.Type.Validate(ctx)` and `cmd.Validate(ctx)` both succeed (the library's allowlist and message rules are only enforced at runtime), and the value survives the serialisation the library performs — `event, err := base.ParseEvent(ctx, cmd)` succeeds, `event["type"]` is `"agent-escalation"`, and `event["target"]` is **absent** (which is how a nil target is proven on the wire);
   - the deeplink helper: a tasks dir containing a space and a task name containing a space both encode as `%20`, never `+`, with the `/` between them preserved.
   Name the new specs so the word "publish" (or "publishes") appears in their descriptions — the verification below runs them with `-ginkgo.focus`.

9. **Specs: `pkg/watcher/watcher_test.go`** — rewrite the suite onto the publisher wiring (real `NewWatcher`, real `publish.NewPublisher`, fake sender, real vault files and the real fsnotify watch, exactly like the current specs). Keep the existing multi-vault and fan-out coverage. Required new coverage, with the word "publish" in each description:
   - a matching task event publishes exactly one command, and that command has `Type == agent-escalation`, a nil `Target`, a non-empty message and the four metadata fields (assert on `ArgsForCall(0)`);
   - silence: a non-matching assignee, a non-matching status and a non-matching phase each publish nothing (`Consistently`, count 0);
   - the guard's presence is asserted **behaviourally**: a task with an empty assignee, a task with an empty status and a task with no phase each publish nothing — do not assert on source text or line numbers;
   - two identical events inside the window produce exactly one publish (the gate is wired into the event path, not merely unit-tested);
   - a failing sender is logged and does not stop later distinct events from publishing, and `Watch` keeps running.
   Delete the notifier-based assertions (`mocknotify.FakeNotifier`, `notifications` from `pkg/notify`); the notifier mock itself stays in the tree until the gated prompt.

10. **Specs: `pkg/factory/factory_test.go`** — update to the new signatures: `CreatePublishers` returns one non-nil publisher per watcher entry (including the empty case) and `CreateWatcher` returns non-nil for a config with publishers. Do not fabricate a Kafka producer fake to "test" `CreateNotificationSender`: its composition is exercised by the CLI path and proven end-to-end by the operator's live proof, and a fake producer would only assert the arguments back. Drop the specs for `CreateNotifiers` if you cannot keep them honest, but do not delete the constructor itself.

11. **`CHANGELOG.md`**: append a `- feat:` bullet for this prompt's work under the existing `## Unreleased` section. If a release has renamed that section in the meantime, create a fresh `## Unreleased` directly beneath the preamble and above the newest released section. Never edit a released section or the preamble.

12. **Do NOT delete anything in this prompt.** `pkg/notify/`, `mocks/notifier.go`, `factory.CreateNotifiers` and the notifier specs all stay in the tree. The deletion is a separate prompt that runs only after the operator has committed the live proof (`docs/live-proof-human-review-push.md`) — and its commit must be strictly later than that proof commit. Deleting before the proof exists is a failure of the spec even if everything else is green.

13. Self-review against `docs/dod.md` before finishing: exported names carry doc comments, errors are wrapped with `github.com/bborbe/errors`, structured logging only (no `fmt.Printf`), no `//nolint` without a reason.
</requirements>

<constraints>
- **Frozen producer contract:** publish a `NotificationPublishCommand` onto the CQRS Kafka command topic through the shared core's command sender — **never** through the notification-controller's `/send` HTTP bridge, which sits behind Google SSO and is human-browser-only.
- **Type and routing:** `agent-escalation`; `Target` is `nil` on every publish — the deployed routing table owns the channel decision, and a non-nil target would override it.
- **No new notification type, and no change to the core:** the library, the controller, the Discord and Telegram services, and the routing tables are not touched.
- **No secret, no role:** no TeamVault key, no token, no chat id; no IAM permission, role, or binding is added, because publishing is not IAM-gated. The initiator name is traceability only.
- **No change to the trigger, the filters, or the empty-assignee / empty-status / nil-phase guard** — this spec changes what happens after a match, not what counts as a match. No Kafka consumer is added anywhere: this process stays a watcher that happens to publish.
- **No dedup inside the core** — duplicate suppression stays producer-side, in the one component required above.
- **No new channels** (no Matrix, mail, Slack, Google Chat), no retry queue, no publish buffer, no metrics, no health endpoint.
- **Do NOT add a per-watcher dry-run, opt-out, or "keep the old channel" flag** — the topic prefix is the test path (dev routing reaches the test channel, prod reaches the real one); a flag that disables the shared path re-creates the bypass this spec removes. Invariant; a future consumer demanding variation needs a separate spec.
- **Ordering:** the live channel is never deleted before the shared path is proven. The proof is committed by the operator, not by this prompt.
- **Do NOT touch `README.md` or `CLAUDE.md` in this prompt**, even though `docs/dod.md` asks for README updates on a change affecting usage. The user-facing docs land with the gated deletion prompt, where the whole story is known — editing them here collides with that prompt's evidence, which requires the deletion commit's diff to carry them. Record the change in the code comments and the `CHANGELOG.md` entry instead.
- **Assumptions:** the shared core is deployed and consuming on dev and prod, `agent-escalation` is registered and resolved by the deployed routing tables, and the host running this watcher can reach the same brokers the controller consumes from. None of these are verifiable from inside this repo — do not add code that tries to.
- **Repo conventions:** Ginkgo v2 / Gomega specs, counterfeiter fakes for local interfaces (and for the library sender interface, per the directive above), factory functions stay pure composition, `github.com/bborbe/errors` for wrapping, structured logging with `log/slog`.
- **Definition of Done:** `docs/dod.md` applies; `CHANGELOG.md` carries an entry under `## Unreleased`.
- Do NOT commit — dark-factory handles git. Do NOT run `go mod vendor` and never use `-mod=vendor`. Existing tests must still pass.
</constraints>

<verification>
```bash
make generate   # run FIRST: the new specs compile against mocks.NotificationPublishCommandSender, which only this step's counterfeiter directive creates
go test -mod=mod ./pkg/publish/...
go test -mod=mod ./pkg/watcher/... -ginkgo.focus="publish"
go test -mod=mod ./...
make precommit
```

All must exit 0. `make precommit` also regenerates `mocks/` — confirm the new fake exists and the old one still does:

```bash
ls mocks/notification_publish_command_sender.go mocks/notifier.go
```

Confirm the dependency landed and the bespoke surface is still present (deletion is the gated prompt's job):

```bash
grep -rn "agent-escalation" --include='*.go' . | wc -l        # >= 1 (the boundary spec pins the literal)
grep -rn "bborbe/notification" --include='*.go' . | wc -l     # >= 1
grep -rn "KAFKA_BROKERS" --include='*.go' . | wc -l           # >= 1
grep -rn "libkafka\|sarama\|Consumer" --include='*.go' . | wc -l   # 0 — no consumer was added
test -d pkg/notify && echo "bespoke package still present"
test -f mocks/notifier.go && echo "notifier mock still present"
```

Prove at runtime that the STARTUP prefix line survived the producer wiring (the process may exit right after, because no broker is reachable — the INFO line is logged before the producer is created, which is why it is still captured):

```bash
go build -mod=mod -o /tmp/task-watcher .
mkdir -p /tmp/tw-vault/tasks
cat > /tmp/tw-minimal.yaml <<'YAML'
vaults:
  test:
    path: /tmp/tw-vault
    tasks_dir: tasks
watchers:
  - name: review
    assignee: TestUser
    phases: [human_review]
YAML

KAFKA_BROKERS=127.0.0.1:9092 TOPIC_PREFIX=develop /tmp/task-watcher --config /tmp/tw-minimal.yaml > /tmp/tw-startup.log 2>&1 &
TW_PID=$!
sleep 3
kill "$TW_PID" 2>/dev/null || true
grep -iE 'level=INFO.*topic_prefix' /tmp/tw-startup.log   # expect >= 1 line with prefix=develop
```

Report in your summary: the pinned `github.com/bborbe/notification` version, whether `go mod tidy` raised the `go` directive, **the full dependency diff the tidy produced** — every direct and indirect module whose version moved, not just the new requires (the fleet records dependency bumps as their own `chore:` commit, so the reviewer needs the list to judge whether this one needs splitting), the exact spec descriptions of the publish and dedup specs, and the file:line of the assertions on `Type`, `Target` and the message.
</verification>

<!--
REVIEWER NOTES — surfaced deliberately, resolve at audit time.

1. AC9's evidence greps for `libkafka|sarama|Consumer` (`--include='*.go'`, must be 0 lines) as proof that no
   Kafka consumer was added. The frozen producer contract needs the kafka library (sync producer, brokers type),
   so requirement 6 writes the import unaliased (`github.com/bborbe/kafka` -> package `kafka`) and never names
   `sarama` or a consumer. The spec's intent holds; the token-level grep passes. Aliasing the import as `libkafka`
   would make that grep non-zero and falsify the criterion without changing any behaviour, so the import stays
   unaliased — the constraint is on the alias, not on the contract.

2. Naming is not fixed by the spec (it only says `pkg/notify/` dies). This prompt names the new package
   `pkg/publish`, the transport value `publish.TaskEvent`, its interface `publish.Publisher`, and the
   message/metadata builder `publish.BuildCommand`. If the reviewer prefers other names, they must be changed in prompt 1b, prompt 2 and
   the specs before approval — nothing downstream depends on them.

3. Dedup remains keyed on task name + phase, and `WatcherConfig.DedupTTL` stays a stdlib `time.Duration`
   (`libtime.Duration` would require reworking the existing config parsing — out of scope here). The new
   component reads its clock from an injected `libtime.CurrentDateTimeGetter` per the time-injection guide, so
   `SetNow` drives expiry and no test sleeps. Note that the three implementations being deleted read a package
   variable clock instead; do not copy that.

4. Spec-side analysis (two of the spec's AC evidence wordings, and the factory-switch reassignment
   between prompts 1a and 2) is recorded in the vault task's Progress entry, not here — this block is read by
   the executing agent, and an agent must not read a reviewer's analysis of the acceptance criteria as licence
   to deviate from them.
-->
