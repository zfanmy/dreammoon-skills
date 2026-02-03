#!/bin/bash
# Export portable packages for deployment

set -e

OUTPUT_DIR="/home/zfanmy/openclaw_docker/export"
PORTABLE_DIR="/home/zfanmy/openclaw_docker/portable"

echo "=========================================="
echo "Exporting OpenClaw Portable Packages"
echo "=========================================="

mkdir -p "$OUTPUT_DIR"

# Create clean package
echo ""
echo "📦 Packaging clean version..."
tar -czf "$OUTPUT_DIR/openclaw-clean-portable.tar.gz" -C "$PORTABLE_DIR" clean/
echo "✅ Clean: openclaw-clean-portable.tar.gz ($(du -h $OUTPUT_DIR/openclaw-clean-portable.tar.gz | cut -f1))"

# Create full package
echo ""
echo "📦 Packaging full version..."
tar -czf "$OUTPUT_DIR/openclaw-full-portable.tar.gz" -C "$PORTABLE_DIR" full/
echo "✅ Full: openclaw-full-portable.tar.gz ($(du -h $OUTPUT_DIR/openclaw-full-portable.tar.gz | cut -f1))"

# Create deployment script
cat > "$OUTPUT_DIR/deploy.sh" << 'EOF'
#!/bin/bash
# Deploy OpenClaw to remote server

if [ $# -lt 3 ]; then
    echo "Usage: $0 <user@host> <clean|full> <remote-path>"
    exit 1
fi

REMOTE="$1"
VERSION="$2"
REMOTE_PATH="$3"

echo "🚀 Deploying openclaw-$VERSION-portable to $REMOTE:$REMOTE_PATH"

ssh "$REMOTE" "mkdir -p $REMOTE_PATH"
scp "openclaw-$VERSION-portable.tar.gz" "$REMOTE:$REMOTE_PATH/"
ssh "$REMOTE" "cd $REMOTE_PATH && tar -xzf openclaw-$VERSION-portable.tar.gz && rm openclaw-$VERSION-portable.tar.gz"

echo "✅ Deployed successfully!"
echo "To start: ssh $REMOTE 'cd $REMOTE_PATH/$VERSION && ./start.sh'"
EOF
chmod +x "$OUTPUT_DIR/deploy.sh"

# Copy install-node.sh
cp "$PORTABLE_DIR/install-node.sh" "$OUTPUT_DIR/"

# Create README
cat > "$OUTPUT_DIR/README.md" << 'EOF'
# OpenClaw 部署包

## 文件说明

- `openclaw-clean-portable.tar.gz` - 纯净版 (约 590MB)
- `openclaw-full-portable.tar.gz` - 完整版含配置 (约 1.5GB)
- `install-node.sh` - Node.js 安装脚本
- `deploy.sh` - 远程部署脚本

## 部署步骤

### 1. 在目标服务器安装 Node.js
```bash
./install-node.sh
```

### 2. 解压并启动
```bash
# 纯净版
tar -xzf openclaw-clean-portable.tar.gz
cd clean
./start.sh

# 完整版
tar -xzf openclaw-full-portable.tar.gz
cd full
./start.sh
```

### 3. 或使用部署脚本
```bash
./deploy.sh user@remote-server clean /opt/openclaw
./deploy.sh user@remote-server full /opt/openclaw
```

## 访问服务
- WebUI: http://localhost:18789

## 区别

| 版本 | 大小 | 说明 |
|------|------|------|
| Clean | 590MB | 纯净版，需自行配置 API Key 和渠道 |
| Full | 1.5GB | 含 DreamMoon 配置、飞书渠道、历史对话 |
EOF

echo ""
echo "=========================================="
echo "Export completed!"
echo ""
echo "Files in: $OUTPUT_DIR"
ls -lh "$OUTPUT_DIR"
echo ""
echo "Deploy example:"
echo "  $OUTPUT_DIR/deploy.sh user@remote clean /opt/openclaw"
echo "=========================================="
