# Changelog

All notable changes to this project will be documented in this file.

## v0.18.0

- feat: Fan out vault events to all configured watchers with per-watcher filter and dedup
- refactor: Remove --dry-run flag (use log action type in config instead)

## v0.17.0

- feat: add Telegram Bot API notifier (`pkg/notify/telegram.go`) with TTL-based dedup and token-safe logging
- feat: add log notifier (`pkg/notify/log.go`) replacing dry-run mode for debug/dry-run observation
- refactor: delete `pkg/notify/dry_run.go` and `pkg/notify/dry_run_test.go`, replace factory references with `NewLogNotifier`

## v0.16.0

- feat: Replace flat config with watchers list supporting multiple independent notifiers per process

## v0.15.0

- feat: replace permanent dedup with TTL-based dedup — webhooks re-fire after configurable `dedup_ttl` (default 5 minutes) so missed notifications are retried

## v0.14.2

- refactor: migrate OpenClaw notifier from /hooks/agent to /hooks/wake endpoint with simplified payload shape (text + mode)

## v0.14.1

- fix: remove exclude directives from go.mod so `go install @latest` works

## v0.14.0

- feat: log version field on startup so operators can confirm which build is running

## v0.13.0

- feat: add slog logging to real notifiers — debug log before HTTP call (method, url, headers, masked auth), info log after successful delivery, debug log on dedup skip

## v0.12.2

- fix: tests no longer hang when ~/.task-watcher/config.yaml exists on host
- docs: add install and replace/exclude rules to DoD

## v0.12.1

- docs: update README with webhook formats section, link to OpenClaw webhook spec, trim phases to actionable set

## v0.12.0

- feat: replace single `vault.path` config with `vaults` map supporting multiple Obsidian vaults, each with its own `path` and `tasks_dir`, watched simultaneously by one process

## v0.11.0

- feat: add OpenClaw webhook format support with `format: openclaw` config field, `webhook_token` auth header, and OpenClaw-shaped payload `{name, message, wakeMode, deliver}` sent to `/hooks/agent`

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

## v0.10.1

- fix: dry-run notifier now logs JSON body and Content-Type header to show the full HTTP request that would be sent

## v0.10.0

- feat: add --dry-run flag that replaces HTTP webhook notifier with a logging-only notifier for safe testing

## v0.9.0

- feat: add default config path ~/.task-watcher/config.yaml so --config flag is optional

## v0.8.0

- feat: add --version flag to CLI using cobra's built-in version support with ldflags injection

## v0.7.0

- feat: replace stdlib flag with cobra in pkg/cli, eliminating glog flag pollution from --help output

## v0.6.0

- feat: implement main.go with flag parsing, config loading, signal handling, and graceful shutdown

## v0.5.0

- feat: add pkg/factory with pure composition factory functions CreateConfigLoader, CreateNotifier, and CreateWatcher

## v0.4.1

- refactor: consolidate counterfeiter mocks to project-root mocks/ directory, matching vault-cli convention

## v0.4.0

- feat: add pkg/watcher with vault file-watching, task frontmatter filtering by assignee/status/phase, and notifier integration

## v0.3.0

- feat: add pkg/notify with HTTP webhook notifier, in-memory deduplication, and Counterfeiter mock

## v0.2.0

- feat: add pkg/config with YAML config loader, field validation, and ~/  expansion

## v0.1.1

- chore: remove skeleton-specific code (Kafka, BoltDB, Sentry, HTTP handlers, build-info-metrics) and replace main.go with minimal placeholder

## v0.1.0

### Fixed
- Exclude golangci-lint v1 transitive dep, update kafka to v1.22.8
- fix: pin opencontainers/runtime-spec to v1.2.1 to resolve containerd v1.7.30 compilation failure with Go 1.26

### Added
- Initial project structure from go-skeleton
- Module github.com/bborbe/task-watcher
