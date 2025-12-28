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

	// 统计全部任务的完成/失败数
	totalCompleted := 0
	totalFailed := 0
	totalInProgress := 0

	for _, task := range tasks {
		if task.IsDone {
			if task.Percentage >= 100 {
				totalCompleted++
			} else {
				totalFailed++
			}
		} else {
			totalInProgress++
		}
	}

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

		taskData := map[string]interface{}{
			"taskId":     task.TaskID,
			"name":       task.Name,
			"progress":   task.Percentage,
			"message":    task.Message,
			"detailMsg":  task.DetailMsg,
			"status":     status,
			"startedAt":  task.StartedAt.Format("2006-01-02 15:04:05"),
			"taskType":   task.TaskType,
			"targetId":   task.TargetID,
			"targetName": task.TargetName,
		}

		if task.FinishedAt != nil {
			taskData["finishedAt"] = task.FinishedAt.Format("2006-01-02 15:04:05")
		}

		// 添加子任务
		if len(task.SubTasks) > 0 {
			subTasks := make([]map[string]interface{}, 0, len(task.SubTasks))
			for _, st := range task.SubTasks {
				subTask := map[string]interface{}{
					"name":    st.Name,
					"status":  st.Status,
					"message": st.Message,
				}
				if st.StartedAt != nil {
					subTask["startedAt"] = st.StartedAt.Format("2006-01-02 15:04:05")
				}
				if st.FinishedAt != nil {
					subTask["finishedAt"] = st.FinishedAt.Format("2006-01-02 15:04:05")
				}
				subTasks = append(subTasks, subTask)
			}
			taskData["subTasks"] = subTasks
		}

		filteredTasks = append(filteredTasks, taskData)
	}

	if filteredTasks == nil {
		filteredTasks = []map[string]interface{}{}
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"tasks": filteredTasks,
		"stats": map[string]int{
			"completed":  totalCompleted,
			"failed":     totalFailed,
			"inProgress": totalInProgress,
			"total":      len(tasks),
		},
	}
	return resp, nil
}
