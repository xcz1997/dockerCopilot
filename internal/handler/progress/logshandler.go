package progress

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/golang-jwt/jwt/v4"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// LogLine 日志行结构
type LogLine struct {
	Time   string `json:"time"`
	Stream string `json:"stream"` // stdout 或 stderr
	Text   string `json:"text"`
}

func ContainerLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ContainerLogsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 手动验证 JWT token
		token := req.Token
		if token == "" {
			// 尝试从 Authorization header 获取
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

		// 检查容器是否存在
		_, err = svcCtx.DockerClient.ContainerInspect(r.Context(), req.Id)
		if err != nil {
			http.Error(w, fmt.Sprintf("容器不存在: %v", err), http.StatusNotFound)
			return
		}

		// 设置 SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE 不支持", http.StatusInternalServerError)
			return
		}

		// Docker 日志选项
		tail := req.Tail
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
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// 监听客户端断开
		go func() {
			<-r.Context().Done()
			cancel()
		}()

		// 获取日志流
		reader, err := svcCtx.DockerClient.ContainerLogs(ctx, req.Id, options)
		if err != nil {
			sendSSEError(w, flusher, fmt.Sprintf("获取日志失败: %v", err))
			return
		}
		defer reader.Close()

		// 发送连接成功事件
		sendSSEEvent(w, flusher, "connected", map[string]string{
			"containerId": req.Id,
			"message":     "日志连接成功",
		})

		// 解析并发送日志
		parseAndSendLogs(ctx, reader, w, flusher)
	}
}

// parseAndSendLogs 解析 Docker 日志格式并发送 SSE 事件
// Docker 日志格式：8 字节头 + 日志内容
// 头部结构：[stream_type(1)] [0(3)] [size(4 big-endian)]
func parseAndSendLogs(ctx context.Context, reader io.Reader, w http.ResponseWriter, flusher http.Flusher) {
	header := make([]byte, 8)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 读取 8 字节头
		_, err := io.ReadFull(reader, header)
		if err != nil {
			// context canceled 或连接关闭是正常的断开行为，不需要记录错误
			if err == io.EOF || strings.Contains(err.Error(), "closed") || strings.Contains(err.Error(), "canceled") {
				sendSSEEvent(w, flusher, "disconnected", map[string]string{
					"message": "日志流已关闭",
				})
				return
			}
			logx.Errorf("读取日志头失败: %v", err)
			return
		}

		// 解析流类型
		streamType := header[0]
		stream := "stdout"
		if streamType == 2 {
			stream = "stderr"
		}

		// 解析消息长度（大端序）
		size := binary.BigEndian.Uint32(header[4:8])
		if size == 0 {
			continue
		}

		// 读取日志内容
		content := make([]byte, size)
		_, err = io.ReadFull(reader, content)
		if err != nil {
			logx.Errorf("读取日志内容失败: %v", err)
			return
		}

		// 解析时间戳和日志文本
		line := strings.TrimSpace(string(content))
		if line == "" {
			continue
		}

		// Docker 日志格式: "2024-01-01T12:00:00.000000000Z 日志内容"
		timestamp := ""
		text := line
		if len(line) > 30 && line[4] == '-' && line[10] == 'T' {
			spaceIdx := strings.Index(line, " ")
			if spaceIdx > 0 {
				timestamp = line[:spaceIdx]
				text = line[spaceIdx+1:]
			}
		}

		// 格式化时间戳
		if timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
				timestamp = t.Local().Format("2006-01-02 15:04:05")
			}
		}

		logLine := LogLine{
			Time:   timestamp,
			Stream: stream,
			Text:   text,
		}

		sendSSEEvent(w, flusher, "log", logLine)
	}
}

// parseAndSendLogsSimple 简化版日志解析（按行读取）
// 用于 TTY 模式的容器（没有 8 字节头）
func parseAndSendLogsSimple(ctx context.Context, reader io.Reader, w http.ResponseWriter, flusher http.Flusher) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// 解析时间戳
		timestamp := ""
		text := line
		if len(line) > 30 && line[4] == '-' && line[10] == 'T' {
			spaceIdx := strings.Index(line, " ")
			if spaceIdx > 0 {
				timestamp = line[:spaceIdx]
				text = line[spaceIdx+1:]
			}
		}

		// 格式化时间戳
		if timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
				timestamp = t.Local().Format("2006-01-02 15:04:05")
			}
		}

		logLine := LogLine{
			Time:   timestamp,
			Stream: "stdout",
			Text:   text,
		}

		sendSSEEvent(w, flusher, "log", logLine)
	}
}

// sendSSEEvent 发送 SSE 事件
func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		logx.Errorf("JSON 序列化失败: %v", err)
		return
	}

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

// sendSSEError 发送 SSE 错误事件
func sendSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	sendSSEEvent(w, flusher, "error", map[string]string{
		"message": message,
	})
}
