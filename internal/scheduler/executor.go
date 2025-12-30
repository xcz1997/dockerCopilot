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

// ProgressCallback 进度回调函数类型
// percentage: 进度百分比 (0-100)
// message: 简短消息
// detailMsg: 详细信息
type ProgressCallback func(percentage int, message string, detailMsg string)

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

// getImageNameWithTag 获取带标签的镜像名称
// 优先使用镜像的 RepoTags，避免使用摘要格式导致更新后标签丢失
func (e *Executor) getImageNameWithTag(ctx context.Context, imageID string, fallbackImage string) string {
	// 尝试通过 ImageID 获取镜像信息
	if imageID != "" {
		imgInspect, _, err := e.dockerClient.ImageInspectWithRaw(ctx, imageID)
		if err == nil {
			// 优先使用 RepoTags
			if len(imgInspect.RepoTags) > 0 {
				// 选择第一个非 <none> 的标签
				for _, tag := range imgInspect.RepoTags {
					if tag != "<none>:<none>" && !strings.HasPrefix(tag, "<none>") {
						return tag
					}
				}
			}
			// 如果没有 RepoTags，从 RepoDigests 中提取镜像名
			if len(imgInspect.RepoDigests) > 0 {
				for _, digest := range imgInspect.RepoDigests {
					// 格式: repo@sha256:xxx
					if idx := strings.Index(digest, "@"); idx > 0 {
						repoName := digest[:idx]
						// 添加 latest 标签
						if !strings.Contains(repoName, ":") {
							return repoName + ":latest"
						}
						return repoName
					}
				}
			}
		}
	}

	// 回退到原始镜像名称
	imageName := fallbackImage

	// 检查是否是纯摘要格式 (sha256:xxx)
	if strings.HasPrefix(imageName, "sha256:") {
		logx.Errorf("镜像名称为纯摘要格式，无法确定正确的标签: %s", imageName)
		return imageName
	}

	// 如果没有标签，添加 :latest
	if !strings.Contains(imageName, ":") {
		imageName += ":latest"
	}

	return imageName
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

	// 获取正确的镜像名称（带标签）
	imageName := e.getImageNameWithTag(ctx, container.ImageID, container.Image)
	logx.Infof("容器[%s]检查更新使用镜像名称: %s", container.Name, imageName)

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
	return e.UpdateWithProgress(ctx, mc, nil)
}

// UpdateWithProgress 更新容器（带进度回调）
func (e *Executor) UpdateWithProgress(ctx context.Context, mc MatchedContainer, onProgress ProgressCallback) (*UpdateResult, error) {
	result := &UpdateResult{
		ContainerID:   mc.ID,
		ContainerName: mc.Name,
		OldImage:      mc.Image,
	}

	// 进度报告辅助函数
	reportProgress := func(pct int, msg, detail string) {
		if onProgress != nil {
			onProgress(pct, msg, detail)
		}
	}

	logx.Infof("开始更新容器[%s]", mc.Name)
	reportProgress(5, "获取容器配置", "正在获取容器配置信息")

	// 1. 获取容器配置
	inspect, err := e.dockerClient.ContainerInspect(ctx, mc.ID)
	if err != nil {
		result.Message = fmt.Sprintf("获取容器配置失败: %v", err)
		return result, err
	}

	// 2. 停止容器
	wasRunning := inspect.State.Running
	if wasRunning {
		reportProgress(10, "停止容器", "正在停止旧容器")
		logx.Infof("停止容器[%s]", mc.Name)
		timeout := 30
		if err := e.dockerClient.ContainerStop(ctx, mc.ID, container.StopOptions{Timeout: &timeout}); err != nil {
			result.Message = fmt.Sprintf("停止容器失败: %v", err)
			return result, err
		}
	}

	// 3. 重命名旧容器
	reportProgress(20, "重命名旧容器", "正在重命名旧容器作为备份")
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

	// 4. 获取正确的镜像名称（带标签）
	// 优先从镜像的 RepoTags 获取，避免使用摘要格式导致标签丢失
	imageName := e.getImageNameWithTag(ctx, mc.ImageID, mc.Image)
	logx.Infof("容器[%s]使用镜像名称: %s (原始: %s)", mc.Name, imageName, mc.Image)
	reportProgress(30, "拉取新镜像", fmt.Sprintf("正在拉取镜像 %s", imageName))
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

	// 读取拉取进度
	buf := make([]byte, 1024)
	for {
		n, readErr := pullOut.Read(buf)
		if n > 0 {
			// 简单解析进度（拉取过程在30%-60%之间）
			reportProgress(45, "拉取新镜像", "正在下载镜像层...")
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			break
		}
	}
	reportProgress(60, "拉取完成", "镜像拉取完成")

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
	reportProgress(70, "创建新容器", "正在使用新镜像创建容器")
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
		reportProgress(80, "启动新容器", "正在启动新容器")
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
	reportProgress(90, "清理旧容器", "正在删除旧容器备份")
	logx.Infof("删除旧容器[%s]", backupName)
	if err := e.dockerClient.ContainerRemove(ctx, mc.ID, container.RemoveOptions{}); err != nil {
		logx.Errorf("删除旧容器失败: %v", err)
		// 不影响结果
	}

	result.Success = true
	result.NewImage = newInspect.ID
	result.Message = "更新成功"

	// 更新缓存：将新镜像标记为不需要更新
	if e.hubImageInfo != nil {
		e.hubImageInfo.MarkAsUpdated(newInspect.ID, imageName)
		logx.Infof("已更新镜像缓存: %s (ID: %s)", imageName, newInspect.ID[:12])
	}

	reportProgress(100, "更新完成", "容器更新成功")
	logx.Infof("容器[%s]更新完成", oldName)
	return result, nil
}

