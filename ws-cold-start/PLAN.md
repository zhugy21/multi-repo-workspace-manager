# Multi-Repo Workspace Manager (ws) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI tool (`ws`) that manages multiple microservice repositories from a single workspace config, supporting concurrent sync/build/health/status/monitor operations with fail-fast support and a Bubble Tea TUI dashboard.

**Architecture:** Three layers — CLI (Cobra) → Runner (sync/build/health logic) → Executor (concurrent engine with context cancellation). Workspace config parsed from YAML by Viper. Shared types in `pkg/types`. No database.

**Tech Stack:** Go 1.22+, Cobra, Viper, Bubble Tea + Bubbles + Lipgloss, `os/exec`, `testing` + `testify`.

**Spec:** `docs/SPEC.md`

---

## Dependency Graph

```
Phase A — Foundation (sequential)
─────────────────────────────────
  T1: go mod init + directory scaffold
   │
  T2: pkg/types (errors, Repo, Task, Result, Status, Summarize)
   │
  T3: internal/workspace (config parse, validate, filter)
   │
  ┌─────────────┬──────────────────┐
  ▼             ▼                  ▼
Phase B — Engine + Runner (T4 ∥ T5+T6 parallel)
───────────────────────────────────────────────
  T4: executor  T5: runner interface  T6: build+health
     engine          + SyncRunner          runners
     │               │                    │
     └───────┬───────┴────────────────────┘
             ▼
Phase C — CLI Core (T7 first, then T8∥T9∥T10∥T11 parallel)
───────────────────────────────────────────────────────────
  T7: root command + global flags + common helpers
   │
  ┌────────┬────────┬────────┬──────────┐
  ▼        ▼        ▼        ▼          ▼
 T8:     T9:     T10:     T11:      T12:
 sync    build   health   status    management
 cmd      cmd     cmd      cmd       commands

Phase D — TUI (parallel with Phase C)
─────────────────────────────────────
  T13: Bubble Tea Model + Table + Styles

Phase E — Integration (depends on Phase C + Phase D)
────────────────────────────────────────────────────
  T14: main.go wire-up + exit codes
   │
  T15: E2E tests (init, sync, build, health)
```

**Key:** `∥` = tasks can run in parallel (no shared state dependencies)

---

### Task 1: Go module scaffold + directory tree

**Depends on:** nothing
**Parallelizable with:** nothing (must be first)
**Goal:** Initialize Go module and create all empty directory structure.

**Files:**
- Create: `go.mod` (via `go mod init`)
- Create: `cmd/ws/main.go` (minimal placeholder)

**Implementation points:**
- Module path: `github.com/user/ws`
- All directories from SPEC section 10: `internal/{cli,executor,workspace,runner,tui}`, `pkg/types`, `internal/workspace/testdata`, `test/e2e`
- Minimal `main.go` that prints "ws" and exits 0

**Verification:**
- [ ] `go build ./cmd/ws/ && ./ws` → prints "ws", exits 0
- [ ] `find . -type d | sort` → all 10 directories present
- [ ] `cat go.mod` → module path correct, Go version 1.22+

---

### Task 2: Shared types — errors, Repo, Task, Result, Status, Summary

**Depends on:** T1 (need `go.mod`)
**Parallelizable with:** nothing (types used by everything else)
**Goal:** Implement all shared domain types and error types in `pkg/types/`.

**Files:**
- Create: `pkg/types/repo.go`
- Create: `pkg/types/errors.go`
- Create: `pkg/types/repo_test.go`

**Implementation points:**

