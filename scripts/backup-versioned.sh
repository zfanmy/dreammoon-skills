#!/bin/bash
# backup-versioned.sh - Backup only when version changes
# Usage: ./backup-versioned.sh <source_dir> <version_file_or_cmd> <backup_dir> [name]

set -e

SOURCE_DIR="${1:-}"
VERSION_SOURCE="${2:-}"
BACKUP_DIR="${3:-}"
BACKUP_NAME="${4:-$(basename "$SOURCE_DIR")}"

# Validate arguments
if [ -z "$SOURCE_DIR" ] || [ -z "$VERSION_SOURCE" ] || [ -z "$BACKUP_DIR" ]; then
    echo "Usage: $0 <source_dir> <version_file_or_cmd> <backup_dir> [name]"
    echo "Example: $0 /opt/app /opt/app/version.txt /backups/app"
    echo "Example: $0 /opt/app 'myapp --version' /backups/app"
    exit 1
fi

# Create backup directory and version tracking
mkdir -p "$BACKUP_DIR"
VERSION_RECORD="$BACKUP_DIR/.version_record"

# Get current version
if [ -f "$VERSION_SOURCE" ]; then
    # Version from file
    CURRENT_VERSION=$(cat "$VERSION_SOURCE" 2>/dev/null || echo "unknown")
elif command -v "$VERSION_SOURCE" >/dev/null 2>1; then
    # Version from command
    CURRENT_VERSION=$($VERSION_SOURCE 2>/dev/null || echo "unknown")
else
    # Try as command string
    CURRENT_VERSION=$(eval "$VERSION_SOURCE" 2>/dev/null || echo "unknown")
fi

# Get last backed up version
if [ -f "$VERSION_RECORD" ]; then
    LAST_VERSION=$(cat "$VERSION_RECORD")
else
    LAST_VERSION=""
fi

# Compare versions
if [ "$CURRENT_VERSION" = "$LAST_VERSION" ]; then
    echo "Version unchanged ($CURRENT_VERSION), skipping backup"
    exit 0
fi

echo "Version changed: $LAST_VERSION -> $CURRENT_VERSION"
echo "Creating backup..."

# Create backup with comprehensive exclusions
tar -czf "${BACKUP_DIR}/${BACKUP_NAME}_${CURRENT_VERSION}.tar.gz" \
    --exclude='*/node_modules' \
    --exclude='*/node_modules/*' \
    --exclude='*/.git' \
    --exclude='*/.git/*' \
    --exclude='*.log' \
    --exclude='*/logs/*.log' \
    --exclude='*/.tmp' \
    --exclude='*/tmp' \
    --exclude='*.tmp' \
    --exclude='*/openclaw-portable-output' \
    --exclude='*/openclaw-portable-output/*' \
    --exclude='*/builds' \
    --exclude='*/builds/*' \
    --exclude='*.pdf' \
    --exclude='*.tar.gz' \
    --exclude='*.tar' \
    --exclude='*.zip' \
    -C "$(dirname "$SOURCE_DIR")" \
    "$(basename "$SOURCE_DIR")"

# Update version record
echo "$CURRENT_VERSION" > "$VERSION_RECORD"
echo "✓ Version record updated"
echo "✓ Backup created: ${BACKUP_DIR}/${BACKUP_NAME}_${CURRENT_VERSION}.tar.gz"