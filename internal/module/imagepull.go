package module

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	ref "github.com/distribution/reference"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// PullImageOptions 镜像拉取选项
type PullImageOptions struct {
	// ImageName 原始镜像名称（如 nginx:latest）
	ImageName string
	// UseAccelerator 是否使用镜像加速器（仅对 Docker Hub 镜像有效）
	UseAccelerator bool
	// RegistryAuth 私有 Registry 认证信息（如果为空，会自动从配置查找）
	RegistryAuth string
}

// PullResult 镜像拉取结果
type PullResult struct {
	// Success 是否成功
	Success bool
	// ActualImageName 实际使用的镜像名称（可能被加速器重写）
	ActualImageName string
	// Message 消息
	Message string
}

// PullImage 统一的镜像拉取方法
// 支持镜像加速器和私有 Registry 认证
func PullImage(ctx context.Context, dockerClient *client.Client, opts PullImageOptions) (io.ReadCloser, *PullResult, error) {
	result := &PullResult{
		Success:         false,
		ActualImageName: opts.ImageName,
	}

	// 解析镜像名称
	normalizedRef, err := ref.ParseNormalizedNamed(opts.ImageName)
	if err != nil {
		result.Message = fmt.Sprintf("解析镜像名称失败: %v", err)
		return nil, result, err
	}

	domain := ref.Domain(normalizedRef)
	imagePath := ref.Path(normalizedRef)

	// 获取标签
	tag := "latest"
	if tagged, ok := normalizedRef.(ref.NamedTagged); ok {
		tag = tagged.Tag()
	}

	// 确定实际使用的镜像名称
	actualImageName := opts.ImageName

	// 如果是 Docker Hub 镜像且启用加速器
	if opts.UseAccelerator && (domain == DefaultRegistryDomain || domain == DefaultRegistryHost) {
		acceleratorHost := getWorkingAcceleratorHost()
		if acceleratorHost != "" && acceleratorHost != DefaultRegistryHost {
			// 使用加速器重写镜像名
			actualImageName = fmt.Sprintf("%s/%s:%s", acceleratorHost, imagePath, tag)
			logx.Infof("使用镜像加速器: %s -> %s", opts.ImageName, actualImageName)
		}
	}

	result.ActualImageName = actualImageName

	// 确定认证信息
	registryAuth := opts.RegistryAuth
	if registryAuth == "" {
		// 尝试从私有 Registry 配置中查找认证
		registryAuth = findPrivateRegistryAuth(opts.ImageName)
	}

	// 构建拉取选项
	pullOpts := image.PullOptions{}
	if registryAuth != "" {
		// 构建认证配置
		authConfig := registry.AuthConfig{
			// registryAuth 是 base64 编码的 username:password
			// 需要解码后设置
		}

		// 解码 registryAuth
		decoded, err := base64.StdEncoding.DecodeString(registryAuth)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				authConfig.Username = parts[0]
				authConfig.Password = parts[1]
			}
		}

		// 编码为 Docker 需要的格式
		encodedAuth, err := encodeAuthToBase64(authConfig)
		if err == nil {
			pullOpts.RegistryAuth = encodedAuth
			logx.Infof("使用 Registry 认证: %s@%s", authConfig.Username, domain)
		}
	}

	// 执行拉取
	reader, err := dockerClient.ImagePull(ctx, actualImageName, pullOpts)
	if err != nil {
		// 如果使用加速器失败，尝试直接拉取
		if opts.UseAccelerator && actualImageName != opts.ImageName {
			logx.Infof("使用加速器拉取失败，尝试直接拉取: %s", opts.ImageName)
			result.ActualImageName = opts.ImageName
			reader, err = dockerClient.ImagePull(ctx, opts.ImageName, pullOpts)
			if err != nil {
				result.Message = fmt.Sprintf("拉取镜像失败: %v", err)
				return nil, result, err
			}
		} else {
			result.Message = fmt.Sprintf("拉取镜像失败: %v", err)
			return nil, result, err
		}
	}

	result.Success = true
	result.Message = "镜像拉取成功"
	return reader, result, nil
}

// PullImageWithTagging 拉取镜像并重新打标签
// 当使用加速器时，拉取后会将镜像重新打上原始标签
func PullImageWithTagging(ctx context.Context, dockerClient *client.Client, opts PullImageOptions) (io.ReadCloser, *PullResult, error) {
	reader, result, err := PullImage(ctx, dockerClient, opts)
	if err != nil {
		return nil, result, err
	}

	// 如果实际使用的镜像名与原始名不同（使用了加速器），需要重新打标签
	if result.ActualImageName != opts.ImageName {
		// 注意：这里只是标记，实际打标签需要在读取完 reader 后执行
		result.Message = fmt.Sprintf("镜像拉取成功，需要重新打标签: %s -> %s", result.ActualImageName, opts.ImageName)
	}

	return reader, result, nil
}

// TagImage 给镜像打标签
// 用于将加速器拉取的镜像重新打上原始标签
func TagImage(ctx context.Context, dockerClient *client.Client, source, target string) error {
	return dockerClient.ImageTag(ctx, source, target)
}

// getWorkingAcceleratorHost 获取可用的加速器地址
func getWorkingAcceleratorHost() string {
	// 1. 首先尝试用户配置的镜像地址
	mirrors, err := model.GetRegistryMirrorsConfig()
	if err == nil && mirrors.Enabled && len(mirrors.Mirrors) > 0 {
		for _, mirror := range mirrors.Mirrors {
			if checkHost(mirror) {
				return mirror
			}
		}
	}

	// 2. 然后尝试官方 Docker Hub
	if checkHost(DefaultRegistryHost) {
		return DefaultRegistryHost
	}

	// 3. 最后回退到内置加速器列表
	for _, host := range DefaultAcceleratorHostList {
		if checkHost(host) {
			return host
		}
	}

	return DefaultRegistryHost
}

// encodeAuthToBase64 将认证配置编码为 base64
func encodeAuthToBase64(authConfig registry.AuthConfig) (string, error) {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

// GetPullAuthForImage 获取镜像的拉取认证信息
// 根据镜像名称查找对应的私有 Registry 认证
func GetPullAuthForImage(imageName string) string {
	return findPrivateRegistryAuth(imageName)
}

// GetPullAuthForImageByMeta 根据镜像元数据获取拉取认证信息
// 如果镜像绑定了私有 Registry，使用绑定的 Registry 认证
func GetPullAuthForImageByMeta(imageID string, imageName string) string {
	meta, err := model.GetImageMetadata(imageID)
	if err == nil && meta != nil && meta.SourceType == model.SourceTypePrivate && meta.RegistryHost != "" {
		auth := findPrivateRegistryAuthByHost(meta.RegistryHost)
		if auth != "" {
			logx.Infof("使用绑定的私有 Registry 认证: %s", meta.RegistryHost)
			return auth
		}
	}
	// 回退到自动查找
	return findPrivateRegistryAuth(imageName)
}
