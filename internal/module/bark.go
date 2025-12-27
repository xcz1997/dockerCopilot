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
