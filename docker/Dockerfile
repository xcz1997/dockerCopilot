# Stage 1: 构建前端
FROM --platform=linux/amd64 node:20-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package*.json ./
RUN npm install --registry=https://registry.npmmirror.com
COPY frontend/ ./
RUN npm run build

# Stage 2: 构建后端
FROM --platform=linux/amd64 golang:1.23-alpine AS backend-builder
WORKDIR /src
# 安装 CGO 依赖（go-sqlite3 需要）
RUN apk add --no-cache gcc musl-dev sqlite-dev
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /build/front ./front
RUN go build -ldflags "-s -w" -o dockerCopilot .

# Stage 3: 运行时镜像
FROM --platform=linux/amd64 alpine
WORKDIR /app

COPY --from=backend-builder /src/dockerCopilot ./dockerCopilot
COPY etc/. ./etc
COPY start.sh ./start.sh

RUN apk add --no-cache tzdata sqlite-libs \
    && chmod +x start.sh \
    && chmod +x dockerCopilot

ENV secretKey="" \
    DOCKER_HOST="unix:///var/run/docker.sock" \
    BACKUP_DIR="/data/backups" \
    TZ="Asia/Shanghai"

VOLUME ["/data"]
CMD ["./start.sh"]