package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegistryTestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegistryTestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegistryTestLogic {
	return &RegistryTestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegistryTestLogic) RegistryTest(req *types.RegistryTestReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	if req.Address == "" {
		resp.Code = 400
		resp.Msg = "地址不能为空"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 测试 Registry 地址连通性
	if err := module.TestRegistryMirror(req.Address); err != nil {
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
