package scheduler

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/robfig/cron/v3"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/zeromicro/go-zero/core/logx"
)

// TaskProgressUpdater 任务进度更新接口
type TaskProgressUpdater interface {
	UpdateProgress(taskID string, percentage int, message string, name string, detailMsg string, isDone bool)
	UpdateProgressWithMeta(taskID string, percentage int, message string, name string, detailMsg string, isDone bool, taskType string, targetID string, targetName string)
	UpdateSubTask(taskID string, subTaskName string, status string, message string)
	UpdateSubTaskWithProgress(taskID string, subTaskName string, status string, message string, detailMsg string, percentage int)
}

// GroupScheduler 群组调度器
type GroupScheduler struct {
	cron            *cron.Cron
	dockerClient    *client.Client
	hubImageInfo    *module.ImageUpdateData
	jobs            map[int64]cron.EntryID // groupID -> entryID
	mu              sync.RWMutex
	progressUpdater TaskProgressUpdater
}

// NewGroupScheduler 创建群组调度器
func NewGroupScheduler(dockerClient *client.Client, hubImageInfo *module.ImageUpdateData) *GroupScheduler {
	// 使用标准的 5 字段 cron 格式 (分 时 日 月 周)，与前端预设一致
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	// 获取时区，优先使用 TZ 环境变量，默认 Asia/Shanghai
	var loc *time.Location
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		logx.Errorf("加载时区[%s]失败: %v，使用本地时区", tz, err)
		loc = time.Local
	} else {
		logx.Infof("调度器使用时区: %s", tz)
	}

	return &GroupScheduler{
		cron:         cron.New(cron.WithParser(parser), cron.WithLocation(loc)),
		dockerClient: dockerClient,
		hubImageInfo: hubImageInfo,
		jobs:         make(map[int64]cron.EntryID),
	}
}

// SetProgressUpdater 设置进度更新器
func (s *GroupScheduler) SetProgressUpdater(updater TaskProgressUpdater) {
	s.progressUpdater = updater
}

// updateProgress 更新任务进度
func (s *GroupScheduler) updateProgress(taskID string, percentage int, message string, name string, detailMsg string, isDone bool) {
	if s.progressUpdater != nil {
		s.progressUpdater.UpdateProgress(taskID, percentage, message, name, detailMsg, isDone)
	}
}

// updateProgressWithMeta 更新任务进度（带元数据）
func (s *GroupScheduler) updateProgressWithMeta(taskID string, percentage int, message string, name string, detailMsg string, isDone bool, taskType string, targetID string, targetName string) {
	if s.progressUpdater != nil {
		s.progressUpdater.UpdateProgressWithMeta(taskID, percentage, message, name, detailMsg, isDone, taskType, targetID, targetName)
	}
}

// updateSubTask 更新子任务
func (s *GroupScheduler) updateSubTask(taskID string, subTaskName string, status string, message string) {
	if s.progressUpdater != nil {
		s.progressUpdater.UpdateSubTask(taskID, subTaskName, status, message)
	}
}

// updateSubTaskWithProgress 更新子任务（带详细进度）
func (s *GroupScheduler) updateSubTaskWithProgress(taskID string, subTaskName string, status string, message string, detailMsg string, percentage int) {
	if s.progressUpdater != nil {
		s.progressUpdater.UpdateSubTaskWithProgress(taskID, subTaskName, status, message, detailMsg, percentage)
	}
}

// generateTaskID 生成任务ID
func (s *GroupScheduler) generateTaskID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// Start 启动调度器
func (s *GroupScheduler) Start() error {
	// 加载所有启用的群组
	groups, err := model.GetEnabledGroups()
	if err != nil {
		logx.Errorf("加载群组失败: %v", err)
		return err
	}

	logx.Infof("找到 %d 个启用的群组", len(groups))

	loadedCount := 0
	for _, group := range groups {
		if group.CronExpr != "" {
			if err := s.AddJob(group); err != nil {
				logx.Errorf("添加群组[%s]定时任务失败: %v", group.Name, err)
			} else {
				loadedCount++
			}
		} else {
			logx.Infof("群组[%s]没有设置 cron 表达式，跳过", group.Name)
		}
	}

	s.cron.Start()
	logx.Infof("群组调度器已启动，加载了 %d 个定时任务", loadedCount)

	// 启动定期状态报告 (每小时)
	go s.periodicStatusReport()

	return nil
}

