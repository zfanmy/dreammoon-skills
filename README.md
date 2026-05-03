# DreamMoon Skills for ClawHub 🌙

> zfanmy-梦月儿 的 OpenClaw 技能集合

## 技能列表

| 技能 | 功能 | 状态 |
|------|------|------|
| [openclaw-deploy](#1-openclaw-deploy) | OpenClaw 打包部署工具 | ✅ 可用 |
| [searxng-search](#2-searxng-search) | SearXNG 网络搜索 | ✅ 可用 |
| [cron-backup](#3-cron-backup) | 定时自动化备份系统 | ✅ 可用 |
| [skill-from-memory](#4-skill-from-memory) | 从记忆自动创建 skill | ✅ 可用 |
| [env-manager](#5-env-manager) | 多节点环境管理服务 | 🆕 新发布 |

---

## 1. openclaw-deploy

**功能**：OpenClaw 打包部署工具

- Docker 镜像构建（clean/full 版本）
- 便携版打包
- 远程部署脚本

**安装**：

```bash
clawhub install openclaw-deploy
```

---

## 2. searxng-search

**功能**：SearXNG 网络搜索

- MCP 服务器集成
- Bash 搜索脚本
- 可配置 SearXNG 端点

**安装**：

```bash
clawhub install searxng-search
```

---

## 3. cron-backup

**功能**：定时自动化备份系统

- 目录备份（带时间戳）
- 版本触发备份（仅版本变化时备份）
- 定时任务设置
- 自动清理旧备份

**安装**：

```bash
clawhub install cron-backup
```

**快速使用**：

```bash
# 单次备份
./scripts/backup.sh /path/to/source /path/to/backup

# 设置每天2点备份
./scripts/setup-cron.sh daily /source /backup "0 2 * * *"
```

---

## 4. skill-from-memory

**功能**：从记忆/历史任务自动创建并发布 skill

- 从会话历史提取技能模式
- 从记忆文件提取任务
- 自动创建 skill 结构
- 一键发布到 GitHub + ClawHub

**安装**：

```bash
clawhub install skill-from-memory
```

**快速使用**：

```bash
# 一键从对话创建并发布 skill
./scripts/create-and-publish.sh \
  --source session.jsonl \
  --skill-name my-automation \
  --github user/skills \
  --clawhub-slug my-automation
```

---

## 5. env-manager 🆕

**功能**：多节点环境管理服务

> 让 AI Agent 拥有集群感知能力 —— 扫描、预检、分配，部署前就知道能不能做。

- **裸机服务与端口管理** — 扫描裸机上运行的服务和占用的端口
- **容器端口与服务管理** — 扫描 Docker 容器的端口映射和运行状态
- **裸机资源管理** — 部署前预检：CPU/内存/磁盘/GPU 是否足够、端口是否冲突

**核心特性**：

| 特性 | 说明 |
|------|------|
| 零 Agent 部署 | 通过 SSH 远程执行采集命令，节点上无需安装任何东西 |
| 六大工具接口 | `scan_node` / `scan_all` / `list_services` / `find_port` / `preflight_check` / `allocate_port` |
| 预检决策 | 部署前自动检查资源+端口，给出 `ok: true/false` 和替代建议 |
| 默认适配集群 | 配置里已写好 macmini / xgp / ncu / tuf 四个节点 |
| Go 单二进制 | 交叉编译后 ~6MB，零依赖运行 |

**安装**：

```bash
clawhub install env-manager
```

**使用示例**：

```bash
# 扫描单个节点
./env-manager -cmd scan -node macmini -scope all

# 查看版本
./env-manager -cmd version
```

**自然语言调用（通过 LLM）**：

- "主节点 上还有多少内存？"
- "帮我在 备份节点 上找一个 8000-9000 之间的可用端口"
- "我要在 从节点 上部署一个需要 4GB 内存和 6379 端口的 Redis，可以吗？"
- "gpu节点 的 GPU 还剩多少显存？"

**技术栈**：Go 1.19+ | stdio JSON-RPC | SSH 远程采集 | JSON 文件存储

---

## 作者

**zfanmy-梦月儿** 🌙

## 许可证

MIT
