package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type RuleDeleteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRuleDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RuleDeleteLogic {
	return &RuleDeleteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RuleDeleteLogic) RuleDelete(req *types.RuleIdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查规则是否存在
	_, err = model.GetRuleByID(req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			resp.Code = 404
			resp.Msg = "规则不存在"
			resp.Data = map[string]interface{}{}
			return resp, err
		}
		resp.Code = 500
		resp.Msg = "获取规则失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	// 删除规则
	if err := model.DeleteRule(req.Id); err != nil {
		resp.Code = 500
		resp.Msg = "删除规则失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
