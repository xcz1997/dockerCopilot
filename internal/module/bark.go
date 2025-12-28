package module

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// SendBarkNotification 发送 Bark 通知
func SendBarkNotification(title, body string) error {
	config, err := model.GetBarkConfig()
	if err != nil {
		logx.Errorf("获取 Bark 配置失败: %v", err)
		return err
	}

	if !config.Enabled {
		logx.Debug("Bark 推送未启用")
		return nil
	}

	if config.Server == "" || config.Key == "" {
		logx.Debug("Bark 配置不完整")
		return nil
	}

	return sendBark(config.Server, config.Key, title, body)
}

// Docker 图标 URL
const dockerIconURL = "https://www.docker.com/wp-content/uploads/2022/03/Moby-logo.png"

// sendBark 发送 Bark 请求
func sendBark(server, key, title, body string) error {
	// 构建 URL，添加图标参数
	barkURL := fmt.Sprintf("%s/%s/%s/%s?icon=%s&group=DockerCopilot",
		server,
		key,
		url.PathEscape(title),
		url.PathEscape(body),
		url.QueryEscape(dockerIconURL),
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(barkURL)
	if err != nil {
		logx.Errorf("Bark 推送请求失败: %v", err)
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logx.Errorf("Bark 推送返回错误: %d - %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("服务器返回错误: %d", resp.StatusCode)
	}

	logx.Infof("Bark 推送成功: %s", title)
	return nil
}

// NotifyContainerUpdate 容器更新通知
func NotifyContainerUpdate(containerName, oldImage, newImage string, success bool) {
	var title, body string
	if success {
		title = "容器更新成功"
		body = fmt.Sprintf("容器 %s 已从 %s 更新到 %s", containerName, oldImage, newImage)
	} else {
		title = "容器更新失败"
		body = fmt.Sprintf("容器 %s 更新失败，镜像: %s", containerName, newImage)
	}

	if err := SendBarkNotification(title, body); err != nil {
		logx.Errorf("发送容器更新通知失败: %v", err)
	}
}

// NotifyImageUpdate 镜像有更新通知
func NotifyImageUpdate(imageName string) {
	title := "发现镜像更新"
	body := fmt.Sprintf("镜像 %s 有新版本可用", imageName)

	if err := SendBarkNotification(title, body); err != nil {
		logx.Errorf("发送镜像更新通知失败: %v", err)
	}
}

// TaskDetail 任务明细
type TaskDetail struct {
	Name    string // 容器/镜像名称
	Status  string // updated, skipped, failed
	Message string // 详细信息
}

// NotifyGroupTaskComplete 群组任务完成通知
func NotifyGroupTaskComplete(groupName string, taskType string, updated, skipped, failed int) {
	NotifyGroupTaskCompleteWithDetails(groupName, taskType, updated, skipped, failed, nil)
}

// NotifyGroupTaskCompleteWithDetails 群组任务完成通知（带明细）
func NotifyGroupTaskCompleteWithDetails(groupName string, taskType string, updated, skipped, failed int, details []TaskDetail) {
	config, err := model.GetBarkConfig()
	if err != nil {
		logx.Errorf("获取 Bark 配置失败: %v", err)
		return
	}

	if !config.Enabled {
		return
	}

	// 根据通知模式决定是否发送
	hasFailure := failed > 0
	allSuccess := failed == 0

	switch config.NotifyMode {
	case model.NotifyModeFailureOnly:
		if !hasFailure {
			logx.Debug("无失败项，跳过通知")
			return
		}
	case model.NotifyModeSuccessOnly:
		if !allSuccess {
			logx.Debug("有失败项，跳过通知")
			return
		}
	// NotifyModeAlways 或其他值：始终通知
	}

	var title, body string
	total := updated + skipped + failed

	if taskType == "check" {
		title = "群组检查完成"
		if updated > 0 {
			body = fmt.Sprintf("群组「%s」检查完成，发现 %d 个可更新", groupName, updated)
		} else {
			body = fmt.Sprintf("群组「%s」检查完成，共 %d 个容器均为最新", groupName, total)
		}
	} else {
		title = "群组更新完成"
		if failed > 0 {
			body = fmt.Sprintf("群组「%s」：成功 %d，跳过 %d，失败 %d", groupName, updated, skipped, failed)
		} else if updated > 0 {
			body = fmt.Sprintf("群组「%s」：成功更新 %d 个容器", groupName, updated)
		} else {
			body = fmt.Sprintf("群组「%s」：全部 %d 个容器均为最新", groupName, total)
		}
	}

	// 添加明细信息
	if config.ShowDetail && len(details) > 0 {
		body += "\n\n"
		for _, d := range details {
			switch d.Status {
			case "updated":
				body += fmt.Sprintf("✅ %s\n", d.Name)
			case "failed":
				body += fmt.Sprintf("❌ %s: %s\n", d.Name, d.Message)
			case "has_update":
				body += fmt.Sprintf("🔄 %s (有更新)\n", d.Name)
			// skipped 状态可以不显示，减少通知长度
			}
		}
	}

	if err := SendBarkNotification(title, body); err != nil {
		logx.Errorf("发送群组任务通知失败: %v", err)
	}
}
