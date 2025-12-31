package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegistrySaveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegistrySaveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegistrySaveLogic {
	return &RegistrySaveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegistrySaveLogic) RegistrySave(req *types.RegistryMirrorsConfigReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否有环境变量覆盖
	if model.HasRegistryEnvOverride() {
		resp.Code = 403
		resp.Msg = "Registry 配置已通过环境变量设置，无法通过界面修改。请修改 Docker 环境变量后重启容器。"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	config := &model.RegistryMirrorsConfig{
		Enabled: req.Enabled,
		Mirrors: req.Mirrors,
	}

	if err := model.SaveRegistryMirrorsConfig(config); err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
