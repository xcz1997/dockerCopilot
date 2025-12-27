package progress

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTasksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTasksLogic {
	return &ListTasksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTasksLogic) ListTasks(statusFilter string) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	tasks := l.svcCtx.GetAllTasks()

	// 根据状态过滤
	var filteredTasks []map[string]interface{}
	for _, task := range tasks {
		// 计算状态
		status := "in_progress"
		if task.IsDone {
			if task.Percentage >= 100 {
				status = "completed"
			} else {
				status = "failed"
			}
		}

		// 过滤逻辑
		if statusFilter == "current" && task.IsDone {
			continue
		}
		if statusFilter == "history" && !task.IsDone {
			continue
		}

		filteredTasks = append(filteredTasks, map[string]interface{}{
			"taskId":   task.TaskID,
			"name":     task.Name,
			"progress": task.Percentage,
			"message":  task.Message,
			"status":   status,
		})
	}

	if filteredTasks == nil {
		filteredTasks = []map[string]interface{}{}
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = filteredTasks
	return resp, nil
}
