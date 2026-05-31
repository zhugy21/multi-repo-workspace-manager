# Multi-Repo Workspace Manager — SPEC

## 1. 问题陈述

### 1.1 要解决的问题

后端和全栈开发者日常工作中需要维护 10 个以上的微服务仓库。典型的每日操作流程是：在多个仓库之间 `cd`、`git pull`、安装依赖、启动服务、检查状态——这些操作高度重复且完全手动。每次切换上下文都需要记忆每个仓库的路径、构建命令、健康检查方式，认知负担重，效率低下。

### 1.2 目标用户

- 维护 5–20 个微服务仓库的后端/全栈开发者
- 使用 Docker 容器化部署的团队
- 需要频繁在多个仓库之间同步、构建、检查状态的工程师

### 1.3 为什么值得做

- **减少重复操作**：将 N 次 `cd && git pull && build && check` 压缩为一条命令
- **降低认知负担**：工具记住每个仓库的配置，开发者只需关心要做什么，不需要记住怎么做
- **快速发现问题**：跨仓库的健康检查一览无余，问题定位从"挨个检查"变为"一眼看到"
- **团队一致性**：workspace.yaml 可提交到仓库共享，全团队使用一致的工作流程

---

## 2. 用户故事

| # | 故事 | 验收条件 |
|---|------|----------|
| 1 | **每日开工同步** — 作为开发者，早上开始工作前，我想用一条命令拉取所有微服务仓库的最新代码，并知道哪些仓库有未提交的改动 | `ws sync --all` 并发拉取所有仓库，显示每个仓库的结果和 dirty 警告，最后展示汇总 |
| 2 | **按组构建** — 作为后端开发者，我只想构建 backend 组的服务，不需要等待前端仓库的构建 | `ws build --group backend` 仅构建 backend 组仓库，其余仓库不参与 |
| 3 | **快速健康检查** — 作为值班 on-call 工程师，出问题时我想快速检查所有服务的健康状态，定位到具体出问题的服务 | `ws health --all --output json` 返回结构化结果，一眼定位失败服务 |
| 4 | **切换工作区** — 作为在多个项目间切换的开发者，我想一键切换到目标项目的工作区配置，避免每次手动指定 | `ws switch project-b && ws sync` 切换到项目 B 的工作区后执行同步 |
| 5 | **分支切换** — 作为需要在新功能分支上工作的开发者，我想把一组相关仓库批量切换到同一个 feature 分支 | `ws sync --checkout feature-x --group backend` 切换 backend 组所有仓库到 feature-x 分支，dirty 仓库被跳过并警告 |
| 6 | **实时监控** — 作为需要持续观察服务状态的开发者，我想进入一个实时更新的监控界面，看到所有仓库的状态变化 | `ws monitor` 进入 TUI 模式，按键刷新，显示所有仓库当前状态 |
| 7 | **遇错即停** — 作为在 CI 流水线中使用 ws 的工程师，当任一仓库失败时我希望立即中止，避免浪费构建时间 | `ws build --all --fail-fast` 任一仓库失败立即传播取消，其余仓库标记为 cancelled/skipped |
| 8 | **安全分支切换** — 作为开发者，批量切换分支时，我不想丢失未提交的工作 | `ws sync --checkout feature-x` 默认跳过 dirty 仓库，显示警告；需显式 `--force` 才能覆盖 |

---

## 3. 功能规约

### 3.1 核心操作

#### 3.1.1 `ws sync`

- **输入**：目标范围（`--all` / `--group` / `--repo`），可选 `--checkout <branch>`，可选 `--force`
- **行为**：
  - 不带 `--checkout`：对每个仓库执行 `git fetch origin && git merge --ff-only`，使用仓库的 `default_branch`（若配置）或保留当前分支
  - 带 `--checkout <branch>`：切换所有匹配仓库到目标分支；若仓库 dirty → 跳过并报告警告；若同时指定 `--force` → `git stash` 后切换并 `git pull`
  - 仓库不存在于本地且配置了 `url` → 自动 `git clone`
