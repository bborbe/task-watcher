# task-watcher

[![Go Reference](https://pkg.go.dev/badge/github.com/bborbe/task-watcher.svg)](https://pkg.go.dev/github.com/bborbe/task-watcher)
[![CI](https://github.com/bborbe/task-watcher/actions/workflows/ci.yml/badge.svg)](https://github.com/bborbe/task-watcher/actions/workflows/ci.yml)

Watches vault task files for phase/status changes and notifies configured agents via webhook.

## Usage

```bash
# default config: ~/.task-watcher/config.yaml
task-watcher

# custom config path
task-watcher --config /etc/task-watcher/config.yaml

# dry-run mode (logs instead of sending webhooks)
task-watcher --dry-run

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

assignee: bborbe

statuses:
  - in_progress

phases:
  - todo
  - planning
  - in_progress
  - ai_review
  - human_review
  - done

# "generic" (default) or "openclaw"
format: openclaw

webhook: http://localhost:9999/hooks/agent

# required when format is "openclaw"
webhook_token: my-secret-token
```

## Development

```bash
make precommit
```

## License

BSD-2-Clause
