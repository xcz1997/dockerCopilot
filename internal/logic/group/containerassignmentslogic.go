package group

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/scheduler"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type ContainerAssignmentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContainerAssignmentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainerAssignmentsLogic {
	return &ContainerAssignmentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ContainerAssignment 容器分配信息
type ContainerAssignment struct {
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	GroupID       *int64 `json:"groupId"`
	GroupName     string `json:"groupName"`
}

func (l *ContainerAssignmentsLogic) ContainerAssignments() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	matcher := scheduler.NewMatcher(l.svcCtx.DockerClient)
	assignments, err := matcher.GetAllContainerAssignments(l.ctx)
	if err != nil {
		resp.Code = 500
		resp.Msg = "获取容器分配情况失败: " + err.Error()
		resp.Data = []interface{}{}
		return resp, err
	}

	// 转换为响应格式
	var result []ContainerAssignment
	for containerID, group := range assignments {
		assignment := ContainerAssignment{
			ContainerID: containerID,
		}
		if group != nil {
			assignment.GroupID = &group.ID
			assignment.GroupName = group.Name
		}
		result = append(result, assignment)
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = result
	return resp, nil
}
