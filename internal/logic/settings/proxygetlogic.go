package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyGetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProxyGetLogic {
	return &ProxyGetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyGetLogic) ProxyGet() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	config, err := model.GetProxyConfig()
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 获取环境变量覆盖状态
	envOverride := model.GetProxyEnvOverride()

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"enabled":     config.Enabled,
		"type":        config.Type,
		"host":        config.Host,
		"port":        config.Port,
		"username":    config.Username,
		"password":    config.Password,
		"envOverride": envOverride,
	}
	return resp, nil
}
