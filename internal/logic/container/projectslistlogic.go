package container

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProjectsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProjectsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProjectsListLogic {
	return &ProjectsListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type ProjectInfo struct {
	Name        string        `json:"name"`
	Containers  int           `json:"containers"`
	Running     int           `json:"running"`
	Stopped     int           `json:"stopped"`
	HaveUpdate  bool          `json:"haveUpdate"`
	UpdateCount int           `json:"updateCount"`
	Services    []ServiceInfo `json:"services"`
}

type ServiceInfo struct {
	Name          string        `json:"name"`
	ContainerId   string        `json:"containerId"`
	ContainerName string        `json:"containerName"`
	Status        string        `json:"status"`
	Image         string        `json:"image"`
	HaveUpdate    bool          `json:"haveUpdate"`
	Ports         []PortMapping `json:"ports,omitempty"`
	NetworkMode   string        `json:"networkMode,omitempty"`
	Networks      []NetworkInfo `json:"networks,omitempty"`
}

func (l *ProjectsListLogic) ProjectsList() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取所有容器
	containers, err := l.svcCtx.DockerClient.ContainerList(l.ctx, container.ListOptions{All: true})
	if err != nil {
		resp.Code = 500
		resp.Msg = "获取容器列表失败: " + err.Error()
		resp.Data = []interface{}{}
		return resp, nil
	}

	// 获取所有镜像，建立 ImageID 到镜像名称的映射
	imageMap := make(map[string]string)
	images, err := l.svcCtx.DockerClient.ImageList(l.ctx, image.ListOptions{})
	if err == nil {
		for _, img := range images {
			// 使用 RepoTags 中的第一个作为镜像名称
			if len(img.RepoTags) > 0 {
				imageMap[img.ID] = img.RepoTags[0]
			} else if len(img.RepoDigests) > 0 {
				// 如果没有 tag，使用 RepoDigests 中的镜像名称部分
				parts := strings.Split(img.RepoDigests[0], "@")
				if len(parts) > 0 {
					imageMap[img.ID] = parts[0] + ":latest"
				}
			}
		}
	}

	// 按 Compose 项目分组
	projects := make(map[string]*ProjectInfo)

	for _, c := range containers {
		// 获取 Compose 项目名称
		projectName := c.Labels["com.docker.compose.project"]
		if projectName == "" {
			continue // 跳过非 Compose 容器
		}

		serviceName := c.Labels["com.docker.compose.service"]
		if serviceName == "" {
			serviceName = "unknown"
		}

		// 初始化项目
		if projects[projectName] == nil {
			projects[projectName] = &ProjectInfo{
				Name:     projectName,
				Services: []ServiceInfo{},
			}
		}

		project := projects[projectName]
		project.Containers++

		// 统计运行状态
		if c.State == "running" {
			project.Running++
		} else {
			project.Stopped++
		}

		// 获取容器名称（去掉前缀斜杠）
		containerName := ""
		if len(c.Names) > 0 {
			containerName = c.Names[0]
			if len(containerName) > 0 && containerName[0] == '/' {
				containerName = containerName[1:]
			}
		}

		// 检查是否有更新（通过 ImageID 查找）
		haveUpdate := false
		if l.svcCtx.HubImageInfo != nil && l.svcCtx.HubImageInfo.Data != nil {
			if info, ok := l.svcCtx.HubImageInfo.Data[c.ImageID]; ok {
				haveUpdate = info.NeedUpdate
			}
		}

		if haveUpdate {
			project.HaveUpdate = true
			project.UpdateCount++
		}

		// 获取镜像名称：优先使用 imageMap 中的完整名称
		imageName := c.Image
		if mappedName, ok := imageMap[c.ImageID]; ok {
			imageName = mappedName
		} else if strings.HasPrefix(c.Image, "sha256:") {
			// 如果仍然是 sha256 格式，尝试截取显示
			imageName = c.Image[:19] + "..."
		}

		// 提取端口映射信息
		var ports []PortMapping
		for _, port := range c.Ports {
			if port.PublicPort > 0 {
				ports = append(ports, PortMapping{
					HostIP:        port.IP,
					HostPort:      fmt.Sprintf("%d", port.PublicPort),
					ContainerPort: fmt.Sprintf("%d", port.PrivatePort),
					Protocol:      port.Type,
				})
			}
		}

		// 获取容器详情以提取网络信息
		var networkMode string
		var networks []NetworkInfo
		containerInspect, err := l.svcCtx.DockerClient.ContainerInspect(l.ctx, c.ID)
		if err == nil {
			if containerInspect.HostConfig != nil {
				networkMode = string(containerInspect.HostConfig.NetworkMode)
			}
			if containerInspect.NetworkSettings != nil && containerInspect.NetworkSettings.Networks != nil {
				for netName, net := range containerInspect.NetworkSettings.Networks {
					networks = append(networks, NetworkInfo{
						Name:      netName,
						IPAddress: net.IPAddress,
					})
				}
			}
		}

		// 添加服务信息
		project.Services = append(project.Services, ServiceInfo{
			Name:          serviceName,
			ContainerId:   c.ID[:12],
			ContainerName: containerName,
			Status:        c.State,
			Image:         imageName,
			HaveUpdate:    haveUpdate,
			Ports:         ports,
			NetworkMode:   networkMode,
			Networks:      networks,
		})
	}

	// 转换为列表并排序
	var projectList []ProjectInfo
	for _, p := range projects {
		// 按服务名排序
		sort.Slice(p.Services, func(i, j int) bool {
			return p.Services[i].Name < p.Services[j].Name
		})
		projectList = append(projectList, *p)
	}

	// 按项目名排序
	sort.Slice(projectList, func(i, j int) bool {
		return projectList[i].Name < projectList[j].Name
	})

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = projectList
	return resp, nil
}
