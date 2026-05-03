

# OpenClaw 技能：环境管理服务（Environment Manager）设计文档

## 1. 整体架构设计

### 核心理念

不做 Prometheus/Grafana 的替代品，而是做 AI Agent 的"眼睛和手"——让 LLM 在部署服务前能看到集群状态、做出判断、执行操作。

### 模块划分

```
┌─────────────────────────────────────────────────────┐
│                   OpenClaw Runtime                   │
│                                                      │
│  ┌───────────────────────────────────────────────┐   │
│  │         env-manager (Skill Process)           │   │
│  │                                                │   │
│  │  ┌─────────┐  ┌──────────┐  ┌─────────────┐  │   │
│  │  │ Scanner │  │ Registry │  │  Allocator   │  │   │
│  │  │ Module  │  │  Module  │  │   Module     │  │   │
│  │  └────┬────┘  └────┬─────┘  └──────┬──────┘  │   │
│  │       │             │               │          │   │
│  │  ┌────▼─────────────▼───────────────▼──────┐  │   │
│  │  │            Node Connector                │  │   │
│  │  │     (SSH Pool / Local Executor)          │  │   │
│  │  └────┬─────────┬──────────┬───────┬───────┘  │   │
│  │       │         │          │       │           │   │
│  └───────┼─────────┼──────────┼───────┼──────────┘   │
│          │         │          │       │               │
└──────────┼─────────┼──────────┼───────┼──────────────┘
           │         │          │       │
      ┌────▼──┐ ┌───▼───┐ ┌───▼──┐ ┌──▼──┐
      │local-  │ │ remote-│ │ gpu-   │ │ backup│
      └───────┘ └───────┘ └──────┘ └─────┘
```

### 三大模块职责

| 模块 | 职责 | 暴露的工具 |
|------|------|-----------|
| Scanner | 采集裸机/容器的端口、服务、资源 | `scan_node`, `scan_all` |
| Registry | 维护服务-端口-节点的映射表 | `list_services`, `find_port`, `register_service` |
| Allocator | 部署前预检：资源是否足够、端口是否冲突 | `preflight_check`, `allocate_port` |

### 数据流

```
用户/LLM 发起意图: "在 remote-server 上部署 redis"
        │
        ▼
  ┌─ preflight_check(node="remote-server", requirements={port:6379, mem:"256MB"})
  │     │
  │     ├─ Scanner.scan_node("remote-server")  ──SSH──▶  remote-server  ──▶ 采集快照
  │     │
  │     ├─ Registry.find_port(6379)   ──▶ 检查端口冲突
  │     │
  │     └─ 返回: { ok: true, warnings: [], snapshot: {...} }
  │           或: { ok: false, reason: "端口6379已被占用", suggestion: 6380 }
  │
  ▼
  LLM 根据结果决定下一步操作
```

## 2. 技术选型

### 语言：Go

理由：
- 单二进制分发，零依赖，符合轻量化要求
- 原生 SSH 库（`golang.org/x/crypto/ssh`）成熟
- 交叉编译覆盖 amd64/arm64（generic ARM64）
- OpenClaw 生态本身偏 Go 友好

### 关键依赖

```
golang.org/x/crypto/ssh     # SSH 连接池
github.com/shirou/gopsutil  # 本地资源采集（跨平台）
github.com/docker/docker    # Docker API 客户端
gopkg.in/yaml.v3            # 配置文件解析
```

### 为什么不选 Python

Python 适合快速原型，但作为常驻技能进程，Go 的内存占用（~10MB）远低于 Python（~50MB+），且不需要用户预装 runtime。

## 3. 与 OpenClaw 的集成方式

### 技能接口规范

env-manager 作为 OpenClaw 的 Tool Provider，通过 stdio JSON-RPC 协议通信（与 MCP 兼容）。

### 暴露的工具定义

