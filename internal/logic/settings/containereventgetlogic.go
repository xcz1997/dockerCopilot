package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContainerEventGetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContainerEventGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainerEventGetLogic {
	return &ContainerEventGetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContainerEventGetLogic) ContainerEventGet() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	config, err := model.GetContainerEventConfig()
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"enabled":           config.Enabled,
		"notifyOnStart":     config.NotifyOnStart,
		"notifyOnStop":      config.NotifyOnStop,
		"notifyOnDie":       config.NotifyOnDie,
		"notifyOnRestart":   config.NotifyOnRestart,
		"notifyOnCreate":    config.NotifyOnCreate,
		"notifyOnDestroy":   config.NotifyOnDestroy,
		"notifyOnHealthy":   config.NotifyOnHealthy,
		"notifyOnUnhealthy": config.NotifyOnUnhealthy,
	}
	return resp, nil
}
