package image

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTagLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTagLogic {
	return &UpdateTagLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateTagLogic) UpdateTag(req *types.UpdateImageTagReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 验证参数
	if req.NewTag == "" {
		resp.Code = 400
		resp.Msg = "tag 不能为空"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 获取镜像信息
	imageInspect, _, err := l.svcCtx.DockerClient.ImageInspectWithRaw(l.ctx, req.Id)
	if err != nil {
		resp.Code = 404
		resp.Msg = fmt.Sprintf("镜像不存在: %v", err)
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 获取原始镜像名称
	var originalImageName string
	var originalTag string
	if len(imageInspect.RepoTags) > 0 {
		// 使用第一个有效的 RepoTag
		for _, tag := range imageInspect.RepoTags {
			if tag != "<none>:<none>" && !strings.HasPrefix(tag, "<none>") {
				parts := strings.SplitN(tag, ":", 2)
				if len(parts) == 2 {
					originalImageName = parts[0]
					originalTag = parts[1]
					break
				}
			}
		}
	}

	if originalImageName == "" {
		resp.Code = 400
		resp.Msg = "无法获取镜像名称，可能是无效的镜像"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 构建新的镜像名称
	newImageRef := fmt.Sprintf("%s:%s", originalImageName, req.NewTag)
	oldImageRef := fmt.Sprintf("%s:%s", originalImageName, originalTag)

	logx.Infof("修改镜像 tag: %s -> %s", oldImageRef, newImageRef)

	// 给镜像打上新 tag
	err = l.svcCtx.DockerClient.ImageTag(l.ctx, req.Id, newImageRef)
	if err != nil {
		resp.Code = 500
		resp.Msg = fmt.Sprintf("打 tag 失败: %v", err)
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	logx.Infof("镜像 tag 修改成功: %s", newImageRef)

	// 查找使用该镜像的容器并级联重建
	rebuiltContainers, failedContainers := l.cascadeRebuildContainers(req.Id, newImageRef)

	// 更新镜像元数据
	now := time.Now()
	meta := &model.ImageMetadata{
		ImageID:     req.Id,
		ImageName:   originalImageName,
		ImageTag:    req.NewTag,
		SourceType:  model.SourceTypeLocal,
		LastCheckAt: &now,
	}
	if err := model.UpsertImageMetadata(meta); err != nil {
		logx.Errorf("更新镜像元数据失败: %v", err)
	}

	resp.Code = 200
	resp.Msg = "镜像 tag 修改成功"
	resp.Data = map[string]interface{}{
		"oldTag":            originalTag,
		"newTag":            req.NewTag,
		"newImageRef":       newImageRef,
		"rebuiltContainers": rebuiltContainers,
		"failedContainers":  failedContainers,
	}
	return resp, nil
}

// cascadeRebuildContainers 级联重建使用该镜像的容器
func (l *UpdateTagLogic) cascadeRebuildContainers(imageID string, newImageRef string) ([]string, []string) {
	var rebuiltContainers []string
	var failedContainers []string

	// 获取所有容器
	containers, err := l.svcCtx.DockerClient.ContainerList(l.ctx, container.ListOptions{All: true})
	if err != nil {
		logx.Errorf("获取容器列表失败: %v", err)
		return rebuiltContainers, failedContainers
	}

	// 找到使用该镜像的容器
	for _, c := range containers {
		if c.ImageID == imageID || c.ImageID == "sha256:"+imageID {
			containerName := ""
			if len(c.Names) > 0 {
				containerName = strings.TrimPrefix(c.Names[0], "/")
			}
			if containerName == "" {
				continue
			}

			logx.Infof("级联重建容器: %s (镜像: %s)", containerName, newImageRef)

			err := l.rebuildContainer(c.ID, containerName, newImageRef)
			if err != nil {
				logx.Errorf("重建容器 %s 失败: %v", containerName, err)
				failedContainers = append(failedContainers, containerName)
			} else {
				logx.Infof("容器 %s 重建成功", containerName)
				rebuiltContainers = append(rebuiltContainers, containerName)
			}
		}
	}

	if len(rebuiltContainers) == 0 && len(failedContainers) == 0 {
		logx.Info("没有容器使用该镜像，无需重建")
	}

	return rebuiltContainers, failedContainers
}

// rebuildContainer 重建单个容器
func (l *UpdateTagLogic) rebuildContainer(containerID, containerName, newImageRef string) error {
	ctx := l.ctx

	// 获取容器详细信息
	inspect, err := l.svcCtx.DockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("获取容器信息失败: %w", err)
	}

	wasRunning := inspect.State.Running

	// 停止容器
	if wasRunning {
		timeout := 10
		if err := l.svcCtx.DockerClient.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
			return fmt.Errorf("停止容器失败: %w", err)
		}
	}

	// 重命名旧容器
	backupName := fmt.Sprintf("%s_backup_%d", containerName, time.Now().Unix())
	if err := l.svcCtx.DockerClient.ContainerRename(ctx, containerID, backupName); err != nil {
		// 尝试恢复
		if wasRunning {
			_ = l.svcCtx.DockerClient.ContainerStart(ctx, containerID, container.StartOptions{})
		}
		return fmt.Errorf("重命名容器失败: %w", err)
	}

	// 使用新镜像创建容器
	inspect.Config.Hostname = ""
	inspect.Config.Image = newImageRef
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: inspect.NetworkSettings.Networks,
	}

	newContainer, err := l.svcCtx.DockerClient.ContainerCreate(ctx, inspect.Config, inspect.HostConfig, networkingConfig, nil, containerName)
	if err != nil {
		// 尝试恢复
		_ = l.svcCtx.DockerClient.ContainerRename(ctx, containerID, containerName)
		if wasRunning {
			_ = l.svcCtx.DockerClient.ContainerStart(ctx, containerID, container.StartOptions{})
		}
		return fmt.Errorf("创建容器失败: %w", err)
	}

	// 启动新容器
	if wasRunning {
		if err := l.svcCtx.DockerClient.ContainerStart(ctx, newContainer.ID, container.StartOptions{}); err != nil {
			// 删除新容器，恢复旧容器
			_ = l.svcCtx.DockerClient.ContainerRemove(ctx, newContainer.ID, container.RemoveOptions{})
			_ = l.svcCtx.DockerClient.ContainerRename(ctx, containerID, containerName)
			_ = l.svcCtx.DockerClient.ContainerStart(ctx, containerID, container.StartOptions{})
			return fmt.Errorf("启动容器失败: %w", err)
		}
	}

	// 删除旧容器
	if err := l.svcCtx.DockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
		logx.Errorf("删除旧容器 %s 失败: %v", backupName, err)
		// 不影响整体流程
	}

	return nil
}
