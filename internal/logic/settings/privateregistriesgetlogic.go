package settings

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type PrivateRegistriesGetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPrivateRegistriesGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PrivateRegistriesGetLogic {
	return &PrivateRegistriesGetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PrivateRegistriesGetLogic) PrivateRegistriesGet() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	config, err := model.GetPrivateRegistriesConfig()
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 转换为响应格式，解密密码用于前端展示（前端会mask处理）
	registries := make([]map[string]interface{}, 0, len(config.Registries))
	encryptKey := l.svcCtx.Config.Auth.AccessSecret

	for _, reg := range config.Registries {
		// 解密密码
		password := reg.Password
		if password != "" {
			decrypted, err := utiles.Decrypt(password, encryptKey)
			if err == nil {
				password = decrypted
			}
			// 解密失败则返回空密码，让用户重新输入
		}

		registries = append(registries, map[string]interface{}{
			"name":     reg.Name,
			"host":     reg.Host,
			"username": reg.Username,
			"password": password,
			"insecure": reg.Insecure,
		})
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"enabled":    config.Enabled,
		"registries": registries,
	}
	return resp, nil
}
