package settings

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BarkTestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBarkTestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BarkTestLogic {
	return &BarkTestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BarkTestLogic) BarkTest(req *types.BarkTestReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	if req.Server == "" || req.Key == "" {
		resp.Code = 400
		resp.Msg = "服务器地址和密钥不能为空"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 发送测试推送
	err = sendBarkNotification(req.Server, req.Key, "DockerCopilot 测试", "Bark 推送配置成功！")
	if err != nil {
		resp.Code = 500
		resp.Msg = fmt.Sprintf("推送失败: %s", err.Error())
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "测试推送已发送"
	resp.Data = map[string]interface{}{}
	return resp, nil
}

// Docker 图标 URL
const dockerIconURL = "https://www.docker.com/wp-content/uploads/2022/03/Moby-logo.png"

// sendBarkNotification 发送 Bark 通知
func sendBarkNotification(server, key, title, body string) error {
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
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("服务器返回错误: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
