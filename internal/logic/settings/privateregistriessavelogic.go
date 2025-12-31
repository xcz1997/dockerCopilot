package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type PrivateRegistriesSaveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPrivateRegistriesSaveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PrivateRegistriesSaveLogic {
	return &PrivateRegistriesSaveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PrivateRegistriesSaveLogic) PrivateRegistriesSave(req *types.PrivateRegistriesConfigReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	encryptKey := l.svcCtx.Config.Auth.AccessSecret

	// 转换请求为模型，加密密码
	registries := make([]model.PrivateRegistry, 0, len(req.Registries))
	for _, reg := range req.Registries {
		password := reg.Password
		if password != "" {
			// 加密密码
			encrypted, err := utiles.Encrypt(password, encryptKey)
			if err != nil {
				resp.Code = 500
				resp.Msg = "密码加密失败: " + err.Error()
				resp.Data = map[string]interface{}{}
				return resp, nil
			}
			password = encrypted
		}

		registries = append(registries, model.PrivateRegistry{
			Name:     reg.Name,
			Host:     reg.Host,
			Username: reg.Username,
			Password: password,
			Insecure: reg.Insecure,
		})
	}

	config := &model.PrivateRegistriesConfig{
		Enabled:    req.Enabled,
		Registries: registries,
	}

	if err := model.SavePrivateRegistriesConfig(config); err != nil {
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