```yaml
# tools.yaml
tools:
  - name: scan_node
    description: "扫描指定节点的资源和服务状态"
    parameters:
      node:
        type: string
        description: "节点名称: local-server | remote-server | gpu-node"
        required: true
      scope:
        type: string
        enum: [all, bare, container, ports, resources]
        default: all
    returns:
      type: object
      description: "节点快照，包含 CPU/内存/磁盘/GPU/端口/服务信息"

  - name: scan_all
    description: "并行扫描所有节点，返回集群全景"
    parameters: {}
    returns:
      type: object
      description: "所有节点的快照集合"

  - name: list_services
    description: "查询已注册的服务列表，支持按节点/端口/名称过滤"
    parameters:
      node:
        type: string
        required: false
      port:
        type: integer
        required: false
      name:
        type: string
        required: false

  - name: find_port
    description: "查找端口在哪个节点被谁占用"
    parameters:
      port:
        type: integer
        required: true
      node:
        type: string
        required: false
        description: "不指定则搜索所有节点"

  - name: preflight_check
    description: "部署前预检：检查目标节点是否满足资源和端口要求"
    parameters:
      node:
        type: string
        required: true
      requirements:
        type: object
        properties:
          ports:
            type: array
            items: { type: integer }
          min_memory_mb:
            type: integer
          min_disk_gb:
            type: integer
          need_gpu:
            type: boolean

  - name: allocate_port
    description: "在指定节点上分配一个可用端口"
    parameters:
      node:
        type: string
        required: true
      preferred:
        type: integer
        required: false
        description: "优先尝试的端口号"
      range_start:
        type: integer
        default: 8000
      range_end:
        type: integer
        default: 9000
```

### OpenClaw 注册方式

```yaml
# skill.yaml (OpenClaw 技能清单)
name: env-manager
version: 0.1.0
runtime: binary
entry: ./env-manager serve --config ./config.yaml
transport: stdio
capabilities:
  tools: true
  alerts: true
```

## 4. 多节点支持的实现方案

### 节点配置

```yaml
# config.yaml
nodes:
  - name: local-server
    host: 192.168.1.10
    user: admin
    auth: key          # key | password | agent
    key_path: ~/.ssh/id_ed25519
    tags: [arm64, macos]

  - name: remote-server
    host: 192.168.1.20
    user: user
    auth: agent
    tags: [amd64, linux, gpu]
    gpu_type: nvidia

  - name: gpu-node
    host: 192.168.1.30
    user: user
    auth: key
    key_path: ~/.ssh/id_ed25519
    tags: [amd64, linux, gpu]
    gpu_type: nvidia

  - name: backup-node
    host: localhost     # 本机直接执行，不走 SSH
    local: true
    tags: [amd64, linux, gpu]
    gpu_type: nvidia

settings:
  scan_timeout: 10s
  ssh_pool_size: 2      # 每节点保持的 SSH 连接数
  cache_ttl: 30s        # 扫描结果缓存时间
  alert_thresholds:
    cpu_percent: 90
    memory_percent: 85
    disk_percent: 90
```

### 连接器设计

```go
// connector.go - 核心抽象
type Executor interface {
    Run(ctx context.Context, cmd string) (stdout string, err error)
    Close() error
}

// 本地节点 → 直接 exec.Command
type LocalExecutor struct{}

// 远程节点 → SSH 连接池
type SSHExecutor struct {
    pool *SSHPool
    node NodeConfig
}
```

关键设计决策：
- 本地节点（backup-node）直接执行命令，零开销
- 远程节点维护 SSH 连接池，避免每次扫描重新握手
- 所有采集逻辑统一为 shell 命令，通过 Executor 接口透明执行
- 并行扫描所有节点，单节点超时不阻塞其他节点

## 5. 资源扫描的具体实现

### 采集命令映射表

每项资源对应一条轻量 shell 命令，远程/本地统一执行：

```go
var collectors = map[string]Collector{
    "cpu": {
        // Linux: /proc/stat 计算使用率; macOS: vm_stat
        Linux: `top -bn1 | grep 'Cpu(s)' | awk '{print $2+$4}'`,
        Darwin: `ps -A -o %cpu | awk '{s+=$1} END {print s}'`,
        Parse: parseCPU,
    },
    "memory": {
        Linux:  `free -b | awk '/Mem:/{printf "%d %d %d",$2,$3,$7}'`,
        Darwin: `vm_stat | head -10`,
        Parse:  parseMemory,
    },
    "disk": {
        // 通用
        Cmd:   `df -B1 --output=source,size,used,avail,target 2>/dev/null || df -k`,
        Parse: parseDisk,
    },
    "gpu": {
        // NVIDIA GPU
        Cmd:   `nvidia-smi --query-gpu=index,name,memory.total,memory.used,utilization.gpu,temperature.gpu --format=csv,noheader,nounits 2>/dev/null || echo "no-gpu"`,
        Parse: parseGPU,
    },
    "ports_bare": {
        // 裸机端口 + 进程名
        Linux:  `ss -tlnp 2>/dev/null | tail -n +2`,
        Darwin: `lsof -iTCP -sTCP:LISTEN -nP`,
        Parse:  parsePorts,
    },
    "containers": {
        // Docker 容器 + 端口映射
        Cmd:   `docker ps --format '{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Ports}}\t{{.Status}}' 2>/dev/null || echo "no-docker"`,
        Parse: parseContainers,
    },
}
```

### 扫描结果数据结构

