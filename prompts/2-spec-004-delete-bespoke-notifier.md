---
spec: ["004-retire-bespoke-notifier"]
status: draft
created: "2026-09-13T20:54:32Z"
---

# Retire the bespoke notifier 2: delete the repo's own channel implementation

<summary>
- This repository stops implementing a delivery channel: the Telegram client, the OpenClaw webhook client, the log sender, the per-type sender selection, the generated mock and the specs that covered them leave the tree
- The removal is a deletion, not an unwiring — after this change the repository contains no channel code, no chat id and no bot token anywhere
- The notifier factory entry point goes with them, so nothing can construct a bespoke sender again
- The project documentation starts describing what the watcher actually does: filter vault task files and publish one notification into the shared delivery core
- The README's configuration section describes a filter-only watcher entry and the two required environment variables, instead of channel fields and a flag that no longer exists
- The rule that made this change safe is written into the repository's tracked documentation: the live channel is never deleted before the shared path is proven, because the proof is what makes the deletion safe
- That sentence is the part meant to outlive the spec
- The deletion commit has to be strictly later than the operator's committed proof — this step must not run before that proof exists
</summary>

<objective>
Remove every trace of this repo's own notification channel implementation, now that the shared publish path has been deployed and proven live by the operator, and record in the repository's tracked documentation why the deletion had to come second — the proof is what makes the deletion safe. The end state is a repo with no Telegram or OpenClaw HTTP code, no chat id, no bot token, no per-type sender selection, and no channel fields in config.
</objective>

<context>
Read `CLAUDE.md` for project conventions; this repo is edited **only** through the dark-factory pipeline.
Read `docs/dod.md` — the repo's own Definition of Done, used as the post-implementation self-review checklist.

Coding plugin docs (in-container paths):
- `/home/node/.claude/plugins/marketplaces/coding/docs/readme-guide.md` — README structure for a public CLI tool, and the split between README (user-facing) and CLAUDE.md (agent context)
- `/home/node/.claude/plugins/marketplaces/coding/docs/documentation-guide.md` — what belongs in README vs `docs/`
- `/home/node/.claude/plugins/marketplaces/coding/docs/changelog-guide.md` — `## Unreleased` entry format, prefixes, and the frozen-preamble rule

Read before changing anything:
- `README.md` — currently documents a `--dry-run` flag that no longer exists, the old flat config format, and a "Webhook Formats" section that describes a webhook this repo no longer posts
- `CLAUDE.md` — the Architecture section still lists `pkg/notify/` and still describes `main.go` as wiring "watcher + notifier"; the Key Design Decisions section carries the in-memory dedup note
- `pkg/notify/notify.go`, `pkg/notify/telegram.go`, `pkg/notify/openclaw.go`, `pkg/notify/log.go` and their specs — the files being deleted
- `pkg/factory/factory.go` — `CreateNotifiers` (the interim placeholder) and the `pkg/notify` import are the last production references
- `pkg/factory/factory_test.go` — the specs of `CreateNotifiers`
- `pkg/config/config.go` — `validateNoRemovedChannelFields` and the raw watcher fields **stay**; only stale prose about channel fields goes
- `CHANGELOG.md` — note whether `## Unreleased` still exists, and the frozen preamble above the newest released section
- `docs/` — currently only `dod.md`; the ordering rule needs a tracked home here

The operator's proof (`docs/live-proof-human-review-push.md`) is written and committed by the operator, not by this prompt — do not create, rewrite or extend it. Its existence and its timestamp relative to this commit are the gate for approving this prompt.
</context>

<requirements>
0. **Gate check before any edit.** Run `test -f docs/live-proof-human-review-push.md`. If it is absent, stop and report the blocker — do not delete `pkg/notify/`, do not edit `README.md`, do not touch the tree. The deletion must not run before the operator's proof is committed. (The "deletion commit strictly later than the proof commit" half is the operator's check, not one this container can run — see the note at the end of `<verification>`.)

