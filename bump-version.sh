#!/bin/bash

# 版本号递增脚本
# 用法: ./bump-version.sh [major|minor|patch]
# 默认递增 patch 版本

set -e

VERSION_FILE="version"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 读取当前版本
if [ ! -f "$VERSION_FILE" ]; then
    echo "v0.0.0" > "$VERSION_FILE"
fi

CURRENT_VERSION=$(cat "$VERSION_FILE" | tr -d '\n')

# 移除 v 前缀
VERSION_NUM=${CURRENT_VERSION#v}

# 解析版本号
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION_NUM"

# 默认递增 patch
BUMP_TYPE=${1:-patch}

case $BUMP_TYPE in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    *)
        echo "用法: $0 [major|minor|patch]"
        exit 1
        ;;
esac

NEW_VERSION="v${MAJOR}.${MINOR}.${PATCH}"

# 写入新版本
echo "$NEW_VERSION" > "$VERSION_FILE"

echo -e "${GREEN}版本号已更新:${NC} ${YELLOW}${CURRENT_VERSION}${NC} -> ${GREEN}${NEW_VERSION}${NC}"
echo "$NEW_VERSION"
