package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type GroupCheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGroupCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupCheckLogic {
	return &GroupCheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupCheckLogic) GroupCheck(req *types.GroupIdReq) (resp *types.Resp, err error) {
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

	// 触发检查（只检查不更新，忽略群组的autoUpdate设置），获取任务ID
	var taskId string
	if l.svcCtx.GroupScheduler != nil {
		taskId = l.svcCtx.GroupScheduler.TriggerGroup(group.ID, false, true)
	}

	resp.Code = 200
	resp.Msg = "检查任务已触发"
	resp.Data = map[string]interface{}{
		"groupId":   group.ID,
		"groupName": group.Name,
		"taskId":    taskId,
	}
	return resp, nil
}
