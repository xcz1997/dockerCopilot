package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContainersListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// PortMapping 端口映射信息
type PortMapping struct {
	HostIP        string `json:"hostIP,omitempty"`        // 主机 IP
	HostPort      string `json:"hostPort"`                // 主机端口
	ContainerPort string `json:"containerPort"`           // 容器端口
	Protocol      string `json:"protocol"`                // 协议 tcp/udp
}

// NetworkInfo 网络信息
type NetworkInfo struct {
	Name      string `json:"name"`                // 网络名称
	IPAddress string `json:"ipAddress,omitempty"` // IP 地址
}

type Info struct {
	Id          string `json:"id"`
	Status      string `json:"status"`
	Name        string `json:"name"`
	UsingImage  string `json:"usingImage"`
	CreateImage string `json:"createImage"`
	CreateTime  string `json:"createTime"`
	RunningTime string `json:"runningTime"`
	HaveUpdate  bool   `json:"haveUpdate"`
	IsSelf      bool   `json:"isSelf"`
	// Compose 相关信息
	ComposeProject string `json:"composeProject,omitempty"`
	ComposeService string `json:"composeService,omitempty"`
	// 网络相关信息
	Ports       []PortMapping `json:"ports,omitempty"`       // 端口映射列表
	NetworkMode string        `json:"networkMode,omitempty"` // 网络模式
	Networks    []NetworkInfo `json:"networks,omitempty"`    // 网络列表（含 IP）
}

func NewContainersListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainersListLogic {
	return &ContainersListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContainersListLogic) ContainersList() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否为远程环境
	if l.svcCtx.CurrentEnvironment != nil && l.svcCtx.CurrentEnvironment.EnvType == model.EnvTypeRemote {
		return l.getRemoteContainers()
	}

	// 本地环境：获取所有容器（包括停止的容器）
	list, err := utiles.GetContainerList(l.svcCtx)
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	// 获取所有镜像，建立 ImageID 到镜像名称的映射
	imageMap := make(map[string]string)
	images, err := l.svcCtx.DockerClient.ImageList(l.ctx, image.ListOptions{})
	if err == nil {
		for _, img := range images {
			if len(img.RepoTags) > 0 {
				imageMap[img.ID] = img.RepoTags[0]
			} else if len(img.RepoDigests) > 0 {
				parts := strings.Split(img.RepoDigests[0], "@")
				if len(parts) > 0 {
					imageMap[img.ID] = parts[0] + ":latest"
				}
			}
		}
	}

	resp.Code = 200
	resp.Msg = "success"
	var containerInfoList []Info
	list = utiles.CheckImageUpdate(l.svcCtx, list)
	for _, v := range list {
		var containerInfo Info
		containerInfo.Id = v.ID
		containerInfo.Status = v.State
		if len(v.Names) > 0 {
			ContainerName := v.Names[0][1:]
			containerInfo.Name = ContainerName
		} else {
			containerInfo.Name = "get container name error"
			l.Error("get container name error" + v.ID)
		}
		// 获取镜像名称：优先使用 imageMap 中的完整名称
		if mappedName, ok := imageMap[v.ImageID]; ok {
			containerInfo.UsingImage = mappedName
		} else if v.Image != "" && !strings.HasPrefix(v.Image, "sha256:") {
			containerInfo.UsingImage = v.Image
		} else {
			containerInfo.UsingImage = v.ImageID[:19] + "..."
			l.Error("image dont have name" + v.ID)
		}
		containerInspect, err := utiles.GetContainerInspect(l.svcCtx, v.ID)
		if err != nil {
			containerInfo.CreateImage = ""
			l.Error("get image name error" + v.ID)
		}
		containerInfo.CreateImage = containerInspect.Config.Image
		t := time.Unix(v.Created, 0)
		containerInfo.CreateTime = t.Format("2006-01-02 15:04:05")
		containerInfo.RunningTime = v.Status
		containerInfo.HaveUpdate = v.Update
		// 检测是否为自身容器
		containerInfo.IsSelf = module.IsSelfImage(containerInfo.UsingImage)
		// 提取 Compose 信息
		containerInfo.ComposeProject = v.Labels["com.docker.compose.project"]
		containerInfo.ComposeService = v.Labels["com.docker.compose.service"]

		// 提取端口映射信息
		for _, port := range v.Ports {
			if port.PublicPort > 0 {
				containerInfo.Ports = append(containerInfo.Ports, PortMapping{
					HostIP:        port.IP,
					HostPort:      fmt.Sprintf("%d", port.PublicPort),
					ContainerPort: fmt.Sprintf("%d", port.PrivatePort),
					Protocol:      port.Type,
				})
			}
		}

		// 提取网络模式和网络信息
		if containerInspect.HostConfig != nil {
			containerInfo.NetworkMode = string(containerInspect.HostConfig.NetworkMode)
		}
		if containerInspect.NetworkSettings != nil && containerInspect.NetworkSettings.Networks != nil {
			for networkName, network := range containerInspect.NetworkSettings.Networks {
				containerInfo.Networks = append(containerInfo.Networks, NetworkInfo{
					Name:      networkName,
					IPAddress: network.IPAddress,
				})
			}
		}

		containerInfoList = append(containerInfoList, containerInfo)
	}
	resp.Data = containerInfoList
	return resp, nil
}

// getRemoteContainers 从远程环境获取容器列表
func (l *ContainersListLogic) getRemoteContainers() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取或创建远程客户端
	client := l.svcCtx.RemoteClient
	if client == nil {
		env := l.svcCtx.CurrentEnvironment
		client = module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	}

	// 从远程获取容器列表
	data, err := client.ProxyRequest("GET", "/api/containers", nil)
	if err != nil {
		resp.Code = 500
		resp.Msg = "获取远程容器列表失败: " + err.Error()
		resp.Data = []interface{}{}
		return resp, nil
	}

	// 解析远程返回的数据
	var containers []Info
	if err := json.Unmarshal(data, &containers); err != nil {
		resp.Code = 500
		resp.Msg = "解析远程容器数据失败: " + err.Error()
		resp.Data = []interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = containers
	return resp, nil
}