1. **Status** — `type Status string` with consts: `pending`, `running`, `success`, `failed`, `cancelled`, `skipped`, `warning`
2. **Repo** — struct with YAML+JSON tags: `name`(required), `path`(required), `url`, `type`, `group`, `default_branch`, `build_command`, `build_timeout`, `health_command`, `health_timeout`, `env_file`, `docker_compose_file`, `sync_command`, `smart_build`
3. **Task** — struct: `ID string`, `RepoName string`, `Group string`, `Command []string`, `Timeout time.Duration` (0 = use global), `EnvFiles []string`
4. **Result** — struct: `TaskID`, `RepoName`, `Group`, `Status`, `Detail`, `Error` (unexported), `ErrorStr`, `Duration`, `ExitCode int`
5. **Summary** — struct: `Total`, `Success`, `Failed`, `Cancelled`, `Skipped`, `Warning int` — plus method `HasFailures() bool`
6. **Filter** — struct: `All bool`, `Group string`, `Repo string`
7. **Summarize(results []Result) Summary** — iterate results, count by status
8. **Errors** — 6 error types: `ConfigError`, `RepoError`, `CommandError`, `TimeoutError`, `CancelError`, `WorkspaceError`. Each with `Error()` using `fmt.Errorf` wrapping.

**Verification:**
- [ ] Write test `TestRepoJSONRoundTrip` — marshal/unmarshal Repo with all fields
- [ ] Write test `TestResultSummary` — 5 results with mixed statuses, verify counts
- [ ] Write test `TestSummarizeHasFailures` — verify `HasFailures()` returns true when Failed>0
- [ ] Write test `TestErrorTypes` — create each error type, verify `Error()` string contains key info
- [ ] `go test ./pkg/types/... -v` → all 4 tests pass
- [ ] `go build ./...` → clean compile

---

### Task 3: Workspace config — parse, validate, filter

**Depends on:** T2 (needs `pkg/types`)
**Parallelizable with:** nothing (executor and runners depend on this)
**Goal:** Implement workspace.yaml parsing with Viper, validation, and repo filtering.

**Files:**
- Create: `internal/workspace/types.go`
- Create: `internal/workspace/config.go`
- Create: `internal/workspace/filter.go`
- Create: `internal/workspace/config_test.go`
- Create: `internal/workspace/testdata/valid.yaml`
- Create: `internal/workspace/testdata/invalid_missing_name.yaml`
- Create: `internal/workspace/testdata/invalid_group_ref.yaml`

**Implementation points:**

1. **Workspace struct** — `WorkspaceName string`, `Description string`, `Config Config`, `Groups map[string][]string`, `Repos []types.Repo`
2. **Config struct** — `DefaultConcurrency int`, `DefaultTimeout time.Duration`, `FailFast bool`, `OutputFormat string`
3. **ActiveConfig struct** — `Active string` (for `~/.ws/config.yaml`)
4. **Parse(path string) (*Workspace, error)** — use Viper, unmarshal YAML, call validate(), call applyDefaults()
5. **validate()** — check: name required, path required, no duplicate names, group references must match `groups` entries
6. **applyDefaults(basePath string)** — concurrency→4, timeout→120s, output→tui
7. **Filter(f types.Filter) []types.Repo** — `--repo` takes precedence over `--group`; if both specified, emit warning to stderr and use repo
8. **ParseActiveConfig(path string) (*ActiveConfig, error)** — parse `~/.ws/config.yaml`

**Verification:**
- [ ] Write test `TestParseValidConfig` — valid.yaml with 2 repos, 1 group → verify all fields parsed correctly including time.Duration fields
- [ ] Write test `TestParseMissingName` — yaml with repo missing name → returns `*ConfigError`
- [ ] Write test `TestParseInvalidGroupRef` — repo references nonexistent group → returns error containing "group"
- [ ] Write test `TestParseNonexistentFile` → returns error
- [ ] Write test `TestParseDefaults` — yaml without concurrency/timeout/format → defaults applied (4, 120s, tui)
- [ ] Write test `TestFilterByRepo` — Filter with Repo="svc-a" → returns only svc-a
- [ ] Write test `TestFilterByGroup` — Filter with Group="backend" → returns repos in backend group
- [ ] Write test `TestFilterByAll` — Filter with All=true → returns all repos
- [ ] Write test `TestFilterRepoOverGroup` — both Repo and Group set → Repo wins, warning to stderr
- [ ] `go test ./internal/workspace/... -v` → all 9 tests pass

