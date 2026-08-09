---
status: completed
summary: Deleted tools.go and rewrote go.mod to remove all tool-only dependencies — go.mod shrank from 443 to 49 lines, all replace/exclude directives removed, generator directives pinned to v6.12.2, all bborbe and golang.org/x/* deps bumped to latest
execution_id: task-watcher-drop-tools-go-exec-025-delete-tools-go-and-clean-gomod
dark-factory-version: v0.192.9
created: "2026-08-09T15:22:03Z"
queued: "2026-08-09T15:22:03Z"
started: "2026-08-09T15:22:21Z"
completed: "2026-08-09T15:24:38Z"
---

# Delete tools.go and remove tool-dependency pollution from go.mod

<summary>
- Tool CLIs are no longer declared as Go module dependencies of this project
- The project's dependency list shrinks to only what the application actually uses
- Linters, scanners, and code generators keep running at exactly the same pinned versions
- Long-standing version-conflict workarounds are no longer needed and go away
- Two live CVEs disappear because the package carrying them was never a real dependency
- Mock generation keeps working, pinned to the same generator version as before
- Transitive dependencies are kept at safe versions rather than sliding back to vulnerable minimums
- No application behavior changes — this is dependency hygiene only
</summary>

<objective>
Delete `tools.go` and rewrite `go.mod` so tool CLIs are no longer module dependencies, because importing them pulls every tool's transitive dependency tree into this project — inflating `go.mod` to 443 lines, forcing five `replace` workarounds plus an `exclude` directive that the project's own `docs/dod.md` forbids, and dragging in `github.com/go-git/go-git/v5` — a package no source file here imports, which has repeatedly surfaced CVEs (most recently GHSA-hc8v-wwc9-vgxm and GHSA-qgq7-7hm3-q39j, already patched on this branch by a preceding commit). Removing it entirely means future go-git advisories never apply to this repo at all. Tool versions stay pinned via `tools.env` and `go run pkg@$(VERSION)`, which already works in the Makefile.
</objective>

<context>
Read the coding plugin's `docs/go-tools-versioning-guide.md` (in-container path: `/home/node/.claude/plugins/marketplaces/coding/docs/go-tools-versioning-guide.md`) — it is the canonical source for this migration. Requirements 4 and 7 are taken verbatim from its "Migration Steps" §5 and §7 respectively. Read its "Pitfalls" section: one unbumped `bborbe/*` dep brings the entire cascade back, and `go mod tidy -e` can truncate `go.mod`. Requirement 6 has NO counterpart in that guide — it is added here from a regression observed on the sibling `git-sync` migration earlier today, and is described in full at that requirement.

Read `docs/dod.md` — this repo's own Definition of Done, used as the post-implementation self-review checklist. Note its rule: "No `exclude` or `replace` directives in go.mod (break remote install)". The current `go.mod` violates it on both counts; requirement 3 fixes that.

Read `CLAUDE.md` for project conventions.

Read `tools.go` — the file to delete.

Read `Makefile` — it already has `include tools.env` and already invokes every tool as `go run pkg@$(VERSION)`. **No Makefile change is needed.** The `-mod=mod` usages (`go run -mod=mod main.go`, `go generate -mod=mod`, `go test -mod=mod`, `go vet -mod=mod`, `go list -mod=mod`) are correct and must stay.

Read `tools.env` — pinned tool versions; `COUNTERFEITER_VERSION` is `v6.12.2`. Leave the file untouched.

Read `go.mod` — note the five-entry `replace` block and the direct-require list mixing six real application dependencies with ten tool-only ones.
</context>

<requirements>
1. Update all five `//go:generate` counterfeiter directives to pin the version explicitly. In each of:
   - `pkg/pkg_suite_test.go`
   - `pkg/watcher/suite_test.go`
   - `pkg/notify/suite_test.go`
   - `pkg/cli/suite_test.go`
   - `pkg/config/suite_test.go`

   Change:
   ```
   //go:generate go run -mod=mod github.com/maxbrunsfeld/counterfeiter/v6 -generate
   ```
   to:
   ```
   //go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.12.2 -generate
   ```
   The version must match `COUNTERFEITER_VERSION` in `tools.env`.

2. Delete `tools.go` entirely.

3. Rewrite `go.mod` to a minimal form: keep `module github.com/bborbe/task-watcher` and `go 1.26.5`, delete the entire `replace` block, **delete the `exclude (cloud.google.com/go v0.26.0)` block as well** — `docs/dod.md` forbids both directives because they break remote install — and keep ONLY these six direct requires — each verified as genuinely imported by non-tool code:
   ```
   github.com/bborbe/errors
   github.com/bborbe/vault-cli
   github.com/onsi/ginkgo/v2
   github.com/onsi/gomega
   github.com/spf13/cobra
   gopkg.in/yaml.v3
   ```
   Keep each dep's existing version. Drop every tool-only direct require: `golangci-lint/v2`, `google/addlicense`, `osv-scanner/v2`, `goimports-reviser/v3`, `kisielk/errcheck`, `counterfeiter/v6`, `gosec/v2`, `segmentio/golines`, `shoenig/go-modtool`, `golang.org/x/vuln`.

   Delete the entire indirect require block — `go mod tidy` repopulates it.

4. Bump every `github.com/bborbe/*` dependency — direct AND indirect, do not filter on `// indirect` — to its latest release. There are eight today (2 direct, 6 indirect). An unbumped bborbe dep still carrying its own `tools.go` re-drags the whole pollution cascade back in:
   ```
   grep '^	github.com/bborbe/' go.mod | awk '{print $1}' | xargs -I {} go get {}@latest
   ```
   Run this after step 3, and again after `go mod tidy` in case tidy surfaces more indirect bborbe deps.

5. Run `go mod tidy` to repopulate legitimate indirect requires. Do NOT use `go mod tidy -e` — it can truncate `go.mod` when resolution partially fails.

6. **Bump the `golang.org/x/*` transitive dependencies to latest** after tidy:
   ```
   go get golang.org/x/net@latest golang.org/x/sync@latest golang.org/x/sys@latest golang.org/x/text@latest golang.org/x/crypto@latest golang.org/x/term@latest
   ```
   This step is required, not optional, and it is NOT in the referenced migration guide — it comes from a regression hit on the sibling `git-sync` repo during the same migration effort. Removing the tool dependencies drops the version floor those tools were imposing, so Go's minimal-version selection slides these packages *backwards* to older, CVE-carrying releases. On `git-sync`, `x/net` fell from v0.56.0 to v0.53.0 and `trivy` then failed with 4 HIGH CVEs that had been masked. Run this before the pollution checks so the final `make precommit` reflects the real end state.

7. Confirm zero tools.go-era pollution remains:
   ```
   grep -E '(cellbuf|go-header|go-diskfs|golangci-lint|osv-scanner|ginkgolinter|charmbracelet/x|denis-tingaikin)' go.mod
   ```
   Must return no matches. If any appear, run `go mod why <package>` — it will name the unbumped `bborbe/*` dep still pulling the old cascade — then re-run step 4 for it.

8. Confirm `go-git` is gone entirely: `grep go-git go.mod` must return no matches. It was only ever reachable through the tool imports; no source file here imports a git library.

9. **Confirm `mocks/mocks.go` still carries its BSD license header.** `go generate` rewrites that file without one, and `addlicense` (the last `precommit` step) re-adds it — so if anything fails earlier in `precommit`, the file is left stripped. This actually happened on the sibling `git-sync` migration. After `make precommit` passes, verify:
   ```
   head -1 mocks/mocks.go
   ```
   It must print a `// Copyright (c) <year> Benjamin Borbe All rights reserved.` line. If it does not, run `make addlicense` and re-check.

   Do NOT extend this check to the other files in `mocks/` — `mocks/config_loader.go`, `mocks/notifier.go`, and `mocks/watcher.go` are counterfeiter output beginning with `// Code generated by counterfeiter. DO NOT EDIT.` and carry no copyright banner by design; `addlicense` deliberately skips them, and flagging them would be a false positive.
</requirements>

<constraints>
- Do NOT commit — dark-factory handles git.
- Do NOT run `go mod vendor`, and never use `-mod=vendor` in any command. This repo has no `vendor/` directory. If any generic dependency-fix guidance you encounter suggests a `go mod vendor` step, ignore it — this constraint takes precedence.
- Do NOT change any application code beyond the five `//go:generate` comment lines in requirement 1.
- Do NOT modify `tools.env` — no version changes, no reordering.
- Do NOT modify any tool invocation in the `Makefile` — every one is already correct.
- Do NOT edit `.osv-scanner.toml`. Some ignore entries will become unused once the pollution is gone; that is expected and is not an error.
- Existing tests must still pass unchanged; mock regeneration must produce no behavior change.
- If one of the six retained direct deps in requirement 3 turns out to be unused, let `go mod tidy` drop it rather than forcing it back in.
</constraints>

<verification>
Run `make precommit` — must pass (exit 0). Both `osv-scanner` and `trivy` must be clean; `trivy` in particular is what catches the requirement-6 regression if the `x/*` bumps were skipped.

Then confirm the end state:

```
grep -E '(cellbuf|go-header|go-diskfs|golangci-lint|osv-scanner|ginkgolinter|charmbracelet/x|denis-tingaikin)' go.mod
grep go-git go.mod
grep -E '^(replace|exclude)' go.mod
ls tools.go
```

The three greps must produce no output, and `ls tools.go` must report that the file does not exist. The `replace`/`exclude` grep enforces the `docs/dod.md` rule that neither directive may remain.

Confirm `go.mod` shrank substantially — `wc -l go.mod` should report well under 150 lines (it was 443).

Confirm the generator pin landed:

```
grep -rn 'counterfeiter/v6' --include='*_test.go' .
```

All five matches must read `go run github.com/maxbrunsfeld/counterfeiter/v6@v6.12.2 -generate`; none may still contain `-mod=mod`.

Confirm the license header survived, per requirement 9 — `head -1 mocks/mocks.go` must print a `// Copyright` line. Do not check the counterfeiter-generated files in `mocks/`; they legitimately have no copyright banner.
</verification>
