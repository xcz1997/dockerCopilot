package group

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type HistoryListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHistoryListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HistoryListLogic {
	return &HistoryListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type HistoryListResp struct {
	List  []model.UpdateHistory `json:"list"`
	Total int                   `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

func (l *HistoryListLogic) HistoryList(req *types.HistoryListReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	page := req.Page
	if page < 1 {
		page = 1
	}
	size := req.Size
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var histories []model.UpdateHistory
	var total int

	if req.GroupId > 0 {
		// 获取指定群组的历史
		histories, err = model.GetHistoryByGroupID(req.GroupId, size)
		if err != nil {
			resp.Code = 500
			resp.Msg = "获取历史记录失败: " + err.Error()
			resp.Data = HistoryListResp{}
			return resp, err
		}
		total = len(histories)
	} else {
		// 获取所有历史
		histories, total, err = model.GetHistoryWithPagination(offset, size)
		if err != nil {
			resp.Code = 500
			resp.Msg = "获取历史记录失败: " + err.Error()
			resp.Data = HistoryListResp{}
			return resp, err
		}
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = HistoryListResp{
		List:  histories,
		Total: total,
		Page:  page,
		Size:  size,
	}
	return resp, nil
}
