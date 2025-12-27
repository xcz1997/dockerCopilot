#!/bin/bash

# DockerCopilot 发布脚本
# 用法: ./release.sh [major|minor|patch] [--no-push]
# 默认递增 patch 版本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
IMAGE_NAME="muuua/docker-copilot"
PLATFORMS="linux/amd64"

# 默认参数
BUMP_TYPE="patch"
DO_PUSH=true

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        major|minor|patch)
            BUMP_TYPE="$1"
            shift
            ;;
        --no-push)
            DO_PUSH=false
            shift
            ;;
        -h|--help)
            echo "DockerCopilot 发布脚本"
            echo ""
            echo "用法: $0 [major|minor|patch] [--no-push]"
            echo ""
            echo "参数:"
            echo "  major      递增主版本号 (x.0.0)"
            echo "  minor      递增次版本号 (0.x.0)"
            echo "  patch      递增修订号 (0.0.x) [默认]"
            echo "  --no-push  构建后不推送镜像"
            echo ""
            echo "示例:"
            echo "  $0              # 递增 patch 并发布"
            echo "  $0 minor        # 递增 minor 并发布"
            echo "  $0 --no-push    # 只构建不推送"
            exit 0
            ;;
        *)
            echo -e "${RED}未知参数: $1${NC}"
            exit 1
            ;;
    esac
done

print_step() {
    echo ""
    echo -e "${BLUE}===================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}===================================${NC}"
    echo ""
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 切换到项目根目录
cd "$(dirname "$0")"

# Step 1: 递增版本号
print_step "Step 1/5: 递增版本号"
OLD_VERSION=$(cat version | tr -d '\n')
./bump-version.sh "$BUMP_TYPE"
VERSION=$(cat version | tr -d '\n')
print_success "版本号: ${OLD_VERSION} -> ${VERSION}"

# Step 2: 构建前端
print_step "Step 2/5: 构建前端"
cd frontend
print_info "安装依赖..."
npm install --registry=https://registry.npmmirror.com --silent
print_info "构建前端..."
npm run build
cd ..
print_success "前端构建完成"

# Step 3: 构建 Docker 镜像
print_step "Step 3/5: 构建 Docker 镜像"
print_info "平台: ${PLATFORMS}"
print_info "标签: ${IMAGE_NAME}:${VERSION}"
print_info "标签: ${IMAGE_NAME}:latest"

docker buildx build \
    --platform "${PLATFORMS}" \
    -f docker/Dockerfile \
    -t "${IMAGE_NAME}:${VERSION}" \
    -t "${IMAGE_NAME}:latest" \
    --load \
    .

print_success "Docker 镜像构建完成（两个 tag 指向同一镜像）"

# Step 4: 推送镜像
if [ "$DO_PUSH" = true ]; then
    print_step "Step 4/5: 推送镜像"
    print_info "推送 ${IMAGE_NAME}:${VERSION}..."
    docker push "${IMAGE_NAME}:${VERSION}"
    print_success "已推送 ${IMAGE_NAME}:${VERSION}"

    print_info "推送 ${IMAGE_NAME}:latest..."
    docker push "${IMAGE_NAME}:latest"
    print_success "已推送 ${IMAGE_NAME}:latest"

    print_success "镜像推送完成（${VERSION} 和 latest 为同一镜像）"
else
    print_step "Step 4/5: 跳过推送"
    print_info "使用 --no-push 参数，跳过镜像推送"
fi

# Step 5: 提交版本文件
print_step "Step 5/5: 提交版本变更"
if git diff --quiet version; then
    print_info "版本文件无变化，跳过提交"
else
    git add version
    git commit -m "chore: bump version to ${VERSION}"
    print_success "版本变更已提交"

    if [ "$DO_PUSH" = true ]; then
        print_info "推送到远程仓库..."
        git push
        print_success "已推送到远程仓库"
    fi
fi

# 完成
echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  发布完成!${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""
echo -e "版本: ${YELLOW}${VERSION}${NC}"
echo -e "镜像: ${YELLOW}${IMAGE_NAME}:${VERSION}${NC}"
echo -e "镜像: ${YELLOW}${IMAGE_NAME}:latest${NC}"
echo ""

if [ "$DO_PUSH" = false ]; then
    echo -e "手动推送命令:"
    echo -e "  docker push ${IMAGE_NAME}:${VERSION}"
    echo -e "  docker push ${IMAGE_NAME}:latest"
    echo ""
fi
