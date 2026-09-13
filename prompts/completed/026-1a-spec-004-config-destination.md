---
status: completed
spec: [004-retire-bespoke-notifier]
summary: Made watcher entries filter-only with a loud migration error for the four removed channel fields, moved the publish destination to KAFKA_BROKERS/TOPIC_PREFIX with fail-fast startup and an INFO topic-prefix log line (kafka v1.26.0, cqrs v0.6.11), and kept the tree compiling with an interim log-only notifier; make precommit exits 0.
execution_id: task-watcher-exec-026-1a-spec-004-config-destination
dark-factory-version: dev
created: "2026-09-13T20:54:32Z"
queued: "2026-09-13T21:42:20Z"
started: "2026-09-13T21:43:03Z"
completed: "2026-09-13T21:47:04Z"
---

# Retire the bespoke notifier 1a: filter-only watcher config and an env-owned publish destination

<summary>
- A watcher entry becomes a pure filter — name, assignee, statuses, phases and the dedup window — and no longer selects a delivery channel
- A config that still carries a channel field refuses to load, and the error names the field that has to go, so a stale config can never start and silently deliver nothing
- The publish destination is read from the environment under the names every other producer in the fleet already uses
- Startup fails fast and names the missing variable when either destination variable is unset or empty
- The effective topic prefix is printed at startup at INFO, so a deployment pointing at the wrong prefix is visible in the log instead of silent
- The per-entry dedup window keeps its 5-minute default and its per-entry independence
- The vault trigger and every filter are untouched — only what a config entry is allowed to contain changes
- The bespoke notifier is still in the tree and still compiles; nothing is deleted in this step
- No delivery path is wired in this step: every entry is built with the log-only sender, so the tree has working Telegram/OpenClaw delivery again only once the shared publish path lands in the next prompt — that is why 1a and 1b are approved and merged together, with the deletion waiting for the operator's live proof after both
</summary>

<objective>
Make a watcher entry a pure filter and move the publish destination into the environment with fail-fast startup, so a stale channel config cannot start silently and a wrong-prefix deployment is visible at INFO. The bespoke notifier package stays in the tree untouched — it is removed only by the gated deletion prompt, after the shared path has been proven live.
</objective>

<context>
Read `CLAUDE.md` for project conventions; note that this repo is edited **only** through the dark-factory pipeline.
Read `docs/dod.md` — the repo's own Definition of Done, used as the post-implementation self-review checklist.

Coding plugin docs (in-container paths — read the ones this change touches):
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md` — Ginkgo v2 / Gomega conventions
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-error-wrapping-guide.md` — `github.com/bborbe/errors` wrapping
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-factory-pattern.md` — factory functions stay pure composition
- `/home/node/.claude/plugins/marketplaces/coding/docs/changelog-guide.md` — `## Unreleased` entry format and prefixes

Read these files before changing anything:
- `pkg/config/config.go` — `WatcherConfig`, `rawWatcherEntry`, `rawConfig`, `validateWatcherType`, `parseDedupTTL`, `parseWatchers`, `loader.Load`
- `pkg/config/config_test.go` — every fixture that carries a `type:` field has to change
- `pkg/factory/factory.go` — `CreateNotifiers` / `createNotifierForWatcher` dispatch on `WatcherConfig.Type`
- `pkg/cli/cli.go` — `Run` loads the config and logs each configured watcher's `type`
- `pkg/notify/notify.go` — the `Notifier` interface, needed by the interim factory construction

Optional: if the module cache is mounted (`/home/node/go/pkg/mod/github.com/bborbe/`), the pinned library sources are readable there, including `/home/node/go/pkg/mod/github.com/bborbe/kafka@*/kafka_brokers.go` and `/home/node/go/pkg/mod/github.com/bborbe/cqrs@*/base/base_topic-prefix.go`. Nothing in this prompt depends on reading them.
</context>

