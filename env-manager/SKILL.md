# env-manager

> OpenClaw 技能：多节点环境管理服务
> 让 AI Agent 拥有集群感知能力 —— 扫描、预检、分配，部署前就知道能不能做。

## 概述

`env-manager` 是一个轻量化的多节点环境管理技能，专为 AI Agent 设计。它不替代 Prometheus/Grafana，而是做 Agent 的"眼睛和手"——让 LLM 在部署服务前能看到集群状态、做出判断、执行操作。

**核心能力：**
1. **裸机服务与端口管理** — 扫描裸机上运行的服务和占用的端口
2. **容器端口与服务管理** — 扫描 Docker 容器的端口映射和运行状态
3. **裸机资源管理** — 部署前预检：CPU/内存/磁盘/GPU 是否足够、端口是否冲突

## 为什么需要它

在多节点开发/生产环境中（如 `server-1` + `web-node` + `gpu-cluster`），经常遇到：
- "我要在 web-node 上跑个新服务，但不知道 8080 端口被谁占了"
- "gpu-cluster 的 GPU 还剩多少显存？能不能再跑一个推理服务？"
- "准备部署 Redis，先帮我看看 server-1 的资源够不够"

传统监控工具（Prometheus/Netdata）回答的是"过去发生了什么"。`env-manager` 回答的是"现在能不能做这件事"。

## 差异化定位

| | Prometheus + Grafana | Netdata | env-manager (本技能) |
|---|---|---|---|
| 定位 | 生产级监控 | 实时可视化 | AI Agent 的环境感知层 |
| 部署复杂度 | 高（多组件） | 中（每节点装 Agent） | 低（单二进制，SSH 远程采集） |
| 数据消费者 | 人（看图表） | 人（看图表） | LLM（结构化 JSON） |
| 核心能力 | 时序存储 + 告警规则 | 实时展示 | 预检 + 决策支持 + 端口分配 |
| 每节点需装 Agent | 是 | 是 | **否**（SSH 远程执行命令） |
| 与 LLM 集成 | 需额外适配 | 无 | 原生工具接口 |

## 技术架构

```
┌─────────────────────────────────────────────────────┐
│                   OpenClaw Runtime                   │
│                                                      │
│  ┌───────────────────────────────────────────────┐ │
│  │         env-manager (Skill Process)             │ │
│  │                                                │ │
│  │  ┌─────────┐  ┌──────────┐  ┌─────────────┐   │ │
│  │  │ Scanner │  │ Registry │  │  Allocator   │   │ │
│  │  │ Module  │  │  Module  │  │   Module     │   │ │
│  │  └────┬────┘  └────┬─────┘  └──────┬──────┘   │ │
│  │       │            │               │            │ │
│  │  ┌────▼────────────▼───────────────▼──────┐    │ │
│  │  │          Node Connector                 │    │ │
│  │  │    (SSH Pool / Local Executor)         │    │ │
│  │  └────┬─────────┬──────────┬───────┬────┘    │ │
│  └───────┼─────────┼──────────┼───────┼─────────┘
│          │         │          │       │
     ┌────▼──┐ ┌───▼────┐ ┌───▼────┐ ┌──▼────┐
     │local-  │ │ remote-│ │ gpu-   │ │ backup│
     │server  │ │ server │ │ node   │ │ node  │
     └───────┘ └────────┘ └────────┘ └───────┘
```

### 三大模块

| 模块 | 职责 | 暴露的工具 |
|------|------|-----------|
| **Scanner** | 采集裸机/容器的端口、服务、资源 | `scan_node`, `scan_all` |
| **Registry** | 维护服务-端口-节点的映射表 | `list_services`, `find_port`, `register_service` |
| **Allocator** | 部署前预检：资源是否足够、端口是否冲突 | `preflight_check`, `allocate_port` |

## 安装

```bash
# 通过 ClawHub 安装
openclaw skill install github.com/zfanmy/dreammoon-skills/skills/env-manager

# 或直接克隆
openclaw skill clone https://github.com/zfanmy/dreammoon-skills.git --subpath skills/env-manager
```

## 配置

首次运行自动生成 `~/.openclaw/skills/env-manager/config.yaml`，编辑节点信息：

```yaml
nodes:
  - name: local-server
    host: localhost
    local: true
    tags: [amd64, linux]

  - name: remote-server
    host: 192.168.1.10
    user: admin
    port: 22
    auth: key
    key_path: ~/.ssh/id_rsa
    tags: [amd64, linux]

  - name: gpu-node
    host: 10.0.0.5
    user: admin
    port: 22
    auth: key
    key_path: ~/.ssh/id_rsa
    tags: [amd64, linux, gpu]
    gpu_type: nvidia

settings:
  scan_timeout: 10s
  ssh_pool_size: 2
  cache_ttl: 30s
  alert_thresholds:
    cpu_percent: 90
    memory_percent: 85
    disk_percent: 90
```

## 提供的工具

