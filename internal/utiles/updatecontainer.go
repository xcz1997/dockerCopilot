package utiles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	dockerMsgType "github.com/docker/docker/pkg/jsonmessage"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	MyType "github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

func UpdateContainer(serviceContext *svc.ServiceContext, id string, name string, imageNameAndTag string, delOldContainer bool, taskID string) error {
	ctx := context.Background()
	serviceContext.UpdateProgress(taskID, svc.TaskProgress{
		TaskID:     taskID,
		Percentage: 0,
		Name:       name,
		Message:    "正在连接Docker",
		DetailMsg:  "正在连接Docker",
		IsDone:     false,
	})
	var oldTaskProgress, result = serviceContext.GetProgress(taskID)
	if !result {
		oldTaskProgress = svc.TaskProgress{
			Percentage: 0,
			Name:       "",
			Message:    "",
			DetailMsg:  "",
			IsDone:     false,
		}
	}
	timeout := 10
	signal := "SIGINT"

	serviceContext.DockerClient.NegotiateAPIVersion(ctx)

	// 记录旧镜像 ID（用于更新后清理）
	var oldImageID string
	inspectedContainerForImage, err := serviceContext.DockerClient.ContainerInspect(ctx, id)
	if err == nil {
		oldImageID = inspectedContainerForImage.Image
	}

	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在拉取新镜像"
	oldTaskProgress.Percentage = 10
	oldTaskProgress.DetailMsg = "正在拉取新镜像"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)

	// 使用统一的镜像拉取方法（支持加速器和私有 Registry 认证）
	pullOpts := module.PullImageOptions{
		ImageName:      imageNameAndTag,
		UseAccelerator: true,
		RegistryAuth:   module.GetPullAuthForImageByMeta(oldImageID, imageNameAndTag),
	}
	reader, pullResult, err := module.PullImage(ctx, serviceContext.DockerClient, pullOpts)
	if err != nil {
		oldTaskProgress.Message = "拉取镜像失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		logx.Errorf("Failed to pull image: %s", err)
		return err
	}
	err = decodePullResp(reader, serviceContext, taskID)
	if err != nil {
		oldTaskProgress.Message = "拉取镜像失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		logx.Errorf("Failed to pull image: %s", err)
		return err
	}

	// 如果使用了加速器，需要重新打标签
	if pullResult.ActualImageName != imageNameAndTag {
		logx.Infof("重新打标签: %s -> %s", pullResult.ActualImageName, imageNameAndTag)
		if err := module.TagImage(ctx, serviceContext.DockerClient, pullResult.ActualImageName, imageNameAndTag); err != nil {
			logx.Errorf("重新打标签失败: %s", err.Error())
			// 打标签失败不影响整体流程，继续使用加速器的镜像名
		}
	}
	oldTaskProgress, result = serviceContext.GetProgress(taskID)
	if !result {
		oldTaskProgress = svc.TaskProgress{
			Percentage: 0,
			Name:       "",
			Message:    "",
			DetailMsg:  "",
			IsDone:     false,
		}
	}
	oldTaskProgress.Message = "拉取镜像成功"
	oldTaskProgress.DetailMsg = "拉取镜像成功"

	oldTaskProgress.Percentage = 30
	oldTaskProgress.Message = "正在停止容器"
	oldTaskProgress.DetailMsg = "正在停止容器"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	stopOptions := container.StopOptions{
		Signal:  signal,
		Timeout: &timeout,
	}
	err = serviceContext.DockerClient.ContainerStop(context.Background(), id, stopOptions)
	if err != nil {
		oldTaskProgress.Message = "停止容器失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "容器停止成功"
	oldTaskProgress.DetailMsg = "容器停止成功"

	oldTaskProgress.Percentage = 40
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在重命名旧容器"
	oldTaskProgress.DetailMsg = "正在重命名旧容器"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	currentDate := time.Now().Format("2006-01-02-15-04-05")
	err = serviceContext.DockerClient.ContainerRename(context.Background(), id, name+"-"+currentDate)
	if err != nil {
		oldTaskProgress.Message = "重命名旧容器失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "重命名旧容器成功"
	oldTaskProgress.DetailMsg = "重命名旧容器成功"
	oldTaskProgress.Percentage = 60
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在创建新容器"
	oldTaskProgress.DetailMsg = "正在创建新容器"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	inspectedContainer, err := serviceContext.DockerClient.ContainerInspect(ctx, id)
	if err != nil {
		oldTaskProgress.Message = "获取容器信息失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		logx.Error("获取容器信息失败" + err.Error())
		return err
	}
	inspectedContainer.Config.Hostname = ""
	inspectedContainer.Config.Image = imageNameAndTag
	inspectedContainer.Image = imageNameAndTag
	config := inspectedContainer.Config
	hostConfig := inspectedContainer.HostConfig
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: inspectedContainer.NetworkSettings.Networks,
	}
	containerName := name
	_, err = serviceContext.DockerClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		oldTaskProgress.Message = "创建新容器失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "创建新容器成功"
	oldTaskProgress.DetailMsg = "创建新容器成功"
	oldTaskProgress.Percentage = 80
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在启动新容器以及删除旧容器(如果不保留旧容器)"
	oldTaskProgress.DetailMsg = "正在启动新容器以及删除旧容器(如果不保留旧容器)"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	err = serviceContext.DockerClient.ContainerStart(context.Background(), containerName, container.StartOptions{
		CheckpointID:  "",
		CheckpointDir: "",
	})
	if err != nil {
		oldTaskProgress.Message = "启动新容器失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	if delOldContainer {
		err = serviceContext.DockerClient.ContainerRemove(context.Background(), id, container.RemoveOptions{})
		if err != nil {
			oldTaskProgress.Message = "删除旧容器失败"
			oldTaskProgress.DetailMsg = err.Error()
			oldTaskProgress.IsDone = true
			serviceContext.UpdateProgress(taskID, oldTaskProgress)
			return err
		}
	}

	// 级联更新使用同一旧镜像的其他容器，然后清理旧镜像
	if oldImageID != "" {
		// 获取新镜像 ID
		newImageInspect, _, err := serviceContext.DockerClient.ImageInspectWithRaw(ctx, imageNameAndTag)
		if err == nil && oldImageID != newImageInspect.ID {
			// 找到使用旧镜像的其他容器并级联更新
			cascadeUpdateContainersUsingImage(serviceContext, oldImageID, imageNameAndTag, id, taskID)

			// 再次检查旧镜像是否还被使用（级联更新后应该没有了）
			if !isImageInUse(serviceContext, oldImageID) {
				logx.Infof("清理旧镜像: %s", oldImageID[:12])
				_, err := serviceContext.DockerClient.ImageRemove(ctx, oldImageID, image.RemoveOptions{})
				if err != nil {
					// 删除旧镜像失败不影响更新结果，只记录日志
					logx.Errorf("删除旧镜像失败: %s", err.Error())
				}
			}

			// 标记新镜像为已更新（清除 haveUpdate 状态），并保存远程 digest
			if serviceContext.HubImageInfo != nil {
				remoteDigest := serviceContext.HubImageInfo.GetRemoteDigest(imageNameAndTag)
				serviceContext.HubImageInfo.MarkAsUpdatedWithDigest(newImageInspect.ID, imageNameAndTag, remoteDigest)
				logx.Infof("标记镜像 %s 为已更新", imageNameAndTag)
			}
		}
	}

	oldTaskProgress.Message = "更新成功"
	oldTaskProgress.DetailMsg = "更新成功"
	oldTaskProgress.Percentage = 100
	oldTaskProgress.IsDone = true
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	return nil
}

