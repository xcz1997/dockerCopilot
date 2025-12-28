package progress

import (
	"context"
	"strconv"

	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RetryTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRetryTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RetryTaskLogic {
	return &RetryTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RetryTaskLogic) RetryTask(taskId string) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取原任务信息
	task, exists := l.svcCtx.GetProgress(taskId)
	if !exists {
		resp.Code = 404
		resp.Msg = "任务不存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 检查任务是否已完成且失败
	if !task.IsDone {
		resp.Code = 400
		resp.Msg = "任务正在进行中，无法重试"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	if task.Percentage >= 100 {
		resp.Code = 400
		resp.Msg = "任务已成功完成，无需重试"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 检查任务类型和目标信息
	if task.TaskType == "" || task.TargetID == "" {
		resp.Code = 400
		resp.Msg = "任务缺少重试所需的元数据"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	var newTaskId string

	switch task.TaskType {
	case "group_check":
		groupID, err := strconv.ParseInt(task.TargetID, 10, 64)
		if err != nil {
			resp.Code = 400
			resp.Msg = "无效的群组ID"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
		newTaskId = l.svcCtx.GroupScheduler.TriggerGroup(groupID, false)

	case "group_update":
		groupID, err := strconv.ParseInt(task.TargetID, 10, 64)
		if err != nil {
			resp.Code = 400
			resp.Msg = "无效的群组ID"
			resp.Data = map[string]interface{}{}
			return resp, nil
		}
		newTaskId = l.svcCtx.GroupScheduler.TriggerGroup(groupID, true)

	case "container_update":
		// 容器更新需要调用容器更新逻辑
		resp.Code = 400
		resp.Msg = "容器更新任务请从容器页面重新发起"
		resp.Data = map[string]interface{}{}
		return resp, nil

	default:
		resp.Code = 400
		resp.Msg = "不支持重试的任务类型: " + task.TaskType
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "已创建重试任务"
	resp.Data = map[string]interface{}{
		"newTaskId":   newTaskId,
		"oldTaskId":   taskId,
		"taskType":    task.TaskType,
		"targetId":    task.TargetID,
		"targetName":  task.TargetName,
	}
	return resp, nil
}
