package module

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// RemoteClient 远程 DockerCopilot 客户端
type RemoteClient struct {
	BaseURL   string
	SecretKey string
	JWTToken  string
	Client    *http.Client
}

// RemoteResponse 远程 API 响应结构
type RemoteResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Jwt string `json:"jwt"`
}

// NewRemoteClient 创建远程客户端
func NewRemoteClient(baseURL, secretKey string) *RemoteClient {
	// 确保 URL 没有尾部斜杠
	baseURL = strings.TrimRight(baseURL, "/")

	return &RemoteClient{
		BaseURL:   baseURL,
		SecretKey: secretKey,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewRemoteClientWithToken 使用已有 token 创建远程客户端
func NewRemoteClientWithToken(baseURL, secretKey, token string) *RemoteClient {
	client := NewRemoteClient(baseURL, secretKey)
	client.JWTToken = token
	return client
}

// Authenticate 认证并获取 JWT token
func (c *RemoteClient) Authenticate() (string, error) {
	// 构建认证请求
	authURL := c.BaseURL + "/api/auth"

	// 使用 form 格式
	data := fmt.Sprintf("secretKey=%s", c.SecretKey)
	req, err := http.NewRequest("POST", authURL, strings.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var result RemoteResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 200 {
		return "", fmt.Errorf("认证失败: %s", result.Msg)
	}

	// 解析 JWT token
	var authResp AuthResponse
	if err := json.Unmarshal(result.Data, &authResp); err != nil {
		return "", fmt.Errorf("解析 token 失败: %w", err)
	}

	c.JWTToken = authResp.Jwt
	return authResp.Jwt, nil
}

// TestConnection 测试连接
func (c *RemoteClient) TestConnection() error {
	_, err := c.Authenticate()
	return err
}

// doRequest 执行带认证的请求
func (c *RemoteClient) doRequest(method, path string, body interface{}) (*RemoteResponse, error) {
	// 确保有 token
	if c.JWTToken == "" {
		if _, err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.JWTToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result RemoteResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 如果是 401 错误，尝试重新认证
	if result.Code == 401 {
		if _, err := c.Authenticate(); err != nil {
			return nil, fmt.Errorf("重新认证失败: %w", err)
		}
		// 重新发送请求
		return c.doRequest(method, path, body)
	}

	return &result, nil
}

// GetContainers 获取远程容器列表
func (c *RemoteClient) GetContainers() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/containers", nil)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("获取容器列表失败: %s", resp.Msg)
	}

	var containers []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &containers); err != nil {
		return nil, fmt.Errorf("解析容器列表失败: %w", err)
	}

	return containers, nil
}

// GetImages 获取远程镜像列表
func (c *RemoteClient) GetImages() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/api/images", nil)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("获取镜像列表失败: %s", resp.Msg)
	}

	var images []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &images); err != nil {
		return nil, fmt.Errorf("解析镜像列表失败: %w", err)
	}

	return images, nil
}

// GetStats 获取环境统计信息
func (c *RemoteClient) GetStats() (*model.EnvironmentStats, error) {
	containers, err := c.GetContainers()
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}

	images, err := c.GetImages()
	if err != nil {
		return nil, fmt.Errorf("获取镜像列表失败: %w", err)
	}

	// 计算统计信息
	stats := &model.EnvironmentStats{
		ContainerCount: len(containers),
		ImageCount:     len(images),
	}

	// 计算运行中和停止的容器数量
	for _, container := range containers {
		if status, ok := container["status"].(string); ok {
			if strings.HasPrefix(strings.ToLower(status), "up") || status == "running" {
				stats.RunningCount++
			} else {
				stats.StoppedCount++
			}
		}
	}

	// 获取系统信息（Volume、CPU、内存）
	sysInfo, err := c.GetSystemInfo()
	if err == nil && sysInfo != nil {
		stats.VolumeCount = sysInfo.VolumeCount
		stats.CPUCores = sysInfo.CPUCores
		stats.MemoryTotal = sysInfo.MemoryTotal
	}

	return stats, nil
}

// SystemInfo 系统信息结构
type SystemInfo struct {
	VolumeCount int   `json:"volumeCount"`
	CPUCores    int   `json:"cpuCores"`
	MemoryTotal int64 `json:"memoryTotal"`
}

// GetSystemInfo 获取远程系统信息
func (c *RemoteClient) GetSystemInfo() (*SystemInfo, error) {
	resp, err := c.doRequest("GET", "/api/system/info", nil)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("获取系统信息失败: %s", resp.Msg)
	}

	var info SystemInfo
	if err := json.Unmarshal(resp.Data, &info); err != nil {
		return nil, fmt.Errorf("解析系统信息失败: %w", err)
	}

	return &info, nil
}

// ProxyRequest 代理 HTTP 请求到远程环境
func (c *RemoteClient) ProxyRequest(method, path string, body interface{}) (json.RawMessage, error) {
	resp, err := c.doRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("远程请求失败: %s", resp.Msg)
	}

	return resp.Data, nil
}

// Restart 重启远程服务
func (c *RemoteClient) Restart() error {
	resp, err := c.doRequest("POST", "/api/system/restart", nil)
	if err != nil {
		return err
	}

	if resp.Code != 200 {
		return fmt.Errorf("重启远程服务失败: %s", resp.Msg)
	}

	return nil
}

// RefreshEnvironmentStats 刷新并更新环境统计信息
func RefreshEnvironmentStats(env *model.Environment) error {
	if env.EnvType == model.EnvTypeLocal {
		// 本地环境的统计由 ServiceContext 处理
		return nil
	}

	// 远程环境
	client := NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	stats, err := client.GetStats()
	if err != nil {
		logx.Errorf("刷新远程环境 %s 统计信息失败: %v", env.Name, err)
		// 更新状态为错误
		model.UpdateEnvironmentStatus(env.ID, model.EnvStatusError, err.Error())
		return err
	}

	// 更新统计信息
	if err := model.UpdateEnvironmentStats(env.ID, stats); err != nil {
		logx.Errorf("更新环境 %s 统计信息失败: %v", env.Name, err)
		return err
	}

	// 如果获取到了新 token，更新它
	if client.JWTToken != env.JWTToken && client.JWTToken != "" {
		tokenExpires := time.Now().Add(24 * time.Hour) // 假设 24 小时过期
		model.UpdateEnvironmentToken(env.ID, client.JWTToken, tokenExpires)
	}

	return nil
}
