package group

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/scheduler"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type GroupStopLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGroupStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupStopLogic {
	return &GroupStopLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupStopLogic) GroupStop(req *types.GroupIdReq) (resp *types.Resp, err error) {
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

	// 仅容器类型群组支持此操作
	if group.GroupType != model.GroupTypeContainer {
		resp.Code = 400
		resp.Msg = "仅容器类型群组支持此操作"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 获取匹配的容器
	matcher := scheduler.NewMatcher(l.svcCtx.DockerClient)
	containers, err := matcher.GetMatchedContainers(l.ctx, req.Id)
	if err != nil {
		resp.Code = 500
		resp.Msg = "获取匹配容器失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	if len(containers) == 0 {
		resp.Code = 200
		resp.Msg = "没有匹配的容器"
		resp.Data = map[string]interface{}{
			"groupId":   group.ID,
			"groupName": group.Name,
			"count":     0,
		}
		return resp, nil
	}

	// 停止所有运行中的容器
	successCount := 0
	failedCount := 0
	skippedCount := 0
	var results []map[string]interface{}

	for _, c := range containers {
		// 跳过已停止的容器
		if c.State != "running" {
			skippedCount++
			results = append(results, map[string]interface{}{
				"id":      c.ID,
				"name":    c.Name,
				"status":  "skipped",
				"message": "容器已停止",
			})
			continue
		}

		err := l.svcCtx.DockerClient.ContainerStop(l.ctx, c.ID, container.StopOptions{})
		if err != nil {
			l.Errorf("停止容器 %s 失败: %v", c.Name, err)
			failedCount++
			results = append(results, map[string]interface{}{
				"id":      c.ID,
				"name":    c.Name,
				"status":  "failed",
				"message": err.Error(),
			})
		} else {
			l.Infof("停止容器 %s 成功", c.Name)
			successCount++
			results = append(results, map[string]interface{}{
				"id":     c.ID,
				"name":   c.Name,
				"status": "success",
			})
		}
	}

	resp.Code = 200
	resp.Msg = fmt.Sprintf("停止完成: %d 成功, %d 失败, %d 跳过", successCount, failedCount, skippedCount)
	resp.Data = map[string]interface{}{
		"groupId":      group.ID,
		"groupName":    group.Name,
		"total":        len(containers),
		"successCount": successCount,
		"failedCount":  failedCount,
		"skippedCount": skippedCount,
		"results":      results,
	}
	return resp, nil
}
