# DreamMoon Skills for ClawHub

这是 zfanmy-梦月儿 创建的 OpenClaw 技能集合。

## 技能列表

### 1. openclaw-deploy
**功能**：OpenClaw 打包部署工具

- Docker 镜像构建（clean/full 版本）
- 便携版打包
- 远程部署脚本

**安装**：
\`\`\`bash
clawhub install openclaw-deploy
\`\`\`

### 2. searxng-search
**功能**：SearXNG 网络搜索

- MCP 服务器集成
- Bash 搜索脚本
- 可配置 SearXNG 端点

**安装**：
\`\`\`bash
clawhub install searxng-search
\`\`\`

### 3. cron-backup
**功能**：定时自动化备份系统

- 目录备份（带时间戳）
- 版本触发备份（仅版本变化时备份）
- 定时任务设置
- 自动清理旧备份

**安装**：
\`\`\`bash
clawhub install cron-backup
\`\`\`

**快速使用**：
\`\`\`bash
# 单次备份
./scripts/backup.sh /path/to/source /path/to/backup

# 设置每天2点备份
./scripts/setup-cron.sh daily /source /backup "0 2 * * *"
\`\`\`

### 4. skill-from-memory
**功能**：从记忆/历史任务自动创建并发布 skill

- 从会话历史提取技能模式
- 从记忆文件提取任务
- 自动创建 skill 结构
- 一键发布到 GitHub + ClawHub

**安装**：
\`\`\`bash
clawhub install skill-from-memory
\`\`\`

**快速使用**：
\`\`\`bash
# 一键从对话创建并发布 skill
./scripts/create-and-publish.sh \\
  --source session.jsonl \\
  --skill-name my-automation \\
  --github user/skills \\
  --clawhub-slug my-automation
\`\`\`

## 作者

**zfanmy-梦月儿**

## 许可证

MIT
