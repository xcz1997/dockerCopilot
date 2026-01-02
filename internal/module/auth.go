package module

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	ref "github.com/distribution/reference"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

const ChallengeHeader = "WWW-Authenticate"
const (
	DefaultRegistryDomain = "docker.io"
	DefaultRegistryHost   = "index.docker.io"
)

var DefaultAcceleratorHostList = []string{"docker.1ms.run", "docker.m.daocloud.io",
	"docker.1panel.top", "docker.1panel.live", "proxy.1panel.live", "dockerproxy.1panel.live", "docker.1panel.dev",
	"docker.anye.in", "hub.rat.dev", "docker.amingg.com"}

func GetToken(image types.Image, registryAuth string) (string, error) {
	logx.Debugf("image name %s", image.ImageName)
	normalizedRef, err := ref.ParseNormalizedNamed(image.ImageName)
	if err != nil {
		return "", err
	}

	// 如果没有提供认证信息，尝试从私有 Registry 配置查找
	if registryAuth == "" {
		registryAuth = findPrivateRegistryAuth(image.ImageName)
	}

	URL := GetChallengeURL(normalizedRef)

	var req *http.Request
	if req, err = GetChallengeRequest(URL); err != nil {
		return "", err
	}

	// 使用统一的 HTTP 客户端（支持代理）
	client := GetHTTPClient(30 * time.Second)
	var res *http.Response
	if res, err = client.Do(req); err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("GetToken关闭Body失败" + err.Error())
		}
	}(res.Body)
	v := res.Header.Get(ChallengeHeader)

	challenge := strings.ToLower(v)
	if strings.HasPrefix(challenge, "basic") {
		if registryAuth == "" {
			return "", fmt.Errorf("no credentials available")
		}

		return fmt.Sprintf("Basic %s", registryAuth), nil
	}
	if strings.HasPrefix(challenge, "bearer") {
		return GetBearerHeader(challenge, normalizedRef, registryAuth)
	}

	return "", errors.New("unsupported challenge type from registry")
}

func GetChallengeRequest(URL url.URL) (*http.Request, error) {
	req, err := http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Watchtower (Docker)")
	return req, nil
}

func GetBearerHeader(challenge string, imageRef ref.Named, registryAuth string) (string, error) {
	// 使用统一的 HTTP 客户端（支持代理）
	client := GetHTTPClient(30 * time.Second)
	authURL, err := GetAuthURL(challenge, imageRef)

	if err != nil {
		return "", err
	}

	var r *http.Request
	if r, err = http.NewRequest("GET", authURL.String(), nil); err != nil {
		return "", err
	}

	if registryAuth != "" {
		logx.Info("私有镜像，无法获取是否有更新")
		r.Header.Add("Authorization", fmt.Sprintf("Basic %s", registryAuth))
	} else {
		logx.Info("No credentials found.")
	}

	var authResponse *http.Response
	if authResponse, err = client.Do(r); err != nil {
		return "", err
	}
	defer authResponse.Body.Close()

	body, _ := io.ReadAll(authResponse.Body)
	tokenResponse := &types.TokenResponse{}

	err = json.Unmarshal(body, tokenResponse)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Bearer %s", tokenResponse.Token), nil
}

func GetAuthURL(challenge string, imageRef ref.Named) (*url.URL, error) {
	loweredChallenge := strings.ToLower(challenge)
	raw := strings.TrimPrefix(loweredChallenge, "bearer")

	pairs := strings.Split(raw, ",")
	values := make(map[string]string, len(pairs))

	for _, pair := range pairs {
		trimmed := strings.Trim(pair, " ")
		if key, val, ok := strings.Cut(trimmed, "="); ok {
			values[key] = strings.Trim(val, `"`)
		}
	}
	if values["realm"] == "" || values["service"] == "" {

		return nil, fmt.Errorf("challenge header did not include all values needed to construct an auth url")
	}

	authURL, _ := url.Parse(values["realm"])
	q := authURL.Query()
	q.Add("service", values["service"])

	scopeImage := ref.Path(imageRef)

	scope := fmt.Sprintf("repository:%s:pull", scopeImage)
	q.Add("scope", scope)

	authURL.RawQuery = q.Encode()
	return authURL, nil
}

