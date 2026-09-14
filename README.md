# task-watcher

[![Go Reference](https://pkg.go.dev/badge/github.com/bborbe/task-watcher.svg)](https://pkg.go.dev/github.com/bborbe/task-watcher)
[![CI](https://github.com/bborbe/task-watcher/actions/workflows/ci.yml/badge.svg)](https://github.com/bborbe/task-watcher/actions/workflows/ci.yml)

Watches vault task files for phase/status changes and publishes matching events into the shared
delivery core. It implements no delivery channel of its own: which channel a notification reaches
is decided by the deployed routing table, not by this process.

## Usage

```bash
# default config: ~/.config/task-watcher/config.yaml (fallback: ~/.task-watcher/config.yaml)
task-watcher

# custom config path
task-watcher --config /etc/task-watcher/config.yaml

# verbose logging
task-watcher --verbose

# version
task-watcher --version
```

## Configuration

```yaml
vaults:
  personal:
    path: ~/Documents/Obsidian/Personal
    tasks_dir: "24 Tasks"
  work:
    path: ~/Documents/Obsidian/Work
    tasks_dir: "Tasks"

watchers:
  - name: human-review
    assignee: bborbe
    statuses:
      - in_progress
    phases:
      - human_review
    # optional, defaults to 5m
    dedup_ttl: 5m
```

Each entry in `watchers` is a pure filter: it selects which task events match. The delivery
destination is owned by the environment, not by this file.

A config that still carries a channel field on a watcher entry — `type`, `url`, `token` or
`chat_id` — refuses to load, and the error names the offending field. This is deliberate: a stale
config must fail loudly rather than start and silently deliver nothing.

## Delivery

Two environment variables are required. The process refuses to start when either is unset or
empty, and the error names the missing variable.

| Variable | Meaning |
|----------|---------|
| `KAFKA_BROKERS` | Comma-separated broker list |
| `TOPIC_PREFIX` | Topic prefix for the shared core's command topic |

Every matched task event is published as a single `agent-escalation` notification with no target,
so the deployed routing table owns the channel decision. Repeats of the same task and phase inside
that entry's `dedup_ttl` (default 5 minutes) are suppressed. A failed publish is logged and does
not stop the watcher.

The effective topic prefix is logged at INFO at startup, and every publish attempt carries the
same prefix.

## Development

```bash
make precommit
```

## License

BSD-2-Clause
