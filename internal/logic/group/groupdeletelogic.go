package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type GroupDeleteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGroupDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupDeleteLogic {
	return &GroupDeleteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupDeleteLogic) GroupDelete(req *types.GroupIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查群组是否存在
	_, err = model.GetGroupByID(req.Id)
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

	// 从调度器移除
	if l.svcCtx.GroupScheduler != nil {
		l.svcCtx.GroupScheduler.RemoveJob(req.Id)
	}

	// 删除群组（级联删除规则和容器分配）
	if err := model.DeleteGroup(req.Id); err != nil {
		resp.Code = 500
		resp.Msg = "删除群组失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
