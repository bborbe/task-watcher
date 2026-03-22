---
status: prompted
approved: "2026-03-22T10:46:42Z"
prompted: "2026-03-22T11:07:46Z"
branch: dark-factory/multi-watcher-config
---

## Summary

- Config restructured from single filter+action to a list of 0-N independent watchers
- Three concrete action types: `openclaw-wake`, `telegram`, `log`
- Each watcher has its own filter (assignee, statuses, phases), action, and dedup TTL
- Backward-incompatible config change — existing configs must be migrated
- `log` type replaces dry-run mode

## Problem

Current config supports exactly one filter + one action. To add a second use case (e.g. Telegram notification on `phase: human_review`), you'd need a second task-watcher process with a separate config. This doesn't scale — N use cases = N processes watching the same files.

## Goal

A single task-watcher process supports 0-N independent watchers, each with its own filter criteria and action. Adding a new notification channel means adding one entry to the watchers list, not deploying a new service.

## Assumptions

- Telegram Bot API token and chat ID are provided in config (no OAuth flow)
- All watchers share the same vault file events (single fsnotify watcher, fan-out to N consumers)
- Dedup is per-watcher (each watcher has its own TTL and seen map)
- Order of watcher execution doesn't matter (independent, no dependencies between watchers)

## Non-goals

- Hot-reload of config (restart required for config changes)
- Watcher-specific vault filtering (all watchers see all vaults)
- Rate limiting or batching across watchers
- Generic webhook type (add later as a new concrete type if needed)
- Matrix, Slack, or other notification types (add as concrete types later)

## Desired Behavior

New config format:

```yaml
vaults:
  openclaw:
    path: ~/vault
    tasks_dir: tasks
  personal:
    path: ~/obsidian-personal
    tasks_dir: "24 Tasks"

watchers:
  - name: wake-tradingclaw
    type: openclaw-wake
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [planning, in_progress, ai_review]
    dedup_ttl: "5m"
    url: http://127.0.0.1:18789/hooks/wake
    token: "secret"

  - name: notify-review
    type: telegram
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [human_review]
    dedup_ttl: "30m"
    chat_id: "456"
    token: "bot123:ABC..."

  - name: debug
    type: log
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [planning]
```

All fields flat per watcher — type determines which fields are required:
- `openclaw-wake` — requires `url`, `token`
- `telegram` — requires `token`, `chat_id`
- `log` — no extra fields

Behavior:
1. Single file watcher monitors all vaults, fans out events to all configured watchers
2. Each watcher independently applies its own filter (assignee, statuses, phases)
3. Matching watchers execute their action type
4. Dedup is per-watcher with configurable TTL (default 5m)

## Constraints

- Config is backward-incompatible — document migration path in CHANGELOG
- Notifier contract unchanged — each action type implements the same notify behavior
- Factory functions remain pure composition (no I/O at construction time)
- Ginkgo/Gomega tests, Counterfeiter mocks
- `make precommit` must pass after all prompts

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---------|-------------------|----------|
| Empty watchers list | Process starts, watches files, but never notifies | Add watcher entries |
| Invalid action type | Config load error with descriptive message | Fix config |
| Telegram API down | Error logged, dedup entry set (retries after TTL) | Telegram recovers |
| One watcher fails | Other watchers unaffected, error logged | Fix failing watcher config |
| Missing telegram token | Config load error | Add token |

## Security / Abuse

- Telegram bot token is sensitive — stored in config file, not logged
- OpenClaw tokens already handled the same way
- No new network listeners opened

## Acceptance Criteria

- [ ] Config supports `watchers` list with per-watcher filters and actions
- [ ] `openclaw-wake` action type sends OpenClaw-formatted payloads with Bearer token
- [ ] `telegram` action type sends messages via Telegram Bot API
- [ ] Each watcher has independent dedup with configurable TTL
- [ ] Single fsnotify watcher fans out to all configured watchers
- [ ] `log` action type works as debug/dry-run replacement
- [ ] Old flat config format produces a clear error with migration hint
- [ ] All tests pass, `make precommit` passes

## Verification

```
make precommit
```

## Do-Nothing Option

Keep one action per process. Each new notification channel requires deploying a separate task-watcher instance with its own config, systemd unit, and resource overhead. N channels = N processes watching the same files.
