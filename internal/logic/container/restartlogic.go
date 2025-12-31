package container

import (
	"context"
	"fmt"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type RestartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRestartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestartLogic {
	return &RestartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RestartLogic) Restart(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否为远程环境
	if l.svcCtx.CurrentEnvironment != nil && l.svcCtx.CurrentEnvironment.EnvType == model.EnvTypeRemote {
		return l.remoteRestart(req)
	}

	err = utiles.RestartContainer(l.svcCtx, req.Id)
	if err != nil {
		resp.Code = 400
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}

func (l *RestartLogic) remoteRestart(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	client := l.svcCtx.RemoteClient
	if client == nil {
		env := l.svcCtx.CurrentEnvironment
		client = module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	}

	_, err = client.ProxyRequest("POST", fmt.Sprintf("/api/container/%s/restart", req.Id), nil)
	if err != nil {
		resp.Code = 400
		resp.Msg = "远程重启容器失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
