package utiles

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	MyType "github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContainerImageInfo 容器镜像信息
type ContainerImageInfo struct {
	ImageID   string `json:"imageId"`
	ImageName string `json:"imageName"`
}

// RelatedContainer 关联容器信息
type RelatedContainer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// RemoveContainerResult 删除容器结果
type RemoveContainerResult struct {
	DeletedContainers []string `json:"deletedContainers,omitempty"`
	DeletedImage      string   `json:"deletedImage,omitempty"`
	ImageDeleteError  string   `json:"imageDeleteError,omitempty"`
}

// GetContainerImageInfo 获取容器使用的镜像信息
func GetContainerImageInfo(ctx *svc.ServiceContext, containerID string) (*ContainerImageInfo, error) {
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inspect, err := ctx.DockerClient.ContainerInspect(c, containerID)
	if err != nil {
		return nil, err
	}

	return &ContainerImageInfo{
		ImageID:   inspect.Image,
		ImageName: inspect.Config.Image,
	}, nil
}

// GetContainersUsingImage 获取使用指定镜像的所有容器（排除指定容器）
func GetContainersUsingImage(ctx *svc.ServiceContext, imageID string, excludeContainerID string) ([]RelatedContainer, error) {
	containers, err := GetContainerList(ctx)
	if err != nil {
		return nil, err
	}

	var related []RelatedContainer
	for _, c := range containers {
		if c.ImageID == imageID && c.ID != excludeContainerID {
			name := ""
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			related = append(related, RelatedContainer{
				ID:     c.ID,
				Name:   name,
				Status: c.State,
			})
		}
	}
	return related, nil
}

// RemoveContainerOptions 删除容器选项
type RemoveContainerOptions struct {
	Force                   bool
	DeleteImage             bool
	DeleteRelatedContainers bool
}

// RemoveContainerWithOptions 删除容器（带选项）
func RemoveContainerWithOptions(ctx *svc.ServiceContext, id string, opts RemoveContainerOptions) (*RemoveContainerResult, error) {
	c, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result := &RemoveContainerResult{}

	// 获取容器的镜像信息
	var imageID string
	if opts.DeleteImage {
		info, err := GetContainerImageInfo(ctx, id)
		if err != nil {
			logx.Errorf("获取容器镜像信息失败: %s", err.Error())
		} else {
			imageID = info.ImageID
		}
	}

	// 如果需要删除关联容器
	if opts.DeleteRelatedContainers && imageID != "" {
		relatedContainers, err := GetContainersUsingImage(ctx, imageID, id)
		if err != nil {
			logx.Errorf("获取关联容器失败: %s", err.Error())
		} else {
			for _, rc := range relatedContainers {
				// 先停止容器
				_ = ctx.DockerClient.ContainerStop(c, rc.ID, container.StopOptions{})
				// 删除容器
				err := ctx.DockerClient.ContainerRemove(c, rc.ID, container.RemoveOptions{
					Force:         true,
					RemoveVolumes: false,
				})
				if err != nil {
					logx.Errorf("删除关联容器 %s 失败: %s", rc.Name, err.Error())
				} else {
					result.DeletedContainers = append(result.DeletedContainers, rc.Name)
					logx.Infof("已删除关联容器: %s", rc.Name)
				}
			}
		}
	}

	// 如果 force 为 true，先停止容器
	if opts.Force {
		_ = ctx.DockerClient.ContainerStop(c, id, container.StopOptions{})
	}

	// 删除主容器
	err := ctx.DockerClient.ContainerRemove(c, id, container.RemoveOptions{
		Force:         opts.Force,
		RemoveVolumes: false,
	})
	if err != nil {
		return result, err
	}

	// 如果需要删除镜像
	if opts.DeleteImage && imageID != "" {
		// 再次检查镜像是否还被其他容器使用
		stillInUse := isImageInUse(ctx, imageID)
		if !stillInUse {
			_, err := ctx.DockerClient.ImageRemove(c, imageID, image.RemoveOptions{
				Force:         false,
				PruneChildren: true,
			})
			if err != nil {
				result.ImageDeleteError = err.Error()
				logx.Errorf("删除镜像失败: %s", err.Error())
			} else {
				result.DeletedImage = imageID[:12]
				logx.Infof("已删除镜像: %s", imageID[:12])
			}
		} else {
			result.ImageDeleteError = "镜像仍被其他容器使用"
		}
	}

	return result, nil
}

// RemoveContainer 删除容器（简单版本，保持向后兼容）
func RemoveContainer(ctx *svc.ServiceContext, id string, force bool) error {
	_, err := RemoveContainerWithOptions(ctx, id, RemoveContainerOptions{
		Force:                   force,
		DeleteImage:             false,
		DeleteRelatedContainers: false,
	})
	return err
}

// GetImageDependencyInfo 获取镜像依赖信息（用于前端预检查）
func GetImageDependencyInfo(ctx *svc.ServiceContext, containerID string) (*MyType.ImageDependencyInfo, error) {
	// 获取容器的镜像信息
	info, err := GetContainerImageInfo(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// 获取使用该镜像的其他容器
	relatedContainers, err := GetContainersUsingImage(ctx, info.ImageID, containerID)
	if err != nil {
		return nil, err
	}

	var containers []MyType.DependentContainer
	for _, c := range relatedContainers {
		containers = append(containers, MyType.DependentContainer{
			ID:     c.ID,
			Name:   c.Name,
			Status: c.Status,
		})
	}

	return &MyType.ImageDependencyInfo{
		ImageID:             info.ImageID,
		ImageName:           info.ImageName,
		DependentContainers: containers,
	}, nil
}
