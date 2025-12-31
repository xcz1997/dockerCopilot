package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContainerEventSaveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContainerEventSaveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainerEventSaveLogic {
	return &ContainerEventSaveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContainerEventSaveLogic) ContainerEventSave(req *types.ContainerEventConfigReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	config := &model.ContainerEventConfig{
		Enabled:           req.Enabled,
		NotifyOnStart:     req.NotifyOnStart,
		NotifyOnStop:      req.NotifyOnStop,
		NotifyOnDie:       req.NotifyOnDie,
		NotifyOnRestart:   req.NotifyOnRestart,
		NotifyOnCreate:    req.NotifyOnCreate,
		NotifyOnDestroy:   req.NotifyOnDestroy,
		NotifyOnHealthy:   req.NotifyOnHealthy,
		NotifyOnUnhealthy: req.NotifyOnUnhealthy,
	}

	if err := model.SaveContainerEventConfig(config); err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "保存成功"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
