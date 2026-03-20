---
status: verifying
tags:
    - dark-factory
    - spec
approved: "2026-03-20T19:03:01Z"
prompted: "2026-03-20T19:05:55Z"
verifying: "2026-03-20T22:30:18Z"
branch: dark-factory/core-packages
---

## Summary

- Implement three core packages (config, notify, watcher) that form the foundation of task-watcher
- Config package loads and validates YAML configuration from a file path
- Notify package sends HTTP webhook notifications with in-memory deduplication
- Watcher package observes vault task files and triggers notifications when tasks match configured filters
- All packages are testable in isolation with Counterfeiter-generated interface mocks
- No main.go wiring -- that belongs to a separate spec

## Problem

task-watcher has no implementation beyond a placeholder main.go. The system needs to watch an Obsidian vault for task file changes, filter tasks by assignee/status/phase, and notify an external agent via webhook. Without the core packages, nothing can be built on top.

## Goal

Three independent, tested packages exist that can be composed into a working binary. Each package has a clear interface boundary: config produces a validated configuration, watcher observes file changes and reads task metadata, notifier delivers webhook calls. A separate spec will wire them together in main.go.

## Non-goals

- No CLI flags or main.go wiring (Spec 2)
- No persistent deduplication across process restarts
- No retry logic for failed webhook calls
- No graceful shutdown handling (Spec 2)
- No metrics or structured logging

## Desired Behavior

1. Given a YAML file path, config returns a validated configuration or a descriptive error explaining what is missing
2. Config rejects files missing required fields: vault path, assignee, at least one status, at least one phase, webhook URL
3. Config expands `~/` prefixes in the vault path to the user's home directory
4. Notifier sends an HTTP POST with JSON body containing task name, phase, and assignee to the configured webhook URL
5. Notifier returns an error when the webhook responds with a non-2xx status code
6. Notifier suppresses duplicate notifications: the same task+phase combination only fires once per process lifetime
7. Watcher detects when task files are created or modified in the vault tasks directory
8. Watcher reads task frontmatter via vault-cli's storage package, filters by assignee/status/phase against config, and calls the notifier on match

## Assumptions

- vault-cli's `WatchOperation` and `storage` package APIs are stable and suitable for library import
- YAML is the config format (no env-var or flag-based config in this spec)
- Counterfeiter is available via `tools.go` for mock generation
- The host filesystem supports inotify-style events (Linux/macOS)

## Constraints

- vault-cli is imported as a Go library (`github.com/bborbe/vault-cli`), never executed via shell
- Factory functions use pure composition: no I/O, no `context.Background()` inside constructors
- Tests use Ginkgo/Gomega with Counterfeiter-generated mocks for all cross-package interfaces
- Error handling follows `github.com/bborbe/errors` patterns
- `make precommit` must pass (linting, tests, formatting)
- Existing `pkg/pkg_suite_test.go` must continue to work
- No `//nolint` directives without explanation

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---------|-------------------|----------|
| Config file not found | Return descriptive error with file path | Caller logs and exits |
| Config missing required field | Return error naming the missing field | User fixes config |
| Webhook returns 5xx | Return error to caller with status code | Caller decides retry policy |
| Webhook URL unreachable | Return error (connection refused / timeout) | Caller decides retry policy |
| Duplicate task+phase event | Silently suppress, no error | None needed |
| Vault path does not exist | Not validated at config time (watcher will fail at runtime) | Watcher surfaces error |
| Task file has no frontmatter | Skip file, no notification | None needed |
| Task file frontmatter missing assignee/status/phase | Skip file, no notification | None needed |

## Security / Abuse Cases

- Webhook URL comes from config file on disk -- no user-controlled HTTP input at runtime
- Config file path is provided by the process owner (CLI arg), not external input
- Vault path expansion only handles `~/` prefix, no arbitrary environment variable expansion
- HTTP POST body is constructed from task metadata read from local files -- no injection vector unless vault files are compromised
- No secrets in the webhook JSON body

## Acceptance Criteria

- [ ] Config loads a valid YAML file and returns a typed configuration
- [ ] Config returns an error for each missing required field (vault.path, assignee, statuses, phases, webhook)
- [ ] Config expands `~/` to the home directory in vault.path
- [ ] Notifier sends HTTP POST with correct JSON body to webhook URL
- [ ] Notifier returns error on non-2xx response
- [ ] Notifier deduplicates: second call with same task+phase is a no-op
- [ ] Watcher calls notifier when a matching task file is created or modified
- [ ] Watcher does not call notifier for tasks that do not match the configured filters
- [ ] All packages have Ginkgo/Gomega tests with Counterfeiter mocks
- [ ] `make precommit` passes

## Verification

```
make precommit
```

## Do-Nothing Option

Without these packages, task-watcher remains a placeholder. No file watching, no webhook delivery, no progress toward the goal of automated task notification. The entire project is blocked.
