package group

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/scheduler"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type RulePreviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRulePreviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RulePreviewLogic {
	return &RulePreviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RulePreviewLogic) RulePreview(req *types.RulePreviewReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

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
		resp.Data = []interface{}{}
		return resp, nil
	}

	// 使用匹配器预览
	matcher := scheduler.NewMatcher(l.svcCtx.DockerClient)
	matched, err := matcher.PreviewRuleMatches(l.ctx, ruleType, req.Pattern)
	if err != nil {
		resp.Code = 500
		resp.Msg = "预览匹配失败: " + err.Error()
		resp.Data = []interface{}{}
		return resp, err
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = matched
	return resp, nil
}
