package group

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// cronParser 标准 5 字段 cron 解析器，用于验证 cron 表达式
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

type GroupCreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGroupCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupCreateLogic {
	return &GroupCreateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupCreateLogic) GroupCreate(req *types.GroupCreateReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查名称是否为空
	if req.Name == "" {
		resp.Code = 400
		resp.Msg = "群组名称不能为空"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 检查名称是否已存在
	existing, _ := model.GetGroupByName(req.Name)
	if existing != nil {
		resp.Code = 400
		resp.Msg = "群组名称已存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 验证 cron 表达式格式
	if req.CronExpr != "" {
		if _, err := cronParser.Parse(req.CronExpr); err != nil {
			resp.Code = 400
			resp.Msg = "Cron 表达式格式错误: " + err.Error() + "。请使用标准 5 字段格式 (分 时 日 月 周)"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
	}

	// 创建群组
	groupType := req.GroupType
	if groupType == "" {
		groupType = model.GroupTypeContainer
	}
	group := &model.ContainerGroup{
		Name:        req.Name,
		GroupType:   groupType,
		CronExpr:    req.CronExpr,
		AutoUpdate:  req.AutoUpdate,
		CheckUpdate: req.CheckUpdate,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
	}

	id, err := model.CreateGroup(group)
	if err != nil {
		resp.Code = 500
		resp.Msg = "创建群组失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	group.ID = id

	// 如果启用且有 cron 表达式，添加到调度器
	if group.Enabled && group.CronExpr != "" && l.svcCtx.GroupScheduler != nil {
		if err := l.svcCtx.GroupScheduler.AddJob(*group); err != nil {
			l.Errorf("添加群组定时任务失败: %v", err)
		}
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = group
	return resp, nil
}
