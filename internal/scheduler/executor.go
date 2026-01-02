package scheduler

import (
	"bufio"
	"context"
	"encoding/json"
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

// dockerPullProgress Docker 镜像拉取进度结构
type dockerPullProgress struct {
	Status         string                 `json:"status"`
	ID             string                 `json:"id"`
	Progress       string                 `json:"progress"`
	ProgressDetail dockerProgressDetail   `json:"progressDetail"`
	Error          string                 `json:"error,omitempty"`
}

// dockerProgressDetail 进度详情
type dockerProgressDetail struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

// layerProgress 单层进度
// Docker 拉取镜像的阶段顺序:
// Pulling fs layer -> Waiting -> Downloading -> Download complete ->
// Verifying Checksum -> Extracting -> Pull complete
// 或者: Already exists (跳过下载)
type layerProgress struct {
	Status          string // 当前状态
	DownloadCurrent int64  // 下载进度
	DownloadTotal   int64  // 下载总量
	ExtractCurrent  int64  // 解压进度
	ExtractTotal    int64  // 解压总量
	IsComplete      bool   // 是否完成
}

// getLayerPhaseWeight 获取层阶段权重 (用于计算整体进度)
// 下载占 50%，解压占 50%
func (l *layerProgress) getProgress() float64 {
	if l.IsComplete {
		return 1.0
	}

	var downloadPct, extractPct float64

	// 下载进度 (占总进度的50%)
	if l.DownloadTotal > 0 {
		downloadPct = float64(l.DownloadCurrent) / float64(l.DownloadTotal)
	} else if l.Status == "Download complete" || l.Status == "Verifying Checksum" ||
		l.Status == "Extracting" || l.Status == "Pull complete" || l.Status == "Already exists" {
		downloadPct = 1.0
	}

	// 解压进度 (占总进度的50%)
	if l.ExtractTotal > 0 {
		extractPct = float64(l.ExtractCurrent) / float64(l.ExtractTotal)
	} else if l.Status == "Pull complete" || l.Status == "Already exists" {
		extractPct = 1.0
	}

	return downloadPct*0.5 + extractPct*0.5
}

// parsePullProgress 解析镜像拉取进度并调用回调
// startPct: 开始百分比（如30）
// endPct: 结束百分比（如60）
// onProgress: 进度回调
func parsePullProgress(pullOut io.Reader, startPct, endPct int, onProgress ProgressCallback) {
	if onProgress == nil {
		// 无回调时直接读取完成
		_, _ = io.Copy(io.Discard, pullOut)
		return
	}

	layers := make(map[string]*layerProgress)
	scanner := bufio.NewScanner(pullOut)
	// 增大 scanner 缓冲区以处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lastReportTime := time.Now()
	reportInterval := 200 * time.Millisecond // 限制更新频率

	// 保存最后一条有效的进度信息用于显示
	var lastProgress dockerPullProgress

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var progress dockerPullProgress
		if err := json.Unmarshal(line, &progress); err != nil {
			continue
		}

		// 处理错误
		if progress.Error != "" {
			onProgress(endPct, "拉取失败", progress.Error)
			return
		}

		// 更新层进度
		if progress.ID != "" {
			layer, exists := layers[progress.ID]
			if !exists {
				layer = &layerProgress{}
				layers[progress.ID] = layer
			}

			layer.Status = progress.Status

			// 根据状态更新对应的进度
			switch progress.Status {
			case "Downloading":
				if progress.ProgressDetail.Total > 0 {
					layer.DownloadCurrent = progress.ProgressDetail.Current
					layer.DownloadTotal = progress.ProgressDetail.Total
				}
			case "Download complete":
				layer.DownloadCurrent = layer.DownloadTotal
				if layer.DownloadTotal == 0 {
					layer.DownloadTotal = 1
					layer.DownloadCurrent = 1
				}
			case "Extracting":
				// 下载已完成
				layer.DownloadCurrent = layer.DownloadTotal
				if layer.DownloadTotal == 0 {
					layer.DownloadTotal = 1
					layer.DownloadCurrent = 1
				}
				// 更新解压进度
				if progress.ProgressDetail.Total > 0 {
					layer.ExtractCurrent = progress.ProgressDetail.Current
					layer.ExtractTotal = progress.ProgressDetail.Total
				}
			case "Pull complete":
				layer.IsComplete = true
				layer.DownloadCurrent = layer.DownloadTotal
				layer.ExtractCurrent = layer.ExtractTotal
			case "Already exists":
				layer.IsComplete = true
			}

			// 记录有进度条的最后一条
			if progress.Progress != "" {
				lastProgress = progress
			}
		} else if progress.Status != "" {
			// 没有 ID 的状态消息（如 Digest, Status 等）
			lastProgress = progress
		}

		// 限制更新频率
		if time.Since(lastReportTime) < reportInterval {
			continue
		}
		lastReportTime = time.Now()

		// 计算总体进度
		var totalProgress float64
		completeCount := 0
		activeCount := 0
		var activeStatus string

		for _, layer := range layers {
			totalProgress += layer.getProgress()
			if layer.IsComplete {
				completeCount++
			}
			// 记录活跃状态（正在进行的操作）
			if layer.Status == "Downloading" || layer.Status == "Extracting" ||
				layer.Status == "Verifying Checksum" {
				activeCount++
				activeStatus = layer.Status
			}
		}

		// 计算百分比
		var layerPct int
		if len(layers) > 0 {
			layerPct = int(totalProgress * 100 / float64(len(layers)))
		}

		// 映射到指定范围
		pct := startPct + layerPct*(endPct-startPct)/100

		// 构建详细信息
		var detailMsg string
		if lastProgress.Progress != "" {
			// 使用 Docker 原生进度条格式，显示当前操作的层
			detailMsg = fmt.Sprintf("%s: %s", lastProgress.Status, lastProgress.Progress)
		} else if activeStatus != "" {
			// 显示活跃状态
			detailMsg = fmt.Sprintf("%s (%d 层处理中, %d/%d 完成)",
				translateStatus(activeStatus), activeCount, completeCount, len(layers))
		} else if len(layers) > 0 {
			detailMsg = fmt.Sprintf("处理中: %d/%d 层完成", completeCount, len(layers))
		} else if lastProgress.Status != "" {
			detailMsg = lastProgress.Status
		}

		// 根据当前主要操作设置消息
		var msg string
		switch activeStatus {
		case "Downloading":
			msg = "下载镜像"
		case "Extracting":
			msg = "解压镜像"
		case "Verifying Checksum":
			msg = "校验镜像"
		default:
			msg = "拉取镜像"
		}

		onProgress(pct, msg, detailMsg)
	}

	// 读取完毕，报告结束进度
	onProgress(endPct, "拉取完成", "镜像拉取完成")
}

