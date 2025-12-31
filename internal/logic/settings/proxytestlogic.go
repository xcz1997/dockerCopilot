package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyTestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyTestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProxyTestLogic {
	return &ProxyTestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyTestLogic) ProxyTest(req *types.ProxyTestReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 验证参数
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

	// 测试代理连通性
	if err := module.TestProxy(req.Type, req.Host, req.Port, req.Username, req.Password); err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{
			"success": false,
		}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "连接成功"
	resp.Data = map[string]interface{}{
		"success": true,
	}
	return resp, nil
}
