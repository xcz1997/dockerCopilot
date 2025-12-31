package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentConnectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentConnectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentConnectLogic {
	return &EnvironmentConnectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentConnectLogic) EnvironmentConnect(req *types.EnvironmentIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取环境
	env, err := model.GetEnvironmentByID(req.Id)
	if err != nil {
		resp.Code = 404
		resp.Msg = "环境不存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 如果是远程环境，测试连接
	if env.EnvType == model.EnvTypeRemote {
		client := module.NewRemoteClient(env.URL, env.SecretKey)
		if err := client.TestConnection(); err != nil {
			// 更新状态为离线
			model.UpdateEnvironmentStatus(env.ID, model.EnvStatusOffline, err.Error())
			resp.Code = 400
			resp.Msg = "无法连接到远程环境: " + err.Error()
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
		// 更新状态为在线
		model.UpdateEnvironmentStatus(env.ID, model.EnvStatusOnline, "")
	}

	// 切换到该环境
	l.svcCtx.SwitchToEnvironment(env)

	// 重新获取环境（状态可能已更新）
	env, _ = model.GetEnvironmentByID(req.Id)

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = env
	return resp, nil
}