---

### Task 4: Executor — concurrent engine with fail-fast

**Depends on:** T2 (needs `pkg/types`)
**Parallelizable with:** T5 (runner interface, different package)
**Goal:** Implement the concurrent task execution engine with semaphore-based throttling, context cancellation, timeout per task, and fail-fast propagation.

**Files:**
- Create: `internal/executor/executor.go`
- Create: `internal/executor/executor_test.go`

**Implementation points:**

1. **Executor struct** — `Concurrency int`, `FailFast bool`
2. **Run(ctx context.Context, tasks []types.Task) []types.Result** — follow SPEC 3.6 pseudocode exactly:
   - Create `ctx, cancel = context.WithCancel(ctx)`; defer cancel()
   - Semaphore: `make(chan struct{}, Concurrency)`
   - Pre-allocate `[]Result` of len(tasks)
   - For each task: if ctx cancelled → mark `skipped`; else acquire sem → goroutine
   - Each goroutine: defer release sem; create taskCtx with timeout (task.Timeout or 2min default); `exec.CommandContext`; capture stdout/stderr; if error → if FailFast → cancel(); write Result to index
   - WaitGroup.Wait(); return results
3. **Timeout resolution** — `task.Timeout ?? GlobalTimeout(2min)`
4. **Fail-fast** — first failure triggers `cancel()` which causes all goroutines that check `ctx.Done()` to stop
5. **Result status mapping** — exit 0 → success; timeout → failed with "timeout" detail; ctx.Done() → cancelled; non-zero exit → failed with stderr detail

**Verification:**
- [ ] Write test `TestExecutorAllSuccess` — 2 tasks echo "ok", concurrency 2 → both success
- [ ] Write test `TestExecutorFailFast` — 3 tasks (slow-ok, fast-fail, slow-ok), concurrency 2, failFast=true → failed task exists AND at least one task is cancelled/skipped
- [ ] Write test `TestExecutorTimeout` — 1 task `sleep 10` with 100ms timeout → status failed, detail contains "timeout"
- [ ] Write test `TestExecutorContinueOnError` — 3 tasks (ok, fail, ok), concurrency 2, failFast=false → 3 total, 2 success, 1 failed
- [ ] Write test `TestExecutorAllSkippedIfContextCancelled` — pass pre-cancelled context → all tasks skipped
- [ ] `go test ./internal/executor/... -v -timeout 30s` → all 5 tests pass

---

### Task 5: Runner interface + SyncRunner

**Depends on:** T2 (types), T3 (workspace)
**Parallelizable with:** T4 (executor, different package), T6 (build/health runners share runner.go but different files)
**Goal:** Define the Runner interface and implement SyncRunner with git operations.

**Files:**
- Create: `internal/runner/runner.go`
- Create: `internal/runner/sync.go`
- Create: `internal/runner/sync_test.go`

**Implementation points:**

1. **Runner interface** — `Name() string`, `Prepare(ctx context.Context, ws *workspace.Workspace, filter types.Filter) ([]types.Task, error)`
2. **SyncRunner struct** — `Checkout string`, `Force bool`
3. **Prepare logic** (without checkout):
   - For each repo: if path doesn't exist and URL set → Task: `git clone <url> <path>`
   - If path exists → two Tasks: `git fetch origin` then `git merge --ff-only`
   - Use `default_branch` from repo config if set
4. **Prepare logic** (with checkout):
   - Run `git status --porcelain` to check dirty
   - If dirty and not Force → Task with Status=warning, skip
   - If dirty and Force → `git stash && git checkout <branch>`
   - If clean → `git checkout <branch>`
5. **isRepoDirty(path string) (bool, error)** — helper running `git status --porcelain`

