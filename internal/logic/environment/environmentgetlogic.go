package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentGetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentGetLogic {
	return &EnvironmentGetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentGetLogic) EnvironmentGet(req *types.EnvironmentIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	env, err := model.GetEnvironmentByID(req.Id)
	if err != nil {
		resp.Code = 404
		resp.Msg = "环境不存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = env
	return resp, nil
}