1. **Delete `pkg/notify/` in full** — `notify.go`, `telegram.go`, `openclaw.go`, `log.go` and the specs that covered them (`notify_test.go`, `telegram_test.go`, `openclaw_test.go`, `log_test.go`, `suite_test.go`). The whole directory goes; do not leave a stub, a deprecation note, or commented-out code.

2. **Delete the bespoke construction** in `pkg/factory/factory.go`: remove `CreateNotifiers` entirely (the interim placeholder included) and the now-unused `pkg/notify` import. Leave `CreateConfigLoader`, `CreateNotificationSender`, `CreatePublishers` and `CreateWatcher` exactly as they are — they are the path production now uses. The file must contain no reference to a per-type sender selection and no placeholder comment about the bespoke path.

3. **Let regeneration remove the mock, do not hand-delete it.** `mocks/notifier.go` is counterfeiter output; `make generate` wipes `mocks/` and rebuilds it from the surviving `//counterfeiter:generate` directives. After the final `make precommit`, `mocks/notifier.go` must be gone while `mocks/config_loader.go`, `mocks/watcher.go` and `mocks/notification_publish_command_sender.go` are present. Never hand-edit a file under `mocks/`. Note that `make generate` recreates `mocks/mocks.go` without its BSD banner and `addlicense` (the last `precommit` step) re-adds it — if an earlier step fails, run `make addlicense` and confirm `head -1 mocks/mocks.go` prints a `// Copyright` line.

4. **Update every spec that referenced the deleted code**: in `pkg/factory/factory_test.go`, drop the `pkg/notify` import and the `CreateNotifiers` specs, and keep the coverage of `CreateConfigLoader`, `CreatePublishers` and `CreateWatcher`. Delete specs that no longer compile rather than commenting them out or `t.Skip`-ing them. Do not remove coverage of the publish path, the dedup gate or the config migration errors.

5. **Leave the config migration detector in place.** `pkg/config/config.go`'s `validateNoRemovedChannelFields` and the raw watcher fields exist so a stale config fails loudly with an error naming the field — that behaviour is an acceptance criterion of the spec (a config that still carries a channel field must not start and silently deliver nothing). Do not delete it, do not weaken it, and keep its specs. Only stale *prose* about channel fields (comments, docs) is removed.

6. **Rewrite `README.md` so it matches the code** — this diff is required to touch README.md, and the current content is wrong in three places:
   - One-line description: the watcher watches vault task files and publishes notifications into the shared delivery core; it implements no channel of its own.
   - Usage: keep `--config`, `--verbose`, `--version`; delete the `--dry-run` example (the flag does not exist any more).
   - Configuration: show a filter-only watcher entry — `name`, `assignee`, `statuses`, `phases`, `dedup_ttl` — alongside the `vaults` map, and state that a config still carrying `type`, `url`, `token` or `chat_id` fails to load with an error naming that field (so a stale config cannot start and silently deliver nothing). Name those fields in prose only — the YAML example must contain none of them. Delete the old flat-format example (`assignee`/`format`/`webhook`/`webhook_token` at the top level), and remove every surviving "via webhook" / OpenClaw / Telegram claim from the prose, not just the section headings.
   - New Delivery section: the process requires `KAFKA_BROKERS` (comma-separated) and `TOPIC_PREFIX`, and refuses to start when either is unset or empty; every matched task event is published as one `agent-escalation` notification with no target, so the deployed routing table owns the channel decision; repeats of the same task and phase inside that entry's `dedup_ttl` (default 5 minutes) are suppressed, and a failed publish is logged without stopping the watcher; the effective topic prefix is logged at INFO at startup, and every publish attempt carries the same prefix.
   - Delete the "Webhook Formats" section entirely — nothing here formats webhooks any more.
   - Keep the badge block, Development and License sections. Keep it user-facing: no internal reasoning, no acceptance-criteria vocabulary.

7. **Update `CLAUDE.md`'s Architecture section** (agent-facing context): `main.go`'s line, `pkg/config/` (filter-only watcher entries plus the environment-owned destination), the new publish package, and `pkg/watcher/`'s publish wording. Delete the `pkg/notify/` line — it is stale and partial. In Key Design Decisions, replace the in-memory dedup wording with the current truth (dedup is per watcher entry, in memory, keyed on task plus phase, measured on the injectable clock, cleared by a restart) and add the ordering rule from requirement 8.