// isImageInUse 检查镜像是否被任何容器使用
func isImageInUse(serviceContext *svc.ServiceContext, imageID string) bool {
	containers, err := GetContainerList(serviceContext)
	if err != nil {
		logx.Errorf("获取容器列表失败: %s", err.Error())
		return true // 出错时保守处理，不删除
	}
	for _, c := range containers {
		if c.ImageID == imageID {
			return true
		}
	}
	return false
}

// cascadeUpdateContainersUsingImage 级联更新使用指定旧镜像的其他容器
// oldImageID: 旧镜像 ID
// newImageNameAndTag: 新镜像名称和标签
// excludeContainerID: 排除的容器 ID（已经更新过的主容器）
// taskID: 任务 ID（用于进度报告）
func cascadeUpdateContainersUsingImage(serviceContext *svc.ServiceContext, oldImageID string, newImageNameAndTag string, excludeContainerID string, taskID string) {
	ctx := context.Background()
	containers, err := GetContainerList(serviceContext)
	if err != nil {
		logx.Errorf("获取容器列表失败，跳过级联更新: %s", err.Error())
		return
	}

	// 找到使用旧镜像的其他容器
	var containersToUpdate []MyType.Container
	for _, c := range containers {
		if c.ImageID == oldImageID && c.ID != excludeContainerID {
			containersToUpdate = append(containersToUpdate, c)
		}
	}

	if len(containersToUpdate) == 0 {
		logx.Infof("没有其他容器使用旧镜像 %s", oldImageID[:12])
		return
	}

	logx.Infof("发现 %d 个容器使用旧镜像，开始级联更新", len(containersToUpdate))

	for _, c := range containersToUpdate {
		// 获取容器名称（去掉前缀 /）
		containerName := ""
		if len(c.Names) > 0 {
			containerName = strings.TrimPrefix(c.Names[0], "/")
		}
		if containerName == "" {
			logx.Errorf("容器 %s 没有名称，跳过", c.ID[:12])
			continue
		}

		logx.Infof("级联更新容器: %s (ID: %s)", containerName, c.ID[:12])

		// 获取容器详细信息
		inspect, err := serviceContext.DockerClient.ContainerInspect(ctx, c.ID)
		if err != nil {
			logx.Errorf("获取容器 %s 信息失败: %s", containerName, err.Error())
			continue
		}

		wasRunning := inspect.State.Running

		// 停止容器
		if wasRunning {
			timeout := 10
			if err := serviceContext.DockerClient.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: &timeout}); err != nil {
				logx.Errorf("停止容器 %s 失败: %s", containerName, err.Error())
				continue
			}
		}

		// 重命名旧容器
		backupName := fmt.Sprintf("%s_backup_%d", containerName, time.Now().Unix())
		if err := serviceContext.DockerClient.ContainerRename(ctx, c.ID, backupName); err != nil {
			logx.Errorf("重命名容器 %s 失败: %s", containerName, err.Error())
			// 尝试恢复
			if wasRunning {
				_ = serviceContext.DockerClient.ContainerStart(ctx, c.ID, container.StartOptions{})
			}
			continue
		}

		// 使用新镜像创建容器
		inspect.Config.Hostname = ""
		inspect.Config.Image = newImageNameAndTag
		networkingConfig := &network.NetworkingConfig{
			EndpointsConfig: inspect.NetworkSettings.Networks,
		}

		newContainer, err := serviceContext.DockerClient.ContainerCreate(ctx, inspect.Config, inspect.HostConfig, networkingConfig, nil, containerName)
		if err != nil {
			logx.Errorf("创建容器 %s 失败: %s", containerName, err.Error())
			// 尝试恢复
			_ = serviceContext.DockerClient.ContainerRename(ctx, c.ID, containerName)
			if wasRunning {
				_ = serviceContext.DockerClient.ContainerStart(ctx, c.ID, container.StartOptions{})
			}
			continue
		}

		// 启动新容器
		if wasRunning {
			if err := serviceContext.DockerClient.ContainerStart(ctx, newContainer.ID, container.StartOptions{}); err != nil {
				logx.Errorf("启动容器 %s 失败: %s", containerName, err.Error())
				// 删除新容器，恢复旧容器
				_ = serviceContext.DockerClient.ContainerRemove(ctx, newContainer.ID, container.RemoveOptions{})
				_ = serviceContext.DockerClient.ContainerRename(ctx, c.ID, containerName)
				_ = serviceContext.DockerClient.ContainerStart(ctx, c.ID, container.StartOptions{})
				continue
			}
		}

		// 删除旧容器
		if err := serviceContext.DockerClient.ContainerRemove(ctx, c.ID, container.RemoveOptions{}); err != nil {
			logx.Errorf("删除旧容器 %s 失败: %s", backupName, err.Error())
			// 不影响整体流程
		}

		logx.Infof("容器 %s 级联更新成功", containerName)
	}
}

