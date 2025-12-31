package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxySaveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxySaveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProxySaveLogic {
	return &ProxySaveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxySaveLogic) ProxySave(req *types.ProxyConfigReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否有环境变量覆盖
	if model.HasProxyEnvOverride() {
		resp.Code = 403
		resp.Msg = "代理配置已通过环境变量设置，无法通过界面修改。请修改 Docker 环境变量后重启容器。"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 验证代理类型
	if req.Enabled {
		if req.Type != "http" && req.Type != "https" && req.Type != "socks5" {
			resp.Code = 400
			resp.Msg = "代理类型必须是 http、https 或 socks5"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
		if req.Host == "" {
			resp.Code = 400
			resp.Msg = "代理服务器地址不能为空"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
		if req.Port <= 0 || req.Port > 65535 {
			resp.Code = 400
			resp.Msg = "代理端口必须在 1-65535 之间"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
	}

	config := &model.ProxyConfig{
		Enabled:  req.Enabled,
		Type:     req.Type,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	}

	if err := model.SaveProxyConfig(config); err != nil {
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
