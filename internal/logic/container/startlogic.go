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

type StartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartLogic {
	return &StartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartLogic) Start(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否为远程环境
	if l.svcCtx.CurrentEnvironment != nil && l.svcCtx.CurrentEnvironment.EnvType == model.EnvTypeRemote {
		return l.remoteStart(req)
	}

	err = utiles.StartContainer(l.svcCtx, req.Id)
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

func (l *StartLogic) remoteStart(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	client := l.svcCtx.RemoteClient
	if client == nil {
		env := l.svcCtx.CurrentEnvironment
		client = module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	}

	_, err = client.ProxyRequest("POST", fmt.Sprintf("/api/container/%s/start", req.Id), nil)
	if err != nil {
		resp.Code = 400
		resp.Msg = "远程启动容器失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
