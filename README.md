# ws — Multi-Repo Workspace Manager

一条命令管理本地所有微服务仓库的 sync / build / health check。

## 项目简介

`ws` 是一个 Go CLI 工具，通过一个 `workspace.yaml` 配置文件管理多个微服务仓库。支持并发执行、fail-fast 容错、Bubble Tea TUI 仪表盘。适合维护 10+ 个微服务仓库的后端和全栈开发者。

### 核心命令

| 命令 | 功能 |
|------|------|
| `ws sync` | Git 同步（fetch + merge --ff-only）+ 分支切换 |
| `ws build` | 执行用户配置的构建命令 |
| `ws health` | 执行用户配置的健康检查命令 |
| `ws status` | 仓库状态快照（分支、dirty、最近 commit） |
| `ws monitor` | TUI 持续监控模式 |

### 管理命令

| 命令 | 功能 |
|------|------|
| `ws init [--name]` | 初始化 workspace.yaml |
| `ws add --name --path [--url] [--group]` | 添加仓库 |
| `ws remove <name>` | 移除仓库 |
| `ws list [--format]` | 列出所有工作区 |
| `ws switch <name>` | 切换默认工作区 |
| `ws config --show\|--edit` | 查看/编辑配置 |

---

## 安装

### Go Install

```bash
go install github.com/zhugy21/multi-repo-workspace-manager/cmd/ws@latest
```

### 从源码构建

```bash
git clone git@github.com:zhugy21/multi-repo-workspace-manager.git
cd multi-repo-workspace-manager
go build -o ws ./cmd/ws/
./ws --help
```

**前置依赖：** Go 1.24+、Git

---

## Docker

### 公开镜像

```
ghcr.io/zhugy21/multi-repo-workspace-manager:latest
```

### 拉取

```bash
docker pull ghcr.io/zhugy21/multi-repo-workspace-manager:latest
```

### 运行

```bash
# 同步所有仓库
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/zhugy21/multi-repo-workspace-manager:latest sync --all

# 构建 backend 组
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/zhugy21/multi-repo-workspace-manager:latest build --group backend

# 健康检查（JSON 输出）
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/zhugy21/multi-repo-workspace-manager:latest health --output json

# 查看状态
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/zhugy21/multi-repo-workspace-manager:latest status

# TUI 监控模式
docker run --rm -it -v $(pwd):/workspace -w /workspace \
  ghcr.io/zhugy21/multi-repo-workspace-manager:latest monitor
```

### 构建本地镜像

```bash
docker build \
  --build-arg http_proxy=$http_proxy \
  --build-arg https_proxy=$https_proxy \
  -t ws:latest .
```

### 容器说明

| 属性 | 值 |
|------|-----|
| **基础镜像** | alpine:3.21 |
| **内置工具** | git（sync 操作需要） |
| **入口点** | `ws`（容器即命令） |
| **端口** | 无（纯 CLI 工具，不监听端口） |
| **环境变量** | `http_proxy`, `https_proxy`（构建时可选） |

---

## 快速开始

### 1. 初始化工作区

```bash
ws init --name myproject
```

生成 `workspace.yaml`：

```yaml
workspace_name: myproject
description: ""
config:
  default_concurrency: 4
  default_timeout: 120s
  fail_fast: false
  output_format: tui
groups: {}
repos: []
```

### 2. 添加仓库

```bash
ws add --name auth-service \
  --path ./auth-service \
  --url git@github.com:org/auth-service.git \
  --group backend
```

### 3. 配置构建和健康检查

编辑 `workspace.yaml`：

```yaml
repos:
  - name: auth-service
    path: ./auth-service
    url: git@github.com:org/auth-service.git
    type: go
    group: backend
    default_branch: main
    build_command: "make build"
    build_timeout: 300s
    health_command: "curl -sf http://localhost:8080/health"
    health_timeout: 30s
```

### 4. 日常使用

```bash
ws sync --all                    # 拉取所有仓库最新代码
ws build --group backend         # 构建 backend 组
ws health --all --output json    # 健康检查
ws status                        # 查看所有仓库状态
```

---

## 全局 Flag

所有核心操作共用的 flag：

