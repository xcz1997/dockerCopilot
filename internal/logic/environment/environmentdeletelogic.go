package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentDeleteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentDeleteLogic {
	return &EnvironmentDeleteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentDeleteLogic) EnvironmentDelete(req *types.EnvironmentIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取环境
	env, err := model.GetEnvironmentByID(req.Id)
	if err != nil {
		resp.Code = 404
		resp.Msg = "环境不存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 本地环境不能删除
	if env.EnvType == model.EnvTypeLocal {
		resp.Code = 400
		resp.Msg = "本地环境不能删除"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 如果是当前环境，切换到本地环境
	if l.svcCtx.CurrentEnvironment != nil && l.svcCtx.CurrentEnvironment.ID == req.Id {
		localEnv, _ := model.GetLocalEnvironment()
		if localEnv != nil {
			l.svcCtx.SwitchToEnvironment(localEnv)
		}
	}

	// 删除环境
	if err := model.DeleteEnvironment(req.Id); err != nil {
		resp.Code = 500
		resp.Msg = "删除环境失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
