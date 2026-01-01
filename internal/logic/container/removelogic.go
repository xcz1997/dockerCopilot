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

type RemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveLogic {
	return &RemoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveLogic) Remove(req *types.RemoveContainerReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否为远程环境
	if l.svcCtx.CurrentEnvironment != nil && l.svcCtx.CurrentEnvironment.EnvType == model.EnvTypeRemote {
		return l.remoteRemove(req)
	}

	err = utiles.RemoveContainer(l.svcCtx, req.Id, req.Force)
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

func (l *RemoveLogic) remoteRemove(req *types.RemoveContainerReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	client := l.svcCtx.RemoteClient
	if client == nil {
		env := l.svcCtx.CurrentEnvironment
		client = module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	}

	url := fmt.Sprintf("/api/container/%s", req.Id)
	if req.Force {
		url += "?force=true"
	}

	_, err = client.ProxyRequest("DELETE", url, nil)
	if err != nil {
		resp.Code = 400
		resp.Msg = "远程删除容器失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
