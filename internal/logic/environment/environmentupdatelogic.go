package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentUpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentUpdateLogic {
	return &EnvironmentUpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentUpdateLogic) EnvironmentUpdate(req *types.EnvironmentUpdateReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取现有环境
	env, err := model.GetEnvironmentByID(req.Id)
	if err != nil {
		resp.Code = 404
		resp.Msg = "环境不存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 本地环境不允许修改类型和 URL
	if env.EnvType == model.EnvTypeLocal {
		if req.Name != "" {
			env.Name = req.Name
		}
		if req.Description != "" {
			env.Description = req.Description
		}
		if err := model.UpdateEnvironment(env); err != nil {
			resp.Code = 500
			resp.Msg = "更新环境失败: " + err.Error()
			resp.Data = map[string]interface{}{}
			return resp, err
		}
	} else {
		// 远程环境
		if req.Name != "" {
			// 检查名称是否已被其他环境使用
			existing, _ := model.GetEnvironmentByName(req.Name)
			if existing != nil && existing.ID != req.Id {
				resp.Code = 400
				resp.Msg = "环境名称已存在"
				resp.Data = map[string]interface{}{}
				return resp, nil
			}
			env.Name = req.Name
		}
		if req.Description != "" {
			env.Description = req.Description
		}
		if req.URL != "" {
			env.URL = req.URL
		}

		// 如果提供了新的 SecretKey，更新它
		if req.SecretKey != "" {
			env.SecretKey = req.SecretKey
			// 测试新的连接
			client := module.NewRemoteClient(env.URL, env.SecretKey)
			if err := client.TestConnection(); err != nil {
				resp.Code = 400
				resp.Msg = "连接测试失败: " + err.Error()
				resp.Data = map[string]interface{}{}
				return resp, nil
			}
			if err := model.UpdateEnvironmentWithSecret(env); err != nil {
				resp.Code = 500
				resp.Msg = "更新环境失败: " + err.Error()
				resp.Data = map[string]interface{}{}
				return resp, err
			}
		} else {
			if err := model.UpdateEnvironment(env); err != nil {
				resp.Code = 500
				resp.Msg = "更新环境失败: " + err.Error()
				resp.Data = map[string]interface{}{}
				return resp, err
			}
		}
	}

	// 重新获取更新后的环境
	env, _ = model.GetEnvironmentByID(req.Id)

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = env
	return resp, nil
}
