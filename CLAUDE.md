# DockerCopilot 项目说明

## 项目概述

DockerCopilot 是一个 Docker 容器管理工具，提供 Web UI 来管理 Docker 容器、镜像，支持容器备份/恢复、镜像更新检查等功能。

## 技术栈

- **后端**: Go 1.23 + go-zero 框架
- **前端**: Vue 3 + Vite + Pinia + TailwindCSS
- **数据库**: SQLite3
- **容器化**: Docker 多阶段构建

## 项目结构

```
.
├── dockercopilot.go          # 主入口文件
├── dockercopilot.api         # go-zero API 定义文件
├── internal/
│   ├── config/               # 配置结构定义
│   ├── handler/              # HTTP 处理器 (由 goctl 生成)
│   │   ├── auth/             # 认证相关
│   │   ├── container/        # 容器操作
│   │   ├── image/            # 镜像操作
│   │   ├── group/            # 群组管理
│   │   ├── progress/         # 进度查询
│   │   └── version/          # 版本管理
│   ├── logic/                # 业务逻辑层
│   ├── model/                # 数据模型 (SQLite)
│   ├── module/               # 核心模块
│   │   ├── auth.go           # Docker Registry 认证
│   │   └── checkupdate.go    # 镜像更新检查
│   ├── scheduler/            # 定时任务调度
│   ├── svc/                  # 服务上下文
│   ├── types/                # 类型定义
│   └── utiles/               # 工具函数
├── frontend/                 # Vue 前端源码
│   ├── src/
│   │   ├── api/              # API 封装
│   │   ├── stores/           # Pinia 状态管理
│   │   └── views/            # 页面组件
│   └── package.json
├── front/                    # 前端构建产物 (嵌入后端)
├── docker/
│   └── Dockerfile            # 多阶段构建
├── etc/
│   └── dockerCopilot.yaml    # 配置文件
└── data/                     # 运行时数据目录
```

## 重要注意事项

### API 响应格式

后端 API 统一返回格式：
```json
{
  "code": 200,      // 成功码是 200，不是 0
  "msg": "success",
  "data": {}
}
```

**前端判断成功必须用 `response.code === 200`**

### 镜像更新检查

`internal/module/checkupdate.go` 中的镜像检查逻辑：

1. 跳过自身镜像: `0nlylty/dockercopilot`
2. 跳过无效镜像: `ImageName == "None"` 或 `ImageTag == "None"`
   - 这些是没有 RepoTags 和 RepoDigests 的孤立/dangling 镜像

### 镜像名称解析

`internal/utiles/image.go` 中 `splitImageNameAndTag()` 的逻辑：

| 条件 | ImageName | ImageTag |
|------|-----------|----------|
| 有 RepoTags | RepoTags[0] 的 `:` 前部分 | RepoTags[0] 的 `:` 后部分 |
| 无 RepoTags，有 RepoDigests | RepoDigests[0] 的 `@` 前部分 | "None" |
| 两者都没有 | "None" | "None" |

### JWT 认证

除 `/api/auth` 外，所有 `/api/*` 路由都需要 JWT 认证。

配置在 `etc/dockerCopilot.yaml`:
```yaml
Auth:
  AccessSecret: "your-secret-key"
  AccessExpire: 86400
```

### Docker Registry 认证

支持的 Registry 加速器（`internal/module/auth.go`）:
- docker.1ms.run
- docker.m.daocloud.io
- docker.anye.in
- 等

### 定时任务

程序启动时会：
1. 获取镜像列表并异步检查更新
2. 设置 cron 任务每 30 分钟检查一次镜像更新

## 开发命令

```bash
# 开发模式运行
./dev.sh

# 构建前端
cd frontend && npm run build

# 构建后端
go build -o dockerCopilot .

# 构建 Docker 镜像
docker build -f docker/Dockerfile -t muuua/docker-copilot:latest .

# 推送镜像
docker push muuua/docker-copilot:latest
```

## Release 注意事项

**重要：Release 默认必须构建 linux/amd64 平台**

```bash
# 构建 linux/amd64 Docker 镜像
docker buildx build --platform linux/amd64 -f docker/Dockerfile -t muuua/docker-copilot:latest --push .

# 或使用 GOOS/GOARCH 构建二进制
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dockerCopilot .
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| secretKey | JWT 密钥 | "" |
| DOCKER_HOST | Docker socket 路径 | unix:///var/run/docker.sock |
| DATA_DIR | 数据目录（SQLite 数据库存放位置） | /data |
| BACKUP_DIR | 备份目录 | /data/backups |
| TZ | 时区 | Asia/Shanghai |

## 常见问题

### 镜像列表为空

检查前端 store 中的成功码判断是否为 `response.code === 200`

### 镜像检查报错 "None"

这是正常的，表示存在孤立镜像。可通过 `docker image prune` 清理。

### Registry 404 错误

可能原因：
1. 镜像不存在或已删除
2. Registry 认证问题
3. 网络问题（TLS 超时等）

## Git Commit 规则

**重要：本项目的 commit 信息不需要任何署名信息**

- 不要添加 `Co-Authored-By` 行
- 不要添加 `Generated with` 标记
- 只需要简洁的 commit message 描述变更内容