// periodicStatusReport 定期打印调度器状态
func (s *GroupScheduler) periodicStatusReport() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.PrintStatus()
	}
}

// PrintStatus 打印调度器当前状态
func (s *GroupScheduler) PrintStatus() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.cron.Entries()
	logx.Infof("=== 调度器状态: 共 %d 个定时任务 ===", len(entries))

	for groupID, entryID := range s.jobs {
		entry := s.cron.Entry(entryID)
		if entry.Valid() {
			logx.Infof("  群组ID=%d: 下次执行=%s", groupID, entry.Next.Format("2006-01-02 15:04:05"))
		} else {
			logx.Errorf("  群组ID=%d: 任务无效", groupID)
		}
	}
}

// Stop 停止调度器
func (s *GroupScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	logx.Info("群组调度器已停止")
}

// AddJob 添加群组定时任务
func (s *GroupScheduler) AddJob(group model.ContainerGroup) error {
	if group.CronExpr == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已存在，先移除
	if entryID, exists := s.jobs[group.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, group.ID)
	}

	// 显式捕获 groupID 和 groupName，避免闭包捕获问题
	groupID := group.ID
	groupName := group.Name
	cronExpr := group.CronExpr

	// 添加新任务 - 使用带进度跟踪的版本，这样任务会显示在任务列表中
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		logx.Infof("=== Cron 任务触发: 群组[%s] ID=%d, 时间=%s ===", groupName, groupID, time.Now().Format("2006-01-02 15:04:05"))
		taskID := s.generateTaskID("cron")
		s.executeGroupWithProgress(groupID, taskID)
	})
	if err != nil {
		logx.Errorf("添加群组[%s] cron 任务失败: %v (表达式: %s)", group.Name, err, cronExpr)
		return err
	}

	s.jobs[group.ID] = entryID

	// 获取下次执行时间（使用 Schedule.Next 手动计算，避免 cron 未 Start 时 entry.Next 为零值）
	entry := s.cron.Entry(entryID)
	var nextRun string
	if entry.Schedule != nil {
		nextRun = entry.Schedule.Next(time.Now()).Format("2006-01-02 15:04:05")
	} else {
		nextRun = "未知"
	}
	logx.Infof("群组[%s]定时任务已添加: cron=%s, 下次执行=%s", group.Name, cronExpr, nextRun)
	return nil
}

// RemoveJob 移除群组定时任务
func (s *GroupScheduler) RemoveJob(groupID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[groupID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, groupID)
		logx.Infof("群组[%d]定时任务已移除", groupID)
	}
}

// UpdateJob 更新群组定时任务
func (s *GroupScheduler) UpdateJob(group model.ContainerGroup) error {
	s.RemoveJob(group.ID)
	if group.Enabled && group.CronExpr != "" {
		return s.AddJob(group)
	}
	return nil
}

// TriggerGroup 手动触发群组检查/更新，返回任务ID
func (s *GroupScheduler) TriggerGroup(groupID int64, forceUpdate bool) string {
	taskID := s.generateTaskID("group")
	go func() {
		if forceUpdate {
			s.executeGroupUpdateWithProgress(groupID, taskID)
		} else {
			s.executeGroupWithProgress(groupID, taskID)
		}
	}()
	return taskID
}

// executeGroupUpdate 强制执行群组更新
func (s *GroupScheduler) executeGroupUpdate(groupID int64) {
	group, err := model.GetGroupByID(groupID)
	if err != nil {
		logx.Errorf("获取群组[%d]失败: %v", groupID, err)
		return
	}

	logx.Infof("开始执行群组[%s]强制更新", group.Name)

	containers, err := s.getMatchedContainers(groupID)
	if err != nil {
		logx.Errorf("获取群组[%s]容器失败: %v", group.Name, err)
		return
	}

	for _, container := range containers {
		s.checkAndUpdate(group, container)
	}

	logx.Infof("群组[%s]强制更新完成", group.Name)
}