// CheckImageUpdate 检查镜像是否有更新
func (e *Executor) CheckImageUpdate(ctx context.Context, img MatchedImage) (bool, error) {
	// 优先使用 hubImageInfo 中已检测的结果
	if e.hubImageInfo != nil {
		if info, ok := e.hubImageInfo.GetImageCheck(img.ID); ok {
			logx.Infof("镜像[%s]使用缓存的更新状态: needUpdate=%v", img.FullName, info.NeedUpdate)
			return info.NeedUpdate, nil
		}
		if info, ok := e.hubImageInfo.GetImageCheckByName(img.FullName); ok {
			logx.Infof("镜像[%s]使用缓存的更新状态(按名称): needUpdate=%v", img.FullName, info.NeedUpdate)
			return info.NeedUpdate, nil
		}
	}

	// 缓存中没有，则实际拉取检查
	logx.Infof("镜像[%s]缓存中无更新状态，开始拉取检查", img.FullName)

	// 获取本地镜像信息
	localInspect, _, err := e.dockerClient.ImageInspectWithRaw(ctx, img.ID)
	if err != nil {
		return false, fmt.Errorf("获取本地镜像信息失败: %w", err)
	}

	// 拉取远程镜像
	pullOut, err := e.dockerClient.ImagePull(ctx, img.FullName, image.PullOptions{})
	if err != nil {
		return false, fmt.Errorf("拉取镜像失败: %w", err)
	}
	defer pullOut.Close()
	_, _ = io.Copy(io.Discard, pullOut)

	// 重新获取镜像信息
	remoteInspect, _, err := e.dockerClient.ImageInspectWithRaw(ctx, img.FullName)
	if err != nil {
		return false, fmt.Errorf("获取远程镜像信息失败: %w", err)
	}

	hasUpdate := localInspect.ID != remoteInspect.ID
	logx.Infof("镜像[%s]检查: 本地=%s, 远程=%s, 有更新=%v",
		img.FullName, localInspect.ID[:12], remoteInspect.ID[:12], hasUpdate)

	return hasUpdate, nil
}

// PullImageWithProgress 拉取镜像（带进度回调）
func (e *Executor) PullImageWithProgress(ctx context.Context, img MatchedImage, onProgress ProgressCallback) error {
	reportProgress := func(pct int, msg, detail string) {
		if onProgress != nil {
			onProgress(pct, msg, detail)
		}
	}

	reportProgress(10, "开始拉取", fmt.Sprintf("正在拉取镜像 %s", img.FullName))

	pullOut, err := e.dockerClient.ImagePull(ctx, img.FullName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("拉取镜像失败: %w", err)
	}
	defer pullOut.Close()

	// 读取拉取进度
	buf := make([]byte, 1024)
	for {
		n, readErr := pullOut.Read(buf)
		if n > 0 {
			reportProgress(50, "拉取中", "正在下载镜像层...")
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			break
		}
	}

	// 获取新镜像ID并更新缓存
	newInspect, _, err := e.dockerClient.ImageInspectWithRaw(ctx, img.FullName)
	if err == nil && e.hubImageInfo != nil {
		e.hubImageInfo.MarkAsUpdated(newInspect.ID, img.FullName)
		logx.Infof("已更新镜像缓存: %s (ID: %s)", img.FullName, newInspect.ID[:12])
	}

	reportProgress(100, "拉取完成", "镜像更新成功")
	logx.Infof("镜像[%s]拉取完成", img.FullName)
	return nil
}