- **输出**：每个仓库的状态（success/failed/warning for dirty/cancelled/skipped）+ 耗时 + 详情
- **边界条件**：无网络连接 → 报告网络错误详情；空仓库（无 commit）→ 跳过并标记 warning；ssh key 问题 → 标记 failed 并提示
- **错误处理**：dirty 仓库在非 force 模式 → warning（不计入失败）；merge conflict → failed；网络/认证 → failed

#### 3.1.2 `ws build`

- **输入**：目标范围 + 通用控制 flag
- **行为**：对每个仓库执行 `build_command`（若未配置 `build_command` → skipped；v1 不内置 `type` 默认命令），超时使用 `build_timeout`（若配置）或全局超时
- **输出**：每个仓库的构建状态 + stdout/stderr 摘要 + 耗时
- **边界条件**：仓库未配置 `build_command` 且 `type` 无默认映射 → skipped；构建命令返回非零 → failed
- **错误处理**：超时 → `TimeoutError`；命令不存在 → `CommandError` with "command not found"

#### 3.1.3 `ws health`

- **输入**：目标范围 + 通用控制 flag
- **行为**：对每个仓库执行 `health_command`，根据退出码判断健康（0 = healthy, 非0 = unhealthy）
- **输出**：每个仓库健康状态 + 退出码 + 输出摘要
- **边界条件**：仓库未配置 `health_command` → skipped；命令超时 → unhealthy
- **错误处理**：超时 → unhealthy；命令不存在 → unhealthy with error detail

#### 3.1.4 `ws status`

- **输入**：目标范围 + 通用控制 flag
- **行为**：对每个仓库收集：当前分支、是否 dirty、最近一次 commit hash 和 message、与 remote 的 ahead/behind 状态
- **输出**：每个仓库的状态快照
- **边界条件**：仓库路径不存在 → 标记 MISSING；非 git 目录 → 标记 NOT_A_REPO
- **错误处理**：git 命令失败 → 在状态行显示错误信息，继续处理

#### 3.1.5 `ws monitor`

- **输入**：目标范围 + 通用控制 flag
- **行为**：进入 TUI 持续模式，首次展示 status 结果，用户按 `r` 刷新，按 `q` 退出
- **输出**：TUI 实时界面
- **退出码**：正常退出（`q` / `Ctrl+C`）→ `130`；配置错误 → `2`

### 3.2 管理命令

#### 3.2.1 `ws init`

- **输入**：可选 `--name <workspace_name>`
- **行为**：
  - 带 `--name`：一步生成模板 workspace.yaml
  - 不带 `--name`：进入交互式创建（询问名称、默认并发、默认超时等）
- **输出**：在当前目录生成 `workspace.yaml` 模板
- **边界条件**：文件已存在 → 提示覆盖确认

#### 3.2.2 `ws add`

- **输入**：`--name <repo>`，可选 `--path <dir>`，`--url <remote>`，`--group <group>`
- **行为**：在当前 workspace.yaml 中添加仓库条目
- **输出**：确认添加结果

#### 3.2.3 `ws remove`

- **输入**：`<name>`
- **行为**：从 workspace.yaml 中移除指定仓库
- **输出**：确认移除结果

#### 3.2.4 `ws list`

- **输入**：可选 `--format table|json`
- **行为**：列出 `~/.ws/workspaces/` 下所有工作区及其仓库数量、最后使用时间
- **输出**：表格或 JSON

#### 3.2.5 `ws switch`

- **输入**：`<workspace_name>`
- **行为**：更新 `~/.ws/config.yaml` 中的 `active` 字段
- **输出**：确认切换结果

#### 3.2.6 `ws config`

- **输入**：`--show` 或 `--edit`
- **行为**：`--show` 输出当前生效配置；`--edit` 用 `$EDITOR` 打开文件
- **输出**：配置内容或编辑器

### 3.3 全局 flag

