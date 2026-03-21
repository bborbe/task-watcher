---
status: completed
summary: Moved counterfeiter mocks to project-root mocks/ directory, updated Makefile generate target, added go:generate directives to notify and watcher suite test files, updated test imports, and deleted stale per-package mock directories.
container: task-watcher-006-spec-001-fix-mock-paths
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T10:58:06Z"
queued: "2026-03-21T10:58:06Z"
started: "2026-03-21T10:58:24Z"
completed: "2026-03-21T11:03:37Z"
---

<summary>
- Mock files move from `pkg/*/mocks/` to project-root `mocks/` directory
- Makefile generate target matches vault-cli pattern (rm, mkdir, package file, go generate)
- All counterfeiter `-o` paths updated to `../../mocks/` (relative from `pkg/*/`)
- `make precommit` passes after changes
</summary>

<objective>
Fix mock generation so all counterfeiter output goes to the project-root `mocks/` directory instead of per-package `pkg/*/mocks/` directories, matching the vault-cli convention.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/docs/go-patterns.md` for counterfeiter patterns.
Read `/home/node/.claude/docs/go-testing.md` for mock conventions.

Before writing any code, read:
- `pkg/config/config.go` — find `//counterfeiter:generate` directive
- `pkg/notify/notify.go` — find `//counterfeiter:generate` directive
- `pkg/watcher/watcher.go` — find `//counterfeiter:generate` directive
- `Makefile` — find `generate` target

Reference: vault-cli uses `../../mocks/` from `pkg/ops/` and its Makefile generate target is:
```makefile
.PHONY: generate
generate:
	rm -rf mocks
	mkdir -p mocks
	echo "package mocks" > mocks/mocks.go
	go generate -mod=mod ./...
```
</context>

<requirements>
1. Update all `//counterfeiter:generate` directives to output to project-root `mocks/`:
   - `pkg/config/config.go`: change `-o mocks/config_loader.go` to `-o ../../mocks/config_loader.go`
   - `pkg/notify/notify.go`: change `-o mocks/notifier.go` to `-o ../../mocks/notifier.go`
   - `pkg/watcher/watcher.go`: change `-o mocks/watcher.go` to `-o ../../mocks/watcher.go`

2. Update `Makefile` generate target to match vault-cli pattern:
   ```makefile
   .PHONY: generate
   generate:
   	rm -rf mocks
   	mkdir -p mocks
   	echo "package mocks" > mocks/mocks.go
   	go generate -mod=mod ./...
   ```

3. Delete stale per-package mock directories:
   - `rm -rf pkg/config/mocks/`
   - `rm -rf pkg/notify/mocks/`
   - `rm -rf pkg/watcher/mocks/`

4. Run `make generate` to regenerate mocks in the correct location.

5. Update test imports from per-package mock paths to project-root `mocks` package:
   - `pkg/watcher/watcher_test.go`: change `github.com/bborbe/task-watcher/pkg/notify/mocks` to `github.com/bborbe/task-watcher/mocks`
   - Check all other `*_test.go` files for any additional mock imports and fix them the same way

6. Run `make test` — all tests must pass with new mock paths.

7. Run `make precommit` — must pass.
</requirements>

<constraints>
- Only change counterfeiter `-o` paths, Makefile generate target, mock imports in tests, and delete stale mock dirs
- Do NOT modify any business logic or interfaces
- Do NOT modify `pkg/pkg_suite_test.go`
- `make precommit` must pass
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
```bash
# Mocks exist in project root
ls mocks/*.go

# No per-package mocks remain
test ! -d pkg/config/mocks
test ! -d pkg/notify/mocks
test ! -d pkg/watcher/mocks

# Tests pass
make test

# Full check
make precommit
```
All must exit with code 0.
</verification>
