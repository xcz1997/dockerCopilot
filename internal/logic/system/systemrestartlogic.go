package system

import (
	"context"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type SystemRestartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSystemRestartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SystemRestartLogic {
	return &SystemRestartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// getSelfContainerID 获取自身容器 ID
func getSelfContainerID() string {
	// Docker 容器的 hostname 默认就是容器 ID 的前 12 位
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	return hostname
}

// SystemRestart 重启服务（重启自身容器）
func (l *SystemRestartLogic) SystemRestart() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	if l.svcCtx.DockerClient == nil {
		resp.Code = 500
		resp.Msg = "Docker client not initialized"
		return resp, nil
	}

	containerID := getSelfContainerID()
	if containerID == "" {
		resp.Code = 500
		resp.Msg = "无法获取容器 ID，可能不是在 Docker 容器中运行"
		return resp, nil
	}

	// 验证容器是否存在
	_, err = l.svcCtx.DockerClient.ContainerInspect(l.ctx, containerID)
	if err != nil {
		resp.Code = 500
		resp.Msg = "无法找到自身容器: " + err.Error()
		return resp, nil
	}

	logx.Infof("准备重启服务，容器 ID: %s", containerID)

	// 异步执行重启，使用独立的 Docker client 避免 context canceled
	go func() {
		// 等待响应发送
		time.Sleep(time.Second)

		// 创建独立的 Docker client，避免主程序关闭时连接被断开
		dockerHost := os.Getenv("DOCKER_HOST")
		if dockerHost == "" {
			dockerHost = "unix:///var/run/docker.sock"
		}

		cli, err := client.NewClientWithOpts(
			client.WithHost(dockerHost),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			logx.Errorf("创建 Docker client 失败: %v", err)
			return
		}
		defer cli.Close()

		timeout := 10
		err = cli.ContainerRestart(context.Background(), containerID, container.StopOptions{
			Timeout: &timeout,
		})
		if err != nil {
			logx.Errorf("重启容器失败: %v", err)
		}
	}()

	resp.Code = 200
	resp.Msg = "服务正在重启..."
	resp.Data = map[string]interface{}{
		"containerId": containerID,
	}
	return resp, nil
}