**Verification:**
- [ ] Write test `TestSyncRunnerPrepare` — workspace with 1 repo (git init'd in TempDir) → 2 tasks (fetch + merge)
- [ ] Write test `TestSyncRunnerPrepareCheckout` — Checkout="feature-x" → task command is `git checkout feature-x`
- [ ] Write test `TestSyncRunnerPrepareDirtySkip` — dirty repo, checkout, Force=false → 1 warning task, no checkout
- [ ] Write test `TestSyncRunnerPrepareClone` — repo path missing, URL set → task command is `git clone`
- [ ] Write test `TestSyncRunnerPrepareNoBuildCommandIsEmpty` — repo without build_command → 0 tasks
- [ ] `go test ./internal/runner/... -v` → all 5 tests pass

---

### Task 6: BuildRunner + HealthRunner

**Depends on:** T2 (types), T3 (workspace), T5 (runner.go — must exist)
**Parallelizable with:** T4 (executor, different package)
**Goal:** Implement BuildRunner and HealthRunner — thin wrappers that create Tasks from user-configured commands.

**Files:**
- Create: `internal/runner/build.go`
- Create: `internal/runner/health.go`
- Create: `internal/runner/build_test.go`
- Create: `internal/runner/health_test.go`

**Implementation points:**

1. **BuildRunner** — struct{}, `Name() "build"`, Prepare iterates repos: if `BuildCommand == ""` → skip; else create Task with `strings.Fields(BuildCommand)` as Command, `BuildTimeout` as Timeout
2. **HealthRunner** — struct{}, `Name() "health"`, Prepare iterates repos: if `HealthCommand == ""` → skip; else create Task with `strings.Fields(HealthCommand)` as Command, `HealthTimeout` as Timeout
3. Both set `Task.RepoName` and `Task.Group` from repo struct
4. Neither does v1 default-command-by-type (per SPEC: v1 不内置 type 默认命令)

**Verification:**
- [ ] Write test `TestBuildRunnerPrepareWithCommand` — repo with build_command → 1 task with parsed command
- [ ] Write test `TestBuildRunnerPrepareWithoutCommand` — repo without build_command → 0 tasks
- [ ] Write test `TestBuildRunnerTimeout` — repo with build_timeout=5s → task.Timeout=5s
- [ ] Write test `TestHealthRunnerPrepareWithCommand` — repo with health_command → 1 task
- [ ] Write test `TestHealthRunnerPrepareWithoutCommand` — repo without health_command → 0 tasks
- [ ] Write test `TestHealthRunnerTimeout` — repo with health_timeout=10s → task.Timeout=10s
- [ ] `go test ./internal/runner/... -v` → all 11 tests pass (including T5's 5 tests)

---

### Task 7: CLI root command + global flags + common helpers

**Depends on:** T2 (types), T3 (workspace)
**Parallelizable with:** T4, T5, T6, T13 (different packages)
**Goal:** Implement the Cobra root command with all global flags, workspace resolution, result display, and exit code mapping. Create command stubs that T8–T12 will flesh out.

**Files:**
- Create: `internal/cli/root.go`
- Create: `internal/cli/common.go`
- Create: `internal/cli/root_test.go`
- Create: `internal/cli/sync.go` (stub)
- Create: `internal/cli/build.go` (stub)
- Create: `internal/cli/health.go` (stub)
- Create: `internal/cli/status.go` (stub)
- Create: `internal/cli/monitor.go` (stub)
- Create: `internal/cli/init.go` (stub)
- Create: `internal/cli/add.go` (stub)
- Create: `internal/cli/remove.go` (stub)
- Create: `internal/cli/list.go` (stub)
- Create: `internal/cli/switch.go` (stub)
- Create: `internal/cli/config_cmd.go` (stub)

**Implementation points:**

1. **Global flag variables** — `workspaceFlag`, `allFlag`, `groupFlag`, `repoFlag`, `concurrency`, `timeoutFlag`, `failFast`, `continueOnErr`, `outputFlag`, `formatFlag`, `checkoutFlag`, `forceFlag`, `initName`
2. **NewRootCommand() *cobra.Command** — `Use: "ws"`, add all subcommand stubs, register PersistentFlags for all global flags
3. **resolveWorkspacePath() (string, error)** — check `--workspace` flag → check `./workspace.yaml` → check `~/.ws/config.yaml` active pointer → fallback to `./workspace.yaml`
4. **resolveFilter() types.Filter** — repo takes precedence over group (warn to stderr if both set), all=true default
5. **resolveFailFast() bool, resolveConcurrency() int, resolveTimeout() time.Duration**
6. **displayResults([]Result, format, command)** — dispatch to `displayPlain` / `displayJSON`
7. **displayPlain** — one line per result: `[repo-name] ICON STATUS  detail  duration`; then `---\nN repos: X passed, Y failed...`
8. **displayJSON** — `{workspace, command, timestamp, results[], summary{}}`
9. **exitCodeFromResults** — return error if `summary.Failed > 0`; main.go maps error type to exit code
10. **Each stub** — `func newXxxCmd() *cobra.Command { return &cobra.Command{Use: "xxx", Short: "...", RunE: func(...){return nil}} }`

**Verification:**
- [ ] Write test `TestRootCommandHasAllSubcommands` — verify names: sync, build, health, status, monitor, init, add, remove, list, switch, config
- [ ] Write test `TestRootCommandHasGlobalFlags` — verify workspace, all, group, repo, concurrency, timeout, fail-fast, continue-on-error, output flags exist
- [ ] Write test `TestResolveFilterRepoOverGroup` — both set → Filter.Repo wins, warning on stderr
- [ ] Write test `TestResolveFilterAll` — neither set → Filter.All=true
- [ ] Write test `TestResolveWorkspacePathFromFlag` — `--workspace /tmp/test.yaml` with file existing → returns path
- [ ] Write test `TestDisplayPlainOutput` — capture stdout, verify format matches SPEC: `[repo] ICON STATUS detail duration`
- [ ] Write test `TestDisplayJSONOutput` — capture stdout, verify valid JSON with results[] and summary{}
- [ ] `go test ./internal/cli/... -v` → all 7 tests pass
- [ ] `go build ./cmd/ws/ && ./ws --help` → shows all subcommands

---

### Task 8: CLI sync command

**Depends on:** T7 (root + common), T5 (SyncRunner)
**Parallelizable with:** T9, T10, T11, T12 (all independent CLI commands)
**Goal:** Implement the `ws sync` command wiring Cobra flags → SyncRunner → Executor → display.

**Files:**
- Modify: `internal/cli/sync.go` (replace stub)
- Create: `internal/cli/sync_test.go`

**Implementation points:**
- Add `--checkout` and `--force` flags to sync subcommand
- `runSync`: resolve workspace → parse config → create SyncRunner{Checkout, Force} → Prepare tasks → create Executor with concurrency/failFast from flags or config → Run → displayResults → exitCode
- Edge case: if no repos match filter → 0 tasks, display "0 repos" summary, exit 0

**Verification:**
- [ ] Write test `TestSyncCommandFlags` — verify --checkout and --force flags registered
- [ ] Write test `TestSyncNoWorkspace` — run sync in empty dir → returns error
- [ ] Write test `TestSyncIntegration` — create temp workspace.yaml with 1 git-initialized repo, run sync → captures output with success status
- [ ] `go test ./internal/cli/... -v -run Sync` → tests pass

---

### Task 9: CLI build command

**Depends on:** T7 (root + common), T6 (BuildRunner)
**Parallelizable with:** T8, T10, T11, T12
**Goal:** Implement the `ws build` command.

**Files:**
- Modify: `internal/cli/build.go` (replace stub)
- Create: `internal/cli/build_test.go`

**Implementation points:**
- `runBuild`: resolve workspace → parse → BuildRunner.Prepare → Executor.Run → display → exitCode
- No build-specific flags in v1; uses only global flags
- If no repos have build_command → all skipped, exit 0

**Verification:**
- [ ] Write test `TestBuildCommandFlagInheritance` — verify global flags (concurrency, fail-fast) accessible from build command
- [ ] Write test `TestBuildNoBuildCommand` — repo without build_command → skipped in output
- [ ] `go test ./internal/cli/... -v -run Build` → tests pass

---

### Task 10: CLI health command

**Depends on:** T7 (root + common), T6 (HealthRunner)
**Parallelizable with:** T8, T9, T11, T12
**Goal:** Implement the `ws health` command.

**Files:**
- Modify: `internal/cli/health.go` (replace stub)
- Create: `internal/cli/health_test.go`

**Implementation points:**
- `runHealth`: resolve workspace → parse → HealthRunner.Prepare → Executor.Run → display → exitCode
- No health-specific flags in v1
- If no repos have health_command → all skipped, exit 0

**Verification:**
- [ ] Write test `TestHealthCommandJSONOutput` — `--output json` → valid JSON with results array
- [ ] Write test `TestHealthNoHealthCommand` — repo without health_command → skipped
- [ ] `go test ./internal/cli/... -v -run Health` → tests pass

---

### Task 11: CLI status command

**Depends on:** T7 (root + common), T3 (workspace filter)
**Parallelizable with:** T8, T9, T10, T12
**Goal:** Implement the `ws status` command that collects git metadata per repo.

**Files:**
- Modify: `internal/cli/status.go` (replace stub)
- Create: `internal/cli/status_test.go`

**Implementation points:**
- `runStatus`: resolve workspace → parse → Filter repos → for each repo create a Task running a shell script: `cd <path>; echo "branch:$(git branch --show-current)"; echo "dirty:$(...); echo "last_commit:$(git log -1 --format='%h %s')"`
- Handle missing path → Task with command `echo MISSING`
- Handle non-git dir → Task with command `echo NOT_A_REPO`
- `statusScript(path)` helper generates the inline shell command

**Verification:**
- [ ] Write test `TestStatusCommandGitRepo` — temp git repo → output contains branch, dirty, last_commit
- [ ] Write test `TestStatusCommandMissingPath` — nonexistent path → output contains "MISSING"
- [ ] Write test `TestStatusCommandFilteredByGroup` — only repos in specified group in output
- [ ] `go test ./internal/cli/... -v -run Status` → tests pass

---

### Task 12: CLI management commands

**Depends on:** T7 (root + common), T3 (workspace)
**Parallelizable with:** T8, T9, T10, T11
**Goal:** Implement all 6 management subcommands: init, add, remove, list, switch, config.

**Files:**
- Modify: `internal/cli/init.go` (replace stub)
- Modify: `internal/cli/add.go` (replace stub)
- Modify: `internal/cli/remove.go` (replace stub)
- Modify: `internal/cli/list.go` (replace stub)
- Modify: `internal/cli/switch.go` (replace stub)
- Modify: `internal/cli/config_cmd.go` (replace stub)
- Create: `internal/cli/management_test.go`

**Implementation points:**

1. **init** — `--name` flag (optional). With name: write template workspace.yaml. Without: prompt "Workspace name:", read stdin. Template has workspace_name, config defaults, empty groups/repos. If file exists → prompt overwrite confirmation.
2. **add** — `--name`(required), `--path`, `--url`, `--group`. Read existing workspace.yaml, unmarshal, append Repo, marshal back, write. Use `yaml.v3` for round-trip.
3. **remove** — `<name>` positional arg. Read workspace.yaml, filter out repo by name, write back. Error if name not found.
4. **list** — `--format table|json`. Read `~/.ws/workspaces/` directory. For each `.yaml` file: count repos, get modtime. Print table or JSON.
5. **switch** — `<workspace_name>` arg. Read symlink at `~/.ws/workspaces/<name>.yaml` to resolve actual path. Write `active: <resolved_path>` to `~/.ws/config.yaml`. Create directories if needed.
6. **config** — `--show` (default) prints workspace.yaml content. `--edit` opens `$EDITOR` (fallback: vim) on workspace.yaml.

**Verification:**
- [ ] Write test `TestInitWithName` — `ws init --name test` → workspace.yaml created with workspace_name: test
- [ ] Write test `TestInitWithoutNameInteractive` — simulate stdin "test2\n" → workspace.yaml created
- [ ] Write test `TestInitOverwritePrompt` — existing file, respond "n" → aborted error
- [ ] Write test `TestAddRepo` — add repo to workspace.yaml → file contains new repo entry
- [ ] Write test `TestRemoveRepo` — remove existing repo → file no longer contains it
- [ ] Write test `TestRemoveRepoNotFound` → error
- [ ] Write test `TestSwitchWorkspace` — create symlink + run switch → `~/.ws/config.yaml` has correct active path
- [ ] `go test ./internal/cli/... -v -run 'Init|Add|Remove|List|Switch|Config'` → all pass

---

### Task 13: TUI — Bubble Tea model, table, styles

**Depends on:** T2 (types)
**Parallelizable with:** T7–T12 (different package, no code dep)
**Goal:** Implement the Bubble Tea TUI model with real-time results table, detail view popup, and interactive keybindings.

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/table.go` (or inline in model.go)
- Create: `internal/tui/styles.go`
- Create: `internal/tui/messages.go`
- Create: `internal/tui/model_test.go`

**Implementation points:**

1. **Model struct** — `table table.Model`, `viewport viewport.Model`, `results []types.Result`, `command string`, `workspace string`, `startTime time.Time`, `failReason string`, `mode string` ("auto" | "monitor"), `width/height int`, `detailView bool`, `detailIdx int`
2. **NewModel(command, workspace, mode) Model** — initialize table with 5 columns (Repo, Group, Status, Time, Detail), set styles per SPEC 8.1
3. **Init() tea.Cmd** — no initial command; results arrive via channel externally
4. **Update(msg tea.Msg)** — handle: `WindowSizeMsg` (resize), `KeyMsg` (q/esc/c-c → quit; r → refresh in monitor; enter → toggle detail; ↑↓ → scroll in monitor; tab → future v2), `TickMsg` (external result update)
5. **TickMsg** — `{Results []Result, Complete bool}`. In auto mode: if Complete → Quit
6. **updateRows()** — convert `[]Result` to `[]table.Row` with icons/colors from styles.go, truncate detail to 40 chars
7. **View() string** — header (command + workspace + progress + elapsed) + table + footer (summary) + help. In detail mode: viewport with full logs
8. **Styles** — lipgloss styles matching SPEC 8.3: TitleStyle (bold white on purple), StatusSuccessStyle (green), StatusFailedStyle (red), StatusWarningStyle (yellow), StatusGrayStyle (gray), FooterStyle, HelpStyle
9. **StatusIcon/StatusColor helpers** — map Status string → icon/color

**Verification:**
- [ ] Write test `TestTUIModelInit` — create model, verify initial state (0 results, mode correct)
- [ ] Write test `TestTUIUpdateWithResults` — send TickMsg with 3 results → model.results has 3 entries
- [ ] Write test `TestTUIViewRendersHeader` — View() contains command name and workspace
- [ ] Write test `TestTUIViewRendersSummary` — View() with failed results contains "failed" in footer
- [ ] Write test `TestTUIQuitOnComplete` — TickMsg{Complete: true} in auto mode → returns tea.Quit
- [ ] Write test `TestTUIQuitOnQ` — KeyMsg "q" → returns tea.Quit
- [ ] Write test `TestTUIStatusIcons` — each status maps to correct icon per SPEC 8.3
- [ ] `go test ./internal/tui/... -v` → all 7 tests pass

---

### Task 14: main.go — wire-up, exit codes, TUI integration

**Depends on:** T7–T13 (all CLI commands exist, TUI exists)
**Parallelizable with:** nothing (integration point)
**Goal:** Wire the root command to main, implement proper exit code mapping per SPEC 7.4, and integrate TUI for non-plain/json output modes.

**Files:**
- Modify: `cmd/ws/main.go` (replace placeholder)

**Implementation points:**

1. **main()** — call `cli.NewRootCommand().Execute()`
2. **Exit code mapping** — if error: check `*types.ConfigError` or `*types.WorkspaceError` → `os.Exit(2)`; other errors → `os.Exit(1)`; no error → `os.Exit(0)`; SIGINT → `os.Exit(130)` (handled by Go runtime, but document)
3. **TUI integration decision** — v1: operations default to `--output tui` but actual TUI integration (feeding results channel to Bubble Tea program) is deferred to a follow-up. v1 uses `--output plain` as default. Document this in code comments.
4. **Explicit note** — add a comment in main.go: "TUI mode activated via `--output tui`. In v1, --output plain is the effective default if no TTY detected."

**Verification:**
- [ ] `go build -o ws ./cmd/ws/` → clean build, single binary
- [ ] `./ws --help` → shows help text, exit 0
- [ ] `./ws sync` (no workspace.yaml) → exits 2 (config error)
- [ ] `./ws init --name test && ./ws sync` (no git repos) → exits 0 (0 repos, no failures)
- [ ] `./ws --workspace /nonexistent sync` → exits 2
- [ ] `./ws nonexistent-command` → Cobra error, exits 1

---

### Task 15: E2E tests

**Depends on:** T14 (main.go complete)
**Parallelizable with:** nothing (final verification)
**Goal:** Write end-to-end tests covering the SPEC acceptance criteria.

**Files:**
- Create: `test/e2e/init_test.go`
- Create: `test/e2e/sync_test.go`
- Create: `test/e2e/build_test.go`
- Create: `test/e2e/health_test.go`

**Implementation points:**

1. **init_test.go** — Test init creates valid YAML with all expected fields; Test init overwrite confirm; Test init with name flag
2. **sync_test.go** — Test sync with 2 git repos (init'd in TempDir), verify both succeed; Test sync with merge conflict (create conflicting changes), verify 1 failed, exit 1; Test sync fail-fast: 2 repos, 1 will fail → verify cancelled; Test sync --checkout: dirty repo skipped with warning; Test sync --checkout --force: dirty repo stashed and switched
3. **build_test.go** — Test build with custom build_command; Test build repo without build_command skipped; Test build --group filter
4. **health_test.go** — Test health with custom command; Test health JSON output valid; Test health repo without command skipped

Each E2E test pattern:
```go
func TestXxx(t *testing.T) {
    dir := t.TempDir()
    // Set up workspace.yaml + git repos
    cmd := exec.Command("go", "run", "../../cmd/ws", "sync", "--output", "plain")
    cmd.Dir = dir
    out, err := cmd.CombinedOutput()
    // Assert output contains expected strings
    // Assert exit code
}
```

**Verification:**
- [ ] `go test ./test/e2e/... -v -count=1` → all E2E tests pass
- [ ] Each spec acceptance criterion (SPEC section 14, items 1–11, 13) covered by at least one E2E test
- [ ] Final `go build -o ws ./cmd/ws/` → binary works

---

## Parallel Execution Strategy

For subagent-driven development with worktree isolation:

**Wave 1** (1 agent): T1 → commit
**Wave 2** (1 agent): T2 (types) → commit
**Wave 3** (1 agent): T3 (workspace config) → commit
**Wave 4** (3 agents parallel): T4 (executor) ∥ T5 (runner interface+sync) ∥ T6 (build+health) → each in own worktree, merge sequentially
**Wave 5** (1 agent): T7 (root CLI + common) → commit
**Wave 6** (5 agents parallel): T8 ∥ T9 ∥ T10 ∥ T11 ∥ T12 (all CLI commands) ∥ T13 (TUI) → each in own worktree
**Wave 7** (1 agent): T14 (main.go wire-up) → commit
**Wave 8** (1 agent): T15 (E2E tests) → commit
