---
status: executing
container: task-watcher-018-migrate-openclaw-to-wake
dark-factory-version: v0.63.0
created: "2026-03-21T20:24:10Z"
queued: "2026-03-21T20:24:10Z"
started: "2026-03-21T20:24:12Z"
issue: migrate OpenClaw notifier from /hooks/agent to /hooks/wake
---

## Context

The OpenClaw `/hooks/agent` endpoint has session-routing issues. The simpler `/hooks/wake` endpoint is more reliable for task-watcher's use case (signal that a task changed, wake the main session).

## Current state

`pkg/notify/openclaw.go` sends payload to `/hooks/agent`:
```json
{"name":"task-watcher","message":"Task update: ...","wakeMode":"now","deliver":false}
```

`pkg/notify/dry_run.go` has `dryRunOpenClawNotifier` that logs the same payload shape.

## Required changes

Change the OpenClaw payload from `/hooks/agent` format to `/hooks/wake` format.

### New payload shape

```json
{"text":"Task watcher: <assignee> task changed (task: <taskname>, phase: <phase>)","mode":"now"}
```

### Files to change

1. **`pkg/notify/openclaw.go`**:
   - Rename `openClawPayload` struct fields: `Name`/`Message`/`WakeMode`/`Deliver` → `Text`/`Mode`
   - JSON tags: `"text"` and `"mode"` only (remove `name`, `wakeMode`, `deliver`)
   - Update payload construction: `Text` = `fmt.Sprintf("Task watcher: %s task changed (task: %s, phase: %s)", notification.Assignee, notification.TaskName, notification.Phase)`
   - `Mode` = `"now"`
   - Update doc comments to say `/hooks/wake` instead of `/hooks/agent`

2. **`pkg/notify/dry_run.go`**:
   - Update `dryRunOpenClawNotifier.Notify` to use same new payload shape
   - Update doc comment

3. **`pkg/notify/openclaw_test.go`**:
   - Update payload assertions: check `text` and `mode` fields instead of `name`, `message`, `wakeMode`, `deliver`
   - Expected `text` value: `"Task watcher: alice task changed (task: my-task, phase: planning)"`
   - Expected `mode` value: `"now"`
   - Remove assertions for `name`, `wakeMode`, `deliver`

4. **`CHANGELOG.md`**: create `## Unreleased` section above `## v0.14.1` and add entry

## Validation

`make precommit` must pass.