func GetChallengeURL(imageRef ref.Named) url.URL {
	host, _ := GetRegistryAddress(imageRef.Name())

	URL := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/v2/",
	}
	return URL
}

func GetRegistryAddress(imageRef string) (string, error) {
	normalizedRef, err := ref.ParseNormalizedNamed(imageRef)
	if err != nil {
		return "", err
	}

	address := ref.Domain(normalizedRef)

	if address == DefaultRegistryDomain {
		// 优先级：用户配置 > 官方 Docker Hub > 内置加速器列表

		// 1. 首先尝试用户配置的镜像地址
		mirrors, err := model.GetRegistryMirrorsConfig()
		if err == nil && mirrors.Enabled && len(mirrors.Mirrors) > 0 {
			for _, mirror := range mirrors.Mirrors {
				if checkHost(mirror) {
					logx.Infof("使用自定义镜像地址: %s", mirror)
					return mirror, nil
				}
			}
		}

		// 2. 然后尝试官方 Docker Hub
		if checkHost(DefaultRegistryHost) {
			address = DefaultRegistryHost
		} else {
			// 3. 最后回退到内置加速器列表
			for _, host := range DefaultAcceleratorHostList {
				if checkHost(host) {
					address = host
					break
				}
			}
		}
		if address == DefaultRegistryDomain {
			address = DefaultRegistryHost
		}
	}
	return address, nil
}

func checkHost(host string) bool {
	URL := "https://" + host + "/v2/"
	// 使用统一的 HTTP 客户端（支持代理）
	client := GetHTTPClient(5 * time.Second)
	// 发送 GET 请求
	resp, err := client.Get(URL)
	if err != nil {
		logx.Errorf("Failed to connect to %s: %s", URL, err)
		return false
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Errorf("关闭body失败" + err.Error())
		}
	}(resp.Body)

	// 检查 HTTP 响应状态码
	if resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusUnauthorized {
		return true
	}

	logx.Errorf("Failed to connect to %s: %s", URL, resp.Status)
	return false
}

// findPrivateRegistryAuth 根据镜像名查找私有 Registry 认证信息
func findPrivateRegistryAuth(imageName string) string {
	// 从镜像名中提取 Registry 地址
	host := extractRegistryHost(imageName)
	if host == "" {
		return ""
	}
	return findPrivateRegistryAuthByHost(host)
}

// findPrivateRegistryAuthByHost 根据 Registry 主机名查找认证信息
func findPrivateRegistryAuthByHost(host string) string {
	config, err := model.GetPrivateRegistriesConfig()
	if err != nil || !config.Enabled || len(config.Registries) == 0 {
		return ""
	}

	// 查找匹配的私有 Registry
	for _, reg := range config.Registries {
		if reg.Host == host || strings.HasSuffix(host, "."+reg.Host) {
			if reg.Username == "" {
				return ""
			}
			// 解密密码
			password := reg.Password
			encryptKey := os.Getenv("secretKey")
			if encryptKey != "" && password != "" {
				decrypted, err := decryptPassword(password, encryptKey)
				if err == nil {
					password = decrypted
				}
			}
			// 构建 Basic Auth
			auth := base64.StdEncoding.EncodeToString([]byte(reg.Username + ":" + password))
			logx.Infof("使用私有 Registry 认证: %s@%s", reg.Username, reg.Host)
			return auth
		}
	}
	return ""
}

// extractRegistryHost 从镜像名中提取 Registry 地址
func extractRegistryHost(imageName string) string {
	normalizedRef, err := ref.ParseNormalizedNamed(imageName)
	if err != nil {
		return ""
	}
	domain := ref.Domain(normalizedRef)
	// docker.io 是默认的 Docker Hub，不属于私有 Registry
	if domain == DefaultRegistryDomain {
		return ""
	}
	return domain
}

// decryptPassword 使用 AES-GCM 解密密码
func decryptPassword(cipherText, key string) (string, error) {
	if cipherText == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	// 派生 32 字节密钥
	hash := sha256.Sum256([]byte(key))
	derivedKey := hash[:]

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// 分离 nonce 和密文
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// 解密
	plainText, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
