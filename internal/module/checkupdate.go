package module

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	url2 "net/url"
	"strings"
	"sync"
	"time"

	ref "github.com/distribution/reference"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ImageCheckList 检查更新处理后的镜像列表
type ImageCheckList struct {
	NeedUpdate bool
}

type ImageUpdateData struct {
	Data               map[string]ImageCheckList
	mu                 sync.RWMutex
	maxConcurrentCheck int // 最大并发检查数
}

const ContentDigestHeader = "Docker-Content-Digest"

// 默认并发检测的最大 goroutine 数量
const defaultMaxConcurrentChecks = 10

func NewImageCheck() *ImageUpdateData {
	return &ImageUpdateData{
		Data:               map[string]ImageCheckList{},
		maxConcurrentCheck: defaultMaxConcurrentChecks,
	}
}

// SetMaxConcurrent 设置最大并发数
func (i *ImageUpdateData) SetMaxConcurrent(max int) {
	if max <= 0 {
		max = defaultMaxConcurrentChecks
	}
	if max > 20 {
		max = 20
	}
	i.maxConcurrentCheck = max
	logx.Infof("镜像检查最大并发数设置为: %d", max)
}

// GetMaxConcurrent 获取最大并发数
func (i *ImageUpdateData) GetMaxConcurrent() int {
	if i.maxConcurrentCheck <= 0 {
		return defaultMaxConcurrentChecks
	}
	return i.maxConcurrentCheck
}

// setImageCheck 线程安全地设置镜像检查结果
func (i *ImageUpdateData) setImageCheck(imageID string, result ImageCheckList) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Data[imageID] = result
}

// setImageCheckByName 按镜像名称设置检查结果（用于容器查找）
func (i *ImageUpdateData) setImageCheckByName(imageName string, imageTag string, result ImageCheckList) {
	i.mu.Lock()
	defer i.mu.Unlock()
	// 使用 imageName:imageTag 作为额外的 key
	key := imageName + ":" + imageTag
	i.Data[key] = result
}

// GetImageCheck 线程安全地获取镜像检查结果
func (i *ImageUpdateData) GetImageCheck(imageID string) (ImageCheckList, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result, ok := i.Data[imageID]
	return result, ok
}

// GetImageCheckByName 按镜像名称获取检查结果
func (i *ImageUpdateData) GetImageCheckByName(imageName string) (ImageCheckList, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result, ok := i.Data[imageName]
	return result, ok
}

// MarkAsUpdated 标记镜像为已更新（不需要更新）
// 在容器/镜像更新成功后调用，更新缓存状态
func (i *ImageUpdateData) MarkAsUpdated(imageID string, imageName string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	// 使用 ImageID 作为 key
	i.Data[imageID] = ImageCheckList{NeedUpdate: false}
	// 同时使用镜像名称作为 key（容器列表可能通过名称查询）
	if imageName != "" {
		i.Data[imageName] = ImageCheckList{NeedUpdate: false}
	}
}

// RemoveImage 从缓存中移除镜像（用于本地镜像排除）
func (i *ImageUpdateData) RemoveImage(imageID string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.Data, imageID)
}

// IsSelfImage 判断是否为 DockerCopilot 自身镜像
func IsSelfImage(imageName string) bool {
	lowerName := strings.ToLower(imageName)
	return strings.Contains(lowerName, "dockercopilot") ||
		strings.Contains(lowerName, "docker-copilot")
}

