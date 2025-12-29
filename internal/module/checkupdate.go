package module

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	url2 "net/url"
	"strings"
	"sync"
	"time"

	ref "github.com/distribution/reference"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ImageCheckList 检查更新处理后的镜像列表
type ImageCheckList struct {
	NeedUpdate bool
}

type ImageUpdateData struct {
	Data map[string]ImageCheckList
	mu   sync.RWMutex
}

const ContentDigestHeader = "Docker-Content-Digest"

// 并发检测的最大 goroutine 数量
const maxConcurrentChecks = 10

func NewImageCheck() *ImageUpdateData {
	return &ImageUpdateData{
		Data: map[string]ImageCheckList{},
	}
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

	logx.Infof("开始并发检查 %d 个镜像的更新状态 (并发数: %d)", len(imagesToCheck), maxConcurrentChecks)
	startTime := time.Now()

	// 使用带缓冲的 channel 作为信号量控制并发数
	semaphore := make(chan struct{}, maxConcurrentChecks)
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
	logx.Infof("镜像更新检查完成，耗时: %v", time.Since(startTime))
}

func (i *ImageUpdateData) checkSingleImage(image types.Image) {
	token, err := GetToken(image, "")
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
		logx.Errorf("获取digest失败 [%s:%s]: %s", image.ImageName, image.ImageTag, err.Error())
		return
	}
	if len(image.RepoDigests) == 0 {
		logx.Errorf("未在本地获取到repoDigest: %s:%s", image.ImageName, image.ImageTag)
		return
	}
	needUpdate := false
	for _, localRepoDigests := range image.RepoDigests {
		parts := strings.Split(localRepoDigests, "@")
		if len(parts) < 2 {
			logx.Errorf("无效的本地 digest 格式: %s", localRepoDigests)
			continue
		}
		localDigest := parts[1]
		if remoteDigest != localDigest {
			if remoteDigest == "" || localDigest == "" {
				logx.Errorf("Digest为空 [%s:%s]", image.ImageName, image.ImageTag)
				continue
			}
			logx.Infof("%s:%s 需要更新 (本地: %s, 远程: %s)", image.ImageName, image.ImageTag, localDigest[:12], remoteDigest[:12])
			needUpdate = true
		} else {
			logx.Debugf("%s:%s 已是最新版本", image.ImageName, image.ImageTag)
			needUpdate = false
		}
	}
	// 使用线程安全的方法设置结果
	i.setImageCheck(image.ID, ImageCheckList{NeedUpdate: needUpdate})
	// 同时按镜像名称存储，方便容器通过镜像名查找
	i.setImageCheckByName(image.ImageName, image.ImageTag, ImageCheckList{NeedUpdate: needUpdate})
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
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
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
	client := &http.Client{Transport: tr}

	req, _ := http.NewRequest("HEAD", url, nil)

	if token != "" {
		req.Header.Add("Authorization", token)
	}
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")

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
