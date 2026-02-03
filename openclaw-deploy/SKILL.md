# SKILL.md 元数据格式示例

---
name: openclaw-deploy
description: Build and deploy OpenClaw as Docker images or portable packages
author: zfanmy-梦月儿
version: 1.0.0
homepage: 
license: MIT
keywords:
  - openclaw
  - deploy
  - docker
  - portable
  - backup
  - migration
requires:
  bins:
    - node
    - npm
    - tar
---

# OpenClaw Deploy

Build and deploy OpenClaw as Docker images or portable packages.

## Features

- 🐳 Build Docker images (clean/full versions)
- 📦 Create portable packages for deployment
- 🚀 Deploy to remote servers with one command
- 💾 Backup and restore configurations

## Quick Start

### Build Portable Packages

```bash
# Build both clean and full versions
./scripts/build-portable.sh

# Export for deployment
./scripts/export-portable.sh
```

### Deploy to Remote Server

```bash
# Deploy clean version
./export/deploy.sh user@remote-server clean /opt/openclaw

# Deploy full version
./export/deploy.sh user@remote-server full /opt/openclaw
```

## Directory Structure

```
openclaw-deploy/
├── portable/clean/          # Clean version (no personal data)
├── portable/full/           # Full version (with config)
├── export/                  # Deployment packages
│   ├── openclaw-clean-portable.tar.gz
│   ├── openclaw-full-portable.tar.gz
│   └── deploy.sh
└── scripts/
    ├── build-portable.sh
    ├── export-portable.sh
    └── deploy.sh
```

## Usage on Target Server

```bash
# Install Node.js
./install-node.sh

# Start OpenClaw
cd clean && ./start.sh   # or cd full && ./start.sh

# Access WebUI
open http://localhost:18789
```

## Requirements

- Node.js 22.x
- Docker (optional, for Docker builds)
- curl, rsync (for deployment)

## Author

zfanmy-梦月儿
