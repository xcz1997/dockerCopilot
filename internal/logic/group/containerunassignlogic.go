package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type ContainerUnassignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContainerUnassignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainerUnassignLogic {
	return &ContainerUnassignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContainerUnassignLogic) ContainerUnassign(req *types.ContainerAssignIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查分配是否存在
	_, err = model.GetGroupContainerByID(req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Code = 404
			resp.Msg = "容器分配不存在"
			resp.Data = map[string]interface{}{}
			return resp, err
		}
		resp.Code = 500
		resp.Msg = "获取容器分配失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	// 删除分配
	if err := model.DeleteGroupContainer(req.Id); err != nil {
		resp.Code = 500
		resp.Msg = "取消分配失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
