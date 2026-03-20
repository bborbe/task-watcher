# CLAUDE.md

Watches vault task files for phase/status changes and notifies configured agents via HTTP webhook.

## Dark Factory Workflow

**Never code directly.** All code changes go through the dark-factory pipeline.

### Complete Flow

**Spec-based (multi-prompt features):**

1. Create spec → `/dark-factory:create-spec`
2. Audit spec → `/dark-factory:audit-spec`
3. User confirms → `dark-factory spec approve <name>`
4. dark-factory auto-generates prompts from spec
5. Audit prompts → `/dark-factory:audit-prompt`
6. User confirms → `dark-factory prompt approve <name>`
7. Start daemon → `dark-factory daemon` (use Bash `run_in_background: true`)
8. dark-factory executes prompts automatically

**Standalone prompts (simple changes):**

1. Create prompt → `/dark-factory:create-prompt`
2. Audit prompt → `/dark-factory:audit-prompt`
3. User confirms → `dark-factory prompt approve <name>`
4. Start daemon → `dark-factory daemon` (use Bash `run_in_background: true`)
5. dark-factory executes prompt automatically

### Assess the change size

| Change | Action |
|--------|--------|
| Simple fix, config change, 1-2 files | Write a prompt → `/dark-factory:create-prompt` |
| Multi-prompt feature, unclear edges, shared interfaces | Write a spec first → `/dark-factory:create-spec` |

### Claude Code Commands

| Command | Purpose |
|---------|---------|
| `/dark-factory:create-spec` | Create a spec file interactively |
| `/dark-factory:create-prompt` | Create a prompt file from spec or task description |
| `/dark-factory:audit-spec` | Audit spec against preflight checklist |
| `/dark-factory:audit-prompt` | Audit prompt against Definition of Done |

### CLI Commands

| Command | Purpose |
|---------|---------|
| `dark-factory spec approve <name>` | Approve spec (inbox → queue, triggers prompt generation) |
| `dark-factory prompt approve <name>` | Approve prompt (inbox → queue) |
| `dark-factory daemon` | Start daemon (watches queue, executes prompts) |
| `dark-factory run` | One-shot mode (process all queued, then exit) |
| `dark-factory status` | Show combined status of prompts and specs |
| `dark-factory prompt list` | List all prompts with status |
| `dark-factory spec list` | List all specs with status |
| `dark-factory prompt retry` | Re-queue failed prompts for retry |

### Key rules

- Prompts go to **`prompts/`** (inbox) — never to `prompts/in-progress/` or `prompts/completed/`
- Specs go to **`specs/`** (inbox) — never to `specs/in-progress/` or `specs/completed/`
- Never number filenames — dark-factory assigns numbers on approve
- Never manually edit frontmatter status — use CLI commands above
- Always audit before approving (`/dark-factory:audit-prompt`, `/dark-factory:audit-spec`)
- **BLOCKING: Never run `dark-factory prompt approve`, `dark-factory spec approve`, or `dark-factory daemon` without explicit user confirmation.** Write the prompt/spec, then STOP and ask the user to approve. Do not assume approval from prior context or task momentum.
- **Before starting daemon** — run `dark-factory status` first to check if one is already running. Only start if not running.
- **Start daemon in background** — use Bash tool with `run_in_background: true` (not foreground, not detached with `&`)

## Development Standards

This project follows the [coding-guidelines](https://github.com/bborbe/coding-guidelines).

### Key Reference Guides

- **go-architecture-patterns.md** — Interface → Constructor → Struct → Method
- **go-testing-guide.md** — Ginkgo v2/Gomega testing
- **go-makefile-commands.md** — Build commands
- **git-commit-workflow.md** — Commit process with precommit checks
- **go-mocking-guide.md** — Counterfeiter mock generation

### Build and test

- `make precommit` — lint + format + generate + test + checks
- `make test` — tests only

## Architecture

- `main.go` — entry point, config loading, wires watcher + notifier
- `pkg/config/` — YAML config parsing (vault path, assignee, phases, webhook URL)
- `pkg/watcher/` — wraps `github.com/bborbe/vault-cli/pkg/ops.WatchOperation`, reads task frontmatter, filters by assignee/status/phase
- `pkg/notify/` — HTTP POST to configured webhook URL with deduplication

## Key Design Decisions

- **vault-cli as library** — import `github.com/bborbe/vault-cli/pkg/ops` directly, never shell out to `vault-cli` CLI
- **One instance per OpenClaw VM** — single assignee + single webhook per process, no multi-tenant config
- **Deduplication in-memory** — same `taskID+phase` combo only triggers once per process lifetime
- **No database** — stateless except for in-memory dedup map; restart clears dedup state (acceptable)
- **Factory functions are pure composition** — no conditionals, no I/O, no `context.Background()`
