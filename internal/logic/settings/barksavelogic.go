package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BarkSaveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBarkSaveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BarkSaveLogic {
	return &BarkSaveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BarkSaveLogic) BarkSave(req *types.BarkConfigReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	config := &model.BarkConfig{
		Enabled:    req.Enabled,
		Server:     req.Server,
		Key:        req.Key,
		NotifyMode: req.NotifyMode,
		ShowDetail: req.ShowDetail,
	}

	if err := model.SaveBarkConfig(config); err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "保存成功"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
