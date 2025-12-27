package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

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
