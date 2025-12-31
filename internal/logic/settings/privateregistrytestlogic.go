package settings

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PrivateRegistryTestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPrivateRegistryTestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PrivateRegistryTestLogic {
	return &PrivateRegistryTestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PrivateRegistryTestLogic) PrivateRegistryTest(req *types.PrivateRegistryTestReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	if req.Host == "" {
		resp.Code = 400
		resp.Msg = "Registry 地址不能为空"
		resp.Data = map[string]interface{}{"success": false}
		return resp, nil
	}

	// 构建测试 URL
	host := req.Host
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		if req.Insecure {
			host = "http://" + host
		} else {
			host = "https://" + host
		}
	}

	// 测试 /v2/ 端点
	testURL := strings.TrimSuffix(host, "/") + "/v2/"

	// 创建 HTTP 客户端
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: req.Insecure,
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	// 创建请求
	httpReq, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		resp.Code = 500
		resp.Msg = "创建请求失败: " + err.Error()
		resp.Data = map[string]interface{}{"success": false}
		return resp, nil
	}

	// 添加认证头
	if req.Username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(req.Username + ":" + req.Password))
		httpReq.Header.Set("Authorization", "Basic "+auth)
	}

	// 发送请求
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Code = 500
		resp.Msg = "连接失败: " + err.Error()
		resp.Data = map[string]interface{}{"success": false}
		return resp, nil
	}
	defer httpResp.Body.Close()

	// 检查响应
	if httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusUnauthorized {
		// 200: 无需认证或认证成功
		// 401: 需要认证但地址可达
		if httpResp.StatusCode == http.StatusUnauthorized && req.Username != "" {
			resp.Code = 401
			resp.Msg = "认证失败，请检查用户名和密码"
			resp.Data = map[string]interface{}{"success": false}
			return resp, nil
		}
		resp.Code = 200
		resp.Msg = "连接成功"
		resp.Data = map[string]interface{}{"success": true}
		return resp, nil
	}

	resp.Code = httpResp.StatusCode
	resp.Msg = fmt.Sprintf("Registry 响应异常: %s", httpResp.Status)
	resp.Data = map[string]interface{}{"success": false}
	return resp, nil
}
