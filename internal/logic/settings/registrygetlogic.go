package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegistryGetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegistryGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegistryGetLogic {
	return &RegistryGetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegistryGetLogic) RegistryGet() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	config, err := model.GetRegistryMirrorsConfig()
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 获取环境变量覆盖状态
	envOverride := model.GetRegistryEnvOverride()

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"enabled":     config.Enabled,
		"mirrors":     config.Mirrors,
		"envOverride": envOverride,
	}
	return resp, nil
}
