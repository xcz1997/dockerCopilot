#!/bin/bash

# Docker Copilot 构建脚本
# 使用 Docker 环境编译，支持多平台

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目信息
PROJECT_NAME="dockerCopilot"
GO_VERSION="1.23"
NODE_VERSION="20"

# 获取版本信息
VERSION=${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")}
BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 默认目标平台
DEFAULT_PLATFORMS="linux/amd64,linux/arm64"

# 输出目录
OUTPUT_DIR="./dist"

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    echo "Docker Copilot 构建脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -p, --platforms    目标平台 (默认: $DEFAULT_PLATFORMS)"
    echo "                     可用: linux/amd64, linux/arm64, linux/arm/v7, darwin/amd64, darwin/arm64, windows/amd64"
    echo "  -o, --output       输出目录 (默认: $OUTPUT_DIR)"
    echo "  -v, --version      版本号 (默认: git tag 或 dev)"
    echo "  --skip-frontend    跳过前端构建"
    echo "  --frontend-only    仅构建前端"
    echo "  -h, --help         显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                                    # 构建默认平台"
    echo "  $0 -p linux/amd64                     # 仅构建 linux/amd64"
    echo "  $0 -p \"linux/amd64,linux/arm64\"       # 构建多个平台"
    echo "  $0 --skip-frontend                    # 跳过前端构建"
    echo ""
}

# 解析命令行参数
PLATFORMS="$DEFAULT_PLATFORMS"
SKIP_FRONTEND=false
FRONTEND_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--platforms)
            PLATFORMS="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        --skip-frontend)
            SKIP_FRONTEND=true
            shift
            ;;
        --frontend-only)
            FRONTEND_ONLY=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
done

# 检查 Docker 是否可用
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装或不在 PATH 中"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        print_error "Docker 服务未运行"
        exit 1
    fi

    print_success "Docker 环境检查通过"
}

# 构建前端
build_frontend() {
    print_info "开始构建前端..."

    # 使用 Node Docker 镜像构建前端
    docker run --rm \
        -v "$(pwd)/frontend:/app" \
        -v "$(pwd)/front:/output" \
        -w /app \
        node:${NODE_VERSION}-alpine \
        sh -c "
            npm install --registry=https://registry.npmmirror.com && \
            npm run build && \
            echo 'Frontend build completed'
        "

    if [ -d "./front" ] && [ -f "./front/index.html" ]; then
        print_success "前端构建完成"
    else
        print_error "前端构建失败"
        exit 1
    fi
}

# 构建 Go 程序
build_go() {
    local platform=$1
    local os=$(echo $platform | cut -d'/' -f1)
    local arch=$(echo $platform | cut -d'/' -f2)
    local arm_version=""

    # 处理 ARM 版本
    if [[ "$arch" == "arm" ]]; then
        arm_version=$(echo $platform | cut -d'/' -f3)
        if [[ -z "$arm_version" ]]; then
            arm_version="v7"
        fi
    fi

    # 设置输出文件名
    local output_name="${PROJECT_NAME}_${VERSION}_${os}_${arch}"
    if [[ -n "$arm_version" ]]; then
        output_name="${output_name}_${arm_version}"
    fi
    if [[ "$os" == "windows" ]]; then
        output_name="${output_name}.exe"
    fi

    print_info "构建 ${os}/${arch}${arm_version:+/$arm_version}..."

    # 设置 GOARM
    local goarm_env=""
    if [[ "$arch" == "arm" ]]; then
        goarm_env="-e GOARM=${arm_version#v}"
    fi

    # 使用 Go Docker 镜像编译
    docker run --rm \
        -v "$(pwd):/src" \
        -w /src \
        -e GOOS=$os \
        -e GOARCH=$arch \
        $goarm_env \
        -e CGO_ENABLED=0 \
        -e GOPROXY=https://goproxy.cn,direct \
        golang:${GO_VERSION}-alpine \
        go build -ldflags "-s -w -X 'github.com/xcz1997/dockerCopilot/internal/config.Version=${VERSION}' -X 'github.com/xcz1997/dockerCopilot/internal/config.BuildDate=${BUILD_DATE}' -X 'github.com/xcz1997/dockerCopilot/internal/config.GitCommit=${GIT_COMMIT}'" \
        -o "${OUTPUT_DIR}/${output_name}" \
        .

    if [ -f "${OUTPUT_DIR}/${output_name}" ]; then
        local size=$(ls -lh "${OUTPUT_DIR}/${output_name}" | awk '{print $5}')
        print_success "构建完成: ${output_name} (${size})"
    else
        print_error "构建失败: ${os}/${arch}"
        return 1
    fi
}

# 主函数
main() {
    echo ""
    echo "======================================"
    echo "  Docker Copilot 构建脚本"
    echo "======================================"
    echo ""
    print_info "版本: ${VERSION}"
    print_info "构建时间: ${BUILD_DATE}"
    print_info "Git Commit: ${GIT_COMMIT}"
    print_info "目标平台: ${PLATFORMS}"
    echo ""

    # 检查 Docker
    check_docker

    # 创建输出目录
    mkdir -p "$OUTPUT_DIR"

    # 构建前端
    if [ "$SKIP_FRONTEND" = false ]; then
        build_frontend
    else
        print_warning "跳过前端构建"
    fi

    # 如果仅构建前端，则退出
    if [ "$FRONTEND_ONLY" = true ]; then
        print_success "前端构建完成"
        exit 0
    fi

    # 检查 front 目录是否存在
    if [ ! -d "./front" ] || [ ! -f "./front/index.html" ]; then
        print_error "前端文件不存在，请先构建前端或移除 --skip-frontend 选项"
        exit 1
    fi

    # 构建各平台
    echo ""
    print_info "开始编译 Go 程序..."

    IFS=',' read -ra PLATFORM_ARRAY <<< "$PLATFORMS"
    for platform in "${PLATFORM_ARRAY[@]}"; do
        platform=$(echo "$platform" | xargs) # 去除空格
        build_go "$platform"
    done

    echo ""
    echo "======================================"
    print_success "所有构建完成!"
    echo "======================================"
    echo ""
    print_info "输出目录: ${OUTPUT_DIR}"
    ls -lh "${OUTPUT_DIR}/"
    echo ""
}

# 运行主函数
main
