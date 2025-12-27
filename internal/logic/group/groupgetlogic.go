package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type GroupGetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGroupGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupGetLogic {
	return &GroupGetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupGetLogic) GroupGet(req *types.GroupIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	groupDetail, err := model.GetGroupWithDetails(req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Code = 404
			resp.Msg = "群组不存在"
			resp.Data = map[string]interface{}{}
			return resp, err
		}
		resp.Code = 500
		resp.Msg = "获取群组详情失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = groupDetail
	return resp, nil
}