// executeGroupWithProgress 执行群组定时任务（带进度跟踪）
func (s *GroupScheduler) executeGroupWithProgress(groupID int64, taskID string) {
	taskName := fmt.Sprintf("群组任务 #%d", groupID)
	groupIDStr := fmt.Sprintf("%d", groupID)

	group, err := model.GetGroupByID(groupID)
	if err != nil {
		logx.Errorf("获取群组[%d]失败: %v", groupID, err)
		s.updateProgressWithMeta(taskID, 0, "获取群组失败", taskName, err.Error(), true, "group_task", groupIDStr, "")
		return
	}

	taskName = fmt.Sprintf("群组任务: %s", group.Name)
	s.updateProgressWithMeta(taskID, 5, "开始执行群组任务", taskName, "正在获取匹配的容器", false, "group_task", groupIDStr, group.Name)

	if !group.Enabled {
		s.updateProgressWithMeta(taskID, 100, "群组已禁用", taskName, "跳过执行", true, "group_task", groupIDStr, group.Name)
		return
	}

	containers, err := s.getMatchedContainers(groupID)
	if err != nil {
		logx.Errorf("获取群组[%s]容器失败: %v", group.Name, err)
		s.updateProgressWithMeta(taskID, 0, "获取容器失败", taskName, err.Error(), true, "group_task", groupIDStr, group.Name)
		return
	}

	if len(containers) == 0 {
		s.updateProgressWithMeta(taskID, 100, "完成", taskName, "没有匹配的容器", true, "group_task", groupIDStr, group.Name)
		return
	}

	total := len(containers)
	updated := 0
	failed := 0
	skipped := 0

	s.updateProgress(taskID, 10, fmt.Sprintf("找到 %d 个容器", total), taskName, "开始执行任务", false)

	hasUpdateCount := 0 // 检查模式下，有可用更新的数量
	var taskDetails []module.TaskDetail

	// 执行检查/更新任务（如果启用了 CheckUpdate）
	if group.CheckUpdate {
		for i, ctr := range containers {
			progress := 10 + (i+1)*40/total
			s.updateProgress(taskID, progress, fmt.Sprintf("检查 %s (%d/%d)", ctr.Name, i+1, total), taskName, "", false)

			// 更新子任务状态
			s.updateSubTask(taskID, ctr.Name, "in_progress", "检查中...")

			if group.AutoUpdate {
				result := s.checkAndUpdateWithResult(group, ctr)
				if result == "updated" {
					updated++
					s.updateSubTask(taskID, ctr.Name, "completed", "已更新")
					taskDetails = append(taskDetails, module.TaskDetail{Name: ctr.Name, Status: "updated"})
				} else if result == "failed" {
					failed++
					s.updateSubTask(taskID, ctr.Name, "failed", "更新失败")
					taskDetails = append(taskDetails, module.TaskDetail{Name: ctr.Name, Status: "failed", Message: "更新失败"})
				} else {
					skipped++
					s.updateSubTask(taskID, ctr.Name, "completed", "已是最新")
				}
			} else {
				hasUpdate := s.checkOnly(group, ctr)
				if hasUpdate {
					hasUpdateCount++
					s.updateSubTask(taskID, ctr.Name, "completed", "有可用更新")
					taskDetails = append(taskDetails, module.TaskDetail{Name: ctr.Name, Status: "has_update"})
				} else {
					skipped++
					s.updateSubTask(taskID, ctr.Name, "completed", "已是最新")
				}
			}
		}
	}

	// 执行容器操作（独立于更新，按 cron 定时执行）
	s.executeContainerOperations(group, containers, taskID, taskName, groupIDStr)

	// 生成任务摘要
	var summaryParts []string
	if group.CheckUpdate {
		if group.AutoUpdate {
			summaryParts = append(summaryParts, fmt.Sprintf("更新: %d, 跳过: %d, 失败: %d", updated, skipped, failed))
		} else if hasUpdateCount > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("发现 %d 个可更新", hasUpdateCount))
		}
	}
	if group.RestartAfterUpdate {
		summaryParts = append(summaryParts, "已执行重启")
	}
	if group.StartContainers {
		summaryParts = append(summaryParts, "已执行启动")
	}
	if group.StopContainers {
		summaryParts = append(summaryParts, "已执行停止")
	}

	summary := "任务完成"
	if len(summaryParts) > 0 {
		summary = fmt.Sprintf("%s (%d 个容器)", summaryParts[0], total)
		if len(summaryParts) > 1 {
			summary = fmt.Sprintf("%s; %s", summaryParts[0], summaryParts[1])
		}
	}

	s.updateProgressWithMeta(taskID, 100, "全部完成", taskName, summary, true, "group_task", groupIDStr, group.Name)
	logx.Infof("群组[%s]定时任务执行完成: %s", group.Name, summary)

	// 发送 Bark 通知
	if group.CheckUpdate {
		if group.AutoUpdate {
			module.NotifyGroupTaskCompleteWithDetails(group.Name, "update", updated, skipped, failed, taskDetails)
		} else {
			module.NotifyGroupTaskCompleteWithDetails(group.Name, "check", hasUpdateCount, skipped, 0, taskDetails)
		}
	}
}

