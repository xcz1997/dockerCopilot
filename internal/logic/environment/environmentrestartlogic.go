package environment

import (
	"context"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentRestartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentRestartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentRestartLogic {
	return &EnvironmentRestartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// getSelfContainerID 获取自身容器 ID
func getSelfContainerID() string {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	return hostname
}

func (l *EnvironmentRestartLogic) EnvironmentRestart(req *types.EnvironmentIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取环境
	env, err := model.GetEnvironmentByID(req.Id)
	if err != nil {
		resp.Code = 404
		resp.Msg = "环境不存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	if env.EnvType == model.EnvTypeLocal {
		// 本地环境：重启自身容器
		if l.svcCtx.DockerClient == nil {
			resp.Code = 500
			resp.Msg = "Docker client not initialized"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}

		containerID := getSelfContainerID()
		if containerID == "" {
			resp.Code = 500
			resp.Msg = "无法获取容器 ID，可能不是在 Docker 容器中运行"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}

		// 验证容器是否存在
		_, err = l.svcCtx.DockerClient.ContainerInspect(l.ctx, containerID)
		if err != nil {
			resp.Code = 500
			resp.Msg = "无法找到自身容器: " + err.Error()
			resp.Data = map[string]interface{}{}
			return resp, nil
		}

		logx.Infof("准备重启本地服务，容器 ID: %s", containerID)

		// 异步执行重启
		// 使用 ContainerStop 而非 ContainerRestart，让 Docker 的 restart policy 自动重启容器
		// 这样更可靠，因为 ContainerRestart 在容器重启自己时可能会有竞态条件
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
			// 注意：不使用 defer cli.Close()，因为容器会被停止，这个 goroutine 不会正常结束

			// 使用 Stop 而非 Restart，依赖容器的 restart: always 策略自动重启
			timeout := 10
			err = cli.ContainerStop(context.Background(), containerID, container.StopOptions{
				Timeout: &timeout,
			})
			if err != nil {
				logx.Errorf("停止本地容器失败: %v", err)
			}
			// 容器停止后，Docker daemon 会根据 restart policy 自动重启
		}()

		resp.Code = 200
		resp.Msg = "本地服务正在重启..."
		resp.Data = map[string]interface{}{
			"containerId": containerID,
			"envType":     "local",
		}
	} else {
		// 远程环境：调用远程 API
		client := module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)

		logx.Infof("准备重启远程服务: %s (%s)", env.Name, env.URL)

		// 远程重启可能会导致连接断开，这是正常的
		go func() {
			if err := client.Restart(); err != nil {
				// 连接断开是预期行为，不需要记录错误
				logx.Infof("远程服务 %s 重启请求已发送（连接断开是正常的）", env.Name)
			}
		}()

		resp.Code = 200
		resp.Msg = "远程服务正在重启..."
		resp.Data = map[string]interface{}{
			"envName": env.Name,
			"envType": "remote",
		}
	}

	return resp, nil
}
