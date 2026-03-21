# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

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
