package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PerformanceGetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPerformanceGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PerformanceGetLogic {
	return &PerformanceGetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PerformanceGetLogic) PerformanceGet() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	config, err := model.GetPerformanceConfig()
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 获取环境变量覆盖状态
	envOverride := model.GetPerformanceEnvOverride()

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"lowPowerMode":         config.LowPowerMode,
		"maxConcurrentChecks":  config.MaxConcurrentChecks,
		"checkIntervalMinutes": config.CheckIntervalMinutes,
		"disableAutoCheck":     config.DisableAutoCheck,
		"envOverride":          envOverride,
	}
	return resp, nil
}