```go
type NodeSnapshot struct {
    Node      string        `json:"node"`
    Timestamp time.Time     `json:"timestamp"`
    OS        string        `json:"os"`        // linux | darwin
    Arch      string        `json:"arch"`      // amd64 | arm64
    CPU       CPUInfo       `json:"cpu"`
    Memory    MemoryInfo    `json:"memory"`
    Disks     []DiskInfo    `json:"disks"`
    GPUs      []GPUInfo     `json:"gpus,omitempty"`
    BarePorts []PortInfo    `json:"bare_ports"`
    Containers []ContainerInfo `json:"containers"`
    Alerts    []Alert       `json:"alerts,omitempty"`
}

type CPUInfo struct {
    Cores       int     `json:"cores"`
    UsagePercent float64 `json:"usage_percent"`
}

type MemoryInfo struct {
    TotalMB     int     `json:"total_mb"`
    UsedMB      int     `json:"used_mb"`
    AvailableMB int     `json:"available_mb"`
    UsagePercent float64 `json:"usage_percent"`
}

type GPUInfo struct {
    Index       int    `json:"index"`
    Name        string `json:"name"`
    MemTotalMB  int    `json:"mem_total_mb"`
    MemUsedMB   int    `json:"mem_used_mb"`
    UtilPercent int    `json:"util_percent"`
    TempC       int    `json:"temp_celsius"`
}

type PortInfo struct {
    Port    int    `json:"port"`
    Proto   string `json:"proto"`    // tcp | udp
    Process string `json:"process"`
    PID     int    `json:"pid"`
    Bind    string `json:"bind"`     // 0.0.0.0 | 127.0.0.1 | ::
}

type ContainerInfo struct {
    ID     string            `json:"id"`
    Name   string            `json:"name"`
    Image  string            `json:"image"`
    Status string            `json:"status"`
    Ports  []PortMapping     `json:"ports"`
}
```

## 6. 数据持久化方案

### 选择：SQLite + JSON 快照文件

```
~/.openclaw/skills/env-manager/
├── config.yaml              # 节点配置
├── data/
│   ├── registry.db          # SQLite: 服务注册表 + 历史快照
│   └── snapshots/           # 最近N次全量快照 (JSON, 用于离线分析)
│       ├── 2024-01-15T10:30:00.json
│       └── 2024-01-15T11:00:00.json
└── logs/
    └── env-manager.log
```

### 为什么是 SQLite

- 零部署，单文件，Go 有成熟的 CGO-free 驱动（`modernc.org/sqlite`）
- 支持结构化查询："哪些端口在过去一周被频繁占用"
- 比纯 JSON 文件更适合做服务注册表的 CRUD

### 表结构

```sql
-- 服务注册表（手动注册 + 自动发现）
CREATE TABLE services (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    node        TEXT NOT NULL,
    port        INTEGER NOT NULL,
    proto       TEXT DEFAULT 'tcp',
    source      TEXT DEFAULT 'discovered',  -- discovered | manual
    first_seen  DATETIME,
    last_seen   DATETIME,
    metadata    TEXT,  -- JSON blob
    UNIQUE(node, port, proto)
);

-- 资源快照历史（用于趋势分析）
CREATE TABLE snapshots (
    id          INTEGER PRIMARY KEY,
    node        TEXT NOT NULL,
    timestamp   DATETIME NOT NULL,
    data        TEXT NOT NULL,  -- JSON blob of NodeSnapshot
    INDEX idx_node_time (node, timestamp)
);
```

## 7. 与其他监控工具的差异化定位

```
┌──────────────────┬──────────────┬──────────────┬─────────────────┐
│                  │ Prometheus   │ Netdata      │ env-manager     │
│                  │ + Grafana    │              │ (本技能)         │
├──────────────────┼──────────────┼──────────────┼─────────────────┤
│ 定位             │ 生产级监控    │ 实时可视化    │ AI Agent 的     │
│                  │              │              │ 环境感知层       │
├──────────────────┼──────────────┼──────────────┼─────────────────┤
│ 部署复杂度        │ 高(多组件)    │ 中(每节点装)  │ 低(单二进制)     │
├──────────────────┼──────────────┼──────────────┼─────────────────┤
│ 数据消费者        │ 人(看图表)    │ 人(看图表)    │ LLM(结构化JSON) │
├──────────────────┼──────────────┼──────────────┼─────────────────┤
│ 核心能力          │ 时序存储      │ 实时展示      │ 预检+决策支持    │
│                  │ 告警规则      │              │ 端口分配        │
├──────────────────┼──────────────┼──────────────┼─────────────────┤
│ 每节点需装Agent   │ 是           │ 是           │ 否(SSH远程采集)  │
├──────────────────┼──────────────┼──────────────┼─────────────────┤
│ 与 LLM 集成      │ 需额外适配    │ 无           │ 原生工具接口     │
└──────────────────┴──────────────┴──────────────┴─────────────────┘
```

