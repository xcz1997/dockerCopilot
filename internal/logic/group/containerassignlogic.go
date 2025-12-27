package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type ContainerAssignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContainerAssignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainerAssignLogic {
	return &ContainerAssignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContainerAssignLogic) ContainerAssign(req *types.ContainerAssignReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查群组是否存在
	_, err = model.GetGroupByID(req.GroupId)
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

	// 检查容器是否已分配
	inGroup, _ := model.IsContainerInGroup(req.GroupId, req.ContainerId)
	if inGroup {
		resp.Code = 400
		resp.Msg = "容器已在该群组中"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 创建分配
	gc := &model.GroupContainer{
		GroupID:       req.GroupId,
		ContainerID:   req.ContainerId,
		ContainerName: req.ContainerName,
	}

	id, err := model.CreateGroupContainer(gc)
	if err != nil {
		resp.Code = 500
		resp.Msg = "分配容器失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	gc.ID = id

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = gc
	return resp, nil
}