| Flag | 类型 | 默认 | 说明 |
|------|------|------|------|
| `--workspace <name>` | string | — | 临时指定工作区（根命令层） |
| `--all` | bool | `true` | 对所有仓库执行 |
| `--group` | string | — | 逗号分隔组名 |
| `--repo` | string | — | 逗号分隔仓库名 |
| `--concurrency` | int | `config.default_concurrency` | 并发数 |
| `--timeout` | duration | `config.default_timeout` | 全局超时 |
| `--fail-fast` | bool | — | 遇错即停 |
| `--continue-on-error` | bool | — | 遇错继续（默认） |
| `--output` | string | `config.output_format` | `tui` / `plain` / `json` |

- `--fail-fast` 和 `--continue-on-error` 互斥
- `--group` 和 `--repo` 同时指定时，以 `--repo` 为准并输出警告
- `--workspace` 临时覆盖 `ws switch` 设置的工作区

### 3.4 sync 专属 flag

| Flag | 说明 |
|------|------|
| `--checkout <branch>` | 切换匹配仓库到目标分支 |
| `--force` | 配合 `--checkout`，强制切换（stash dirty 改动） |

### 3.5 退出码

| 码 | 含义 |
|------|------|
| `0` | `--continue-on-error` 下全部成功；`--fail-fast` 下全部成功；所有仓库 skipped（无实际操作） |
| `1` | `--continue-on-error` 下部分 failed；`--fail-fast` 下因失败中止 |
| `2` | 配置错误（yaml 解析失败、无效仓库引用、无效 flag 组合等） |
| `130` | 用户中断（`SIGINT` / `Ctrl+C`）；monitor 模式正常退出（`q` / `Ctrl+C`） |

- `skipped` 不计入失败数，计入 total；不影响退出码
- `cancelled` 由 fail-fast 触发，不计入失败数；退出码由 failed 仓库决定

### 3.6 并发执行引擎规约

```
输入: []Task, Concurrency=N, FailFast=bool, GlobalTimeout=D

1. 创建带 cancel 的 context
2. 信号量 (chan struct{}, cap N) 控制并发
3. for each task in sequence:
     if ctx 已取消 → 标记 skipped, continue
     获取信号量（阻塞直到有空位）
     go run(task, ctx):
       defer 释放信号量
       timeout = task.Timeout ?? GlobalTimeout    // 任务自身超时优先
       taskCtx, _ = context.WithTimeout(ctx, timeout)
       执行 task.Command（通过 os/exec）
       写入 Result 到 output channel
       if Result.Failed && FailFast:
         cancel()           // 触发 ctx.Done()，所有 goroutine 在下一轮检查中退出
4. 所有 goroutine 结束后 close(output channel)
```

#### Fail-fast 取消传播时序

```
Task-1  [========== ✗ FAIL ==========]
Task-2  [===== ctx.Done() → ⏹ cancelled =====]
Task-3  [======== ctx.Done() → ⏹ cancelled ===]
Task-4  [                                ◌ skipped]
Task-5  [                                ◌ skipped]
        ↑ cancel() called here
```

### 3.7 配置优先级链

```
per-repo field  >  CLI flag  >  config.<global>  >  内置默认值
```

#### 超时优先级链

```
task.build_timeout / task.health_timeout  >  --timeout flag  >  config.default_timeout  >  120s 内置默认
```

---

## 4. 非功能性需求

### 4.1 性能

- 10 个仓库 sync（并发 4）应在 30 秒内完成（假设网络正常）
- 100 个仓库配置解析应在 10ms 内完成
- TUI 刷新率不低于 10fps
- 内存占用：空闲 < 20MB，运行中（10 仓库并发）< 50MB

### 4.2 安全

- 不持久化任何 git 凭据
- 通过 SSH agent forwarding 或系统 git credential helper 认证
- `.env` 文件路径仅存储引用，不读取内容写入日志
- 不通过网络发送仓库元数据或状态信息

### 4.3 可用性

- 单二进制分发，无运行时依赖
- 支持 Linux、macOS、WSL2
- 启动时间 < 100ms（不含命令执行）
- 提供 `--help` 完整文档和用法示例

### 4.4 可观测性

