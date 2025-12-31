package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PerformanceSaveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPerformanceSaveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PerformanceSaveLogic {
	return &PerformanceSaveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PerformanceSaveLogic) PerformanceSave(req *types.PerformanceConfigReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否有环境变量覆盖
	if model.HasPerformanceEnvOverride() {
		resp.Code = 403
		resp.Msg = "性能配置已通过环境变量设置，无法通过界面修改。请修改 Docker 环境变量后重启容器。"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	config := &model.PerformanceConfig{
		LowPowerMode:         req.LowPowerMode,
		MaxConcurrentChecks:  req.MaxConcurrentChecks,
		CheckIntervalMinutes: req.CheckIntervalMinutes,
		DisableAutoCheck:     req.DisableAutoCheck,
	}

	if err := model.SavePerformanceConfig(config); err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 更新 ServiceContext 中的配置缓存
	l.svcCtx.UpdatePerformanceConfig(config)

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
