package system

import (
	"context"

	"github.com/docker/docker/api/types/volume"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type SystemInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSystemInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SystemInfoLogic {
	return &SystemInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SystemInfo 返回系统信息（Volume数量、CPU核心数、内存大小）
func (l *SystemInfoLogic) SystemInfo() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	if l.svcCtx.DockerClient == nil {
		resp.Code = 500
		resp.Msg = "Docker client not initialized"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 获取 Volume 数量
	volumes, err := l.svcCtx.DockerClient.VolumeList(l.ctx, volume.ListOptions{})
	volumeCount := 0
	if err == nil {
		volumeCount = len(volumes.Volumes)
	}

	// 获取系统信息（CPU、内存）
	info, err := l.svcCtx.DockerClient.Info(l.ctx)
	cpuCores := 0
	var memoryTotal int64 = 0
	if err == nil {
		cpuCores = info.NCPU
		memoryTotal = info.MemTotal
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"volumeCount": volumeCount,
		"cpuCores":    cpuCores,
		"memoryTotal": memoryTotal,
	}
	return resp, nil
}
