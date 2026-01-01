package container

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type ImageDependencyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewImageDependencyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImageDependencyLogic {
	return &ImageDependencyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ImageDependencyLogic) ImageDependency(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否为远程环境
	if l.svcCtx.CurrentEnvironment != nil && l.svcCtx.CurrentEnvironment.EnvType == model.EnvTypeRemote {
		return l.remoteImageDependency(req)
	}

	info, err := utiles.GetImageDependencyInfo(l.svcCtx, req.Id)
	if err != nil {
		resp.Code = 400
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = info
	return resp, nil
}

func (l *ImageDependencyLogic) remoteImageDependency(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	client := l.svcCtx.RemoteClient
	if client == nil {
		env := l.svcCtx.CurrentEnvironment
		client = module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	}

	data, err := client.ProxyRequest("GET", "/api/container/"+req.Id+"/image-dependency", nil)
	if err != nil {
		resp.Code = 400
		resp.Msg = "获取镜像依赖信息失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = data
	return resp, nil
}
