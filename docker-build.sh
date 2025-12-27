#!/bin/bash

# Docker 构建脚本（自动递增版本号）
# 用法: ./docker-build.sh [major|minor|patch] [额外的 docker build 参数]
# 默认递增 patch 版本

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认 bump 类型
BUMP_TYPE="patch"

# 检查第一个参数是否是 bump type
if [[ "$1" =~ ^(major|minor|patch)$ ]]; then
    BUMP_TYPE="$1"
    shift
fi

# 递增版本号
echo -e "${BLUE}[INFO]${NC} 递增版本号 (${BUMP_TYPE})..."
./bump-version.sh "$BUMP_TYPE"

# 读取新版本
VERSION=$(cat version | tr -d '\n')
echo -e "${GREEN}[INFO]${NC} 当前版本: ${YELLOW}${VERSION}${NC}"

# Docker 镜像名称
IMAGE_NAME="muuua/docker-copilot"

# 构建 Docker 镜像
echo -e "${BLUE}[INFO]${NC} 开始构建 Docker 镜像..."
docker buildx build \
    --platform linux/amd64 \
    -f docker/Dockerfile \
    -t "${IMAGE_NAME}:${VERSION}" \
    -t "${IMAGE_NAME}:latest" \
    "$@" \
    .

echo ""
echo -e "${GREEN}[SUCCESS]${NC} 构建完成!"
echo -e "  镜像: ${YELLOW}${IMAGE_NAME}:${VERSION}${NC}"
echo -e "  镜像: ${YELLOW}${IMAGE_NAME}:latest${NC}"
echo ""
echo -e "推送命令:"
echo -e "  docker push ${IMAGE_NAME}:${VERSION}"
echo -e "  docker push ${IMAGE_NAME}:latest"
