package utiles

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	MyType "github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

func GetContainerList(ctx *svc.ServiceContext) ([]MyType.Container, error) {
	// 获取所有容器（包括停止的容器）
	dockerContainerList, err := ctx.DockerClient.ContainerList(context.Background(), container.ListOptions{
		All: true, // 设置为true来获取所有容器
	})
	if err != nil {
		logx.Errorf("get container list error: %v", err)
		return nil, err
	}
	var containerList []MyType.Container
	for _, dockerContainerInfo := range dockerContainerList {
		containerInfo := MyType.Container{
			Container: dockerContainerInfo,
		}
		containerList = append(containerList, containerInfo)
	}
	return containerList, nil
}

func CheckImageUpdate(ctx *svc.ServiceContext, containerListData []MyType.Container) []MyType.Container {
	for i, v := range containerListData {
		// 优先通过 ImageID 查找
		if info, ok := ctx.HubImageInfo.Data[v.ImageID]; ok {
			if info.NeedUpdate {
				containerListData[i].Update = true
			}
		} else {
			// 如果 ImageID 没找到，尝试通过镜像名称查找
			// 这种情况发生在：容器使用旧镜像，但新镜像已被拉取
			imageName := v.Image
			if info, ok := ctx.HubImageInfo.Data[imageName]; ok {
				if info.NeedUpdate {
					containerListData[i].Update = true
				}
			}
		}
	}
	return containerListData
}