// executeGroupUpdateWithProgress 强制执行群组更新（带进度跟踪）
func (s *GroupScheduler) executeGroupUpdateWithProgress(groupID int64, taskID string) {
	taskName := fmt.Sprintf("群组更新 #%d", groupID)
	groupIDStr := fmt.Sprintf("%d", groupID)

	group, err := model.GetGroupByID(groupID)
	if err != nil {
		logx.Errorf("获取群组[%d]失败: %v", groupID, err)
		s.updateProgressWithMeta(taskID, 0, "获取群组失败", taskName, err.Error(), true, "group_update", groupIDStr, "")
		return
	}

	taskName = fmt.Sprintf("群组更新: %s", group.Name)

	// 根据群组类型执行不同的更新逻辑
	switch group.GroupType {
	case model.GroupTypeImage:
		s.executeImageGroupUpdate(group, taskID, taskName, groupIDStr)
	default:
		// container 和 project 类型都使用容器更新逻辑
		s.executeContainerGroupUpdate(group, taskID, taskName, groupIDStr)
	}
}

// executeContainerGroupUpdate 执行容器群组更新
func (s *GroupScheduler) executeContainerGroupUpdate(group *model.ContainerGroup, taskID, taskName, groupIDStr string) {
	s.updateProgressWithMeta(taskID, 5, "开始更新群组", taskName, "正在获取匹配的容器", false, "group_update", groupIDStr, group.Name)

	containers, err := s.getMatchedContainers(group.ID)
	if err != nil {
		logx.Errorf("获取群组[%s]容器失败: %v", group.Name, err)
		s.updateProgressWithMeta(taskID, 0, "获取容器失败", taskName, err.Error(), true, "group_update", groupIDStr, group.Name)
		return
	}

	if len(containers) == 0 {
		s.updateProgressWithMeta(taskID, 100, "完成", taskName, "没有匹配的容器", true, "group_update", groupIDStr, group.Name)
		return
	}

	total := len(containers)
	updated := 0
	failed := 0
	skipped := 0
	var taskDetails []module.TaskDetail

	s.updateProgress(taskID, 10, fmt.Sprintf("找到 %d 个容器", total), taskName, "开始更新", false)

	for i, container := range containers {
		progress := 10 + (i+1)*80/total
		s.updateProgress(taskID, progress, fmt.Sprintf("更新 %s (%d/%d)", container.Name, i+1, total), taskName, "", false)

		// 更新子任务状态
		s.updateSubTaskWithProgress(taskID, container.Name, "in_progress", "检查更新", "正在检查是否有可用更新...", 5)

		executor := NewExecutor(s.dockerClient, s.hubImageInfo)
		hasUpdate, err := executor.CheckUpdate(context.Background(), container)
		if err != nil {
			logx.Errorf("检查容器[%s]更新失败: %v", container.Name, err)
			s.recordHistory(group.ID, container, "", "", model.UpdateStatusFailed, err.Error())
			s.updateSubTaskWithProgress(taskID, container.Name, "failed", "检查失败", err.Error(), 100)
			taskDetails = append(taskDetails, module.TaskDetail{Name: container.Name, Status: "failed", Message: "检查失败"})
			failed++
			continue
		}

		if !hasUpdate {
			s.updateSubTaskWithProgress(taskID, container.Name, "completed", "已是最新", "当前版本已是最新，无需更新", 100)
			skipped++
			continue
		}

		// 创建进度回调，用于报告子任务详细进度
		containerName := container.Name // 捕获变量
		onProgress := func(pct int, msg, detail string) {
			s.updateSubTaskWithProgress(taskID, containerName, "in_progress", msg, detail, pct)
		}

		oldImage := container.Image
		result, err := executor.UpdateWithProgress(context.Background(), container, onProgress)
		if err != nil {
			logx.Errorf("容器[%s]更新失败: %v", container.Name, err)
			s.recordHistory(group.ID, container, oldImage, "", model.UpdateStatusFailed, err.Error())
			s.updateSubTaskWithProgress(taskID, container.Name, "failed", "更新失败", err.Error(), 100)
			taskDetails = append(taskDetails, module.TaskDetail{Name: container.Name, Status: "failed", Message: err.Error()})
			failed++
			continue
		}

		logx.Infof("容器[%s]更新成功: %s -> %s", container.Name, oldImage, result.NewImage)
		s.recordHistory(group.ID, container, oldImage, result.NewImage, model.UpdateStatusSuccess, "")
		s.updateSubTaskWithProgress(taskID, container.Name, "completed", "更新成功", "容器已成功更新到最新版本", 100)
		taskDetails = append(taskDetails, module.TaskDetail{Name: container.Name, Status: "updated"})
		updated++
	}

	summary := fmt.Sprintf("更新: %d, 跳过: %d, 失败: %d", updated, skipped, failed)
	s.updateProgressWithMeta(taskID, 100, "更新完成", taskName, summary, true, "group_update", groupIDStr, group.Name)
	logx.Infof("群组[%s]强制更新完成: %s", group.Name, summary)

	// 发送 Bark 通知
	module.NotifyGroupTaskCompleteWithDetails(group.Name, "update", updated, skipped, failed, taskDetails)
}

