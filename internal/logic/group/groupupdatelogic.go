package group

import (
	"context"
	"database/sql"

	"github.com/robfig/cron/v3"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// cronParserUpdate 标准 5 字段 cron 解析器
var cronParserUpdate = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

type GroupUpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGroupUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupUpdateLogic {
	return &GroupUpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupUpdateLogic) GroupUpdate(req *types.GroupUpdateReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取现有群组
	existing, err := model.GetGroupByID(req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Code = 404
			resp.Msg = "群组不存在"
			resp.Data = map[string]interface{}{}
			return resp, err
		}
		resp.Code = 500
		resp.Msg = "获取群组失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	// 验证 cron 表达式格式
	if req.CronExpr != "" {
		if _, err := cronParserUpdate.Parse(req.CronExpr); err != nil {
			resp.Code = 400
			resp.Msg = "Cron 表达式格式错误: " + err.Error() + "。请使用标准 5 字段格式 (分 时 日 月 周)"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
	}

	// 更新字段
	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.CronExpr = req.CronExpr
	existing.AutoUpdate = req.AutoUpdate
	existing.CheckUpdate = req.CheckUpdate
	if req.Priority > 0 {
		existing.Priority = req.Priority
	}
	existing.Enabled = req.Enabled

	// 保存更新
	if err := model.UpdateGroup(existing); err != nil {
		resp.Code = 500
		resp.Msg = "更新群组失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	// 更新调度器
	if l.svcCtx.GroupScheduler != nil {
		if err := l.svcCtx.GroupScheduler.UpdateJob(*existing); err != nil {
			l.Errorf("更新群组定时任务失败: %v", err)
		}
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = existing
	return resp, nil
}