- 所有外部命令执行记录 stdout/stderr 摘要
- json 模式输出完整的结构化结果
- 退出码语义清晰，可被 CI 管道消费
- 错误信息包含足够上下文：仓库名、命令、退出码、stderr 摘要

---

## 5. 系统架构

### 5.1 组件图

```
┌──────────────────────────────────────────────────────────┐
│                     CLI Layer (Cobra)                     │
│                                                          │
│  ┌────────┬────────┬────────┬────────┬────────┐         │
│  │  sync  │ build  │ health │ status │monitor │         │
│  └───┬────┴───┬────┴───┬────┴───┬────┴───┬────┘         │
│      │        │        │        │        │               │
│  ┌───┴────────┴────────┴────────┴────────┴───┐          │
│  │          管理命令 (init/add/remove/...)      │          │
│  └───────────────────────────────────────────┘          │
├──────────────────────────────────────────────────────────┤
│                  TUI Layer (Bubble Tea)                   │
│                                                          │
│  ┌───────────┐  ┌───────────┐  ┌──────────────────┐    │
│  │  Model    │  │  Table    │  │  Messages/Events  │    │
│  └─────┬─────┘  └───────────┘  └──────────────────┘    │
│        │ ← Result channel                                │
├────────┼─────────────────────────────────────────────────┤
│        │          Core Engine                            │
│  ┌─────┴──────┐  ┌──────────────┐  ┌─────────────────┐ │
│  │  Executor  │  │  Workspace   │  │     Runner      │ │
│  │            │  │              │  │                 │ │
│  │ - 并发调度  │  │ - 配置解析    │  │ - Runner 接口   │ │
│  │ - 取消传播  │  │ - 过滤查询    │  │ - SyncRunner    │ │
│  │ - 超时管理  │  │ - 仓库迭代    │  │ - BuildRunner   │ │
│  │ - 结果汇总  │  │ - 验证       │  │ - HealthRunner  │ │
│  └────────────┘  └──────────────┘  └─────────────────┘ │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Shared Types & Errors                │   │
│  └──────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

### 5.2 数据流

```
CLI flags → workspace.Config.Filter() → []types.Repo
                                                    ↓
                    runner.Prepare(ctx, config, filter) → []executor.Task
                                                                    ↓
                                            executor.Run(ctx, tasks) → <-chan Result
                                                                            ↓
                                          ┌──────────────┬───────────────┐
                                          ↓              ↓               ↓
                                     TUI Model      plain stdout     JSON encoder
```

### 5.3 运行时依赖

- `git`：命令行工具（sync 操作）
- 无其他运行时依赖
- 无数据库、无守护进程

### 5.4 文件系统结构

```
~/.ws/
  config.yaml               # active: /path/to/current/workspace.yaml
  workspaces/
    myproject.yaml          # symlink → 实际的 workspace.yaml
    other-project.yaml

<项目目录>/
  workspace.yaml            # 工作区配置文件
  auth-service/             # 各仓库本地路径
  user-service/
  web-app/
```

---

## 6. 数据模型

### 6.1 核心实体

```yaml
# Workspace — 顶层工作区配置
workspace:
  workspace_name: string          # 可选，仅用于识别
  description: string             # 可选
  config: Config                  # 全局配置
  groups: map[string][]string     # 组名 → 仓库名列表
  repos: []Repo                   # 仓库列表

# Config — 全局配置
Config:
  default_concurrency: int        # 默认 4
  default_timeout: duration       # 默认 120s
  fail_fast: bool                 # 默认 false
  output_format: enum             # tui | plain | json，默认 tui

# Repo — 单个仓库
Repo:
  name: string                    # 必填，唯一标识
  path: string                    # 必填，本地路径（相对或绝对）
  url: string                     # 可选，远程 Git URL（存在时支持自动 clone）
  type: enum                      # 可选，go | node | python | rust（v1 仅标识，不用于默认命令）
  group: string                   # 可选，所属组名
  default_branch: string          # 可选，sync 默认分支
  build_command: string           # 可选，构建命令
  build_timeout: duration         # 可选，构建超时
  health_command: string          # 可选，健康检查命令
  health_timeout: duration        # 可选，健康检查超时
  env_file: string                # V2 保留，v1 解析但不执行
  docker_compose_file: string     # V2 保留
  sync_command: string            # V2 保留
  smart_build: bool               # V2 保留

