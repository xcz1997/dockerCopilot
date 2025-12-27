package progress

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProgressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetProgressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProgressLogic {
	return &GetProgressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetProgressLogic) GetProgress(req *types.GetProgressReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	//progress, exists := l.svcCtx.ProgressStore[req.TaskId]
	progress, exists := l.svcCtx.GetProgress(req.TaskId)
	if !exists {
		resp.Code = 400
		resp.Msg = "taskID 未找到"
		resp.Data = map[string]interface{}{}
		return
	}
	// 计算状态：前端期望 status 为 'completed'/'failed'/其他
	status := "in_progress"
	if progress.IsDone {
		if progress.Percentage >= 100 {
			status = "completed"
		} else {
			status = "failed"
		}
	}

	resp.Code = 200
	resp.Msg = progress.Message
	resp.Data = map[string]interface{}{
		"taskId":   progress.TaskID,
		"progress": progress.Percentage,
		"message":  progress.Message,
		"status":   status,
		"name":     progress.Name,
	}
	return resp, nil
}
