package module

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/net/proxy"
)

// GetHTTPClient 获取带代理配置的 HTTP 客户端
func GetHTTPClient(timeout time.Duration) *http.Client {
	transport := GetHTTPTransport()
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// GetHTTPTransport 获取带代理配置的 Transport
func GetHTTPTransport() *http.Transport {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	// 获取代理配置
	proxyConfig, err := model.GetProxyConfig()
	if err != nil {
		logx.Errorf("获取代理配置失败: %v", err)
		return transport
	}

	if !proxyConfig.Enabled || proxyConfig.Host == "" || proxyConfig.Port == 0 {
		return transport
	}

	// 配置代理
	if proxyConfig.Type == "socks5" {
		// SOCKS5 代理
		var auth *proxy.Auth
		if proxyConfig.Username != "" {
			auth = &proxy.Auth{
				User:     proxyConfig.Username,
				Password: proxyConfig.Password,
			}
		}

		dialer, err := proxy.SOCKS5("tcp",
			fmt.Sprintf("%s:%d", proxyConfig.Host, proxyConfig.Port),
			auth,
			proxy.Direct)
		if err != nil {
			logx.Errorf("创建 SOCKS5 代理失败: %v，将回退到直连", err)
			return transport
		}

		// 包装 DialContext
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.Dial(network, addr)
			if err != nil {
				// 代理失败，回退到直连
				logx.Errorf("SOCKS5 代理连接失败: %v，回退到直连", err)
				directDialer := &net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				return directDialer.DialContext(ctx, network, addr)
			}
			return conn, nil
		}
	} else {
		// HTTP/HTTPS 代理
		proxyURL := proxyConfig.GetProxyURL()
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			logx.Errorf("解析代理 URL 失败: %v，将使用直连", err)
			return transport
		}

		// 使用自定义代理函数，支持降级
		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			// 尝试使用代理
			return parsedURL, nil
		}

		// 保存原始 DialContext 用于降级
		originalDialContext := transport.DialContext
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := originalDialContext(ctx, network, addr)
			if err != nil {
				// 代理失败时回退到直连
				logx.Errorf("代理连接失败: %v，回退到直连", err)
				// 清除代理设置
				transport.Proxy = nil
				directDialer := &net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				return directDialer.DialContext(ctx, network, addr)
			}
			return conn, nil
		}
	}

	logx.Infof("已启用 %s 代理: %s:%d", proxyConfig.Type, proxyConfig.Host, proxyConfig.Port)
	return transport
}

// GetHTTPClientWithCustomTransport 获取带代理配置的 HTTP 客户端（可自定义 Transport 选项）
func GetHTTPClientWithCustomTransport(timeout time.Duration, tlsSkipVerify bool) *http.Client {
	transport := GetHTTPTransport()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = tlsSkipVerify

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// TestProxy 测试代理连通性
func TestProxy(proxyType, host string, port int, username, password string) error {
	config := &model.ProxyConfig{
		Enabled:  true,
		Type:     proxyType,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}

	var transport *http.Transport
	if proxyType == "socks5" {
		var auth *proxy.Auth
		if username != "" {
			auth = &proxy.Auth{
				User:     username,
				Password: password,
			}
		}

		dialer, err := proxy.SOCKS5("tcp",
			fmt.Sprintf("%s:%d", host, port),
			auth,
			proxy.Direct)
		if err != nil {
			return fmt.Errorf("创建 SOCKS5 代理失败: %v", err)
		}

		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		proxyURL := config.GetProxyURL()
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			return fmt.Errorf("解析代理 URL 失败: %v", err)
		}

		transport = &http.Transport{
			Proxy:           http.ProxyURL(parsedURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// 测试访问 Google（或其他可靠的外部站点）
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		return fmt.Errorf("代理连接测试失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("代理连接测试失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// TestRegistryMirror 测试 Registry 镜像地址连通性
func TestRegistryMirror(address string) error {
	client := GetHTTPClient(10 * time.Second)

	testURL := fmt.Sprintf("https://%s/v2/", address)
	resp, err := client.Get(testURL)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	// 200 或 401 都表示 Registry 可访问
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}

	return fmt.Errorf("Registry 响应异常，状态码: %d", resp.StatusCode)
}
