package scheduler

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/zeromicro/go-zero/core/logx"
)

// UpdateResult 更新结果
type UpdateResult struct {
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	OldImage      string `json:"oldImage"`
	NewImage      string `json:"newImage"`
	Success       bool   `json:"success"`
	Message       string `json:"message"`
}

// Executor 更新执行器
type Executor struct {
	dockerClient *client.Client
	hubImageInfo *module.ImageUpdateData
}

// NewExecutor 创建执行器
func NewExecutor(dockerClient *client.Client, hubImageInfo *module.ImageUpdateData) *Executor {
	return &Executor{
		dockerClient: dockerClient,
		hubImageInfo: hubImageInfo,
	}
}

// CheckUpdate 检查容器是否有更新
func (e *Executor) CheckUpdate(ctx context.Context, container MatchedContainer) (bool, error) {
	// 优先使用 hubImageInfo 中已检测的结果（与前端显示一致）
	if e.hubImageInfo != nil {
		// 先尝试通过 ImageID 查找
		if info, ok := e.hubImageInfo.GetImageCheck(container.ImageID); ok {
			logx.Infof("容器[%s]使用缓存的更新状态(ImageID): needUpdate=%v (镜像ID: %s)",
				container.Name, info.NeedUpdate, container.ImageID[:12])
			return info.NeedUpdate, nil
		}
		// 再尝试通过镜像名称查找（容器可能使用旧版本镜像，但镜像检查结果按最新镜像ID存储）
		imageName := container.Image
		if !strings.Contains(imageName, ":") {
			imageName += ":latest"
		}
		if info, ok := e.hubImageInfo.GetImageCheckByName(imageName); ok {
			logx.Infof("容器[%s]使用缓存的更新状态(ImageName): needUpdate=%v (镜像名: %s)",
				container.Name, info.NeedUpdate, imageName)
			return info.NeedUpdate, nil
		}
	}

	// 如果缓存中没有，则实际拉取检查
	logx.Infof("容器[%s]缓存中无更新状态，开始拉取检查", container.Name)

	// 获取本地镜像信息
	localInspect, _, err := e.dockerClient.ImageInspectWithRaw(ctx, container.ImageID)
	if err != nil {
		return false, fmt.Errorf("获取本地镜像信息失败: %w", err)
	}

	// 解析镜像名称
	imageName := container.Image
	if !strings.Contains(imageName, ":") {
		imageName += ":latest"
	}

	// 拉取远程镜像信息
	pullOut, err := e.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return false, fmt.Errorf("拉取镜像失败: %w", err)
	}
	defer pullOut.Close()

	// 读取完成
	_, err = io.Copy(io.Discard, pullOut)
	if err != nil {
		return false, fmt.Errorf("读取拉取结果失败: %w", err)
	}

	// 重新获取镜像信息
	remoteInspect, _, err := e.dockerClient.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return false, fmt.Errorf("获取远程镜像信息失败: %w", err)
	}

	// 比较 digest
	hasUpdate := localInspect.ID != remoteInspect.ID

	logx.Infof("容器[%s]镜像检查: 本地=%s, 远程=%s, 有更新=%v",
		container.Name, localInspect.ID[:12], remoteInspect.ID[:12], hasUpdate)

	return hasUpdate, nil
}

// Update 更新容器
func (e *Executor) Update(ctx context.Context, mc MatchedContainer) (*UpdateResult, error) {
	result := &UpdateResult{
		ContainerID:   mc.ID,
		ContainerName: mc.Name,
		OldImage:      mc.Image,
	}

	logx.Infof("开始更新容器[%s]", mc.Name)

	// 1. 获取容器配置
	inspect, err := e.dockerClient.ContainerInspect(ctx, mc.ID)
	if err != nil {
		result.Message = fmt.Sprintf("获取容器配置失败: %v", err)
		return result, err
	}

	// 2. 停止容器
	wasRunning := inspect.State.Running
	if wasRunning {
		logx.Infof("停止容器[%s]", mc.Name)
		timeout := 30
		if err := e.dockerClient.ContainerStop(ctx, mc.ID, container.StopOptions{Timeout: &timeout}); err != nil {
			result.Message = fmt.Sprintf("停止容器失败: %v", err)
			return result, err
		}
	}

	// 3. 重命名旧容器
	oldName := mc.Name
	backupName := fmt.Sprintf("%s_backup_%d", oldName, time.Now().Unix())
	logx.Infof("重命名容器[%s] -> [%s]", oldName, backupName)
	if err := e.dockerClient.ContainerRename(ctx, mc.ID, backupName); err != nil {
		result.Message = fmt.Sprintf("重命名容器失败: %v", err)
		// 尝试恢复
		if wasRunning {
			_ = e.dockerClient.ContainerStart(ctx, mc.ID, container.StartOptions{})
		}
		return result, err
	}

	// 4. 拉取新镜像
	imageName := mc.Image
	if !strings.Contains(imageName, ":") {
		imageName += ":latest"
	}
	logx.Infof("拉取新镜像[%s]", imageName)
	pullOut, err := e.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		result.Message = fmt.Sprintf("拉取镜像失败: %v", err)
		// 尝试恢复
		_ = e.dockerClient.ContainerRename(ctx, mc.ID, oldName)
		if wasRunning {
			_ = e.dockerClient.ContainerStart(ctx, mc.ID, container.StartOptions{})
		}
		return result, err
	}
	defer pullOut.Close()
	_, _ = io.Copy(io.Discard, pullOut)

	// 获取新镜像信息
	newInspect, _, err := e.dockerClient.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		result.Message = fmt.Sprintf("获取新镜像信息失败: %v", err)
		_ = e.dockerClient.ContainerRename(ctx, mc.ID, oldName)
		if wasRunning {
			_ = e.dockerClient.ContainerStart(ctx, mc.ID, container.StartOptions{})
		}
		return result, err
	}

	// 5. 创建新容器
	logx.Infof("创建新容器[%s]", oldName)
	newContainerConfig := inspect.Config
	newContainerConfig.Image = imageName

	resp, err := e.dockerClient.ContainerCreate(
		ctx,
		newContainerConfig,
		inspect.HostConfig,
		nil,
		nil,
		oldName,
	)
	if err != nil {
		result.Message = fmt.Sprintf("创建新容器失败: %v", err)
		// 尝试恢复
		_ = e.dockerClient.ContainerRename(ctx, mc.ID, oldName)
		if wasRunning {
			_ = e.dockerClient.ContainerStart(ctx, mc.ID, container.StartOptions{})
		}
		return result, err
	}

	// 6. 启动新容器（如果原来是运行状态）
	if wasRunning {
		logx.Infof("启动新容器[%s]", oldName)
		if err := e.dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			result.Message = fmt.Sprintf("启动新容器失败: %v", err)
			// 删除新容器，恢复旧容器
			_ = e.dockerClient.ContainerRemove(ctx, resp.ID, container.RemoveOptions{})
			_ = e.dockerClient.ContainerRename(ctx, mc.ID, oldName)
			_ = e.dockerClient.ContainerStart(ctx, mc.ID, container.StartOptions{})
			return result, err
		}
	}

	// 7. 删除旧容器
	logx.Infof("删除旧容器[%s]", backupName)
	if err := e.dockerClient.ContainerRemove(ctx, mc.ID, container.RemoveOptions{}); err != nil {
		logx.Errorf("删除旧容器失败: %v", err)
		// 不影响结果
	}

	result.Success = true
	result.NewImage = newInspect.ID
	result.Message = "更新成功"

	logx.Infof("容器[%s]更新完成", oldName)
	return result, nil
}
