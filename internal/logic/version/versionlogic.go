package version

import (
	"context"
	"github.com/xcz1997/dockerCopilot/internal/config"

	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
)

type VersionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VersionLogic {
	return &VersionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VersionLogic) Version(req *types.VersionReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	if req.Type == "local" {
		resp.Code = 200
		resp.Msg = "success"
		resp.Data = map[string]interface{}{
			"version":   config.Version,
			"buildDate": config.BuildDate,
			"isDocker":  utiles.IsRunningInDocker(),
		}
		return resp, nil
	} else if req.Type == "remote" {
		versionInfo, err := utiles.GetVersionInfo()
		if err != nil {
			resp.Code = 500
			resp.Msg = "获取版本错误: " + err.Error()
			resp.Data = map[string]interface{}{
				"localVersion":  config.Version,
				"remoteVersion": "获取失败",
				"hasUpdate":     false,
				"isDocker":      utiles.IsRunningInDocker(),
				"canAutoUpdate": false,
				"updateMessage": "获取远程版本失败",
			}
			return resp, err
		}

		if versionInfo.HasUpdate {
			resp.Code = 200
			resp.Msg = "程序有更新"
		} else {
			resp.Code = 200
			resp.Msg = "程序无更新"
		}

		resp.Data = map[string]interface{}{
			"localVersion":  versionInfo.LocalVersion,
			"remoteVersion": versionInfo.RemoteVersion,
			"hasUpdate":     versionInfo.HasUpdate,
			"isDocker":      versionInfo.IsDocker,
			"canAutoUpdate": versionInfo.CanAutoUpdate,
			"updateMessage": versionInfo.UpdateMessage,
		}
		return resp, nil

	} else {
		resp.Code = 400
		resp.Msg = "type 参数错误"
		resp.Data = map[string]string{}
		return resp, nil
	}
}