func (i *ImageUpdateData) CheckUpdate(imageList []types.Image) {
	// 过滤出需要检查的镜像
	var imagesToCheck []types.Image
	for _, image := range imageList {
		// 跳过自身镜像
		if IsSelfImage(image.ImageName) {
			continue
		}
		// 跳过无效镜像（无 RepoTags 和 RepoDigests 的孤立镜像）
		if image.ImageName == "None" || image.ImageTag == "None" {
			logx.Debugf("跳过无效镜像: %s (ID: %s)", image.ImageName, image.ID)
			continue
		}
		imagesToCheck = append(imagesToCheck, image)
	}

	if len(imagesToCheck) == 0 {
		logx.Info("没有需要检查更新的镜像")
		return
	}

	maxConcurrent := i.GetMaxConcurrent()
	logx.Infof("开始检查 %d 个镜像的更新状态 (并发数: %d)", len(imagesToCheck), maxConcurrent)
	startTime := time.Now()

	if maxConcurrent == 1 {
		// 单线程顺序执行（低性能模式）
		for idx, image := range imagesToCheck {
			logx.Debugf("检查镜像 (%d/%d): %s:%s", idx+1, len(imagesToCheck), image.ImageName, image.ImageTag)
			i.checkSingleImage(image)
		}
	} else {
		// 多线程并发执行
		// 使用带缓冲的 channel 作为信号量控制并发数
		semaphore := make(chan struct{}, maxConcurrent)
		var wg sync.WaitGroup

		for _, image := range imagesToCheck {
			wg.Add(1)
			go func(img types.Image) {
				defer wg.Done()
				// 获取信号量
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				i.checkSingleImage(img)
			}(image)
		}

		wg.Wait()
	}
	logx.Infof("镜像更新检查完成，耗时: %v", time.Since(startTime))
}

func (i *ImageUpdateData) checkSingleImage(image types.Image) {
	// 先检查镜像元数据，是否已标记为 local（跳过检查）
	meta, _ := model.GetImageMetadata(image.ID)
	if meta != nil && meta.SourceType == model.SourceTypeLocal {
		logx.Debugf("跳过本地镜像检查: %s:%s", image.ImageName, image.ImageTag)
		return
	}

	// 如果是私有镜像，使用绑定的 Registry 认证
	var registryAuth string
	if meta != nil && meta.SourceType == model.SourceTypePrivate && meta.RegistryHost != "" {
		registryAuth = findPrivateRegistryAuthByHost(meta.RegistryHost)
		if registryAuth != "" {
			logx.Infof("使用绑定的私有 Registry: %s", meta.RegistryHost)
		}
	}

	token, err := GetToken(image, registryAuth)
	if err != nil {
		logx.Debugf("获取token失败或者无需获取token，继续尝试检查: %s", err.Error())
	}
	digestURL, err := BuildManifestURL(image)
	if err != nil {
		logx.Errorf("获取digestURL失败 [%s:%s]: %s", image.ImageName, image.ImageTag, err.Error())
		return
	}
	remoteDigest, err := GetDigest(digestURL, token)
	if err != nil {
		errMsg := err.Error()
		// 检查是否为 404 错误，仅 404 时降级为 local（但私有镜像不自动降级）
		if is404Error(errMsg) && (meta == nil || meta.SourceType != model.SourceTypePrivate) {
			logx.Infof("镜像 %s:%s 远程不存在 (404)，标记为本地镜像", image.ImageName, image.ImageTag)
			updateImageMetadata(image, model.SourceTypeLocal, errMsg)
		} else {
			// 其他错误（网络超时、认证失败等）仅记录，不降级
			logx.Errorf("获取digest失败 [%s:%s]: %s", image.ImageName, image.ImageTag, errMsg)
			// 更新检查时间和错误信息，但保持原来源类型
			updateImageMetadataCheckError(image, errMsg)
		}
		return
	}
	if len(image.RepoDigests) == 0 {
		logx.Errorf("未在本地获取到repoDigest: %s:%s", image.ImageName, image.ImageTag)
		return
	}

	// 检查是否有任意一个本地 digest 与远程匹配
	// 对于 multi-arch 镜像，本地可能存储的是特定架构的 digest
	// 只要有一个匹配就表示不需要更新
	needUpdate := true
	var localDigestForLog string
	for _, localRepoDigests := range image.RepoDigests {
		parts := strings.Split(localRepoDigests, "@")
		if len(parts) < 2 {
			logx.Debugf("无效的本地 digest 格式: %s", localRepoDigests)
			continue
		}
		localDigest := parts[1]
		if localDigest == "" {
			continue
		}
		localDigestForLog = localDigest
		if remoteDigest == localDigest {
			// 找到匹配，不需要更新
			logx.Debugf("%s:%s 已是最新版本", image.ImageName, image.ImageTag)
			needUpdate = false
			break
		}
	}

	if needUpdate && localDigestForLog != "" && remoteDigest != "" {
		logx.Infof("%s:%s 需要更新 (本地: %s, 远程: %s)", image.ImageName, image.ImageTag, localDigestForLog[:12], remoteDigest[:12])
	}
	// 使用线程安全的方法设置结果
	i.setImageCheck(image.ID, ImageCheckList{NeedUpdate: needUpdate})
	// 同时按镜像名称存储，方便容器通过镜像名查找
	i.setImageCheckByName(image.ImageName, image.ImageTag, ImageCheckList{NeedUpdate: needUpdate})
	// 远程检查成功，更新元数据标记为 remote
	updateImageMetadata(image, model.SourceTypeRemote, "")
}

