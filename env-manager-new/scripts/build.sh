#!/bin/bash
# env-manager 交叉编译脚本

set -e

VERSION=${VERSION:-"0.1.0-dev"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

OUTPUT_DIR="./dist"
mkdir -p "${OUTPUT_DIR}"

echo "Building env-manager ${VERSION}..."

# Linux AMD64
echo "  → linux/amd64"
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${OUTPUT_DIR}/env-manager-linux-amd64" .

# Linux ARM64 (for ARM servers)
echo "  → linux/arm64"
GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${OUTPUT_DIR}/env-manager-linux-arm64" .

# macOS ARM64 (Apple Silicon)
echo "  → darwin/arm64"
GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${OUTPUT_DIR}/env-manager-darwin-arm64" .

# macOS AMD64 (Intel Mac)
echo "  → darwin/amd64"
GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${OUTPUT_DIR}/env-manager-darwin-amd64" .

echo ""
echo "Build complete! Binaries in ${OUTPUT_DIR}/"
ls -lh "${OUTPUT_DIR}/"
