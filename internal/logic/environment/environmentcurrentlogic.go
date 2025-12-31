package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentCurrentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentCurrentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentCurrentLogic {
	return &EnvironmentCurrentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentCurrentLogic) EnvironmentCurrent() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取当前环境
	currentEnv := l.svcCtx.CurrentEnvironment
	if currentEnv == nil {
		// 如果没有设置当前环境，尝试获取默认环境
		defaultEnv, err := model.GetDefaultEnvironment()
		if err != nil {
			// 如果没有默认环境，获取本地环境
			localEnv, err := model.GetLocalEnvironment()
			if err != nil {
				resp.Code = 404
				resp.Msg = "未找到任何环境"
				resp.Data = map[string]interface{}{}
				return resp, nil
			}
			currentEnv = localEnv
		} else {
			currentEnv = defaultEnv
		}
		// 设置为当前环境
		l.svcCtx.SwitchToEnvironment(currentEnv)
	}

	// 重新从数据库获取最新数据
	env, err := model.GetEnvironmentByID(currentEnv.ID)
	if err != nil {
		resp.Code = 500
		resp.Msg = "获取环境信息失败"
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = env
	return resp, nil
}
