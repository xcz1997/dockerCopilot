# Stage 1: 构建前端（在构建机器架构上运行）
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package*.json ./
RUN npm install --registry=https://registry.npmmirror.com
COPY frontend/ ./
RUN npm run build

# Stage 2: 构建后端（在目标架构上运行）
FROM golang:1.23-alpine AS backend-builder
WORKDIR /src

# 安装 CGO 依赖（go-sqlite3 需要）
RUN apk add --no-cache gcc musl-dev sqlite-dev

ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=1

COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /build/front ./front

# 读取版本号并注入到二进制
RUN VERSION=$(cat version 2>/dev/null || echo "dev") && \
    BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S') && \
    go build -ldflags "-s -w -X 'github.com/xcz1997/dockerCopilot/internal/config.Version=${VERSION}' -X 'github.com/xcz1997/dockerCopilot/internal/config.BuildDate=${BUILD_DATE}'" -o dockerCopilot .

# Stage 3: 运行时镜像
FROM alpine
WORKDIR /app

COPY --from=backend-builder /src/dockerCopilot ./dockerCopilot
COPY etc/. ./etc
COPY start.sh ./start.sh

RUN apk add --no-cache tzdata sqlite-libs \
    && chmod +x start.sh \
    && chmod +x dockerCopilot

ENV secretKey="" \
    DOCKER_HOST="unix:///var/run/docker.sock" \
    DATA_DIR="/data" \
    BACKUP_DIR="/data/backups" \
    TZ="Asia/Shanghai"

VOLUME ["/data"]
CMD ["./start.sh"]
