package utiles

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/xcz1997/dockerCopilot/internal/svc"
)

// RemoveContainer 删除容器
func RemoveContainer(ctx *svc.ServiceContext, id string, force bool) error {
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 如果 force 为 true，先停止容器
	if force {
		_ = ctx.DockerClient.ContainerStop(c, id, container.StopOptions{})
	}

	return ctx.DockerClient.ContainerRemove(c, id, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false, // 默认不删除卷
	})
}
