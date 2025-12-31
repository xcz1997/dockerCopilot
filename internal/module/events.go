package module

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContainerEventWatcher Docker 容器事件监听器
type ContainerEventWatcher struct {
	client *client.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// NewContainerEventWatcher 创建事件监听器
func NewContainerEventWatcher(cli *client.Client) *ContainerEventWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &ContainerEventWatcher{
		client: cli,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动事件监听
func (w *ContainerEventWatcher) Start() {
	go w.watchEvents()
	logx.Info("Docker 容器事件监听器已启动")
}

// Stop 停止事件监听
func (w *ContainerEventWatcher) Stop() {
	w.cancel()
	logx.Info("Docker 容器事件监听器已停止")
}

// watchEvents 监听 Docker 事件
func (w *ContainerEventWatcher) watchEvents() {
	// 过滤只监听容器事件
	eventChan, errChan := w.client.Events(w.ctx, events.ListOptions{})

	for {
		select {
		case <-w.ctx.Done():
			return
		case err := <-errChan:
			if err != nil {
				logx.Errorf("Docker 事件监听错误: %v", err)
				// 等待一段时间后重试
				time.Sleep(5 * time.Second)
				eventChan, errChan = w.client.Events(w.ctx, events.ListOptions{})
			}
		case event := <-eventChan:
			w.handleEvent(event)
		}
	}
}

// handleEvent 处理单个事件
func (w *ContainerEventWatcher) handleEvent(event events.Message) {
	// 只处理容器事件
	if event.Type != events.ContainerEventType {
		return
	}

	// 获取配置
	config, err := model.GetContainerEventConfig()
	if err != nil {
		logx.Errorf("获取容器事件通知配置失败: %v", err)
		return
	}

	// 检查是否启用
	if !config.Enabled {
		return
	}

	// 获取容器名称
	containerName := event.Actor.Attributes["name"]
	if containerName == "" {
		containerName = event.Actor.ID[:12]
	}

	// 获取镜像名称
	imageName := event.Actor.Attributes["image"]

	// 根据事件类型判断是否需要发送通知
	var shouldNotify bool
	var title, body string

	switch event.Action {
	case "start":
		shouldNotify = config.NotifyOnStart
		title = "容器已启动"
		body = fmt.Sprintf("容器 %s 已启动\n镜像: %s", containerName, imageName)

	case "stop":
		shouldNotify = config.NotifyOnStop
		title = "容器已停止"
		body = fmt.Sprintf("容器 %s 已停止\n镜像: %s", containerName, imageName)

	case "die":
		shouldNotify = config.NotifyOnDie
		exitCode := event.Actor.Attributes["exitCode"]
		title = "容器异常退出"
		body = fmt.Sprintf("容器 %s 异常退出\n退出码: %s\n镜像: %s", containerName, exitCode, imageName)

	case "restart":
		shouldNotify = config.NotifyOnRestart
		title = "容器已重启"
		body = fmt.Sprintf("容器 %s 已重启\n镜像: %s", containerName, imageName)

	case "create":
		shouldNotify = config.NotifyOnCreate
		title = "容器已创建"
		body = fmt.Sprintf("容器 %s 已创建\n镜像: %s", containerName, imageName)

	case "destroy":
		shouldNotify = config.NotifyOnDestroy
		title = "容器已删除"
		body = fmt.Sprintf("容器 %s 已删除", containerName)

	// 健康检查事件
	case "health_status: healthy":
		shouldNotify = config.NotifyOnHealthy
		title = "容器健康检查通过"
		body = fmt.Sprintf("容器 %s 健康检查通过\n镜像: %s", containerName, imageName)

	case "health_status: unhealthy":
		shouldNotify = config.NotifyOnUnhealthy
		title = "容器健康检查失败"
		body = fmt.Sprintf("容器 %s 健康检查失败\n镜像: %s", containerName, imageName)

	default:
		// 处理带前缀的健康检查事件
		if strings.HasPrefix(string(event.Action), "health_status") {
			if strings.Contains(string(event.Action), "healthy") && !strings.Contains(string(event.Action), "unhealthy") {
				shouldNotify = config.NotifyOnHealthy
				title = "容器健康检查通过"
				body = fmt.Sprintf("容器 %s 健康检查通过\n镜像: %s", containerName, imageName)
			} else if strings.Contains(string(event.Action), "unhealthy") {
				shouldNotify = config.NotifyOnUnhealthy
				title = "容器健康检查失败"
				body = fmt.Sprintf("容器 %s 健康检查失败\n镜像: %s", containerName, imageName)
			}
		}
		return
	}

	if !shouldNotify {
		return
	}

	// 发送通知
	if err := SendBarkNotification(title, body); err != nil {
		logx.Errorf("发送容器事件通知失败: %v", err)
	} else {
		logx.Infof("已发送容器事件通知: %s - %s", title, containerName)
	}
}
