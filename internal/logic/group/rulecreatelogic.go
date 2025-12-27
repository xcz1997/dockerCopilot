package group

import (
	"context"
	"database/sql"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type RuleCreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRuleCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RuleCreateLogic {
	return &RuleCreateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RuleCreateLogic) RuleCreate(req *types.RuleCreateReq) (resp *types.Resp, err error) {
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

	// 验证规则类型
	ruleType := model.RuleType(req.RuleType)
	validTypes := map[model.RuleType]bool{
		model.RuleTypeNamePrefix:   true,
		model.RuleTypeNameSuffix:   true,
		model.RuleTypeNameRegex:    true,
		model.RuleTypeNameContains: true,
		model.RuleTypeImagePrefix:  true,
		model.RuleTypeImageRegex:   true,
		model.RuleTypeLabelKey:     true,
		model.RuleTypeLabelValue:   true,
	}
	if !validTypes[ruleType] {
		resp.Code = 400
		resp.Msg = "无效的规则类型"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 检查模式是否为空
	if req.Pattern == "" {
		resp.Code = 400
		resp.Msg = "匹配模式不能为空"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 创建规则
	rule := &model.GroupRule{
		GroupID:  req.GroupId,
		RuleType: ruleType,
		Pattern:  req.Pattern,
	}

	id, err := model.CreateRule(rule)
	if err != nil {
		resp.Code = 500
		resp.Msg = "创建规则失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	rule.ID = id

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = rule
	return resp, nil
}
