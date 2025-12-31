package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentTestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentTestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentTestLogic {
	return &EnvironmentTestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentTestLogic) EnvironmentTest(req *types.EnvironmentTestReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	if req.URL == "" {
		resp.Code = 400
		resp.Msg = "URL 不能为空"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 测试连接
	client := module.NewRemoteClient(req.URL, req.SecretKey)
	if err := client.TestConnection(); err != nil {
		resp.Code = 400
		resp.Msg = "连接测试失败: " + err.Error()
		resp.Data = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		return resp, nil
	}

	// 获取统计信息
	stats, err := client.GetStats()
	if err != nil {
		resp.Code = 200
		resp.Msg = "连接成功，但获取统计信息失败"
		resp.Data = map[string]interface{}{
			"success": true,
			"warning": err.Error(),
		}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "连接成功"
	resp.Data = map[string]interface{}{
		"success":        true,
		"containerCount": stats.ContainerCount,
		"runningCount":   stats.RunningCount,
		"stoppedCount":   stats.StoppedCount,
		"imageCount":     stats.ImageCount,
	}
	return resp, nil
}
