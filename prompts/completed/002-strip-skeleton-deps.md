---
status: completed
summary: Replaced skeleton main.go with minimal placeholder, deleted pkg/handler/, pkg/factory/, pkg/build-info-metrics.go, main_test.go, and mocks/build-info-metrics.go, then ran go mod tidy to clean unused dependencies
container: task-watcher-002-strip-skeleton-deps
dark-factory-version: v0.59.5-dirty
created: "2026-03-20T18:19:15Z"
queued: "2026-03-20T18:19:15Z"
started: "2026-03-20T18:19:22Z"
completed: "2026-03-20T18:27:01Z"
---

<summary>
- Remove unused messaging, database, error-tracking, and web-server code inherited from the project template
- Delete skeleton-specific HTTP handlers and factory functions that task-watcher will never use
- Clean up the module dependency graph so no conflicting linter versions are pulled in transitively
- Replace the entry point with a minimal placeholder that compiles and passes all checks
- All linting, formatting, and tests pass after the cleanup
</summary>

<objective>
task-watcher was bootstrapped from go-skeleton which carries heavy deps (Kafka, BoltDB, Sentry) that task-watcher does not use. These pull in golangci-lint v1 transitively, conflicting with the golangci-lint v2 in tools.go. Strip all skeleton-specific code so go.mod is clean and precommit passes.
</objective>

<context>
Read CLAUDE.md for project conventions.
The current main.go imports: libboltkv, libkafka, libkv, libsentry, libhttp, gorilla/mux, prometheus — none of which task-watcher needs.
pkg/handler/ contains sentry-alert and test-loglevel handlers — skeleton-only.
pkg/factory/ contains factory functions for those handlers — skeleton-only.
pkg/build-info-metrics.go registers a Prometheus build-info metric — skeleton-only.
The real task-watcher logic does not exist yet — this prompt only removes the skeleton noise.
</context>

<requirements>
1. Replace `main.go` with a minimal placeholder:
   - Package main, import only "context" and "fmt"
   - func main() that prints "task-watcher" and exits cleanly
   - No Kafka, BoltDB, Sentry, HTTP, gorilla/mux, prometheus imports
2. Delete `pkg/handler/` directory entirely
3. Delete `pkg/factory/` directory entirely
4. Delete `pkg/build-info-metrics.go`
5. Delete `main_test.go` (tests the skeleton main, not task-watcher logic)
6. Run `go mod tidy` to remove unused dependencies
7. Verify `go build -mod=mod ./...` compiles cleanly
8. Run `make precommit` — must pass
</requirements>

<constraints>
- Do NOT commit — dark-factory handles git
- Do NOT add any task-watcher business logic — only remove skeleton code
- Do NOT remove tools.go, Makefile, .golangci.yml, or any build tooling
- Do NOT remove pkg/pkg_suite_test.go if it exists — keep test infrastructure
</constraints>

<verification>
Run `make precommit` — must pass with exit code 0.
</verification>
