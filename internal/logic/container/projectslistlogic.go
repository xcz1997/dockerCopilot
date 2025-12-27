package container

import (
	"context"
	"sort"

	"github.com/docker/docker/api/types/container"
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
	Name       string        `json:"name"`
	Containers int           `json:"containers"`
	Running    int           `json:"running"`
	Stopped    int           `json:"stopped"`
	Services   []ServiceInfo `json:"services"`
}

type ServiceInfo struct {
	Name        string `json:"name"`
	ContainerId string `json:"containerId"`
	Status      string `json:"status"`
	Image       string `json:"image"`
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

		// 添加服务信息
		project.Services = append(project.Services, ServiceInfo{
			Name:        serviceName,
			ContainerId: c.ID[:12],
			Status:      c.State,
			Image:       c.Image,
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