// executeContainerOperations 执行容器操作（重启/启动/停止）
func (s *GroupScheduler) executeContainerOperations(group *model.ContainerGroup, containers []MatchedContainer, taskID, taskName, groupIDStr string) {
	ctx := context.Background()

	// 重启容器
	if group.RestartAfterUpdate {
		logx.Infof("群组[%s]开始执行重启容器操作", group.Name)
		s.updateProgress(taskID, 96, "执行重启操作", taskName, "正在重启容器...", false)

		restartSuccess := 0
		restartFailed := 0
		for _, c := range containers {
			err := s.dockerClient.ContainerRestart(ctx, c.ID, container.StopOptions{})
			if err != nil {
				logx.Errorf("重启容器[%s]失败: %v", c.Name, err)
				restartFailed++
			} else {
				logx.Infof("重启容器[%s]成功", c.Name)
				restartSuccess++
			}
		}
		logx.Infof("群组[%s]重启操作完成: 成功 %d, 失败 %d", group.Name, restartSuccess, restartFailed)
	}

	// 启动容器
	if group.StartContainers {
		logx.Infof("群组[%s]开始执行启动容器操作", group.Name)
		s.updateProgress(taskID, 97, "执行启动操作", taskName, "正在启动容器...", false)

		startSuccess := 0
		startFailed := 0
		startSkipped := 0
		for _, c := range containers {
			// 检查容器状态，跳过已运行的
			inspect, err := s.dockerClient.ContainerInspect(ctx, c.ID)
			if err != nil {
				logx.Errorf("检查容器[%s]状态失败: %v", c.Name, err)
				startFailed++
				continue
			}

			if inspect.State.Running {
				startSkipped++
				continue
			}

			err = s.dockerClient.ContainerStart(ctx, c.ID, container.StartOptions{})
			if err != nil {
				logx.Errorf("启动容器[%s]失败: %v", c.Name, err)
				startFailed++
			} else {
				logx.Infof("启动容器[%s]成功", c.Name)
				startSuccess++
			}
		}
		logx.Infof("群组[%s]启动操作完成: 成功 %d, 跳过 %d, 失败 %d", group.Name, startSuccess, startSkipped, startFailed)
	}

	// 停止容器
	if group.StopContainers {
		logx.Infof("群组[%s]开始执行停止容器操作", group.Name)
		s.updateProgress(taskID, 98, "执行停止操作", taskName, "正在停止容器...", false)

		stopSuccess := 0
		stopFailed := 0
		stopSkipped := 0
		for _, c := range containers {
			// 检查容器状态，跳过已停止的
			inspect, err := s.dockerClient.ContainerInspect(ctx, c.ID)
			if err != nil {
				logx.Errorf("检查容器[%s]状态失败: %v", c.Name, err)
				stopFailed++
				continue
			}

			if !inspect.State.Running {
				stopSkipped++
				continue
			}

			err = s.dockerClient.ContainerStop(ctx, c.ID, container.StopOptions{})
			if err != nil {
				logx.Errorf("停止容器[%s]失败: %v", c.Name, err)
				stopFailed++
			} else {
				logx.Infof("停止容器[%s]成功", c.Name)
				stopSuccess++
			}
		}
		logx.Infof("群组[%s]停止操作完成: 成功 %d, 跳过 %d, 失败 %d", group.Name, stopSuccess, stopSkipped, stopFailed)
	}
}

