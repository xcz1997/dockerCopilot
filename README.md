# dockerCopilot

<a href="https://www.gnu.org/licenses/agpl-3.0.en.html">
    <img alt="License: AGPLv3" src="https://shields.io/badge/License-AGPL%20v3-blue.svg">
</a>

## 介绍

一个主打便捷的 Docker 容器管理工具，支持所有平台。

### 功能特性

- 一键更新容器（自动拉取最新镜像并重建）
- 指定镜像和 tag 更新
- 启动、停止、重启容器
- 重命名容器
- 删除无 TAG 镜像 / 未使用镜像
- 更新进度实时查看
- 容器配置备份与恢复
- 导出为 docker-compose.yml
- 群组管理（批量更新、定时任务）
- 容器日志实时查看

## 快速开始

### Docker Compose 安装（推荐）

创建 `docker-compose.yml` 文件：

```yaml
services:
  dockercopilot:
    image: muuua/docker-copilot:latest
    container_name: dockercopilot
    restart: always
    privileged: true
    network_mode: bridge
    ports:
      - 12712:12712
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
    environment:
      - TZ=Asia/Shanghai
      - secretKey=your_secret_key_here  # 必填，至少8位且非纯数字
```

启动服务：

```bash
docker-compose up -d
```

访问 `http://localhost:12712/manager` 进入管理界面。

### Docker Run 安装

```bash
docker run -d \
  --name dockercopilot \
  --restart always \
  --privileged \
  -p 12712:12712 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v ./data:/data \
  -e TZ=Asia/Shanghai \
  -e secretKey=your_secret_key_here \
  muuua/docker-copilot:latest
```

## 环境变量

### 基础配置

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `secretKey` | ✅ | - | JWT 认证密钥，要求至少 8 位且非纯数字 |
| `TZ` | ❌ | `Asia/Shanghai` | 时区设置，影响定时任务和日志时间 |
| `DATA_DIR` | ❌ | `/data` | 数据目录，存放 SQLite 数据库 |
| `BACKUP_DIR` | ❌ | `/data/backups` | 容器配置备份目录 |
| `DOCKER_HOST` | ❌ | `unix:///var/run/docker.sock` | Docker API 地址 |
| `DelOldContainer` | ❌ | `true` | 更新容器后是否删除旧容器，设为 `false` 保留 |
| `githubProxy` | ❌ | - | GitHub 代理地址，用于检查程序更新 |

### 性能配置

适用于 NAS、树莓派等低性能设备优化：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `PERFORMANCE_LOWPOWERMODE` | `false` | 低性能模式，启用后所有并发操作改为顺序执行 |
| `PERFORMANCE_MAXCONCURRENTCHECKS` | `10` | 镜像检查最大并发数（范围 1-20，低性能模式自动设为 1） |
| `PERFORMANCE_CHECKINTERVALMINUTES` | `30` | 镜像自动检查间隔（分钟），设为 `0` 禁用自动检查 |
| `PERFORMANCE_DISABLEAUTOCHECK` | `false` | 禁用启动时自动检查镜像更新 |

**低性能模式示例：**

```yaml
services:
  dockercopilot:
    image: muuua/docker-copilot:latest
    container_name: dockercopilot
    restart: always
    privileged: true
    ports:
      - 12712:12712
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
    environment:
      - TZ=Asia/Shanghai
      - secretKey=your_secret_key_here
      - PERFORMANCE_LOWPOWERMODE=true
      - PERFORMANCE_MAXCONCURRENTCHECKS=1
      - PERFORMANCE_CHECKINTERVALMINUTES=60
```

## 高级配置

### 自定义备份目录

```yaml
services:
  dockercopilot:
    # ... 其他配置
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
      - /path/to/backups:/backups  # 自定义备份目录
    environment:
      - BACKUP_DIR=/backups
```

### 使用 TCP 连接远程 Docker

```yaml
services:
  dockercopilot:
    # ... 其他配置
    environment:
      - DOCKER_HOST=tcp://192.168.1.100:2375
    # 不需要挂载 docker.sock
```

## 开发环境

- Go 版本：1.23+
- 前端：Vue 3 + Vite + TailwindCSS
- 数据库：SQLite3

```bash
# 开发模式运行
./dev.sh

# 构建前端
cd frontend && npm run build

# 构建后端
go build -o dockerCopilot .
```

## 常见问题

### 无法连接到 Docker

确保已正确挂载 Docker socket：
```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

并且容器具有访问权限（`privileged: true`）。

### 更新后旧容器未删除

默认情况下更新后会删除旧容器。如需保留，设置环境变量：
```yaml
environment:
  - DelOldContainer=false
```

### 定时任务时间不对

检查时区设置是否正确：
```yaml
environment:
  - TZ=Asia/Shanghai
```

## License

[AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.en.html)
