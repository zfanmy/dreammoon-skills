#!/bin/bash
# backup.sh - Create timestamped backup of a directory with improved exclusions
# Usage: ./backup.sh <source_dir> <backup_dir> [name]

set -e

SOURCE_DIR="${1:-}"
BACKUP_DIR="${2:-}"
BACKUP_NAME="${3:-backup}"

# Validate arguments
if [ -z "$SOURCE_DIR" ] || [ -z "$BACKUP_DIR" ]; then
    echo "Usage: $0 <source_dir> <backup_dir> [name]"
    echo "Example: $0 /home/user/data /backups/data mydata"
    exit 1
fi

# Validate source exists
if [ ! -d "$SOURCE_DIR" ]; then
    echo "Error: Source directory does not exist: $SOURCE_DIR"
    exit 1
fi

# Create backup directory if needed
mkdir -p "$BACKUP_DIR"

# Generate timestamp
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/${BACKUP_NAME}_${TIMESTAMP}.tar.gz"

echo "Creating backup..."
echo "  Source: $SOURCE_DIR"
echo "  Destination: $BACKUP_FILE"

# Create backup with comprehensive exclusions
tar -czf "$BACKUP_FILE" \
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

# Verify backup
if [ -f "$BACKUP_FILE" ]; then
    SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo "✓ Backup completed successfully"
    echo "  File: $BACKUP_FILE"
    echo "  Size: $SIZE"
else
    echo "✗ Backup failed"
    exit 1
fi