# Task — 执行引擎输入（内部类型，非用户可见）
Task:
  id: string
  repo: Repo
  command: []string
  timeout: duration | nil
  env_files: []string

# Result — 执行引擎输出（内部类型）
Result:
  task_id: string
  repo_name: string
  status: enum                    # pending | running | success | failed | cancelled | skipped | warning
  detail: string
  error: error | nil
  duration: duration
  exit_code: int
```

### 6.2 状态机

```
pending ──→ running ──→ success
  │           │  ├──→ failed
  │           │  └──→ warning (dirty 但继续)
  │           └──→ cancelled (fail-fast 触发)
  └──→ skipped (因前序失败取消)
```

### 6.3 关系

- `Repo` N:0..1 `Group`（一个仓库最多属于一个组）
- `Group` 1:N `Repo`（一个组包含多个仓库）
- `Config` 1:1 `Workspace`（每个工作区一个全局配置）
- `Task` 1:1 `Result`（每个任务产生一个结果）
- `Repo.command` 覆盖 `Config.default_*`，遵循优先级链

---

## 7. API 设计

### 7.1 CLI 作为 API

本工具 CLI 本身就是 API。所有功能通过命令行调用。

### 7.2 命令语法

```
ws [--workspace <name>] <command> [flags]

核心操作:
  ws sync     [--all|--group x|--repo y] [--checkout <branch>] [--force] [通用控制]
  ws build    [--all|--group x|--repo y] [通用控制]
  ws health   [--all|--group x|--repo y] [通用控制]
  ws status   [--all|--group x|--repo y] [通用控制]
  ws monitor  [--all|--group x|--repo y] [通用控制]

管理命令:
  ws init     [--name <workspace>]
  ws add      --name <repo> [--path <dir>] [--url <remote>] [--group <group>]
  ws remove   <name>
  ws list     [--format table|json]
  ws switch   <workspace_name>
  ws config   --show|--edit