<requirements>
1. **Make `WatcherConfig` filter-only** in `pkg/config/config.go`:
   - Keep `Name`, `Assignee`, `Statuses`, `Phases`, `DedupTTL`.
   - Remove the fields `Type`, `URL`, `Token`, `ChatID` — including their doc comments about openclaw-wake and telegram.
   - Keep all four fields in the *private* `rawWatcherEntry` struct with their existing `yaml` tags (`type`, `url`, `token`, `chat_id`). They exist only so a stale config can be detected. Mirror the existing comment style already used by `rawConfig`'s old flat fields (`// Old flat fields — present only for migration error detection`), e.g. `// Removed channel fields — present only so a stale config fails loudly.`
   - Do not add `KnownFields(true)` to the YAML decoding, and do not switch to a generic `map`-based key scan.

2. **Delete the per-type validation** in `pkg/config/config.go`:
   - Remove `validateWatcherType` entirely (the per-type required-field checks and the `unknown type %q (must be openclaw-wake, telegram, or log)` error have no meaning once entries are filter-only).
   - In `parseWatchers`, remove the `missing required field: type` check and its call to `validateWatcherType`. Keep the existing `name` check and the `parseDedupTTL` call exactly as they are, and keep building `WatcherConfig` from the fields that remain.
   - Do not touch the top-level "old flat format" check in `loader.Load` (`assignee`/`webhook`/`format`/`webhook_token`) — that error and its message stay byte-identical.

3. **Add the migration check** in `pkg/config/config.go`, called from `parseWatchers` **after** the `name` check and **before** `parseDedupTTL`, so the reported error is deterministic:
   - Fixed check order: `type`, `url`, `token`, `chat_id`.
   - A non-empty value in any of the four is an error naming that field. An absent or explicitly empty field is not an error.
   - The error names both the watcher and the field, using the YAML spelling of the field (`chat_id`, not `ChatID`), e.g.:
     ```go
     func validateNoRemovedChannelFields(ctx context.Context, rw rawWatcherEntry) error {
     	removed := []struct {
     		field string
     		value string
     	}{
     		{"type", rw.Type},
     		{"url", rw.URL},
     		{"token", rw.Token},
     		{"chat_id", rw.ChatID},
     	}
     	for _, r := range removed {
     		if r.value != "" {
     			return errors.Errorf(
     				ctx,
     				"watcher %q: field %q was removed — the notification core resolves the channel from the notification type; remove it from the watcher entry",
     				rw.Name,
     				r.field,
     			)
     		}
     	}
     	return nil
     }
     ```
     Adapt the wording if you can make it more actionable, but keep the watcher name and the field name in the message — the config spec asserts `ContainSubstring` on each of the four field names.
   - Do **not** mention any channel implementation in this code (no `openclaw-wake`, no Telegram vocabulary). The check is generic over the four removed keys.

4. **Add the env-owned destination** in a new file `pkg/config/destination.go` (package `config`):
   - Exported constants for the two variable names: `EnvKafkaBrokers = "KAFKA_BROKERS"` and `EnvTopicPrefix = "TOPIC_PREFIX"`.
   - ```go
     // Destination holds the publish destination read from the environment.
     type Destination struct {
     	KafkaBrokers kafka.Brokers
     	TopicPrefix  base.TopicPrefix
     }
     ```
   - ```go
     // LoadDestinationFromEnv reads the publish destination from the environment.
     // Both KAFKA_BROKERS (comma-separated) and TOPIC_PREFIX are required; an
     // unset or empty variable is an error naming that variable.
     func LoadDestinationFromEnv(ctx context.Context) (Destination, error)
     ```
   - Import style, deliberate: `"github.com/bborbe/kafka"` and `"github.com/bborbe/cqrs/base"` **unaliased** (upstream package names are `kafka` and `base`). This repo has no `libkafka` alias and the spec's "no Kafka consumer was added" evidence greps for that token — see the reviewer note at the end of this file.
   - Parse with `kafka.ParseBrokersFromString(value)` and `base.TopicPrefix(value)`. Both variables are required: report `KAFKA_BROKERS` first, then `TOPIC_PREFIX`, and name the offending variable in the error (e.g. `errors.Errorf(ctx, "environment variable %s is not set", EnvKafkaBrokers)`). Use `github.com/bborbe/errors`, as the rest of `pkg/config` does.
   - **Add both module requirements explicitly, and in this order.** This repo's `go.mod` carries neither `github.com/bborbe/kafka` nor `github.com/bborbe/cqrs` today. Write `destination.go` first, then run `go get github.com/bborbe/kafka@latest github.com/bborbe/cqrs@latest && go mod tidy` — never `go get` before the importing file exists, because a tidy with no importer drops the requirement again. **`base.TopicPrefix` exists only from cqrs v0.6.0**, so the floor is load-bearing: if `@latest` resolves below it, pin a higher version explicitly rather than accepting the resolution. Report both resolved versions, and any direct or indirect dependencies the tidy bumped, in your summary.

