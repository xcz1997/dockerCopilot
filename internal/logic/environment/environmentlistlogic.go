package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentListLogic {
	return &EnvironmentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentListLogic) EnvironmentList() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	environments, err := model.GetAllEnvironments()
	if err != nil {
		resp.Code = 500
		resp.Msg = "获取环境列表失败: " + err.Error()
		resp.Data = []interface{}{}
		return resp, err
	}

	// 如果没有环境，返回空数组
	if environments == nil {
		environments = []model.Environment{}
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = environments
	return resp, nil
}