```

### 7.3 输出结构

#### TUI 模式（默认）

实时表格 + 底部消息栏 + 汇总条（参见 8.1 TUI 布局）

#### plain 模式

```
[auth-service]    ✅ OK       Already up to date            2.3s
[user-service]    ✅ OK       Fast-forward merge main        1.1s
[web-app]         ✗ FAILED    Merge conflict: src/App.tsx    3.2s
[admin-panel]     ⏹ CANCELLED Fail-fast triggered            —
[shared-lib]      ◌ SKIPPED   Skipped (earlier failure)      —
[docs-site]       ⚠ WARN      Dirty: 2 uncommitted files    0.1s
---
6 repos: 2 passed, 1 failed, 1 cancelled, 1 skipped, 1 warning
```

状态标签缩写对照：`✅ OK` / `✗ FAILED` / `⏹ CANCELLED` / `◌ SKIPPED` / `⚠ WARN` / `⏳ ...`

#### json 模式

```json
{
  "workspace": "myproject",
  "command": "sync",
  "timestamp": "2026-05-31T10:30:00Z",
  "results": [
    {
      "repo": "auth-service",
      "group": "backend",
      "status": "success",
      "detail": "Already up to date",
      "duration_ms": 2300,
      "exit_code": 0
    },
    {
      "repo": "web-app",
      "group": "frontend",
      "status": "failed",
      "detail": "Merge conflict: src/App.tsx",
      "duration_ms": 3200,
      "exit_code": 1
    }
  ],
  "summary": {
    "total": 3,
    "success": 2,
    "failed": 1,
    "cancelled": 0,
    "skipped": 0,
    "warning": 0
  }
}
```

### 7.4 退出码（重复）

| 码 | 含义 |
|------|------|
| `0` | 全部成功，或全部 skipped（无实际操作） |
| `1` | 部分 failed，或 fail-fast 中止 |
| `2` | 配置错误 |
| `130` | 用户中断；monitor 正常退出 |

---

## 8. TUI 设计

### 8.1 布局

```
┌─ ws sync ─── workspace: myproject ─── 4/10 completed ─── 00:12 elapsed ─┐
│                                                                         │
│  ┌──────────────┬────────┬──────────┬──────────┬─────────────────────┐ │
│  │ Repo         │ Group  │ Status   │ Time     │ Detail              │ │
│  ├──────────────┼────────┼──────────┼──────────┼─────────────────────┤ │
│  │ auth-service │backend │ ✅ OK    │ 2.3s     │ Already up to date  │ │
│  │ user-service │backend │ ✅ OK    │ 1.1s     │ Fast-forward merge  │ │
│  │ payment-svc  │backend │ ⏳ Running│ 0.8s     │ Pulling...          │ │
│  │ web-app      │frontend│ ✗ FAILED │ 3.2s     │ Merge conflict: src │ │
│  │ admin-panel  │frontend│ ⏹ CANCELLED│ —     │ Fail-fast triggered │ │
│  │ shared-lib   │  —    │ ◌ SKIPPED│ —        │ Skipped (earlier …) │ │
│  └──────────────┴────────┴──────────┴──────────┴─────────────────────┘ │
│                                                                         │
│  ─────────────────────────────────────────────────────────────────────  │
│  Fail-fast triggered by payment-svc: merge conflict in main.go         │
│  Results: 2 passed │ 1 failed │ 1 cancelled │ 1 skipped │ 6 total     │
└─────────────────────────────────────────────────────────────────────────┘
```

- Detail 列默认截断至终端宽度的 40%
- 选中仓库按 `Enter` 可弹窗查看完整输出/日志

### 8.2 交互键

| 键 | 行为 | 适用模式 |
|------|------|----------|
| `q` / `Esc` / `Ctrl+C` | 退出 | 所有 TUI |
| `r` | 重新执行上一次命令 | monitor |
| `↑ ↓` | 滚动仓库列表 | monitor |
| `Enter` | 选中仓库 → 显示详细日志弹窗 | 所有 TUI |
| `tab` | 切换视图（进度 / 日志 / 汇总） | monitor |

- 普通操作（sync/build/health/status）：命令完成后自动退出，不等待按键
- monitor 模式：持续运行，等待用户操作

### 8.3 状态图标与颜色

| 状态 | 图标 | 颜色 |
|------|------|------|
| pending | `○` | 灰色 |
| running | `⏳` | 黄色 |
| success | `✅` | 绿色 |
| failed | `✗` | 红色 |
| cancelled | `⏹` | 灰色 |
| skipped | `◌` | 灰色 |
| warning | `⚠` | 黄色 |

---

## 9. 技术选型与理由

| 组件 | 选择 | 理由 |
|------|------|------|
| **语言** | Go 1.22+ | 编译为单二进制，无运行时依赖；goroutine 天然适合并发执行模型 |
| **CLI 框架** | Cobra | 行业标准，POSIX 风格 flag，子命令组合灵活 |
| **配置** | Viper | Cobra 生态标配，YAML 解析 + 环境变量覆盖 |
| **TUI** | Bubble Tea | Elm Architecture，测试友好，生态活跃（Bubble 组件库） |
| **并发模型** | goroutine + channel + context | 标准库即可满足需求，无第三方依赖 |
| **版本控制** | go-git（评估） | v1 使用 `os/exec` 调用 git CLI；若性能或跨平台问题突出，v2 可切换 go-git 纯 Go 实现 |
| **测试** | `testing` + `testify` | 标准库为主，testify 用于断言和 mock |
| **分发** | `go build` 单二进制 + Homebrew / Go install | 零依赖分发 |

### 不使用

- **数据库**：不需要。workspace.yaml 是唯一数据源
- **网络服务**：不需要。纯本地 CLI
- **容器运行时**：v1 不直接操作 Docker，仅执行用户定义的命令

---

## 10. 包目录树

```
cmd/ws/
  main.go

