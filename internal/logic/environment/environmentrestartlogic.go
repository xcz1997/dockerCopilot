package environment

import (
	"context"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
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
		go func() {
			time.Sleep(time.Second)
			timeout := 10
			err := l.svcCtx.DockerClient.ContainerRestart(context.Background(), containerID, container.StopOptions{
				Timeout: &timeout,
			})
			if err != nil {
				logx.Errorf("重启本地容器失败: %v", err)
			}
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
