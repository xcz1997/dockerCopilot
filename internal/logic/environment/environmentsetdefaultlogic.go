package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentSetDefaultLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentSetDefaultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentSetDefaultLogic {
	return &EnvironmentSetDefaultLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentSetDefaultLogic) EnvironmentSetDefault(req *types.EnvironmentIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查环境是否存在
	env, err := model.GetEnvironmentByID(req.Id)
	if err != nil {
		resp.Code = 404
		resp.Msg = "环境不存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 设置为默认环境
	if err := model.SetDefaultEnvironment(req.Id); err != nil {
		resp.Code = 500
		resp.Msg = "设置默认环境失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	// 重新获取环境
	env, _ = model.GetEnvironmentByID(req.Id)

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = env
	return resp, nil
}