internal/
  cli/
    root.go                     # 根命令 + 全局 flag
    sync.go                     # ws sync
    build.go                    # ws build
    health.go                   # ws health
    status.go                   # ws status
    monitor.go                  # ws monitor
    init.go                     # ws init
    add.go                      # ws add
    remove.go                   # ws remove
    list.go                     # ws list
    switch.go                   # ws switch
    config_cmd.go               # ws config

  executor/
    executor.go                 # 并发执行引擎
    executor_test.go            # 单元测试
    task.go                     # Task 类型定义
    result.go                   # Result 类型定义 + 汇总函数

  workspace/
    config.go                   # workspace.yaml 解析 + 验证
    config_test.go              # 单元测试
    types.go                    # Repo, Group, Config 结构体
    filter.go                   # 按 group/repo/all 过滤
    testdata/
      valid.yaml                # 测试夹具
      missing_repos.yaml
      invalid_group.yaml

  runner/
    runner.go                   # Runner 接口
    sync.go                     # SyncRunner
    sync_test.go                # 单元测试
    build.go                    # BuildRunner
    build_test.go               # 单元测试
    health.go                   # HealthRunner
    health_test.go              # 单元测试

  tui/
    model.go                    # Bubble Tea Model
    model_test.go               # TUI 同步测试
    table.go                    # 实时状态表组件
    styles.go                   # 主题/颜色定义
    messages.go                 # 事件消息类型

pkg/
  types/
    repo.go                     # 共享类型：Repo, Task, Result, Status
    errors.go                   # 错误类型：ConfigError, RepoError, ...

test/
  e2e/
    init_test.go                # 端到端：ws init 生成模板正确性
    sync_test.go                # 端到端：ws sync 流程
    build_test.go               # 端到端：ws build 流程
    health_test.go              # 端到端：ws health 流程
