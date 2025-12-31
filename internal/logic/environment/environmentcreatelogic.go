package environment

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnvironmentCreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnvironmentCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnvironmentCreateLogic {
	return &EnvironmentCreateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnvironmentCreateLogic) EnvironmentCreate(req *types.EnvironmentCreateReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 验证名称不为空
	if req.Name == "" {
		resp.Code = 400
		resp.Msg = "环境名称不能为空"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 检查名称是否已存在
	existing, _ := model.GetEnvironmentByName(req.Name)
	if existing != nil {
		resp.Code = 400
		resp.Msg = "环境名称已存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 设置默认类型
	envType := req.EnvType
	if envType == "" {
		envType = model.EnvTypeRemote
	}

	// 远程环境需要 URL
	if envType == model.EnvTypeRemote {
		if req.URL == "" {
			resp.Code = 400
			resp.Msg = "远程环境需要提供 URL"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}

		// 测试连接
		client := module.NewRemoteClient(req.URL, req.SecretKey)
		if err := client.TestConnection(); err != nil {
			resp.Code = 400
			resp.Msg = "连接测试失败: " + err.Error()
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
	}

	// 创建环境
	env := &model.Environment{
		Name:        req.Name,
		Description: req.Description,
		EnvType:     envType,
		URL:         req.URL,
		SecretKey:   req.SecretKey,
		Status:      model.EnvStatusUnknown,
	}

	id, err := model.CreateEnvironment(env)
	if err != nil {
		resp.Code = 500
		resp.Msg = "创建环境失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	env.ID = id

	// 异步刷新统计信息
	go func() {
		if err := module.RefreshEnvironmentStats(env); err != nil {
			l.Errorf("刷新环境统计信息失败: %v", err)
		}
	}()

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = env
	return resp, nil
}