7b. **`CLAUDE.md` is gitignored and deliberately untracked** — `.gitignore` line `/CLAUDE.md`, and `git ls-files CLAUDE.md` is empty. The edit in requirement 7 is for the working-tree copy only, so the next agent reading it is not misled. Never `git add -f` it, and never treat its absence from the commit's `--stat` as a failure; the durable ordering rule lives in the **tracked** page from requirement 8, which is what the deletion commit's diff will actually show.

8. **Create the tracked ordering rule page** `docs/ordering-live-channel-retirement.md` (short — at most ~25 lines, no operator to-do items, no acceptance-criteria vocabulary). It must contain this sentence verbatim, in prose:
   > The live channel is never deleted before the shared path is proven, because the proof is what makes the deletion safe.

   and around it, factually: what "proven" means here (the operator's capture in `docs/live-proof-human-review-push.md`: this watcher's publish line, the controller's `notification(agent-escalation) routed to ...` lines and the telegram service's delivery line, spanning at most 60 s, for both the dev and the prod topic prefix); why the previous binary and the previous config are kept (they are the rollback for a failed proof); and that the deletion commit must be strictly later than the proof commit, because a reader of the history has only the timestamps to tell which came first.

9. **`CHANGELOG.md`**: add an entry under `## Unreleased` covering the retirement (`- feat:` or `- chore:` prefix per the changelog guide). If a release has renamed that section in the meantime, create a fresh `## Unreleased` directly beneath the preamble and above the newest released section. Never edit a released section and never move the preamble.

10. **Confirm the config documentation in the repo carries no channel field references outside the detector and its specs**, and that no other tracked document still promises a webhook, a Telegram bot or an OpenClaw wake.

11. Self-review against `docs/dod.md` before finishing: exported names carry doc comments, errors are wrapped with `github.com/bborbe/errors`, structured logging only, no `//nolint` without a reason.
</requirements>

<constraints>
- **This prompt is gated.** It must not run before the operator's live proof (`docs/live-proof-human-review-push.md`) is committed, and the deletion commit must be strictly later than the proof commit. If the proof file is absent, or if its timestamps and the deployed version do not back it, stop and report the blocker — deleting the live channel before the shared path is proven is a failure of the spec even if every other check is green.
- **Deletion, not unwiring.** No deprecation shims, no build tags, no commented-out senders, no "keep the old channel" escape hatch.
- **`CLAUDE.md` is gitignored (`.gitignore` line `/CLAUDE.md`) and deliberately untracked** — requirement 7 edits the working-tree copy for the next agent reading it. Never `git add -f` it, and never treat its absence from the commit as a failure.
- **Evidence scoping is deliberate, not a loophole:** the `chat_id` → 0 assertion excludes `pkg/config/` because the migration detector required by this prompt must name the field. Do not delete the detector to make the grep pass.
- **No change to the shared core:** the notification library, the controller, the Discord and Telegram services, and the routing tables are not touched.
- **Do not change behaviour that the spec froze:** the trigger, the filters, the empty-assignee / empty-status / nil-phase guard, the `agent-escalation` type, the nil target on every publish, the single per-entry dedup gate, and the fail-fast destination checks all stay exactly as they are.
- **No new credential, no IAM:** this change removes credentials from the repo, it does not add any. No TeamVault key, no permission, role or binding.
- **Do not delete the config migration detector** (`validateNoRemovedChannelFields`) or its specs — a stale config must keep failing to load with an error naming the field.
- **Do NOT add a per-watcher dry-run, opt-out, or "keep the old channel" flag** — the topic prefix is the test path; a flag that disables the shared path re-creates the bypass this spec removes.
- **Scope of edits:** only this repo.
- **Repo conventions:** Ginkgo v2 / Gomega specs, counterfeiter fakes regenerated via `make generate`, factory functions stay pure composition, `github.com/bborbe/errors`, structured logging (`log/slog`).
- **Definition of Done:** `docs/dod.md` applies, including CHANGELOG under `## Unreleased` and README updated for a change that affects configuration and setup.
- Do NOT commit — dark-factory handles git. Do NOT run `go mod vendor` and never use `-mod=vendor`. Existing tests must still pass.
</constraints>

