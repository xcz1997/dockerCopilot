#!/bin/bash

# Docker Copilot 本地开发启动脚本

set -e

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 默认密钥（仅用于开发测试）
DEFAULT_SECRET_KEY="dev-secret-key-12345"

# 检查是否设置了密钥
if [ -z "$secretKey" ]; then
    echo -e "${YELLOW}[提示]${NC} 未设置 secretKey 环境变量，使用默认开发密钥"
    export secretKey="$DEFAULT_SECRET_KEY"
fi

# 本地开发使用 ./data 目录
if [ -z "$DATA_DIR" ]; then
    export DATA_DIR="./data"
fi

echo ""
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}  Docker Copilot 开发模式${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo -e "${GREEN}访问地址:${NC} http://localhost:12712/manager"
echo -e "${GREEN}登录密钥:${NC} $secretKey"
echo ""

# 检查 front 目录
if [ ! -d "./front" ] || [ ! -f "./front/index.html" ]; then
    echo -e "${YELLOW}[提示]${NC} 前端文件不存在，正在构建..."
    cd frontend && npm install && npm run build && cd ..
fi

# 运行
go run .
