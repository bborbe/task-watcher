---
status: completed
approved: "2026-03-20T19:09:27Z"
prompted: "2026-03-20T19:11:11Z"
verifying: "2026-03-21T11:11:51Z"
completed: "2026-03-21T18:02:19Z"
branch: dark-factory/main-wiring
---

## Summary

- Wire the three core packages (config, watcher, notifier) into a working CLI binary via main.go
- Binary accepts `--config` flag pointing to a YAML config file, loads it, constructs all components, and starts watching
- Graceful shutdown on SIGTERM/SIGINT: cancels context, waits for watcher to drain, then exits
- Invalid or missing config causes immediate exit with a non-zero code and human-readable error
- Depends on Spec 1 (core-packages) being complete -- this spec only covers composition and startup

## Problem

The core packages (config, watcher, notifier) exist as independent units but nothing wires them together into a runnable process. Without main wiring, the project cannot be built, deployed, or tested end-to-end.

## Goal

A single binary that reads a YAML config file, constructs all components via pure factory functions, starts a filesystem watcher filtered by assignee/status/phase, delivers webhook notifications, and shuts down cleanly on OS signals. The binary is deployable as a long-running systemd service or container process.

## Non-goals

- Implementing the core packages themselves (covered by Spec 1)
- Multi-config or hot-reload support
- Health-check endpoint or metrics
- Daemonization or process supervision
- Logging framework selection beyond Go's standard library

## Desired Behavior

1. Running `task-watcher --config path/to/config.yaml` parses the flag, loads the YAML file via the config package, and starts the watcher.
2. On successful startup, a log line is emitted containing the vault path and assignee from config (confirming correct config was loaded).
3. The watcher runs continuously, forwarding matching task changes to the notifier, until an OS signal arrives.
4. On SIGTERM or SIGINT, the binary cancels its root context, waits for the watcher to finish its current cycle, and exits with code 0.
5. If `--config` is omitted or the file is missing/unreadable/invalid, the binary exits immediately with a non-zero code and a message describing the problem.
6. The binary starts, watches, and shuts down without any observable side effects at construction time -- no network calls or file reads occur until the watcher is explicitly started.

## Assumptions

- Spec 1 (core-packages) is complete: pkg/config, pkg/notify, pkg/watcher exist with stable interfaces
- vault-cli library API is stable and importable
- Standard library `flag` package is sufficient for CLI flag parsing
- Graceful shutdown timeout of 5 seconds is acceptable before force exit

## Constraints

- Factory functions must be pure composition: accept config values and dependencies, return constructed objects. No file reads, no HTTP calls, no context creation inside factories.
- `context.Background()` is created exactly once, in main, and threaded through.
- Ginkgo/Gomega for any tests covering main-adjacent logic (e.g., flag parsing, factory wiring).
- `make precommit` must pass after all changes.
- vault-cli imported as library (`github.com/bborbe/vault-cli/pkg/ops`), never shelled out to.
- Single assignee + single webhook per process (no multi-tenant).

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---------|-------------------|----------|
| `--config` flag omitted | Exit code 1, message: "config flag required" | User provides flag |
| Config file not found | Exit code 1, message includes file path | User fixes path |
| Config file invalid YAML | Exit code 1, message describes parse error | User fixes YAML |
| Config missing required fields | Exit code 1, message names missing field(s) | User adds fields |
| SIGTERM during active watch cycle | Context cancelled, watcher finishes current iteration, exit 0 | Normal operation |
| Double signal (SIGTERM then SIGTERM) | Force exit if shutdown exceeds reasonable timeout | Process manager restarts |

## Security / Abuse Cases

- Config file path is user-controlled input; must be validated as a readable file before parsing.
- Webhook URL from config crosses a trust boundary (outbound HTTP); covered by notifier package, not this spec.
- No network listeners opened by main wiring itself.

## Acceptance Criteria

- [ ] `go build .` produces a `task-watcher` binary
- [ ] `task-watcher --config testdata/valid.yaml` starts, logs vault path and assignee, and blocks
- [ ] `task-watcher` without `--config` exits with code 1 and descriptive error
- [ ] `task-watcher --config /nonexistent` exits with code 1 and descriptive error
- [ ] Sending SIGTERM to a running process causes clean exit with code 0
- [ ] No `context.Background()` calls outside of main
- [ ] `make precommit` passes

## Verification

```
make precommit
```

## Do-Nothing Option

Without main wiring, the three core packages cannot be used together. The project remains a collection of unconnected libraries with no deployable artifact. This is not acceptable -- the binary is the deliverable.