5. **Wire the destination in `pkg/cli/cli.go`**, inside `RunE`, after the config load succeeds and before anything that can block:
   - ```go
     destination, err := config.LoadDestinationFromEnv(ctx)
     if err != nil {
     	return errors.Wrapf(ctx, err, "load destination")
     }
     ```
     Add `"github.com/bborbe/errors"` to `pkg/cli/cli.go`'s import block — this file is the repo's one exception to the rule below, and this new line must not extend it. The three pre-existing `fmt.Errorf` calls in the same file are out of scope: leave them, do not widen the diff by rewriting them.
     Do not add a new exit path and do not call `os.Exit` here — `cli.Execute` already prints `Error: <err>` to stderr and exits 1, which is what makes the missing variable visible in `systemctl`/`journalctl` output.
   - Log the destination at **INFO**, immediately after the existing `task-watcher starting` line and before the vault lines:
     ```go
     slog.Info(
     	"publish destination",
     	"kafka_brokers", destination.KafkaBrokers.String(),
     	"topic_prefix", destination.TopicPrefix.String(),
     )
     ```
     The level is pinned at INFO and the field name `topic_prefix` is frozen — it is what makes a wrong-prefix deployment detectable instead of silent. Do not gate this line on `--verbose` and do not move it below the producer construction in a later prompt.
   - Update the `configured watcher` log line: `w.Type` no longer exists. Log the fields an entry still has (`name`, `assignee`, `statuses`, `phases`, `dedupTTL`).
   - The destination is otherwise unused in this step; the log line is the only consumer until the publish path lands in the next prompt. Do not add a placeholder no-op, a build tag, or a commented-out publish call.
   - `Run` in `pkg/cli/cli.go` sits at roughly 71 of the 80 lines `funlen` allows in `.golangci.yml`, so these additions push it over and `make precommit` will fail on the linter rather than on a test. If that happens, extract the new construction (this step's destination load, or later the producer and its `Close`) into a small unexported helper — do **not** raise the `funlen` limit, and do not delete or merge existing statements to make room.

6. **Keep the tree compiling in `pkg/factory/factory.go`**: `CreateNotifiers` currently dispatches on `w.Type`, which no longer exists.
   - Delete `createNotifierForWatcher` and construct the log notifier for every entry (`notify.NewLogNotifier(w.DedupTTL)`) inside a loop. Keep the exported signature `CreateNotifiers(cfg config.Config) []notify.Notifier` unchanged so the call site in `pkg/cli/cli.go` keeps compiling and the constructor survives untouched until the gated deletion prompt, and keep the function pure composition (no conditionals, no I/O).
   - Put the interim state in a doc comment on `CreateNotifiers`, e.g. `// CreateNotifiers is an interim placeholder: watcher entries are filter-only, so the bespoke path can no longer be selected per type. Delivery is log-only until the shared publish path lands in the next prompt; this constructor and pkg/notify are removed by the gated deletion prompt of spec 004.`
   - Remove the now-unused `net/http` import. Leave `CreateConfigLoader` and `CreateWatcher` untouched.
   - Update `pkg/factory/factory_test.go` in the same pass: its fixtures build `config.WatcherConfig` values with `Type`, `URL`, `Token` and `ChatID`, which no longer compile. Rewrite them as filter-only entries and keep the structural expectations (one non-nil notifier per entry, empty slice for no entries, `CreateWatcher` returns non-nil). Do not delete `pkg/factory/factory_test.go` coverage of `CreateConfigLoader` and `CreateWatcher`.
   - **`pkg/factory/factory_test.go` is not the only compile casualty.** `pkg/watcher/watcher_test.go` carries **12** `Type: "log"` keyed literals (lines 59, 145, 237, 238, 256, 257, 280, 281, 299, 300, 322, 323) in its `config.WatcherConfig` fixtures; they stop compiling the moment `Type` is removed, so `go test ./pkg/watcher/...` — and therefore `make precommit` — fails. Delete only the `Type: "log"` key from each literal: keep every spec and every expectation unchanged, so the package's **11** specs stay at 11. Do not delete, merge, `t.Skip` or rewrite any of them; prompt 1b rewrites this file onto the publisher seam later, and the spec's "trigger unchanged" evidence requires the count to be identical before and after.

7. **Update `pkg/config/config_test.go`** — the fixtures are the biggest part of this change:
   - Delete the specs that only asserted the removed per-type validation: "returns error when watcher is missing type", "returns error when watcher has unknown type", "returns error when openclaw-wake watcher is missing url", "returns error when openclaw-wake watcher is missing token", "returns error when telegram watcher is missing token", "returns error when telegram watcher is missing chat_id", and "accepts log type with no extra fields".
   - Rewrite every remaining fixture that carries `type:` (and any `url:`/`token:`/`chat_id:`) into a filter-only entry. The "missing name" spec keeps its intent with a fixture that has no removed field, e.g. `- assignee: alice`.
   - Add **one spec per removed field** — `type`, `url`, `token`, `chat_id` — each loading a config whose watcher entry carries that single field with a non-empty value and asserting the load fails with an error naming both the watcher and the field (`Expect(err.Error()).To(ContainSubstring("chat_id"))` and the watcher name).
   - Keep: the old-flat-format specs (`assignee`, `webhook`, `format`), the vault validation specs, `dedup_ttl` default of 5 minutes, `dedup_ttl: "30m"` parsing, the invalid `dedup_ttl` error, and all `findConfigDir` specs.
   - Rewrite the "full valid config" spec into a filter-only config (two vaults, two or three watchers carrying `assignee`/`statuses`/`phases`/`dedup_ttl`) and assert the parsed `name`/`assignee`/`statuses`/`phases`/`DedupTTL` values. Do not try to assert "no entry exposes a channel field" — the field no longer exists, so that is a compile-time property, not a runnable expectation; the migration specs above are what cover the removed keys.
   - **Record the intended spec-count deltas, so AC9's "unchanged spec count" evidence is unambiguous:** AC9's "unchanged" applies to the pre-existing **trigger** specs — `pkg/watcher/` stays at **11** (unchanged, see requirement 6). The other two necessarily shift, and that is DB3/DB6 working as specified rather than a regression: `pkg/config/` goes **30 → 32** (7 per-type specs deleted, 4 migration specs added, 5 in the new `destination_test.go`); `pkg/cli/` goes **2 → 4**. State the counts before and after in your summary, and say plainly which package is the "unchanged" one.

8. **Add `pkg/config/destination_test.go`** (package `config_test`, same suite) covering `LoadDestinationFromEnv` with `GinkgoT().Setenv`:
   - both variables set → brokers parsed into two entries and the prefix preserved;
   - `KAFKA_BROKERS` empty → error names `KAFKA_BROKERS`; `KAFKA_BROKERS` unset → error names `KAFKA_BROKERS`;
   - `TOPIC_PREFIX` empty → error names `TOPIC_PREFIX`; `TOPIC_PREFIX` unset → error names `TOPIC_PREFIX`.
   For the unset cases, unset the variable explicitly and restore it with `DeferCleanup`, since `GinkgoT().Setenv` cannot express "unset".

9. **Add two specs to `pkg/cli/cli_test.go`** proving the fail-fast behaviour through the real entry point `cli.Run` (not just through the loader): with a valid temp config on disk and `KAFKA_BROKERS` unset, `Run` returns an error naming `KAFKA_BROKERS`; with `TOPIC_PREFIX` empty, `Run` returns an error naming `TOPIC_PREFIX`. Use `GinkgoT().Setenv` for the set side and `DeferCleanup`/`os.Unsetenv` for the unset side. The existing specs (missing config file, `--help` output) must keep passing unchanged.

10. **Add the changelog entry**: insert a `## Unreleased` section directly beneath the existing CHANGELOG preamble and above the newest released section (`## v0.19.5`), with a `- feat:` bullet for this prompt's work. Do not touch the preamble or any released section. If `## Unreleased` already exists, add your bullet to it.

11. **Do NOT touch `README.md` or `CLAUDE.md` in this prompt.** The spec keeps the user-facing docs update with the gated deletion prompt, where the whole story is known. Record the interim factory state in the code comment from requirement 6 instead.

12. Self-review against `docs/dod.md` before finishing: exported names have doc comments, new errors use `github.com/bborbe/errors`, no `fmt.Printf`/debug output, no `//nolint` without a reason.
</requirements>

<constraints>
- **Frozen producer contract (the next prompt implements it; do not implement it here):** notifications are published as a `NotificationPublishCommand` onto the CQRS Kafka command topic through the shared core's command sender — never through the controller's `/send` HTTP bridge, which sits behind Google SSO and is human-browser-only.
- **No new notification type.** The shared core's `agent-escalation` type is reused as-is; the library, the controller and the channel services are not touched.
- **No IAM change** — no new permission, role, or role binding. Publishing is not IAM-gated.
- **No new credential** — no TeamVault key, no token, no chat id in this repo or in any log line. This step *removes* credentials from the config surface.
- **No change to the trigger, the filters, or the empty-assignee / empty-status / nil-phase guard** — this spec changes what happens after a match, not what counts as a match.
- **No dedup inside the core** — duplicate suppression stays producer-side.
- **No new channels, no retry queue, no publish buffer, no metrics endpoint, no health endpoint.**
- **Do NOT add a per-watcher dry-run, opt-out, or "keep the old channel" flag.** The topic prefix is the test path (dev routing reaches the test channel, prod reaches the real one); a flag that disables the shared path re-creates the bypass this spec removes. Invariant.
- **Ordering gate:** the live channel is never deleted before the shared path is proven. Do not delete `pkg/notify/`, `mocks/notifier.go`, `CreateNotifiers`, or the notifier specs in this prompt.
- **Scope of edits:** only this repo.
- **Repo conventions:** Ginkgo v2 / Gomega specs, counterfeiter fakes for local interfaces, factory functions stay pure composition, `github.com/bborbe/errors` for wrapping, structured logging (`log/slog`) — no `fmt.Printf`.
- **Definition of Done:** `docs/dod.md` applies; `CHANGELOG.md` carries an entry under `## Unreleased`.
- Do NOT commit — dark-factory handles git.
- Do NOT run `go mod vendor` and never use `-mod=vendor`. Existing tests must still pass.
</constraints>

<verification>
Run the suite and the full gate — both must exit 0:

```bash
go test -mod=mod ./pkg/config/... ./pkg/cli/... ./pkg/factory/... ./pkg/watcher/...
make precommit
```

Then prove the two fail-fast exits and the INFO prefix line with a real binary and a minimal filter-only config:

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

# 1. missing KAFKA_BROKERS -> non-zero exit, error names the variable
env -u KAFKA_BROKERS /tmp/task-watcher --config /tmp/tw-minimal.yaml > /tmp/tw-nobroker.log 2>&1
echo "exit=$?"                                # expect non-zero
grep -c KAFKA_BROKERS /tmp/tw-nobroker.log    # expect >= 1

# 2. empty TOPIC_PREFIX -> non-zero exit, error names the variable
KAFKA_BROKERS=127.0.0.1:9092 TOPIC_PREFIX= /tmp/task-watcher --config /tmp/tw-minimal.yaml > /tmp/tw-noprefix.log 2>&1
echo "exit=$?"                                # expect non-zero
grep -c TOPIC_PREFIX /tmp/tw-noprefix.log     # expect >= 1

# 3. startup log carries the effective prefix at INFO
KAFKA_BROKERS=127.0.0.1:9092 TOPIC_PREFIX=develop /tmp/task-watcher --config /tmp/tw-minimal.yaml > /tmp/tw-startup.log 2>&1 &
TW_PID=$!
sleep 3
kill "$TW_PID" 2>/dev/null || true
grep -iE '(level=info|INFO).*topic_prefix=develop' /tmp/tw-startup.log   # expect >= 1 line, level INFO + prefix
grep -c 'topic_prefix=develop' /tmp/tw-startup.log        # expect >= 1
```

Finally confirm the migration error fires once per removed field:

```bash
for f in type url token chat_id; do
  printf 'vaults:\n  test:\n    path: /tmp/tw-vault\n    tasks_dir: tasks\nwatchers:\n  - name: stale\n    %s: somevalue\n' "$f" > /tmp/tw-stale.yaml
  env -u KAFKA_BROKERS /tmp/task-watcher --config /tmp/tw-stale.yaml 2>&1 | grep -c "$f"
done
```

Each iteration must print 1 or more (the config error fires before the destination check, because the config is loaded first). `test -d pkg/notify` must still exit 0 — the bespoke package is **not** deleted in this prompt. (The spec's own `test ! -d pkg/notify` check belongs to the deletion prompt, a later step; do not satisfy it here.)

This prompt is the one that introduces the Kafka dependency, so it is also the moment to prove the spec's "no consumer was added" claim and that the publish path exists at all:

```bash
grep -rn "libkafka\|sarama\|Consumer" --include='*.go' . | wc -l              # 0 — no Kafka consumer was added
grep -rn "agent-escalation\|KAFKA_BROKERS" --include='*.go' . | wc -l        # >= 1
```
</verification>

<!--
REVIEWER NOTES — surfaced deliberately, resolve at audit time.

1. AC9's evidence greps for the token `libkafka` (`grep -rn "libkafka\|sarama\|Consumer" --include='*.go' .`
   must return 0 lines) as proof that no Kafka *consumer* was added. The frozen producer contract needs the
   kafka library in this repo (brokers parsing here, the sync producer in prompt 1b), so requirement 4 writes
   the import unaliased (`github.com/bborbe/kafka` -> package `kafka`). The spec's intent — no consumer — holds:
   nothing consumes, `sarama` is never named, and the watch trigger is untouched. Aliasing the import as
   `libkafka` would make that grep non-zero and falsify the criterion without changing any behaviour, so the
   **import stays unaliased** — the constraint is on the alias, not on the contract.

2. The spec's decomposition assigns "the factory type switch" to the deletion prompt. The switch reads
   `WatcherConfig.Type`, `URL`, `Token` and `ChatID`, all of which this prompt removes as required by DB3
   (filter-only entries), so the switch cannot survive here. It is replaced by an interim log-notifier
   construction whose removal stays in the deletion prompt — the end state is unchanged, only the split moves
   by one step. Flagging it because the audited table names the switch as prompt 2's scope.

3. Interim delivery note: after this prompt the tree no longer selects a per-type sender, so the *tree* has no
   working Telegram/OpenClaw delivery until prompt 1b wires the shared publish path. The deployed build is only
   cut after 1a+1b are both merged with the operator's live proof, which is the spec's intended sequence. If
   the reviewer wants the bespoke path to remain selectable in between, DB3 has to be relaxed instead, because
   the channel fields are exactly what selected it.
-->