// is404Error 检查错误信息是否表示 404 Not Found
func is404Error(errMsg string) bool {
	lowerMsg := strings.ToLower(errMsg)
	return strings.Contains(lowerMsg, "404") ||
		strings.Contains(lowerMsg, "not_found") ||
		strings.Contains(lowerMsg, "not found") ||
		strings.Contains(lowerMsg, "name unknown") ||
		strings.Contains(lowerMsg, "manifest unknown")
}

// updateImageMetadata 更新镜像元数据
func updateImageMetadata(image types.Image, sourceType string, checkError string) {
	now := time.Now()
	meta := &model.ImageMetadata{
		ImageID:        image.ID,
		ImageName:      image.ImageName,
		ImageTag:       image.ImageTag,
		SourceType:     sourceType,
		LastCheckAt:    &now,
		LastCheckError: checkError,
	}
	if err := model.UpsertImageMetadata(meta); err != nil {
		logx.Errorf("更新镜像元数据失败 [%s]: %v", image.ImageName, err)
	}
}

// updateImageMetadataCheckError 仅更新检查错误信息，不改变来源类型
func updateImageMetadataCheckError(image types.Image, checkError string) {
	// 先尝试获取现有元数据
	existing, err := model.GetImageMetadata(image.ID)
	if err != nil {
		// 不存在则创建新的，默认 remote
		updateImageMetadata(image, model.SourceTypeRemote, checkError)
		return
	}
	// 保持原来源类型，仅更新检查错误
	now := time.Now()
	existing.LastCheckAt = &now
	existing.LastCheckError = checkError
	if err := model.UpsertImageMetadata(existing); err != nil {
		logx.Errorf("更新镜像元数据失败 [%s]: %v", image.ImageName, err)
	}
}

func BuildManifestURL(image types.Image) (string, error) {
	normalizedRef, err := ref.ParseDockerRef(image.ImageName + ":" + image.ImageTag)
	if err != nil {
		return "", err
	}
	normalizedTaggedRef, isTagged := normalizedRef.(ref.NamedTagged)
	if !isTagged {
		return "", errors.New("镜像无tag" + normalizedRef.String())
	}

	host, ErrGetRegistryAddress := GetRegistryAddress(normalizedTaggedRef.Name())
	img, tag := ref.Path(normalizedTaggedRef), normalizedTaggedRef.Tag()

	if ErrGetRegistryAddress != nil {
		return "", ErrGetRegistryAddress
	}

	url := url2.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/v2/%s/manifests/%s", img, tag),
	}
	return url.String(), nil
}

func GetDigest(url string, token string) (string, error) {
	// 使用统一的 HTTP 客户端（支持代理配置）
	client := GetHTTPClient(30 * time.Second)

	req, _ := http.NewRequest("HEAD", url, nil)

	if token != "" {
		req.Header.Add("Authorization", token)
	}
	// Accept header 顺序
	// 1. 优先请求 manifest list / OCI index（multi-arch 镜像）
	// 2. 再请求单架构 manifest
	// 这样返回的 digest 与本地 RepoDigests 中的 manifest list digest 一致
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("GetDigest关闭body失败" + err.Error())
		}
	}(res.Body)

	if res.StatusCode != 200 {
		wwwAuthHeader := res.Header.Get("www-authenticate")
		if wwwAuthHeader == "" {
			wwwAuthHeader = "not present"
		}
		return "", fmt.Errorf("registry responded to head request with %q, auth: %q", res.Status, wwwAuthHeader)
	}
	return res.Header.Get(ContentDigestHeader), nil
}