// executeImageGroupUpdate 执行镜像群组更新
func (s *GroupScheduler) executeImageGroupUpdate(group *model.ContainerGroup, taskID, taskName, groupIDStr string) {
	s.updateProgressWithMeta(taskID, 5, "开始更新群组", taskName, "正在获取匹配的镜像", false, "group_update", groupIDStr, group.Name)

	images, err := s.getMatchedImages(group.ID)
	if err != nil {
		logx.Errorf("获取群组[%s]镜像失败: %v", group.Name, err)
		s.updateProgressWithMeta(taskID, 0, "获取镜像失败", taskName, err.Error(), true, "group_update", groupIDStr, group.Name)
		return
	}

	if len(images) == 0 {
		s.updateProgressWithMeta(taskID, 100, "完成", taskName, "没有匹配的镜像", true, "group_update", groupIDStr, group.Name)
		return
	}

	total := len(images)
	updated := 0
	failed := 0
	skipped := 0
	var taskDetails []module.TaskDetail

	s.updateProgress(taskID, 10, fmt.Sprintf("找到 %d 个镜像", total), taskName, "开始更新", false)

	executor := NewExecutor(s.dockerClient, s.hubImageInfo)

	for i, img := range images {
		progress := 10 + (i+1)*80/total
		s.updateProgress(taskID, progress, fmt.Sprintf("更新 %s (%d/%d)", img.FullName, i+1, total), taskName, "", false)

		// 更新子任务状态
		imageName := img.FullName // 捕获变量
		s.updateSubTaskWithProgress(taskID, imageName, "in_progress", "检查更新", "正在检查是否有可用更新...", 5)

		// 检查镜像是否有更新
		hasUpdate, err := executor.CheckImageUpdate(context.Background(), img)
		if err != nil {
			logx.Errorf("检查镜像[%s]更新失败: %v", img.FullName, err)
			s.updateSubTaskWithProgress(taskID, imageName, "failed", "检查失败", err.Error(), 100)
			taskDetails = append(taskDetails, module.TaskDetail{Name: img.FullName, Status: "failed", Message: "检查失败"})
			failed++
			continue
		}

		if !hasUpdate {
			s.updateSubTaskWithProgress(taskID, imageName, "completed", "已是最新", "当前版本已是最新，无需更新", 100)
			skipped++
			continue
		}

		// 创建进度回调
		onProgress := func(pct int, msg, detail string) {
			s.updateSubTaskWithProgress(taskID, imageName, "in_progress", msg, detail, pct)
		}

		// 拉取新镜像
		err = executor.PullImageWithProgress(context.Background(), img, onProgress)
		if err != nil {
			logx.Errorf("镜像[%s]更新失败: %v", img.FullName, err)
			s.updateSubTaskWithProgress(taskID, imageName, "failed", "更新失败", err.Error(), 100)
			taskDetails = append(taskDetails, module.TaskDetail{Name: img.FullName, Status: "failed", Message: err.Error()})
			failed++
			continue
		}

		logx.Infof("镜像[%s]更新成功", img.FullName)
		s.updateSubTaskWithProgress(taskID, imageName, "completed", "更新成功", "镜像已成功更新到最新版本", 100)
		taskDetails = append(taskDetails, module.TaskDetail{Name: img.FullName, Status: "updated"})
		updated++
	}

	summary := fmt.Sprintf("更新: %d, 跳过: %d, 失败: %d", updated, skipped, failed)
	s.updateProgressWithMeta(taskID, 100, "更新完成", taskName, summary, true, "group_update", groupIDStr, group.Name)
	logx.Infof("群组[%s]镜像更新完成: %s", group.Name, summary)

	// 发送 Bark 通知
	module.NotifyGroupTaskCompleteWithDetails(group.Name, "update", updated, skipped, failed, taskDetails)
}

