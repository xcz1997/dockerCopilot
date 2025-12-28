package utiles

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/xcz1997/dockerCopilot/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
)

// VersionInfo 版本信息结构
type VersionInfo struct {
	LocalVersion   string `json:"localVersion"`
	RemoteVersion  string `json:"remoteVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	IsDocker       bool   `json:"isDocker"`
	CanAutoUpdate  bool   `json:"canAutoUpdate"`
	UpdateMessage  string `json:"updateMessage"`
}

// IsRunningInDocker 检测是否在 Docker 容器中运行
func IsRunningInDocker() bool {
	// 方法1: 检查 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// 方法2: 检查 cgroup（Linux 特有）
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "kubepods") || strings.Contains(content, "containerd") {
			return true
		}
	}

	// 方法3: 检查环境变量（一些 Docker 镜像会设置）
	if os.Getenv("DOCKER_CONTAINER") != "" {
		return true
	}

	return false
}

// GetRemoteVersion 获取远程版本号
func GetRemoteVersion() (remoteVersion string, err error) {
	githubProxy := os.Getenv("githubProxy")
	if githubProxy != "" {
		githubProxy = strings.TrimRight(githubProxy, "/") + "/"
	}

	// 从 latest 分支获取版本号
	versionURL := githubProxy + "https://raw.githubusercontent.com/xcz1997/dockerCopilot/latest/version"
	remoteVersion, err = fetchVersionFromURL(versionURL)
	if err != nil {
		return "0.0.0", err
	}

	localVersion := config.Version
	if strings.Contains(localVersion, "FNOS") {
		logx.Infof("飞牛版本，无需在线更新")
		return localVersion, nil
	}
	if localVersion == remoteVersion {
		logx.Info("版本一致:", localVersion)
		return remoteVersion, nil
	} else {
		logx.Infof("版本不一致! 本地: %s, 远程: %s", localVersion, remoteVersion)
		return remoteVersion, nil
	}
}

// GetVersionInfo 获取完整的版本信息
func GetVersionInfo() (*VersionInfo, error) {
	localVersion := config.Version
	isDocker := IsRunningInDocker()

	info := &VersionInfo{
		LocalVersion:  localVersion,
		IsDocker:      isDocker,
		CanAutoUpdate: !isDocker && !strings.Contains(localVersion, "FNOS"),
	}

	// 飞牛版本不检查更新
	if strings.Contains(localVersion, "FNOS") {
		info.RemoteVersion = localVersion
		info.HasUpdate = false
		info.UpdateMessage = "飞牛版本，请通过飞牛应用商店更新"
		return info, nil
	}

	remoteVersion, err := GetRemoteVersion()
	if err != nil {
		info.RemoteVersion = "获取失败"
		info.UpdateMessage = "获取远程版本失败: " + err.Error()
		return info, err
	}

	info.RemoteVersion = remoteVersion
	// 只有当远程版本大于本地版本时才提示更新
	info.HasUpdate = compareVersions(remoteVersion, localVersion) > 0

	if info.HasUpdate {
		if isDocker {
			info.UpdateMessage = "发现新版本 " + remoteVersion + "，请拉取最新镜像更新: docker pull muuua/docker-copilot:latest"
		} else {
			info.UpdateMessage = "发现新版本 " + remoteVersion + "，可以自动更新"
		}
	} else {
		info.UpdateMessage = "当前已是最新版本"
	}

	return info, nil
}

// compareVersions 比较两个语义化版本号
// 返回: 1 表示 v1 > v2, -1 表示 v1 < v2, 0 表示相等
func compareVersions(v1, v2 string) int {
	// 去掉 v 前缀
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// 比较每个部分
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int
		if i < len(parts1) {
			num1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			num2, _ = strconv.Atoi(parts2[i])
		}

		if num1 > num2 {
			return 1
		} else if num1 < num2 {
			return -1
		}
	}

	return 0
}

func fetchVersionFromURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("关闭Body失败:", err)
		}
	}(resp.Body)

	versionData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(versionData)), nil
}