| Flag | 类型 | 默认 | 说明 |
|------|------|------|------|
| `--workspace` | string | — | 临时指定工作区 |
| `--all` | bool | `true` | 对所有仓库执行 |
| `--group` | string | — | 逗号分隔组名 |
| `--repo` | string | — | 逗号分隔仓库名 |
| `--concurrency` | int | `4` | 并发数 |
| `--timeout` | duration | `120s` | 全局超时 |
| `--fail-fast` | bool | — | 遇错即停 |
| `--continue-on-error` | bool | `true` | 遇错继续（默认） |
| `--output` | string | `tui` | `tui` / `plain` / `json` |

### sync 专属

| Flag | 说明 |
|------|------|
| `--checkout <branch>` | 切换所有匹配仓库到目标分支 |
| `--force` | 配合 `--checkout`，强制切换（stash dirty 改动） |

### 退出码

| 码 | 含义 |
|------|------|
| `0` | 全部成功 |
| `1` | 部分失败或 fail-fast 中止 |
| `2` | 配置错误 |
| `130` | 用户中断 |

---

## 目录结构

```
.
├── cmd/ws/main.go                  # 入口点，退出码映射
├── internal/
│   ├── cli/                        # Cobra 命令层（11 个子命令）
│   │   ├── root.go                 # 根命令 + 全局 flag
│   │   ├── common.go               # 通用 helper（resolve/display/exitCode）
│   │   ├── sync.go / build.go / health.go / status.go / monitor.go
│   │   └── init.go / add.go / remove.go / list.go / switch.go / config_cmd.go
│   ├── tui/                        # Bubble Tea TUI
│   │   ├── model.go                # TUI Model（Elm Architecture）
│   │   ├── styles.go               # Lipgloss 主题/颜色
│   │   └── messages.go             # 事件消息类型
│   ├── runner/                     # Runner 接口 + 实现
│   │   ├── runner.go               # Runner 接口
│   │   ├── sync.go                 # SyncRunner（git clone/fetch/merge/checkout）
│   │   ├── build.go                # BuildRunner（执行用户构建命令）
│   │   └── health.go               # HealthRunner（执行用户健康检查命令）
│   ├── executor/                   # 并发执行引擎
│   │   └── executor.go             # 信号量 + Context 取消 + fail-fast
│   └── workspace/                  # 配置解析
│       ├── config.go               # Parse + validate + applyDefaults
│       └── filter.go               # 按 name/group/all 过滤
├── pkg/types/                      # 共享领域类型
│   ├── repo.go                     # Status/Repo/Task/Result/Summary/Filter
│   └── errors.go                   # 6 种错误类型
├── test/e2e/                       # 端到端测试（11 个）
├── docs/
│   ├── SPEC.md                     # 15 章功能规约
│   ├── PLAN.md                     # 15 步实现计划 + 依赖图
│   ├── SPEC_PROCESS.md             # Superpowers 协作过程记录
│   └── AGENT_LOG.md                # Subagent 协作日志
├── Dockerfile                      # 多阶段构建（golang → alpine + git）
├── workspace.yaml                  # 示例工作区配置
└── go.mod
```

---

## 开发

### 运行测试

```bash
# 全部测试（7 包，81 测试）
go test ./... -count=1

# 单独包
go test ./pkg/types/... -v           # 14 tests — 类型 + 错误
go test ./internal/workspace/... -v  # 9 tests — 配置解析
go test ./internal/executor/... -v   # 5 tests — 并发引擎
go test ./internal/runner/... -v     # 13 tests — sync/build/health
go test ./internal/cli/... -v        # 22 tests — CLI 命令
go test ./internal/tui/... -v        # 7 tests — TUI
go test ./test/e2e/... -v            # 11 tests — 端到端
```

### 开发流程

本项目使用 [Superpowers](https://github.com/anthropics/claude-code) 技能驱动开发：

1. **Brainstorming** → SPEC.md（设计规约）
2. **Writing Plans** → PLAN.md（实现计划）
3. **Subagent-Driven Development** → TDD + 两阶段 Review
4. **Finishing-a-development-branch** → Merge/PR/保留
