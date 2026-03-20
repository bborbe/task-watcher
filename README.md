# task-watcher

[![Go Reference](https://pkg.go.dev/badge/github.com/bborbe/task-watcher.svg)](https://pkg.go.dev/github.com/bborbe/task-watcher)
[![CI](https://github.com/bborbe/task-watcher/actions/workflows/ci.yml/badge.svg)](https://github.com/bborbe/task-watcher/actions/workflows/ci.yml)

Watches vault task files for phase/status changes and notifies configured agents via webhook.

## Usage

```bash
task-watcher --config /etc/task-watcher/config.yaml
```

## Configuration

```yaml
vault:
  path: ~/vault

openclaw:
  assignee: TradingClaw
  status:
    - in_progress
  phases:
    - planning
  webhook: http://localhost:8080/hooks/wake
```

## Development

```bash
make precommit
```

## License

BSD-2-Clause