| 工具名 | 说明 | 典型用途 |
|--------|------|---------|
| `scan_node` | 扫描单节点资源和服务状态 | 查看某台机器的状态 |
| `scan_all` | 并行扫描所有节点 | 集群全景概览 |
| `list_services` | 查询已注册的服务列表 | 找某个服务跑在哪 |
| `find_port` | 查端口占用 | 端口冲突排查 |
| `preflight_check` | 部署前预检 | 部署前确认资源充足 |
| `allocate_port` | 分配可用端口 | 自动选端口 |

## 使用示例

**自然语言（通过 LLM）：**

- "remote-server 上还有多少内存？"
- "帮我在 gpu-node 上找一个 8000-9000 之间的可用端口"
- "我要在 local-server 上部署一个需要 4GB 内存和 6379 端口的 Redis，可以吗？"
- "gpu-node 的 GPU 还剩多少显存？"

**工具调用：**

```json
// preflight_check
{
  "node": "remote-server",
  "requirements": {
    "ports": [8080, 8443],
    "min_memory_mb": 2048,
    "need_gpu": false
  }
}

// 返回示例
{
  "ok": true,
  "node": "remote-server",
  "warnings": ["磁盘使用率 88%，接近阈值"],
  "snapshot": {
    "cpu": { "cores": 8, "usage_percent": 23.5 },
    "memory": { "total_mb": 16384, "available_mb": 10240, "usage_percent": 37.5 },
    "disk": { "usage_percent": 88.2 },
    "bare_ports": [
      { "port": 22, "process": "sshd", "bind": "0.0.0.0" },
      { "port": 80, "process": "nginx", "bind": "0.0.0.0" }
    ]
  },
  "port_suggestions": {}
}
```

## 权限要求

- SSH 访问远程节点（推荐 key 认证，支持 `~/.ssh/config` 中定义的 Host）
- 远程节点需要的基础命令：`ss`/`netstat`, `free`, `df`, `docker`（可选）, `nvidia-smi`（可选）
- **无需在远程节点安装任何 Agent** —— 这是核心设计决策，用 SSH 远程采集换取零部署负担

## 技术选型

- **语言**: Go（单二进制分发，~6MB，零依赖）
- **采集方式**: 原生 shell 命令（`ss`, `free`, `df`, `nvidia-smi` 等），跨平台兼容
- **远程执行**: 原生 SSH 库 + 连接池
- **持久化**: JSON 文件（轻量，无需 CGO/SQLite）
- **传输协议**: stdio JSON-RPC（与 OpenClaw 原生兼容）

## 项目结构

```
env-manager/
├── SKILL.md                 # 技能描述文件（本文件）
├── skill.yaml               # OpenClaw 技能清单
├── README.md                # 项目介绍
├── DESIGN.md                # 详细设计文档
├── go.mod
├── main.go                  # 入口: serve / scan / version
├── internal/
│   ├── connector/           # 节点连接抽象
│   │   ├── executor.go      # Executor 接口
│   │   ├── local.go         # 本地执行
│   │   └── ssh.go           # SSH 连接池
│   ├── scanner/             # 资源采集模块
│   │   ├── scanner.go       # 采集编排
│   │   ├── cpu.go
│   │   ├── memory.go
│   │   ├── disk.go
│   │   ├── gpu.go
│   │   ├── ports.go         # 裸机端口
│   │   └── containers.go    # Docker 容器
│   ├── registry/            # 服务注册表
│   │   ├── store.go         # JSON 文件存储层
│   │   └── registry.go      # CRUD 逻辑
│   ├── allocator/           # 预检与分配
│   │   └── preflight.go
│   ├── config/              # 配置管理
│   │   └── config.go
│   └── transport/           # 通信层
│       └── jsonrpc.go       # stdio JSON-RPC
├── configs/
│   └── config.example.yaml
└── scripts/
    └── build.sh             # 交叉编译脚本
```

## 实现路线图

```
Phase 1 (MVP) ─ 1~2 周
├── 本地节点扫描
├── scan_node + list_services 工具
├── JSON 文件持久化
└── OpenClaw stdio 集成

Phase 2 ─ 第 3 周
├── SSH 远程扫描 (多节点)
├── preflight_check + allocate_port
├── 扫描结果缓存
└── 服务注册表同步

Phase 3 ─ 第 4 周
├── 主动告警 (阈值触发)
├── GPU 采集 (nvidia-smi)
├── macOS 兼容
└── 发布到 ClawHub
```

## 兼容性

- **节点 OS**: Linux (amd64/arm64), macOS (arm64)
- **OpenClaw**: >= 0.1.0
- **传输协议**: stdio (JSON-RPC)
- **Go 版本**: >= 1.19

## 与其他技能的协作

| 技能 | 协作场景 |
|------|---------|
| `config-manager` | 读取 SSH 配置和节点信息 |
| `cron-backup` | 备份 env-manager 的数据和配置 |
| `network-relay` | 当当前节点无法直连远程节点时，通过中继节点执行扫描 |

## 许可证

MIT © DreamMoon