```

---

## 11. 错误处理

### 11.1 错误类型

| 错误 | 触发条件 | 退出码 |
|------|------|------|
| `ConfigError` | yaml 解析失败、必填字段缺失、无效组引用 | `2` |
| `RepoError` | 仓库不存在、路径不可达、git 操作失败 | `1` |
| `CommandError` | build/health 命令执行失败（非零退出码） | `1` |
| `TimeoutError` | 任务超时 | `1` |
| `CancelError` | fail-fast 触发的取消 | `1`（由首次失败决定） |
| `WorkspaceError` | workspace 切换/查找失败 | `2` |

所有错误包装原始 error：`fmt.Errorf("repo %s: build failed: %w", name, err)`

### 11.2 错误展示

| 模式 | 展示方式 |
|------|------|
| `tui` | 行内状态 + 底部消息栏首错详情；Enter 弹窗完整 stderr |
| `plain` | `[repo] ✗ FAILED <error message>` 逐行 + 汇总 |
| `json` | `"detail"` 字段包含错误信息 |

---

## 12. 测试策略

### 12.1 分层

| 层级 | 测试内容 | 工具 |
|------|------|------|
| **单元** | `executor` 并发调度、取消传播、超时处理 | `go test` + mock Task |
| **单元** | `workspace` 配置解析、过滤逻辑 | `go test` + 夹具 yaml |
| **单元** | `runner` 各 Runner 的 Prepare 逻辑 | `go test` + mock 文件系统 |
| **单元** | `tui` Model 的状态转换与 View 输出 | `tea.NewProgram()` 同步测试模式，模拟输入并断言 View 输出 |
| **集成** | 真实 git 操作（`git init` 临时仓库中执行 sync） | `go test` + `t.TempDir()` |
| **端到端** | 完整 CLI 调用（构建 cobra command，执行并捕获输出） | `go test` |

### 12.2 覆盖率目标

- executor + workspace + runner：≥ 80%
- cli 命令构建：≥ 70%
- tui 渲染逻辑：≥ 60%

---

## 13. v1 功能边界

| 类别 | v1 包含 | V2 延后 |
|------|------|------|
| **sync** | fetch + merge (--ff-only), checkout, dirty 检测 | branch-map, sync_command（完整同步） |
| **build** | 执行自定义 build_command | 增量构建、Docker 集成、smart_build |
| **health** | 自定义命令 + 退出码 | HTTP 端点探测、容器状态检查 |
| **配置** | 解析 workspace.yaml、字段校验、过滤 | env_file 注入、docker_compose_file 功能 |
| **monitor** | TUI 持续模式 + 手动刷新 (`r`) | 定时轮询、告警 |
| **分支** | checkout 单分支、dirty 检测、force | branch-map 批量分支映射 |

---

## 14. 验收标准

| # | 标准 | 判定方式 |
|---|------|----------|
| 1 | `ws sync --all` 并发拉取 10 个仓库，全部成功返回 0 | 自动化测试 |
| 2 | `ws sync --all` 其中 1 个仓库 merge conflict → 其余仓库正常完成，该仓库标记 failed，退出码 1 | 自动化测试 |
| 3 | `ws sync --all --fail-fast` 任一仓库失败 → 其余 running 任务收到 cancel，未开始任务 skipped | 自动化测试 |
| 4 | `ws build --group backend` 仅构建 backend 组，其余仓库不参与 | 自动化测试 |
| 5 | `ws health --output json` 输出合法 JSON，包含所有仓库健康状态 | 自动化测试 |
| 6 | `ws sync --checkout feature-x` dirty 仓库被跳过并标记 warning（非 force 模式） | 自动化测试 |
| 7 | `ws sync --checkout feature-x --force` dirty 仓库 stash 后切换成功 | 自动化测试 |
| 8 | `ws switch other-project` 后 `ws sync` 使用 other-project 的 workspace.yaml | 自动化测试 |
| 9 | `ws init --name test` 生成合法 workspace.yaml 模板 | 自动化测试 |
| 10 | 无效 workspace.yaml（缺失必填字段）→ 退出码 2，stderr 含具体错误信息 | 自动化测试 |
| 11 | `--group` 和 `--repo` 同时指定 → 以 `--repo` 为准，stderr 输出警告 | 自动化测试 |
| 12 | 单二进制可运行于 Linux、macOS、WSL2 | 手动验证 |
| 13 | TUI 表格正确显示所有状态（success/failed/cancelled/skipped/warning），颜色和图标与规约一致 | 自动化 TUI 测试 |

---

## 15. 风险与未决问题

### 15.1 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| **os/exec 调用 git 的跨平台兼容性** | Windows 上 git 路径、shell 行为差异 | v1 优先支持 Linux/macOS/WSL2；`exec.Command` 不依赖 shell |
| **Bubble Tea 大表格性能** | 100+ 仓库时 TUI 渲染可能卡顿 | v1 目标为 10–30 仓库；使用视口虚拟化（`viewport` 组件） |
| **并发 exec 的资源消耗** | 高并发下 fork 大量子进程可能触及 ulimit | 默认并发 4，最大值由用户显式设置；文档注明风险 |
| **workspace.yaml 向后兼容** | v2 新字段可能导致旧版工具解析失败 | 使用 Viper 的宽松解析，未知字段忽略而非报错 |
| **git 操作失败的状态恢复** | pull --ff-only 失败后仓库可能处于不一致状态 | 仅执行非破坏性操作（ff-only merge）；checkout 前检测 dirty |

### 15.2 未决问题

1. **go-git vs os/exec**：v1 使用 os/exec 调用 git CLI。需确认目标平台是否都预装 git。v2 可评估纯 Go 实现以减少依赖
2. **monitor 模式的定时轮询**：v1 只支持手动 `r` 刷新。定时轮询（如每 30 秒）推迟到 v2
3. **workspace.yaml 的共享与版本控制**：团队可将 workspace.yaml 提交到仓库。是否需要在 `ws init` 时提示这一最佳实践？
4. **命令注入风险**：build_command 和 health_command 是用户配置的任意命令。工具不对命令内容做任何转义或校验——这是预期行为还是需要文档警告？
5. **SSH agent 依赖**：sync 依赖 SSH agent 或 credential helper。若用户未配置，git 操作会失败。是否需要在首次运行 `ws sync` 前检测 SSH 可达性？