func decodePullResp(reader io.Reader, ctx *svc.ServiceContext, taskID string) (err error) {
	decoder := json.NewDecoder(reader)
	var oldTaskProgress, result = ctx.GetProgress(taskID)
	if !result {
		oldTaskProgress = svc.TaskProgress{
			Percentage: 0,
			Name:       "",
			Message:    "",
			DetailMsg:  "",
			IsDone:     false,
		}
	}
	for {
		var msg dockerMsgType.JSONMessage
		if err = decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			oldTaskProgress.Message = "拉取镜像失败"
			oldTaskProgress.DetailMsg = err.Error()
			oldTaskProgress.Percentage = 25
			oldTaskProgress.IsDone = true
			ctx.UpdateProgress(taskID, oldTaskProgress)
			logx.Errorf("Failed to decode pull image response: %s", err)
			return fmt.Errorf("拉取镜像失败: %w", err)
		}
		// Print the progress or error information from the response
		if msg.Error != nil {
			oldTaskProgress.Message = "拉取镜像失败"
			oldTaskProgress.DetailMsg = msg.Error.Error()
			oldTaskProgress.Percentage = 25
			oldTaskProgress.IsDone = true
			ctx.UpdateProgress(taskID, oldTaskProgress)
			logx.Errorf("Error: %s", msg.Error)
			return fmt.Errorf("拉取镜像失败: %w", msg.Error)
		} else {
			var formattedMsg string
			if msg.Progress != nil {
				formattedMsg = fmt.Sprintf("进度%s: %s", msg.Status, msg.Progress.String())
			} else {
				formattedMsg = fmt.Sprintf("进度%s", msg.Status)
			}
			oldTaskProgress.DetailMsg = formattedMsg
			oldTaskProgress.Percentage = 25
			ctx.UpdateProgress(taskID, oldTaskProgress)
			logx.Debugf("拉取镜像进度\t %s: %s\n", msg.Status, msg.Progress)
		}
	}
}
