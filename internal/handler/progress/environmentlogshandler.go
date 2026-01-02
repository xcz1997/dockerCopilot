package progress

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/golang-jwt/jwt/v4"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// getSelfContainerID 获取自身容器 ID
func getSelfContainerID() string {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	return hostname
}

// EnvironmentLogsHandler 环境日志 SSE 处理器
func EnvironmentLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EnvironmentLogsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 手动验证 JWT token
		token := req.Token
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			http.Error(w, "未授权：缺少 token", http.StatusUnauthorized)
			return
		}

		// 验证 JWT
		_, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte(svcCtx.Config.Auth.AccessSecret), nil
		})
		if err != nil {
			http.Error(w, "未授权：token 无效", http.StatusUnauthorized)
			return
		}

		// 获取环境信息
		env, err := model.GetEnvironmentByID(req.Id)
		if err != nil {
			http.Error(w, fmt.Sprintf("环境不存在: %v", err), http.StatusNotFound)
			return
		}

		// 设置 SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE 不支持", http.StatusInternalServerError)
			return
		}

		if env.EnvType == model.EnvTypeRemote {
			// 远程环境：代理请求
			handleRemoteEnvironmentLogs(r.Context(), w, flusher, env, req.Tail, token)
		} else {
			// 本地环境：获取自身容器日志
			handleLocalEnvironmentLogs(r.Context(), w, flusher, svcCtx, req.Tail)
		}
	}
}

// handleLocalEnvironmentLogs 处理本地环境日志
func handleLocalEnvironmentLogs(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, svcCtx *svc.ServiceContext, tail string) {
	containerID := getSelfContainerID()
	if containerID == "" {
		sendSSEError(w, flusher, "无法获取自身容器ID")
		return
	}

	// 检查容器是否存在
	_, err := svcCtx.DockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		sendSSEError(w, flusher, fmt.Sprintf("容器不存在: %v", err))
		return
	}

	if tail == "" {
		tail = "100"
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       tail,
		Timestamps: true,
	}

	// 创建可取消的 context
	logCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 监听客户端断开
	go func() {
		<-ctx.Done()
		cancel()
	}()

	// 获取日志流
	reader, err := svcCtx.DockerClient.ContainerLogs(logCtx, containerID, options)
	if err != nil {
		sendSSEError(w, flusher, fmt.Sprintf("获取日志失败: %v", err))
		return
	}
	defer reader.Close()

	// 发送连接成功事件
	sendSSEEvent(w, flusher, "connected", map[string]string{
		"containerId": containerID,
		"message":     "日志连接成功",
	})

	// 解析并发送日志
	parseAndSendLogs(logCtx, reader, w, flusher)
}

// handleRemoteEnvironmentLogs 处理远程环境日志（代理请求）
func handleRemoteEnvironmentLogs(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, env *model.Environment, tail, token string) {
	// 构建远程日志 URL
	// 远程环境需要先获取自身容器ID，然后请求容器日志
	// 为简化实现，我们直接请求远程的 /api/environment/self/logs 端点

	if tail == "" {
		tail = "100"
	}

	// 构建远程 URL - 使用远程环境的 self logs 端点
	remoteURL := fmt.Sprintf("%s/api/environment/self/logs?token=%s&tail=%s",
		strings.TrimRight(env.URL, "/"),
		env.JWTToken, // 使用远程环境的 JWT token
		tail,
	)

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", remoteURL, nil)
	if err != nil {
		sendSSEError(w, flusher, fmt.Sprintf("创建请求失败: %v", err))
		return
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		sendSSEError(w, flusher, fmt.Sprintf("连接远程环境失败: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		sendSSEError(w, flusher, fmt.Sprintf("远程环境返回错误: %s", string(body)))
		return
	}

	// 发送连接成功事件
	sendSSEEvent(w, flusher, "connected", map[string]string{
		"envName": env.Name,
		"message": "远程日志连接成功",
	})

	// 代理远程 SSE 响应
	proxySSEResponse(ctx, resp.Body, w, flusher)
}

// proxySSEResponse 代理 SSE 响应
func proxySSEResponse(ctx context.Context, reader io.Reader, w http.ResponseWriter, flusher http.Flusher) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				sendSSEEvent(w, flusher, "disconnected", map[string]string{
					"message": "远程日志流已关闭",
				})
			}
			return
		}

		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
		}
	}
}

// SelfLogsHandler 获取自身容器日志（供远程环境代理调用）
func SelfLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 手动验证 JWT token
		token := r.URL.Query().Get("token")
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			http.Error(w, "未授权：缺少 token", http.StatusUnauthorized)
			return
		}

		// 验证 JWT
		_, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte(svcCtx.Config.Auth.AccessSecret), nil
		})
		if err != nil {
			http.Error(w, "未授权：token 无效", http.StatusUnauthorized)
			return
		}

		// 设置 SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE 不支持", http.StatusInternalServerError)
			return
		}

		tail := r.URL.Query().Get("tail")
		handleLocalEnvironmentLogs(r.Context(), w, flusher, svcCtx, tail)
	}
}

// EnvironmentSelfContainerHandler 获取自身容器信息
func EnvironmentSelfContainerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		containerID := getSelfContainerID()
		if containerID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 500,
				"msg":  "无法获取自身容器ID",
				"data": map[string]interface{}{},
			})
			return
		}

		// 检查容器是否存在
		inspect, err := svcCtx.DockerClient.ContainerInspect(r.Context(), containerID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 404,
				"msg":  "未在 Docker 容器中运行",
				"data": map[string]interface{}{
					"containerId":   "",
					"containerName": "",
					"isDocker":      false,
				},
			})
			return
		}

		containerName := strings.TrimPrefix(inspect.Name, "/")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"msg":  "success",
			"data": map[string]interface{}{
				"containerId":   containerID,
				"containerName": containerName,
				"isDocker":      true,
			},
		})
	}
}
