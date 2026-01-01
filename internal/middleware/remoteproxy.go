package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// RemoteProxyMiddleware 远程环境代理中间件
// 当当前环境为远程环境时，自动将请求代理转发到远程服务器
type RemoteProxyMiddleware struct {
	svcCtx *svc.ServiceContext
}

// NewRemoteProxyMiddleware 创建远程代理中间件
func NewRemoteProxyMiddleware(svcCtx *svc.ServiceContext) *RemoteProxyMiddleware {
	return &RemoteProxyMiddleware{
		svcCtx: svcCtx,
	}
}

// excludedPaths 不需要代理的路径前缀
var excludedPaths = []string{
	"/api/auth",            // 认证
	"/api/environment",     // 环境管理
	"/api/environments",    // 环境列表
}

// shouldProxy 判断请求是否需要代理
func (m *RemoteProxyMiddleware) shouldProxy(path string) bool {
	// 检查是否在排除列表中
	for _, excluded := range excludedPaths {
		if strings.HasPrefix(path, excluded) {
			return false
		}
	}
	// 只代理 /api/ 开头的请求
	return strings.HasPrefix(path, "/api/")
}

// Handle 中间件处理函数
func (m *RemoteProxyMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 检查是否为远程环境
		if m.svcCtx.CurrentEnvironment == nil ||
			m.svcCtx.CurrentEnvironment.EnvType != model.EnvTypeRemote {
			// 本地环境，正常处理
			next(w, r)
			return
		}

		// 检查路径是否需要代理
		if !m.shouldProxy(r.URL.Path) {
			// 不需要代理的路径，正常处理
			next(w, r)
			return
		}

		// 远程环境，代理请求
		m.proxyRequest(w, r)
	}
}

// proxyRequest 代理请求到远程服务器
func (m *RemoteProxyMiddleware) proxyRequest(w http.ResponseWriter, r *http.Request) {
	env := m.svcCtx.CurrentEnvironment

	// 获取或创建远程客户端
	client := m.svcCtx.RemoteClient
	if client == nil {
		client = module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	}

	// 读取请求体
	var body interface{}
	if r.Body != nil && r.ContentLength > 0 {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			logx.Errorf("读取请求体失败: %v", err)
			writeErrorResponse(w, 500, "读取请求体失败")
			return
		}
		r.Body.Close()

		// 尝试解析 JSON
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &body); err != nil {
				// 非 JSON 格式，使用原始数据
				body = bodyBytes
			}
		}
	}

	// 构建完整路径（包含查询参数）
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path = path + "?" + r.URL.RawQuery
	}

	logx.Infof("代理请求到远程环境: %s %s -> %s", r.Method, path, env.URL)

	// 发送代理请求
	data, err := client.ProxyRequest(r.Method, path, body)
	if err != nil {
		logx.Errorf("代理请求失败: %v", err)
		writeErrorResponse(w, 500, "远程请求失败: "+err.Error())
		return
	}

	// 返回远程响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 构造标准响应格式
	resp := types.Resp{
		Code: 200,
		Msg:  "success",
		Data: json.RawMessage(data),
	}
	respBytes, _ := json.Marshal(resp)
	w.Write(respBytes)
}

// writeErrorResponse 写入错误响应
func writeErrorResponse(w http.ResponseWriter, code int, msg string) {
	resp := types.Resp{
		Code: code,
		Msg:  msg,
		Data: map[string]interface{}{},
	}
	httpx.WriteJson(w, code, resp)
}

// ProxyRequestRaw 代理原始请求（用于需要自定义响应处理的场景）
func (m *RemoteProxyMiddleware) ProxyRequestRaw(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	env := m.svcCtx.CurrentEnvironment

	client := m.svcCtx.RemoteClient
	if client == nil {
		client = module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	}

	// 读取请求体
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body.Close()
		// 重新设置 Body 以便后续使用
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path = path + "?" + r.URL.RawQuery
	}

	var body interface{}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &body)
	}

	return client.ProxyRequest(r.Method, path, body)
}
