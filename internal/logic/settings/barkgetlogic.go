package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BarkGetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBarkGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BarkGetLogic {
	return &BarkGetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BarkGetLogic) BarkGet() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	config, err := model.GetBarkConfig()
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"enabled": config.Enabled,
		"server":  config.Server,
		"key":     config.Key,
	}
	return resp, nil
}
