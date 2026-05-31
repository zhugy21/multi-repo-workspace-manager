# CLAUDE.md — Multi-Repo Workspace Manager (ws)

## Project Overview

`ws` is a Go CLI tool that manages multiple microservice repositories from a single YAML workspace config. One command to sync, build, and health-check all your repos — concurrently, with fail-fast support, and a Bubble Tea TUI dashboard.

**Repository:** `github.com/zhugy21/multi-repo-workspace-manager`

**Key features:**
- `ws sync` — git fetch + merge (--ff-only) across all repos, with branch checkout support
- `ws build` — run user-configured build commands concurrently
- `ws health` — run user-configured health check commands, exit-code-based
- `ws status` — git metadata snapshot per repo (branch, dirty, last commit)
- `ws monitor` — TUI continuous monitoring mode
- `ws init/add/remove/list/switch/config` — workspace management commands

## Environment Setup

```bash
# Go toolchain (installed at /usr/local/go)
export PATH="/usr/local/go/bin:$PATH"
export GOROOT="/usr/local/go"

# HTTP proxy (required for GitHub access from this environment)
export http_proxy=127.0.0.1:7890
export https_proxy=127.0.0.1:7890

# Go module proxy (if sum.golang.org is unreachable)
export GOPROXY=direct

# Verify
which go && go version  # go1.26+
```

## Build & Run

```bash
# Build
go build -o ws ./cmd/ws/

# Run
./ws --help
./ws init --name myproject          # create workspace.yaml
./ws sync --all --output plain      # sync all repos
./ws build --group backend          # build a specific group
./ws health --output json           # health check as JSON
./ws status                         # repo status snapshot
```

## Architecture

```
cmd/ws/main.go           # Entry point, exit code mapping
─────────────────────────────────────────────
internal/cli/            # Cobra commands (thin layer)
  root.go                # Root command + 10 global flags
  sync.go / build.go / health.go / status.go / monitor.go
  init.go / add.go / remove.go / list.go / switch.go / config_cmd.go
  common.go              # resolveWorkspacePath, displayPlain/JSON, exitCode
─────────────────────────────────────────────
internal/tui/            # Bubble Tea TUI (model, table, styles)
─────────────────────────────────────────────
internal/runner/         # Runner interface + implementations
  runner.go              # Runner interface (Name, Prepare)
  sync.go                # SyncRunner — git clone/fetch/merge/checkout
  build.go               # BuildRunner — exec user-configured build_command
  health.go              # HealthRunner — exec user-configured health_command
─────────────────────────────────────────────
internal/executor/       # Concurrent execution engine
  executor.go            # Semaphore + context cancellation + fail-fast
─────────────────────────────────────────────
internal/workspace/      # Config parsing (workspace.yaml)
  config.go              # Parse + validate + defaults
  filter.go              # Filter repos by name/group/all
─────────────────────────────────────────────
pkg/types/               # Shared domain types
  repo.go                # Status, Repo, Task, Result, Summary, Filter
  errors.go              # ConfigError, RepoError, CommandError, ...
─────────────────────────────────────────────
test/e2e/                # End-to-end tests (11 tests)
```

### Data Flow

```
workspace.yaml → workspace.Parse() → []Repo
    → ws.Filter(--group/--repo/--all) → []Repo
    → runner.Prepare(repos) → []Task
    → executor.Run(tasks) → <-chan Result
    → displayPlain / displayJSON / TUI Model
```

## Directory Structure

```
docs/
  SPEC.md              # 15-chapter specification
  PLAN.md              # 15-task implementation plan with dependency graph
  SPEC_PROCESS.md      # Collaboration process record
cmd/ws/main.go
internal/{cli,executor,workspace,runner,tui}/
pkg/types/
test/e2e/
```

## Testing

```bash
# All tests (7 packages, 81 tests)
go test ./... -count=1

# Individual packages
go test ./pkg/types/... -v           # 14 tests — domain types + errors
go test ./internal/workspace/... -v  # 9 tests — config parsing + filter
go test ./internal/executor/... -v   # 5 tests — concurrency + fail-fast
go test ./internal/runner/... -v     # 13 tests — sync/build/health runners
go test ./internal/cli/... -v        # 22 tests — CLI commands
go test ./internal/tui/... -v        # 7 tests — Bubble Tea model
go test ./test/e2e/... -v            # 11 tests — end-to-end
```

## Development Workflow (Superpowers)

### Process (MANDATORY)

1. **Git worktrees** — each independent feature/module gets its own worktree → one PR per worktree
2. **Subagent-driven development** — each task dispatched to a fresh subagent with isolated context
3. **TDD (red→green→refactor)** — write failing test FIRST, verify red, implement minimal code, verify green, refactor. NO implementation code before tests.
4. **Two-stage review per task** — spec compliance check first → code quality check second. Critical issues MUST be fixed before next task.
5. **Finishing-a-development-branch** — after all tasks: decide merge / PR / keep / discard

### Key Skills (invoke via Skill tool)

| Skill | When |
|-------|------|
| `superpowers:brainstorming` | Before any creative work — design before implementation |
| `superpowers:writing-plans` | After spec approved — create implementation plan |
| `superpowers:subagent-driven-development` | Execute plan task-by-task with review gates |
| `superpowers:test-driven-development` | Each feature/bugfix — red→green→refactor |
| `superpowers:requesting-code-review` | After completing tasks, before merging |
| `superpowers:systematic-debugging` | Any bug, test failure, or unexpected behavior |
| `superpowers:finishing-a-development-branch` | All tasks complete — decide merge/PR/keep/discard |

### Git Conventions

- Commit messages: `feat:` / `fix:` / `refactor:` / `test:` prefix
- Commit messages / PR descriptions must annotate: which subagent completed the task, what was manually modified
- PLAN.md updated after each task completion with commit hash
- AGENT_LOG.md maintained as the process evidence log

## GitHub Requirements

- Public repository
- Complete commit history + PR workflow (no single-commit delivery)
- Each worktree → one PR
- Commit/PR annotations: subagent attribution, manual modifications

## Containerization (V2)

- Provide Dockerfile; multi-service projects need docker-compose.yml
- Single `docker build` + single `docker run` for build and launch
- README: run commands, ports, environment variables
- Push to Docker Hub or GHCR

## Deliverables Checklist

- [x] SPEC.md, PLAN.md, SPEC_PROCESS.md
- [x] Complete source code (40 files, ~3300 lines Go)
- [ ] Dockerfile + docker-compose.yml (V2)
- [ ] README.md
- [ ] AGENT_LOG.md
- [ ] CI config (.github/workflows/)
- [ ] REFLECTION.md