一句话定位：**env-manager 不是监控系统，是 AI 的集群感知能力。** 它回答的不是"过去发生了什么"，而是"现在能不能做这件事"。

## 8. SKILL.md 规范

```markdown
# env-manager

> OpenClaw 技能：多节点环境管理服务

## 概述

为 AI Agent 提供集群级别的环境感知能力。扫描裸机/容器的服务、端口、
资源（CPU/内存/磁盘/GPU），支持部署前预检和端口分配。

## 安装

    openclaw skill install github.com/yourname/env-manager

## 配置

首次运行自动生成 `~/.openclaw/skills/env-manager/config.yaml`，
编辑节点信息：

    nodes:
      - name: backup-node
        host: localhost
        local: true
      - name: remote-server
        host: 192.168.1.20
        user: user
        auth: agent

## 提供的工具

| 工具名 | 说明 | 典型用途 |
|--------|------|---------|
| `scan_node` | 扫描单节点 | 查看某台机器的状态 |
| `scan_all` | 扫描全部节点 | 集群全景概览 |
| `list_services` | 查询服务注册表 | 找某个服务跑在哪 |
| `find_port` | 查端口占用 | 端口冲突排查 |
| `preflight_check` | 部署前预检 | 部署前确认资源充足 |
| `allocate_port` | 分配可用端口 | 自动选端口 |

## 使用示例

**自然语言（通过 LLM）：**

- "remote-server 上还有多少内存？"
- "帮我在 gpu-node 上找一个 8000-9000 之间的可用端口"
- "我要在 local-server 上部署一个需要 4GB 内存和 6379 端口的 Redis，可以吗？"

**工具调用：**

    // preflight_check
    {
      "node": "remote-server",
      "requirements": {
        "ports": [8080, 8443],
        "min_memory_mb": 2048,
        "need_gpu": true
      }
    }

## 权限要求

- SSH 访问远程节点（推荐 key 认证）
- 远程节点需要: `ss`, `free`, `df`, `docker`(可选), `nvidia-smi`(可选)
- 无需在远程节点安装任何 agent

## 兼容性

- 节点 OS: Linux (amd64/arm64), macOS (arm64)
- OpenClaw: >= 0.1.0
- 传输协议: stdio (JSON-RPC)

## 许可证

MIT
```

## 9. 实现路线图

```
Phase 1 (MVP) ─ 1~2 周
├── 本地节点扫描
├── scan_node + list_services 工具
├── JSON 文件持久化
└── OpenClaw stdio 集成

Phase 2 ─ 第 3 周
├── SSH 远程扫描 (多节点)
├── preflight_check + allocate_port
├── SQLite 持久化
└── 扫描结果缓存

Phase 3 ─ 第 4 周
├── 主动告警 (阈值触发)
├── GPU 采集 (nvidia-smi)
├── macOS 兼容
└── 发布到 ClawHub
```

### 项目结构

```
env-manager/
├── SKILL.md
├── skill.yaml
├── go.mod
├── main.go                 # 入口: serve / scan / version
├── internal/
│   ├── connector/
│   │   ├── executor.go     # Executor 接口
│   │   ├── local.go        # LocalExecutor
│   │   └── ssh.go          # SSHExecutor + 连接池
│   ├── scanner/
│   │   ├── scanner.go      # 编排所有采集器
│   │   ├── cpu.go
│   │   ├── memory.go
│   │   ├── disk.go
│   │   ├── gpu.go
│   │   ├── ports.go        # 裸机端口
│   │   └── containers.go   # Docker 容器
│   ├── registry/
│   │   ├── store.go        # SQLite 存储层
│   │   └── registry.go     # 服务注册/查询逻辑
│   ├── allocator/
│   │   └── preflight.go    # 预检 + 端口分配
│   ├── config/
│   │   └── config.go       # YAML 配置加载
│   └── transport/
│       └── jsonrpc.go      # stdio JSON-RPC 处理
├── configs/
│   └── config.example.yaml
└── scripts/
    └── build.sh            # 交叉编译脚本
```

这个设计的核心取舍是：用 SSH 远程采集换取零 Agent 部署。对于 4 节点的小集群，SSH 的延迟（~100ms）完全可接受，而省去了在每台机器上装 agent 的运维负担。如果未来节点规模扩大到几十台，可以考虑加一个可选的轻量 agent 模式。