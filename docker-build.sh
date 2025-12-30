#!/bin/bash

# Docker 构建脚本（自动递增版本号）
# 用法: ./docker-build.sh [major|minor|patch] [--push] [额外的 docker build 参数]
# 默认递增 patch 版本

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
IMAGE_NAME="muuua/docker-copilot"
PLATFORMS="linux/amd64,linux/arm64"

# 默认参数
BUMP_TYPE="patch"
DO_PUSH=false

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        major|minor|patch)
            BUMP_TYPE="$1"
            shift
            ;;
        --push)
            DO_PUSH=true
            shift
            ;;
        *)
            break
            ;;
    esac
done

# 递增版本号
echo -e "${BLUE}[INFO]${NC} 递增版本号 (${BUMP_TYPE})..."
./bump-version.sh "$BUMP_TYPE"

# 读取新版本
VERSION=$(cat version | tr -d '\n')
echo -e "${GREEN}[INFO]${NC} 当前版本: ${YELLOW}${VERSION}${NC}"

# 构建 Docker 镜像
echo -e "${BLUE}[INFO]${NC} 开始构建 Docker 镜像..."

if [ "$DO_PUSH" = true ]; then
    # 多架构构建并推送
    echo -e "${BLUE}[INFO]${NC} 平台: ${PLATFORMS}"
    echo -e "${BLUE}[INFO]${NC} 多架构构建，直接推送到 Docker Hub..."
    docker buildx build \
        --platform "${PLATFORMS}" \
        -f docker/Dockerfile \
        -t "${IMAGE_NAME}:${VERSION}" \
        -t "${IMAGE_NAME}:latest" \
        --push \
        "$@" \
        .
    echo ""
    echo -e "${GREEN}[SUCCESS]${NC} 构建并推送完成!"
    echo -e "  镜像: ${YELLOW}${IMAGE_NAME}:${VERSION}${NC} (linux/amd64, linux/arm64)"
    echo -e "  镜像: ${YELLOW}${IMAGE_NAME}:latest${NC} (linux/amd64, linux/arm64)"
else
    # 仅构建单架构用于本地测试
    echo -e "${BLUE}[INFO]${NC} 平台: linux/amd64 (本地测试)"
    docker buildx build \
        --platform linux/amd64 \
        -f docker/Dockerfile \
        -t "${IMAGE_NAME}:${VERSION}" \
        -t "${IMAGE_NAME}:latest" \
        --load \
        "$@" \
        .
    echo ""
    echo -e "${GREEN}[SUCCESS]${NC} 构建完成!"
    echo -e "  镜像: ${YELLOW}${IMAGE_NAME}:${VERSION}${NC}"
    echo -e "  镜像: ${YELLOW}${IMAGE_NAME}:latest${NC}"
    echo ""
    echo -e "多架构构建并推送命令:"
    echo -e "  ./docker-build.sh --push"
fi
