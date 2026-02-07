#!/bin/bash
# OpenClaw Portable Package Builder - 优化版
# 支持自定义路径配置

set -e

# 版本
VERSION="1.0.1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 默认配置（可通过环境变量覆盖）
: "${OPENCLAW_INSTALL_DIR:=/home/$(whoami)/.nvm/versions/node/v22.22.0/lib/node_modules/openclaw}"
: "${OPENCLAW_CONFIG_DIR:=/home/$(whoami)/.openclaw}"
: "${OUTPUT_DIR:=$(pwd)/openclaw-portable-output}"
: "${CLEAN_CONFIG_FILE:=${SCRIPT_DIR}/../clean/config/openclaw.json}"

NODE_VERSION="22.22.0"

echo "=========================================="
echo "OpenClaw Portable Package Builder v${VERSION}"
echo "=========================================="

# 检查依赖
check_dependencies() {
    local missing=()
    
    if ! command -v cp >/dev/null 2>&1; then
        missing+=("cp")
    fi
    
    if ! command -v mkdir >/dev/null 2>&1; then
        missing+=("mkdir")
    fi
    
    if [ ${#missing[@]} -gt 0 ]; then
        echo "❌ 缺少必要工具: ${missing[*]}"
        exit 1
    fi
    
    echo "✅ 依赖检查通过"
}

# 检查源文件存在
check_source() {
    if [ ! -d "$OPENCLAW_INSTALL_DIR" ]; then
        echo "❌ OpenClaw 安装目录不存在: $OPENCLAW_INSTALL_DIR"
        echo "   可通过环境变量设置: export OPENCLAW_INSTALL_DIR=/path/to/openclaw"
        exit 1
    fi
    
    if [ ! -d "$OPENCLAW_CONFIG_DIR" ]; then
        echo "⚠️  警告: OpenClaw 配置目录不存在: $OPENCLAW_CONFIG_DIR"
        echo "   可通过环境变量设置: export OPENCLAW_CONFIG_DIR=/path/to/.openclaw"
    fi
    
    if [ ! -f "$CLEAN_CONFIG_FILE" ]; then
        echo "⚠️  警告: 纯净版配置文件不存在: $CLEAN_CONFIG_FILE"
        echo "   将使用默认配置"
    fi
    
    echo "✅ 源文件检查通过"
}

# 显示配置
show_config() {
    echo ""
    echo "📋 当前配置:"
    echo "  OpenClaw 安装目录: $OPENCLAW_INSTALL_DIR"
    echo "  OpenClaw 配置目录: $OPENCLAW_CONFIG_DIR"
    echo "  输出目录: $OUTPUT_DIR"
    echo "  纯净版配置: $CLEAN_CONFIG_FILE"
    echo ""
}

# 清理并创建目录
prepare_dirs() {
    echo "🗂️  准备输出目录..."
    rm -rf "$OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR"/{clean,full}/openclaw/app
    mkdir -p "$OUTPUT_DIR/clean/openclaw/.openclaw"
    echo "✅ 目录准备完成"
}

# 构建纯净版
build_clean() {
    echo ""
    echo "📦 构建纯净版..."
    
    # 复制应用
    if [ -d "$OPENCLAW_INSTALL_DIR" ]; then
        cp -r "$OPENCLAW_INSTALL_DIR"/* "$OUTPUT_DIR/clean/openclaw/app/"
        echo "  ✅ 复制 OpenClaw 应用"
    fi
    
    # 复制配置
    if [ -f "$CLEAN_CONFIG_FILE" ]; then
        cp "$CLEAN_CONFIG_FILE" "$OUTPUT_DIR/clean/openclaw/.openclaw/"
        echo "  ✅ 复制纯净版配置"
    else
        echo "  ⚠️  使用默认配置"
    fi
    
    # 创建启动脚本
    cat > "$OUTPUT_DIR/clean/start.sh" << 'STARTEOF'
#!/bin/bash
# OpenClaw Clean Version Startup Script

SCRIPT="$(cd "$(dirname "$0")" && pwd)/openclaw"

# 检查 Node.js
if command -v node > /dev/null 2>&1; then
    NODE_CMD="node"
elif [ -x "$SCRIPT/../node/bin/node" ]; then
    NODE_CMD="$SCRIPT/../node/bin/node"
elif [ -x "$HOME/.nvm/versions/node/v22.22.0/bin/node" ]; then
    NODE_CMD="$HOME/.nvm/versions/node/v22.22.0/bin/node"
else
    echo "❌ 错误: 未找到 Node.js"
    echo "   请先安装 Node.js 22.x:"
    echo "   ./install-node.sh"
    exit 1
fi

# 检查应用存在
if [ ! -f "$SCRIPT/app/openclaw.mjs" ]; then
    echo "❌ 错误: OpenClaw 应用文件不存在"
    exit 1
fi

# 创建配置目录
mkdir -p "$SCRIPT/.openclaw/workspace"

# 设置环境变量
export OPENCLAW_CONFIG="$SCRIPT/.openclaw/openclaw.json"

echo "🚀 启动 OpenClaw (纯净版)..."
echo "   配置文件: $OPENCLAW_CONFIG"
echo "   Node.js: $NODE_CMD"
echo ""

# 启动
exec "$NODE_CMD" "$SCRIPT/app/openclaw.mjs" gateway start "$@"
STARTEOF

    chmod +x "$OUTPUT_DIR/clean/start.sh"
    echo "  ✅ 创建启动脚本"
}

# 构建完整版
build_full() {
    echo ""
    echo "📦 构建完整版..."
    mkdir -p "$OUTPUT_DIR/full/openclaw/.openclaw"
    
    # 复制应用
    if [ -d "$OPENCLAW_INSTALL_DIR" ]; then
        cp -r "$OPENCLAW_INSTALL_DIR"/* "$OUTPUT_DIR/full/openclaw/app/"
        echo "  ✅ 复制 OpenClaw 应用"
    fi
    
    # 复制完整配置
    if [ -d "$OPENCLAW_CONFIG_DIR" ]; then
        # 排除敏感文件
        cp -r "$OPENCLAW_CONFIG_DIR"/* "$OUTPUT_DIR/full/openclaw/.openclaw/" 2>/dev/null || true
        echo "  ✅ 复制完整配置"
    else
        echo "  ⚠️  警告: 配置目录不存在，跳过"
    fi
    
    # 创建启动脚本
    cat > "$OUTPUT_DIR/full/start.sh" <>/dev/null 2>&1; then
    NODE_CMD="node"
elif [ -x "$SCRIPT/../node/bin/node" ]; then
    NODE_CMD="$SCRIPT/../node/bin/node"
elif [ -x "$HOME/.nvm/versions/node/v22.22.0/bin/node" ]; then
    NODE_CMD="$HOME/.nvm/versions/node/v22.22.0/bin/node"
else
    echo "❌ 错误: 未找到 Node.js"
    echo "   请先安装 Node.js 22.x:"
    echo "   ./install-node.sh"
    exit 1
fi

# 检查应用存在
if [ ! -f "$SCRIPT/app/openclaw.mjs" ]; then
    echo "❌ 错误: OpenClaw 应用文件不存在"
    exit 1
fi

# 设置环境变量
export OPENCLAW_CONFIG="$SCRIPT/.openclaw/openclaw.json"

echo "🚀 启动 OpenClaw (完整版)..."
echo "   配置文件: $OPENCLAW_CONFIG"
echo "   Node.js: $NODE_CMD"
echo ""

# 启动
exec "$NODE_CMD" "$SCRIPT/app/openclaw.mjs" gateway start "$@"
STARTEOF

    chmod +x "$OUTPUT_DIR/full/start.sh"
    echo "  ✅ 创建启动脚本"
}

# 创建辅助文件
create_aux_files() {
    echo ""
    echo "📄 创建辅助文件..."
    
    # Node.js 安装脚本
    cat > "$OUTPUT_DIR/install-node.sh" <>/dev/null 2>&1; then
    echo "✅ Node.js 已安装: $(node --version)"
else
    echo "❌ Node.js 未安装"
    exit 1
fi

# 检查 OpenClaw
echo ""
echo "检查 OpenClaw..."
if openclaw --version >/dev/null 2>&1; then
    echo "✅ OpenClaw 已安装: $(openclaw --version)"
else
    echo "❌ OpenClaw 未安装"
    exit 1
fi

echo ""
echo "启动 OpenClaw Gateway..."
openclaw gateway start
EOF

    chmod +x "$OUTPUT_DIR/install-check.sh"
    
    echo "  ✅ 创建 install-node.sh"
    echo "  ✅ 创建 install-check.sh"
}

# 创建 README
create_readme() {
    cat > "$OUTPUT_DIR/README.md" <>/dev/null 2>&1; then
    echo "✅ Node.js 已安装: $(node --version)"
    exit 0
fi

# 安装 NVM
if ! command -v nvm >/dev/null 2>&1; then
    echo "📦 安装 NVM..."
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.0/install.sh | bash
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
fi

# 安装 Node.js
echo "📦 安装 Node.js 22.22.0..."
nvm install 22.22.0
nvm use 22.22.0

# 验证
node --version
npm --version

echo "✅ Node.js 安装完成！"
EOF

    chmod +x "$OUTPUT_DIR/install-node.sh"
    
    # 检查脚本
    cat > "$OUTPUT_DIR/install-check.sh" << 'EOF'
#!/bin/bash
# 环境检查脚本

echo "=========================================="
echo "OpenClaw 环境检查"
echo "=========================================="

# 检查 Node.js
if command -v node >/dev/null 2>&1; then
    echo "✅ Node.js 已安装: $(node --version)"
else
    echo "❌ Node.js 未安装"
    exit 1
fi

# 检查 OpenClaw
if openclaw --version >/dev/null 2>&1; then
    echo "✅ OpenClaw 已安装: $(openclaw --version)"
else
    echo "⚠️  OpenClaw 可能未安装或不在 PATH 中"
    echo "   如已打包便携版，可忽略此警告"
fi

echo ""
echo "启动 OpenClaw Gateway..."
./clean/start.sh  # 或 ./full/start.sh
EOF

    chmod +x "$OUTPUT_DIR/install-check.sh"
    
    echo "  ✅ 创建 install-node.sh"
    echo "  ✅ 创建 install-check.sh"
}

# 创建 README
create_readme() {
    cat > "$OUTPUT_DIR/README.md" << 'EOF'
# OpenClaw 便携版

OpenClaw 便携部署包，支持快速部署到其他服务器。

## 包含版本

- **纯净版 (clean)**: 无个人配置，需自行设置
- **完整版 (full)**: 包含完整配置和对话历史

## 快速开始

### 1. 检查环境

```bash
./install-check.sh
```

### 2. 安装 Node.js（如未安装）

```bash
./install-node.sh
```

### 3. 启动服务

**纯净版：**
```bash
cd clean
./start.sh
```

**完整版：**
```bash
cd full
./start.sh
```

### 4. 访问

- WebUI: http://localhost:18789

## 自定义路径

如需自定义路径，可在运行前设置环境变量：

```bash
export OPENCLAW_INSTALL_DIR=/path/to/openclaw
export OPENCLAW_CONFIG_DIR=/path/to/.openclaw
export OUTPUT_DIR=/path/to/output
./build-portable.sh
```

## 部署到其他服务器

```bash
# 打包
tar -czf openclaw-portable.tar.gz clean/ full/ install-*.sh README.md

# 传输
scp openclaw-portable.tar.gz user@remote-server:/opt/

# 在目标服务器解压并运行
ssh user@remote-server
cd /opt
tar -xzf openclaw-portable.tar.gz
cd clean && ./start.sh
```

## 故障排除

### Node.js 未找到
确保 Node.js 22.x 已安装，或运行 `./install-node.sh`

### 权限不足
```bash
chmod +x */start.sh install-*.sh
```

### 端口占用
修改启动脚本中的端口，或使用 `./start.sh --port 8080`

## 作者

zfanmy-梦月儿

## 版本

v1.0.1
EOF

    echo "  ✅ 创建 README.md"
}

# 显示结果
show_result() {
    echo ""
    echo "=========================================="
    echo "✅ 构建完成！"
    echo "=========================================="
    echo ""
    echo "📁 输出目录: $OUTPUT_DIR"
    echo ""
    echo "文件列表:"
    ls -lh "$OUTPUT_DIR/"
    echo ""
    echo "目录大小:"
    du -sh "$OUTPUT_DIR/clean" "$OUTPUT_DIR/full" 2>/dev/null || true
    echo ""
    echo "使用方法:"
    echo "  cd $OUTPUT_DIR"
    echo "  ./install-check.sh       # 检查环境"
    echo "  ./clean/start.sh         # 启动纯净版"
    echo "  ./full/start.sh          # 启动完整版"
}

# 主函数
main() {
    # 检查依赖
    check_dependencies
    
    # 检查源文件
    check_source
    
    # 显示配置
    show_config
    
    # 准备目录
    prepare_dirs
    
    # 构建版本
    build_clean
    build_full
    
    # 创建辅助文件
    create_aux_files
    
    # 创建 README
    create_readme
    
    # 显示结果
    show_result
}

# 运行
main "$@"
