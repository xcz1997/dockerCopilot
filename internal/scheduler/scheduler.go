package scheduler

import (
	"context"
	"sync"

	"github.com/docker/docker/client"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

// GroupScheduler 群组调度器
type GroupScheduler struct {
	cron          *cron.Cron
	dockerClient  *client.Client
	hubImageInfo  *module.ImageUpdateData
	jobs          map[int64]cron.EntryID // groupID -> entryID
	mu            sync.RWMutex
	progressFunc  func(taskID string, percentage int, message string, name string, detailMsg string, isDone bool)
}

// NewGroupScheduler 创建群组调度器
func NewGroupScheduler(dockerClient *client.Client, hubImageInfo *module.ImageUpdateData) *GroupScheduler {
	return &GroupScheduler{
		cron:         cron.New(cron.WithSeconds()),
		dockerClient: dockerClient,
		hubImageInfo: hubImageInfo,
		jobs:         make(map[int64]cron.EntryID),
	}
}

// SetProgressFunc 设置进度回调函数
func (s *GroupScheduler) SetProgressFunc(fn func(taskID string, percentage int, message string, name string, detailMsg string, isDone bool)) {
	s.progressFunc = fn
}

// Start 启动调度器
func (s *GroupScheduler) Start() error {
	// 加载所有启用的群组
	groups, err := model.GetEnabledGroups()
	if err != nil {
		logx.Errorf("加载群组失败: %v", err)
		return err
	}

	for _, group := range groups {
		if group.CronExpr != "" {
			if err := s.AddJob(group); err != nil {
				logx.Errorf("添加群组[%s]定时任务失败: %v", group.Name, err)
			}
		}
	}

	s.cron.Start()
	logx.Info("群组调度器已启动")
	return nil
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

	// 添加新任务
	entryID, err := s.cron.AddFunc(group.CronExpr, func() {
		s.executeGroup(group.ID)
	})
	if err != nil {
		return err
	}

	s.jobs[group.ID] = entryID
	logx.Infof("群组[%s]定时任务已添加: %s", group.Name, group.CronExpr)
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

// TriggerGroup 手动触发群组检查/更新
func (s *GroupScheduler) TriggerGroup(groupID int64, forceUpdate bool) {
	go func() {
		if forceUpdate {
			s.executeGroupUpdate(groupID)
		} else {
			s.executeGroup(groupID)
		}
	}()
}

// executeGroup 执行群组定时任务
func (s *GroupScheduler) executeGroup(groupID int64) {
	group, err := model.GetGroupByID(groupID)
	if err != nil {
		logx.Errorf("获取群组[%d]失败: %v", groupID, err)
		return
	}

	if !group.Enabled {
		logx.Infof("群组[%s]已禁用，跳过执行", group.Name)
		return
	}

	logx.Infof("开始执行群组[%s]定时任务", group.Name)

	// 获取匹配的容器
	containers, err := s.getMatchedContainers(groupID)
	if err != nil {
		logx.Errorf("获取群组[%s]容器失败: %v", group.Name, err)
		return
	}

	if len(containers) == 0 {
		logx.Infof("群组[%s]没有匹配的容器", group.Name)
		return
	}

	logx.Infof("群组[%s]匹配到%d个容器", group.Name, len(containers))

	// 根据群组配置执行
	if group.CheckUpdate {
		for _, container := range containers {
			if group.AutoUpdate {
				// 检查并自动更新
				s.checkAndUpdate(group, container)
			} else {
				// 仅检查更新
				s.checkOnly(group, container)
			}
		}
	}

	logx.Infof("群组[%s]定时任务执行完成", group.Name)
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

// getMatchedContainers 获取匹配群组的容器
func (s *GroupScheduler) getMatchedContainers(groupID int64) ([]MatchedContainer, error) {
	matcher := NewMatcher(s.dockerClient)
	return matcher.GetMatchedContainers(context.Background(), groupID)
}

// checkOnly 仅检查更新
func (s *GroupScheduler) checkOnly(group *model.ContainerGroup, container MatchedContainer) {
	executor := NewExecutor(s.dockerClient, s.hubImageInfo)
	hasUpdate, err := executor.CheckUpdate(context.Background(), container)
	if err != nil {
		logx.Errorf("检查容器[%s]更新失败: %v", container.Name, err)
		return
	}

	if hasUpdate {
		logx.Infof("容器[%s]有可用更新", container.Name)
	} else {
		logx.Infof("容器[%s]已是最新版本", container.Name)
	}
}

// checkAndUpdate 检查并更新
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
