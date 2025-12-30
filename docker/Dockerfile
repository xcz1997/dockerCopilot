# Stage 1: 构建前端（在构建机器架构上运行）
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package*.json ./
RUN npm install --registry=https://registry.npmmirror.com
COPY frontend/ ./
RUN npm run build

# Stage 2: 构建后端（在构建机器架构上交叉编译，避免 QEMU 模拟）
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# 使用纯 Go 的 SQLite 驱动，避免 CGO 交叉编译问题
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

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

RUN apk add --no-cache tzdata \
    && chmod +x start.sh \
    && chmod +x dockerCopilot

ENV secretKey="" \
    DOCKER_HOST="unix:///var/run/docker.sock" \
    DATA_DIR="/data" \
    BACKUP_DIR="/data/backups" \
    TZ="Asia/Shanghai"

VOLUME ["/data"]
CMD ["./start.sh"]
