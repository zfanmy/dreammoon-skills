# Skills Release v1.0.0

两个新技能已创建完成，准备发布到 GitHub 和 ClawHub。

---

## Skill 1: cron-backup (4.4K)

**功能**: 定时自动化备份系统

**包含脚本**:
- `backup.sh` - 单次目录备份（带时间戳）
- `backup-versioned.sh` - 版本触发备份（仅版本变化时备份）
- `setup-cron.sh` - 设置定时任务
- `cleanup.sh` - 自动清理旧备份
- `list-backups.sh` - 列出备份

**使用方法**:
```bash
# 单次备份
./scripts/backup.sh /path/to/source /path/to/backup

# 设置每天2点备份
./scripts/setup-cron.sh daily /source /backup "0 2 * * *"

# 版本感知备份
./scripts/setup-cron.sh versioned /app /backup "0 */6 * * *"
```

---

## Skill 2: skill-from-memory (7.1K)

**功能**: 从记忆/历史任务自动创建并发布 skill

**包含脚本**:
- `extract-from-history.sh` - 从会话历史提取
- `extract-from-memory.sh` - 从记忆文件提取
- `create-skill.sh` - 创建 skill 结构
- `publish.sh` - 发布到 GitHub + ClawHub
- `create-and-publish.sh` - 一键全流程

**使用方法**:
```bash
# 从会话历史创建 skill
./scripts/extract-from-history.sh session.jsonl ./extracted-content

# 创建 skill 结构
./scripts/create-skill.sh ./extracted-content my-skill \
  --description "My automation skill" \
  --type workflow

# 发布
./scripts/publish.sh ./my-skill \
  --github "user/skills-repo" \
  --clawhub-slug "my-skill" \
  --version "1.0.0"

# 一键全流程
./scripts/create-and-publish.sh \
  --source session.jsonl \
  --skill-name my-skill \
  --github user/skills \
  --clawhub-slug my-skill
```

---

## 文件位置

```
/home/zfanmy/.openclaw/workspace/skills/
├── cron-backup/
│   ├── SKILL.md
│   └── scripts/
│       ├── backup.sh
│       ├── backup-versioned.sh
│       ├── cleanup.sh
│       ├── list-backups.sh
│       └── setup-cron.sh
├── skill-from-memory/
│   ├── SKILL.md
│   └── scripts/
│       ├── create-and-publish.sh
│       ├── create-skill.sh
│       ├── extract-from-history.sh
│       ├── extract-from-memory.sh
│       └── publish.sh
├── cron-backup-v1.0.0.tar.gz (4.4K)
└── skill-from-memory-v1.0.0.tar.gz (7.1K)
```

---

## 发布步骤

### 1. GitHub 发布

```bash
# 登录 GitHub（SSH 密钥需已配置）
git config --global user.name "Your Name"
git config --global user.email "your@email.com"

# 为每个 skill 推送
cd /home/zfanmy/.openclaw/workspace/skills/cron-backup
git init
git remote add origin git@github.com:zfanmy/openclaw-skills.git
git add -A
git commit -m "Release cron-backup v1.0.0"
git push origin main

cd /home/zfanmy/.openclaw/workspace/skills/skill-from-memory
git init
git remote add origin git@github.com:zfanmy/openclaw-skills.git
git add -A
git commit -m "Release skill-from-memory v1.0.0"
git push origin main
```

### 2. ClawHub 发布

```bash
# 登录
clawhub login

# 发布 cron-backup
clawhub publish /home/zfanmy/.openclaw/workspace/skills/cron-backup \
  --slug cron-backup \
  --name "Cron Backup" \
  --version 1.0.0 \
  --changelog "Initial release: automated backup scheduling"

# 发布 skill-from-memory
clawhub publish /home/zfanmy/.openclaw/workspace/skills/skill-from-memory \
  --slug skill-from-memory \
  --name "Skill from Memory" \
  --version 1.0.0 \
  --changelog "Initial release: create skills from conversation history"
```

---

## 后续使用

安装技能：
```bash
clawhub install cron-backup
clawhub install skill-from-memory
```

使用 skill-from-memory 创建新技能：
```bash
# 从昨天的对话创建 skill
clawhub skill-from-memory \
  --source ~/.openclaw/agents/main/sessions/2026-02-03.jsonl \
  --skill-name my-new-automation \
  --github zfanmy/skills \
  --clawhub-slug my-new-automation
```

---

**Created**: 2026-02-04  
**Version**: 1.0.0  
**Author**: DreamMoon for 明焱