// getMatchedContainers 获取匹配群组的容器
func (s *GroupScheduler) getMatchedContainers(groupID int64) ([]MatchedContainer, error) {
	matcher := NewMatcher(s.dockerClient)
	return matcher.GetMatchedContainers(context.Background(), groupID)
}

// getMatchedImages 获取匹配群组的镜像
func (s *GroupScheduler) getMatchedImages(groupID int64) ([]MatchedImage, error) {
	matcher := NewMatcher(s.dockerClient)
	return matcher.GetMatchedImages(context.Background(), groupID)
}

// checkOnly 仅检查更新，返回是否有更新
func (s *GroupScheduler) checkOnly(group *model.ContainerGroup, container MatchedContainer) bool {
	executor := NewExecutor(s.dockerClient, s.hubImageInfo)
	hasUpdate, err := executor.CheckUpdate(context.Background(), container)
	if err != nil {
		logx.Errorf("检查容器[%s]更新失败: %v", container.Name, err)
		return false
	}

	if hasUpdate {
		logx.Infof("容器[%s]有可用更新", container.Name)
	} else {
		logx.Infof("容器[%s]已是最新版本", container.Name)
	}
	return hasUpdate
}

// checkAndUpdateWithResult 检查并更新，返回结果: "updated", "skipped", "failed"
func (s *GroupScheduler) checkAndUpdateWithResult(group *model.ContainerGroup, container MatchedContainer) string {
	executor := NewExecutor(s.dockerClient, s.hubImageInfo)

	hasUpdate, err := executor.CheckUpdate(context.Background(), container)
	if err != nil {
		logx.Errorf("检查容器[%s]更新失败: %v", container.Name, err)
		s.recordHistory(group.ID, container, "", "", model.UpdateStatusFailed, err.Error())
		return "failed"
	}

	if !hasUpdate {
		logx.Infof("容器[%s]已是最新版本", container.Name)
		return "skipped"
	}

	logx.Infof("容器[%s]开始更新", container.Name)

	oldImage := container.Image
	result, err := executor.Update(context.Background(), container)
	if err != nil {
		logx.Errorf("容器[%s]更新失败: %v", container.Name, err)
		s.recordHistory(group.ID, container, oldImage, "", model.UpdateStatusFailed, err.Error())
		return "failed"
	}

	logx.Infof("容器[%s]更新成功: %s -> %s", container.Name, oldImage, result.NewImage)
	s.recordHistory(group.ID, container, oldImage, result.NewImage, model.UpdateStatusSuccess, "")
	return "updated"
}

