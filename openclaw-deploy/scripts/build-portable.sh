#!/bin/bash
set -e

OUTPUT_DIR="/home/zfanmy/openclaw_docker/portable"
NODE_VERSION="22.22.0"

echo "=========================================="
echo "Building OpenClaw Portable Package"
echo "=========================================="

rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"/{clean,full}/openclaw/app
mkdir -p "$OUTPUT_DIR/clean/openclaw/.openclaw"

# 1. Clean Version
echo "📦 Building clean version..."
cp -r /home/zfanmy/.nvm/versions/node/v$NODE_VERSION/lib/node_modules/openclaw/* "$OUTPUT_DIR/clean/openclaw/app/"
cp /home/zfanmy/openclaw_docker/clean/config/openclaw.json "$OUTPUT_DIR/clean/openclaw/.openclaw/"

# Create clean start script
printf '%s\n' '#!/bin/bash' \
'SCRIPT="$(cd "$(dirname "$0")" \&\& pwd)/openclaw"' \
'' \
'if command -v node > /dev/null 2>\&1; then' \
'    NODE_CMD="node"' \
'elif [ -x "$SCRIPT/../node/bin/node" ]; then' \
'    NODE_CMD="$SCRIPT/../node/bin/node"' \
'else' \
'    echo "❌ Node.js not found"' \
'    exit 1' \
'fi' \
'' \
'export OPENCLAW_CONFIG="$SCRIPT/.openclaw/openclaw.json"' \
'mkdir -p "$SCRIPT/.openclaw/workspace"' \
'' \
'echo "🚀 Starting OpenClaw (Clean)..."' \
'exec "$NODE_CMD" "$SCRIPT/app/openclaw.mjs" gateway start "$@"' \
> "$OUTPUT_DIR/clean/start.sh"
chmod +x "$OUTPUT_DIR/clean/start.sh"

# 2. Full Version
echo "📦 Building full version..."
mkdir -p "$OUTPUT_DIR/full/openclaw/.openclaw"
cp -r /home/zfanmy/.nvm/versions/node/v$NODE_VERSION/lib/node_modules/openclaw/* "$OUTPUT_DIR/full/openclaw/app/"
cp -r /home/zfanmy/.openclaw/* "$OUTPUT_DIR/full/openclaw/.openclaw/"

# Create full start script
printf '%s\n' '#!/bin/bash' \
'SCRIPT="$(cd "$(dirname "$0")" \&\& pwd)/openclaw"' \
'' \
'if command -v node > /dev/null 2>\&1; then' \
'    NODE_CMD="node"' \
'elif [ -x "$SCRIPT/../node/bin/node" ]; then' \
'    NODE_CMD="$SCRIPT/../node/bin/node"' \
'else' \
'    echo "❌ Node.js not found"' \
'    exit 1' \
'fi' \
'' \
'export OPENCLAW_CONFIG="$SCRIPT/.openclaw/openclaw.json"' \
'' \
'echo "🚀 Starting OpenClaw (Full - DreamMoon)..."' \
'exec "$NODE_CMD" "$SCRIPT/app/openclaw.mjs" gateway start "$@"' \
> "$OUTPUT_DIR/full/start.sh"
chmod +x "$OUTPUT_DIR/full/start.sh"

# Create install-node.sh
cat > "$OUTPUT_DIR/install-node.sh" << 'EOF'
#!/bin/bash
echo "📦 Installing Node.js 22.22.0..."
if ! command -v nvm >/dev/null 2>&1; then
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.0/install.sh | bash
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
fi
nvm install 22.22.0
nvm use 22.22.0
node --version
echo "✅ Node.js installed!"
EOF
chmod +x "$OUTPUT_DIR/install-node.sh"

# Create README
cat > "$OUTPUT_DIR/README.md" << 'EOF'
# OpenClaw 便携版

## 使用方式

### 1. 安装 Node.js
```bash
./install-node.sh
```

### 2. 启动服务
```bash
# 纯净版
cd clean
./start.sh

# 完整版（含 DreamMoon 配置）
cd full
./start.sh
```

### 3. 访问
WebUI: http://localhost:18789

## 部署到其他服务器
```bash
tar -czf openclaw-portable.tar.gz portable/
scp openclaw-portable.tar.gz user@remote:/opt/
```
EOF

echo ""
echo "=========================================="
echo "✅ Portable packages built!"
echo "Location: $OUTPUT_DIR"
ls -lh "$OUTPUT_DIR/"
du -sh "$OUTPUT_DIR/clean" "$OUTPUT_DIR/full"