// translateStatus 翻译状态文本
func translateStatus(status string) string {
	switch status {
	case "Downloading":
		return "下载中"
	case "Extracting":
		return "解压中"
	case "Verifying Checksum":
		return "校验中"
	case "Download complete":
		return "下载完成"
	case "Pull complete":
		return "拉取完成"
	case "Already exists":
		return "已存在"
	case "Waiting":
		return "等待中"
	case "Pulling fs layer":
		return "准备中"
	default:
		return status
	}
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

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

// isImageInUse 检查镜像是否被任何容器使用
func (e *Executor) isImageInUse(ctx context.Context, imageID string) bool {
	containers, err := e.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logx.Errorf("获取容器列表失败: %v", err)
		return true // 出错时保守处理，不删除
	}
	for _, c := range containers {
		if c.ImageID == imageID {
			return true
		}
	}
	return false
}

// cascadeUpdateContainers 级联更新使用指定旧镜像的其他容器
func (e *Executor) cascadeUpdateContainers(ctx context.Context, oldImageID string, newImageNameAndTag string, excludeContainerID string) {
	containers, err := e.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		logx.Errorf("获取容器列表失败，跳过级联更新: %v", err)
		return
	}

	// 找到使用旧镜像的其他容器
	var containersToUpdate []struct {
		ID   string
		Name string
	}
	for _, c := range containers {
		if c.ImageID == oldImageID && c.ID != excludeContainerID {
			name := ""
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			if name != "" {
				containersToUpdate = append(containersToUpdate, struct {
					ID   string
					Name string
				}{ID: c.ID, Name: name})
			}
		}
	}

	if len(containersToUpdate) == 0 {
		logx.Infof("没有其他容器使用旧镜像 %s", oldImageID[:12])
		return
	}

	logx.Infof("发现 %d 个容器使用旧镜像，开始级联更新", len(containersToUpdate))

	for _, c := range containersToUpdate {
		logx.Infof("级联更新容器: %s (ID: %s)", c.Name, c.ID[:12])

		// 获取容器详细信息
		inspect, err := e.dockerClient.ContainerInspect(ctx, c.ID)
		if err != nil {
			logx.Errorf("获取容器 %s 信息失败: %v", c.Name, err)
			continue
		}

		wasRunning := inspect.State.Running

		// 停止容器
		if wasRunning {
			timeout := 30
			if err := e.dockerClient.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: &timeout}); err != nil {
				logx.Errorf("停止容器 %s 失败: %v", c.Name, err)
				continue
			}
		}

		// 重命名旧容器
		backupName := fmt.Sprintf("%s_backup_%d", c.Name, time.Now().Unix())
		if err := e.dockerClient.ContainerRename(ctx, c.ID, backupName); err != nil {
			logx.Errorf("重命名容器 %s 失败: %v", c.Name, err)
			if wasRunning {
				_ = e.dockerClient.ContainerStart(ctx, c.ID, container.StartOptions{})
			}
			continue
		}

		// 使用新镜像创建容器
		inspect.Config.Hostname = ""
		inspect.Config.Image = newImageNameAndTag

		newContainer, err := e.dockerClient.ContainerCreate(ctx, inspect.Config, inspect.HostConfig, nil, nil, c.Name)
		if err != nil {
			logx.Errorf("创建容器 %s 失败: %v", c.Name, err)
			_ = e.dockerClient.ContainerRename(ctx, c.ID, c.Name)
			if wasRunning {
				_ = e.dockerClient.ContainerStart(ctx, c.ID, container.StartOptions{})
			}
			continue
		}

		// 启动新容器
		if wasRunning {
			if err := e.dockerClient.ContainerStart(ctx, newContainer.ID, container.StartOptions{}); err != nil {
				logx.Errorf("启动容器 %s 失败: %v", c.Name, err)
				_ = e.dockerClient.ContainerRemove(ctx, newContainer.ID, container.RemoveOptions{})
				_ = e.dockerClient.ContainerRename(ctx, c.ID, c.Name)
				_ = e.dockerClient.ContainerStart(ctx, c.ID, container.StartOptions{})
				continue
			}
		}

		// 删除旧容器
		if err := e.dockerClient.ContainerRemove(ctx, c.ID, container.RemoveOptions{}); err != nil {
			logx.Errorf("删除旧容器 %s 失败: %v", backupName, err)
		}

		logx.Infof("容器 %s 级联更新成功", c.Name)
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

	// 使用统一的镜像拉取方法（支持加速器和私有 Registry 认证）
	pullOpts := module.PullImageOptions{
		ImageName:      imageName,
		UseAccelerator: true,
		RegistryAuth:   module.GetPullAuthForImageByMeta(container.ImageID, imageName),
	}
	pullOut, pullResult, err := module.PullImage(ctx, e.dockerClient, pullOpts)
	if err != nil {
		return false, fmt.Errorf("拉取镜像失败: %w", err)
	}
	defer pullOut.Close()

	// 读取完成
	_, err = io.Copy(io.Discard, pullOut)
	if err != nil {
		return false, fmt.Errorf("读取拉取结果失败: %w", err)
	}

	// 如果使用了加速器，需要重新打标签
	if pullResult.ActualImageName != imageName {
		logx.Infof("重新打标签: %s -> %s", pullResult.ActualImageName, imageName)
		if err := module.TagImage(ctx, e.dockerClient, pullResult.ActualImageName, imageName); err != nil {
			logx.Errorf("重新打标签失败: %v", err)
		}
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

	// 使用统一的镜像拉取方法（支持加速器和私有 Registry 认证）
	pullOpts := module.PullImageOptions{
		ImageName:      imageName,
		UseAccelerator: true,
		RegistryAuth:   module.GetPullAuthForImageByMeta(mc.ImageID, imageName),
	}
	pullOut, pullResult, err := module.PullImage(ctx, e.dockerClient, pullOpts)
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

	// 解析镜像拉取进度（拉取过程在30%-60%之间）
	parsePullProgress(pullOut, 30, 60, onProgress)

	// 如果使用了加速器，需要重新打标签
	if pullResult.ActualImageName != imageName {
		logx.Infof("重新打标签: %s -> %s", pullResult.ActualImageName, imageName)
		if err := module.TagImage(ctx, e.dockerClient, pullResult.ActualImageName, imageName); err != nil {
			logx.Errorf("重新打标签失败: %v", err)
		}
	}

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

	// 8. 级联更新使用同一旧镜像的其他容器，然后清理旧镜像
	oldImageID := mc.ImageID
	if oldImageID != "" && oldImageID != newInspect.ID {
		reportProgress(92, "级联更新", "正在更新使用同一镜像的其他容器")
		e.cascadeUpdateContainers(ctx, oldImageID, imageName, mc.ID)

		reportProgress(98, "清理旧镜像", "正在清理旧镜像")
		// 再次检查旧镜像是否还被使用（级联更新后应该没有了）
		if !e.isImageInUse(ctx, oldImageID) {
			logx.Infof("清理旧镜像: %s", oldImageID[:12])
			_, err := e.dockerClient.ImageRemove(ctx, oldImageID, image.RemoveOptions{})
			if err != nil {
				// 删除旧镜像失败不影响更新结果，只记录日志
				logx.Errorf("删除旧镜像失败: %v", err)
			}
		}
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

	// 使用统一的镜像拉取方法（支持加速器和私有 Registry 认证）
	pullOpts := module.PullImageOptions{
		ImageName:      img.FullName,
		UseAccelerator: true,
		RegistryAuth:   module.GetPullAuthForImageByMeta(img.ID, img.FullName),
	}
	pullOut, pullResult, err := module.PullImage(ctx, e.dockerClient, pullOpts)
	if err != nil {
		return false, fmt.Errorf("拉取镜像失败: %w", err)
	}
	defer pullOut.Close()
	_, _ = io.Copy(io.Discard, pullOut)

	// 如果使用了加速器，需要重新打标签
	if pullResult.ActualImageName != img.FullName {
		logx.Infof("重新打标签: %s -> %s", pullResult.ActualImageName, img.FullName)
		if err := module.TagImage(ctx, e.dockerClient, pullResult.ActualImageName, img.FullName); err != nil {
			logx.Errorf("重新打标签失败: %v", err)
		}
	}

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

// PullImageWithProgress 拉取镜像并级联重建使用该镜像的容器（带进度回调）
func (e *Executor) PullImageWithProgress(ctx context.Context, img MatchedImage, onProgress ProgressCallback) error {
	if onProgress != nil {
		onProgress(5, "开始拉取", fmt.Sprintf("正在拉取镜像 %s", img.FullName))
	}

	// 记录旧镜像ID，用于后续级联更新和清理
	oldImageID := img.ID

	// 使用统一的镜像拉取方法（支持加速器和私有 Registry 认证）
	pullOpts := module.PullImageOptions{
		ImageName:      img.FullName,
		UseAccelerator: true,
		RegistryAuth:   module.GetPullAuthForImageByMeta(img.ID, img.FullName),
	}
	pullOut, pullResult, err := module.PullImage(ctx, e.dockerClient, pullOpts)
	if err != nil {
		return fmt.Errorf("拉取镜像失败: %w", err)
	}
	defer pullOut.Close()

	// 解析镜像拉取进度（拉取过程在5%-50%之间）
	parsePullProgress(pullOut, 5, 50, onProgress)

	// 如果使用了加速器，需要重新打标签
	if pullResult.ActualImageName != img.FullName {
		logx.Infof("重新打标签: %s -> %s", pullResult.ActualImageName, img.FullName)
		if err := module.TagImage(ctx, e.dockerClient, pullResult.ActualImageName, img.FullName); err != nil {
			logx.Errorf("重新打标签失败: %v", err)
		}
	}

	// 获取新镜像ID
	newInspect, _, err := e.dockerClient.ImageInspectWithRaw(ctx, img.FullName)
	if err != nil {
		return fmt.Errorf("获取新镜像信息失败: %w", err)
	}

	// 如果镜像确实更新了（ID 不同），级联重建使用该镜像的容器
	if oldImageID != newInspect.ID {
		if onProgress != nil {
			onProgress(55, "级联重建容器", "正在重建使用该镜像的容器")
		}
		logx.Infof("镜像[%s]已更新，开始级联重建容器 (旧ID: %s, 新ID: %s)",
			img.FullName, oldImageID[:12], newInspect.ID[:12])

		// 级联更新使用旧镜像的容器
		e.cascadeUpdateContainers(ctx, oldImageID, img.FullName, "")

		// 清理旧镜像
		if onProgress != nil {
			onProgress(90, "清理旧镜像", "正在清理旧镜像")
		}
		if !e.isImageInUse(ctx, oldImageID) {
			logx.Infof("清理旧镜像: %s", oldImageID[:12])
			_, err := e.dockerClient.ImageRemove(ctx, oldImageID, image.RemoveOptions{})
			if err != nil {
				logx.Errorf("删除旧镜像失败: %v", err)
			}
		}
	}

	// 更新缓存
	if e.hubImageInfo != nil {
		e.hubImageInfo.MarkAsUpdated(newInspect.ID, img.FullName)
		logx.Infof("已更新镜像缓存: %s (ID: %s)", img.FullName, newInspect.ID[:12])
	}

	if onProgress != nil {
		onProgress(100, "更新完成", "镜像和容器更新成功")
	}
	logx.Infof("镜像[%s]更新完成", img.FullName)
	return nil
}
