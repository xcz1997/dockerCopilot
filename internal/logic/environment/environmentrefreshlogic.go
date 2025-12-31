package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentRefreshLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentRefreshLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentRefreshLogic {
	return &EnvironmentRefreshLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentRefreshLogic) EnvironmentRefresh(req *types.EnvironmentIdReq) (resp *types.Resp, err error) {
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
		// 本地环境刷新统计
		stats, err := l.svcCtx.GetLocalStats()
		if err != nil {
			resp.Code = 500
			resp.Msg = "获取本地统计信息失败: " + err.Error()
			resp.Data = map[string]interface{}{}
			return resp, err
		}
		if err := model.UpdateEnvironmentStats(env.ID, stats); err != nil {
			resp.Code = 500
			resp.Msg = "更新统计信息失败: " + err.Error()
			resp.Data = map[string]interface{}{}
			return resp, err
		}
	} else {
		// 远程环境刷新统计
		if err := module.RefreshEnvironmentStats(env); err != nil {
			resp.Code = 500
			resp.Msg = "刷新远程环境统计失败: " + err.Error()
			resp.Data = map[string]interface{}{}
			return resp, err
		}
	}

	// 重新获取更新后的环境
	env, _ = model.GetEnvironmentByID(req.Id)

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = env
	return resp, nil
}
