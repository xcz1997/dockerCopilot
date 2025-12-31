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
- 容器网络信息显示（端口映射、网络模式、内部 IP）
- Bark 推送通知（镜像更新、群组任务完成）
- 容器状态变化实时监控与通知

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

### Registry 镜像加速配置

支持自定义 Docker Registry 镜像地址，解决国内访问 Docker Hub 慢的问题：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `REGISTRY_MIRRORS_ENABLED` | `false` | 是否启用自定义镜像地址 |
| `REGISTRY_MIRRORS` | - | 镜像地址列表，逗号分隔（如 `docker.m.daocloud.io,docker.1ms.run`） |

**镜像加速示例：**

```yaml
environment:
  - REGISTRY_MIRRORS_ENABLED=true
  - REGISTRY_MIRRORS=docker.m.daocloud.io,docker.1ms.run
```

> 优先级：用户配置 > 官方 Docker Hub > 内置加速器列表

### 网络代理配置

支持 HTTP/HTTPS/SOCKS5 代理，应用于所有 HTTP 请求：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `PROXY_ENABLED` | `false` | 是否启用代理 |
| `PROXY_TYPE` | `http` | 代理类型：`http`、`https`、`socks5` |
| `PROXY_HOST` | - | 代理服务器地址 |
| `PROXY_PORT` | - | 代理端口 |
| `PROXY_USER` | - | 代理用户名（可选） |
| `PROXY_PASS` | - | 代理密码（可选） |

**代理配置示例：**

```yaml
environment:
  - PROXY_ENABLED=true
  - PROXY_TYPE=http
  - PROXY_HOST=192.168.1.1
  - PROXY_PORT=7890
```

> 代理连接失败时会自动回退到直连。

### 性能配置

适用于 NAS、树莓派等低性能设备优化：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `LOW_POWER_MODE` | `false` | 低性能模式，启用后所有并发操作改为顺序执行 |
| `MAX_CONCURRENT_CHECKS` | `10` | 镜像检查最大并发数（范围 1-20，低性能模式自动设为 1） |
| `CHECK_INTERVAL_MINUTES` | `30` | 镜像自动检查间隔（分钟），设为 `0` 禁用自动检查 |
| `DISABLE_AUTO_CHECK` | `false` | 禁用启动时自动检查镜像更新 |

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
      - LOW_POWER_MODE=true
      - MAX_CONCURRENT_CHECKS=1
      - CHECK_INTERVAL_MINUTES=60
```

### 配置优先级

所有配置支持三种方式设置，优先级从高到低：

1. **环境变量**（最高优先级）
2. **Web 界面设置**（保存到数据库）
3. **默认值**

环境变量设置会覆盖数据库中的配置，适合 Docker 部署时固定配置。

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

## 推送通知

DockerCopilot 支持通过 [Bark](https://github.com/Finb/Bark) 发送推送通知到 iOS 设备。

### Bark 推送配置

在设置页面配置 Bark 推送：

1. 在 iOS 设备上安装 Bark App
2. 获取推送密钥（App 首页显示）
3. 在 DockerCopilot 设置页面填写：
   - **服务器地址**：默认 `https://api.day.app`，可自建服务器
   - **推送密钥**：Bark App 中获取的密钥
   - **通知模式**：
     - 始终通知：所有事件都发送通知
     - 仅失败时通知：只在操作失败时通知
     - 仅全部成功时通知：只在全部成功时通知
   - **显示具体明细**：在通知中显示每个容器的详细结果

### 容器状态变化通知

实时监控 Docker 容器状态变化，在以下事件发生时发送通知：

| 事件类型 | 说明 |
|---------|------|
| 容器启动 | 容器 start 事件 |
| 容器停止 | 容器正常 stop 事件 |
| 异常退出 | 容器 die 事件（推荐开启） |
| 容器重启 | 容器 restart 事件 |
| 容器创建 | 新容器 create 事件 |
| 容器删除 | 容器 destroy 事件 |
| 健康检查通过 | 容器 health_status: healthy 事件 |
| 健康检查失败 | 容器 health_status: unhealthy 事件（推荐开启） |

在设置页面的「容器状态变化通知」区域可以：
- 开启/关闭状态监控总开关
- 选择需要通知的事件类型

> **注意**：容器状态变化通知依赖 Bark 推送配置，请先完成 Bark 配置。

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
