package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type GroupUpdateActionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGroupUpdateActionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupUpdateActionLogic {
	return &GroupUpdateActionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupUpdateActionLogic) GroupUpdateAction(req *types.GroupIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查群组是否存在
	group, err := model.GetGroupByID(req.Id)
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

	// 触发强制更新，获取任务ID
	var taskId string
	if l.svcCtx.GroupScheduler != nil {
		taskId = l.svcCtx.GroupScheduler.TriggerGroup(group.ID, true)
	}

	resp.Code = 200
	resp.Msg = "更新任务已触发"
	resp.Data = map[string]interface{}{
		"groupId":   group.ID,
		"groupName": group.Name,
		"taskId":    taskId,
	}
	return resp, nil
}