<verification>
```bash
make precommit          # run FIRST: it regenerates mocks/ — until then mocks/notifier.go still imports the deleted pkg/notify and `go test ./...` cannot compile
go test -mod=mod ./...
```
Both must exit 0. `make precommit` regenerates `mocks/`, which is what removes the notifier mock — so the order above matters. Do **not** hand-delete `mocks/notifier.go` to make the intermediate state pass (requirement 3).

The gate first — the deletion must not happen before the operator's proof is committed:

```bash
test -f docs/live-proof-human-review-push.md && echo "live proof present"   # must print before anything below is meaningful
```

The bespoke surface must be gone — not merely unreferenced:

```bash
test ! -d pkg/notify && echo "pkg/notify gone"
test ! -f mocks/notifier.go && echo "notifier mock gone"
ls mocks/config_loader.go mocks/watcher.go mocks/notification_publish_command_sender.go
grep -rn "notify.Notifier\|api.telegram.org\|hooks/wake\|sendMessage\|openclaw-wake" --include='*.go' . | wc -l   # 0
grep -rn "chat_id" --include='*.go' . | grep -v '^\./pkg/config/' | wc -l                                     # 0
grep -rn "chat_id" --include='*.go' ./pkg/config | wc -l                                                       # >0: the migration detector and its specs (deliberate — see the constraints)
grep -rn "Permission\|RoleBinding" --include='*.go' . | wc -l                                                  # 0 — no authorization surface was added
grep -rn "bborbe/notification" --include='*.go' . | wc -l                                                      # >= 1 — the shared publish path is what remains
grep -rn "CreateNotifiers\|createNotifierForWatcher" --include='*.go' . | wc -l                                 # 0
```

Documentation evidence:

```bash
grep -A1 '^## Unreleased' CHANGELOG.md                    # >= 2 lines, and the line under the header is non-empty
grep -n "KAFKA_BROKERS\|TOPIC_PREFIX" README.md            # both documented
grep -nE "^[[:space:]]*(type|url|token|chat_id):" README.md | wc -l                          # 0 — no channel field appears in a config example
grep -n "dry-run\|hooks/wake\|Webhook Formats" README.md | wc -l                    # 0 — the stale sections are gone
grep -niE "webhook|openclaw|telegram|hooks/" README.md | wc -l                      # 0 — no flat-format example and no "via webhook" claim survives
test -f docs/ordering-live-channel-retirement.md
grep -c "never deleted before the shared path is proven" docs/ordering-live-channel-retirement.md   # >= 1
test "$(grep -c 'pkg/notify' CLAUDE.md)" = "0" && echo "no stale pkg/notify entry in CLAUDE.md"
grep -rniE "bot[0-9]{6,}:|112230768" --include='*.go' . | wc -l                      # 0 — no credential literal survives in code (this prompt is the one that removes them)
```

Note on scope: the git-history half of the ordering evidence (`git log -1` on the proof file being strictly earlier than the deletion commit) is created by dark-factory's commit *after* this prompt finishes, so it cannot be checked from inside the prompt. It belongs to the spec's verification ladder and the operator's approval gate for this prompt — report the proof file's timestamp and the deployed version you observed, and stop if the proof is missing.
</verification>

<!--
Spec-side notes (AC10 / AC11 evidence wording, and the factory-switch reassignment between prompts 1a and 2) are
recorded in the vault task's Progress entry, not here — this file is read by the executing agent, and an agent
must not read a reviewer's analysis of the acceptance criteria as licence to deviate from them. The two operative
guardrails that analysis produced are now constraints above: `CLAUDE.md` is deliberately untracked, and the
`chat_id` evidence scoping exists because the migration detector is required to name the field.
-->