// checkAndUpdate 检查并更新（保持兼容）
func (s *GroupScheduler) checkAndUpdate(group *model.ContainerGroup, container MatchedContainer) {
	executor := NewExecutor(s.dockerClient, s.hubImageInfo)

	hasUpdate, err := executor.CheckUpdate(context.Background(), container)
	if err != nil {
		logx.Errorf("检查容器[%s]更新失败: %v", container.Name, err)
		// 记录失败历史
		s.recordHistory(group.ID, container, "", "", model.UpdateStatusFailed, err.Error())
		return
	}

	if !hasUpdate {
		logx.Infof("容器[%s]已是最新版本", container.Name)
		return
	}

	logx.Infof("容器[%s]开始更新", container.Name)

	oldImage := container.Image
	result, err := executor.Update(context.Background(), container)
	if err != nil {
		logx.Errorf("容器[%s]更新失败: %v", container.Name, err)
		s.recordHistory(group.ID, container, oldImage, "", model.UpdateStatusFailed, err.Error())
		return
	}

	logx.Infof("容器[%s]更新成功: %s -> %s", container.Name, oldImage, result.NewImage)
	s.recordHistory(group.ID, container, oldImage, result.NewImage, model.UpdateStatusSuccess, "")
}

// recordHistory 记录更新历史
func (s *GroupScheduler) recordHistory(groupID int64, container MatchedContainer, oldImage, newImage string, status model.UpdateStatus, message string) {
	history := &model.UpdateHistory{
		GroupID:       &groupID,
		ContainerID:   container.ID,
		ContainerName: container.Name,
		OldImage:      oldImage,
		NewImage:      newImage,
		Status:        status,
		Message:       message,
	}

	if _, err := model.CreateHistory(history); err != nil {
		logx.Errorf("记录更新历史失败: %v", err)
	}
}

// GetJobStatus 获取任务状态
func (s *GroupScheduler) GetJobStatus(groupID int64) (bool, *cron.Entry) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entryID, exists := s.jobs[groupID]
	if !exists {
		return false, nil
	}

	entry := s.cron.Entry(entryID)
	return true, &entry
}

// SchedulerStatus 调度器状态信息
type SchedulerStatus struct {
	Running    bool              `json:"running"`
	Timezone   string            `json:"timezone"`
	TotalJobs  int               `json:"totalJobs"`
	Jobs       []JobStatus       `json:"jobs"`
}

// JobStatus 单个任务状态
type JobStatus struct {
	GroupID   int64  `json:"groupId"`
	GroupName string `json:"groupName"`
	CronExpr  string `json:"cronExpr"`
	NextRun   string `json:"nextRun"`
	Valid     bool   `json:"valid"`
}

// GetSchedulerStatus 获取调度器状态
func (s *GroupScheduler) GetSchedulerStatus() SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	status := SchedulerStatus{
		Running:   true,
		Timezone:  tz,
		TotalJobs: len(s.jobs),
		Jobs:      make([]JobStatus, 0, len(s.jobs)),
	}

	for groupID, entryID := range s.jobs {
		entry := s.cron.Entry(entryID)

		// 获取群组信息
		group, err := model.GetGroupByID(groupID)
		groupName := fmt.Sprintf("群组#%d", groupID)
		cronExpr := ""
		if err == nil && group != nil {
			groupName = group.Name
			cronExpr = group.CronExpr
		}

		jobStatus := JobStatus{
			GroupID:   groupID,
			GroupName: groupName,
			CronExpr:  cronExpr,
			Valid:     entry.Valid(),
		}

		if entry.Valid() {
			jobStatus.NextRun = entry.Next.Format("2006-01-02 15:04:05")
		} else {
			jobStatus.NextRun = "无效"
		}

		status.Jobs = append(status.Jobs, jobStatus)
	}

	return status
}

// ReloadAllJobs 重新加载所有任务
func (s *GroupScheduler) ReloadAllJobs() error {
	s.mu.Lock()
	// 清除所有现有任务
	for groupID, entryID := range s.jobs {
		s.cron.Remove(entryID)
		delete(s.jobs, groupID)
	}
	s.mu.Unlock()

	// 重新加载
	groups, err := model.GetEnabledGroups()
	if err != nil {
		return err
	}

	for _, group := range groups {
		if group.CronExpr != "" {
			if err := s.AddJob(group); err != nil {
				logx.Errorf("重新加载群组[%s]定时任务失败: %v", group.Name, err)
			}
		}
	}

	return nil
}
