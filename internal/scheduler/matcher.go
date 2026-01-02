package scheduler

import (
	"context"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// MatchedContainer 匹配到的容器信息
type MatchedContainer struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	ImageID   string            `json:"imageId"`
	Labels    map[string]string `json:"labels"`
	State     string            `json:"state"`
	MatchType string            `json:"matchType"` // rule 或 manual
	RuleID    *int64            `json:"ruleId"`    // 匹配的规则ID
}

// MatchedImage 匹配到的镜像信息
type MatchedImage struct {
	ID        string   `json:"id"`        // 镜像ID
	Name      string   `json:"name"`      // 镜像名称 (如 nginx)
	Tag       string   `json:"tag"`       // 镜像标签 (如 latest)
	FullName  string   `json:"fullName"`  // 完整名称 (如 nginx:latest)
	RepoTags  []string `json:"repoTags"`  // 所有标签
	Size      int64    `json:"size"`      // 镜像大小
	MatchType string   `json:"matchType"` // manual
}

// Matcher 容器匹配器
type Matcher struct {
	dockerClient *client.Client
}

// NewMatcher 创建匹配器
func NewMatcher(dockerClient *client.Client) *Matcher {
	return &Matcher{dockerClient: dockerClient}
}

// GetMatchedContainers 获取群组匹配的所有容器
func (m *Matcher) GetMatchedContainers(ctx context.Context, groupID int64) ([]MatchedContainer, error) {
	// 获取群组信息，确定群组类型
	group, err := model.GetGroupByID(groupID)
	if err != nil {
		return nil, err
	}

	// 获取所有 Docker 容器
	containers, err := m.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	// 获取所有镜像，建立 ImageID 到镜像名称的映射
	imageMap := make(map[string]string)
	images, err := m.dockerClient.ImageList(ctx, image.ListOptions{})
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

	// 获取群组规则
	rules, err := model.GetRulesByGroupID(groupID)
	if err != nil {
		return nil, err
	}

	// 获取手动分配的容器/项目
	manualContainers, err := model.GetContainersByGroupID(groupID)
	if err != nil {
		return nil, err
	}

	// 构建手动分配的容器ID映射和名称映射
	manualMapByID := make(map[string]bool)
	manualMapByName := make(map[string]bool)
	// 对于项目类型群组，存储项目名称用于标签匹配
	manualProjectNames := make(map[string]bool)

	logx.Infof("群组[%s]手动分配记录数: %d, 规则数: %d", group.Name, len(manualContainers), len(rules))

	for _, mc := range manualContainers {
		manualMapByID[mc.ContainerID] = true
		logx.Debugf("群组[%s]分配记录: ID=%s, Name=%s", group.Name, mc.ContainerID, mc.ContainerName)
		// 同时按名称映射，用于容器更新后ID变化的情况
		if mc.ContainerName != "" {
			manualMapByName[mc.ContainerName] = true
		}
		// 对于项目类型群组，ContainerID 和 ContainerName 存储的是项目名称
		if group.GroupType == model.GroupTypeProject {
			manualProjectNames[mc.ContainerID] = true
			if mc.ContainerName != "" {
				manualProjectNames[mc.ContainerName] = true
			}
		}
	}

	// 辅助函数：获取正确的镜像名称
	getImageName := func(imageID, fallback string) string {
		if name, ok := imageMap[imageID]; ok {
			return name
		}
		if fallback != "" && !strings.HasPrefix(fallback, "sha256:") {
			return fallback
		}
		if len(imageID) > 19 {
			return imageID[:19] + "..."
		}
		return imageID
	}

	var matched []MatchedContainer
	matchedIDs := make(map[string]bool)

	// 首先处理手动分配的容器（按ID匹配）
	for _, c := range containers {
		if manualMapByID[c.ID] {
			name := strings.TrimPrefix(c.Names[0], "/")
			matched = append(matched, MatchedContainer{
				ID:        c.ID,
				Name:      name,
				Image:     getImageName(c.ImageID, c.Image),
				ImageID:   c.ImageID,
				Labels:    c.Labels,
				State:     c.State,
				MatchType: "manual",
			})
			matchedIDs[c.ID] = true
		}
	}

	// 再处理手动分配的容器（按名称匹配，用于容器更新后ID变化的情况）
	for _, c := range containers {
		if matchedIDs[c.ID] {
			continue
		}
		name := strings.TrimPrefix(c.Names[0], "/")
		if manualMapByName[name] {
			logx.Debugf("容器[%s]通过名称匹配（ID可能已变化）", name)
			matched = append(matched, MatchedContainer{
				ID:        c.ID,
				Name:      name,
				Image:     getImageName(c.ImageID, c.Image),
				ImageID:   c.ImageID,
				Labels:    c.Labels,
				State:     c.State,
				MatchType: "manual",
			})
			matchedIDs[c.ID] = true
		}
	}

	// 对于项目类型群组，通过 com.docker.compose.project 标签匹配所有属于该项目的容器
	if group.GroupType == model.GroupTypeProject && len(manualProjectNames) > 0 {
		for _, c := range containers {
			if matchedIDs[c.ID] {
				continue
			}
			// 检查容器的 Compose 项目标签
			projectName := c.Labels["com.docker.compose.project"]
			if projectName != "" && manualProjectNames[projectName] {
				name := strings.TrimPrefix(c.Names[0], "/")
				logx.Debugf("容器[%s]通过项目标签匹配（项目: %s）", name, projectName)
				matched = append(matched, MatchedContainer{
					ID:        c.ID,
					Name:      name,
					Image:     getImageName(c.ImageID, c.Image),
					ImageID:   c.ImageID,
					Labels:    c.Labels,
					State:     c.State,
					MatchType: "manual",
				})
				matchedIDs[c.ID] = true
			}
		}
	}

	// 然后处理规则匹配
	for _, c := range containers {
		// 跳过已匹配的容器
		if matchedIDs[c.ID] {
			continue
		}

		name := strings.TrimPrefix(c.Names[0], "/")
		imageName := getImageName(c.ImageID, c.Image)

		// 检查是否匹配任何规则
		for _, rule := range rules {
			if m.matchRule(rule, name, imageName, c.Labels) {
				matched = append(matched, MatchedContainer{
					ID:        c.ID,
					Name:      name,
					Image:     imageName,
					ImageID:   c.ImageID,
					Labels:    c.Labels,
					State:     c.State,
					MatchType: "rule",
					RuleID:    &rule.ID,
				})
				matchedIDs[c.ID] = true
				break // 一个容器只匹配一次
			}
		}
	}

	return matched, nil
}

// GetAllContainerAssignments 获取所有容器的群组分配情况
func (m *Matcher) GetAllContainerAssignments(ctx context.Context) (map[string]*model.ContainerGroup, error) {
	// 获取所有容器
	containers, err := m.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	// 获取所有启用的群组（按优先级排序）
	groups, err := model.GetEnabledGroups()
	if err != nil {
		return nil, err
	}

	// 获取所有规则
	allRules, err := model.GetAllRules()
	if err != nil {
		return nil, err
	}

	// 按群组ID分组规则
	groupRules := make(map[int64][]model.GroupRule)
	for _, rule := range allRules {
		groupRules[rule.GroupID] = append(groupRules[rule.GroupID], rule)
	}

	// 获取所有手动分配
	allManual, err := model.GetAllGroupContainers()
	if err != nil {
		return nil, err
	}

	// 按容器ID分组手动分配
	manualByContainer := make(map[string]int64) // containerID -> groupID
	for _, mc := range allManual {
		manualByContainer[mc.ContainerID] = mc.GroupID
	}

	// 构建群组ID到群组的映射
	groupMap := make(map[int64]*model.ContainerGroup)
	for i := range groups {
		groupMap[groups[i].ID] = &groups[i]
	}

	// 为每个容器找到匹配的群组
	result := make(map[string]*model.ContainerGroup)

	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		image := c.Image

		// 首先检查手动分配
		if groupID, ok := manualByContainer[c.ID]; ok {
			if group, exists := groupMap[groupID]; exists {
				result[c.ID] = group
				continue
			}
		}

		// 按优先级检查规则匹配
		for _, group := range groups {
			rules := groupRules[group.ID]
			for _, rule := range rules {
				if m.matchRule(rule, name, image, c.Labels) {
					result[c.ID] = &group
					break
				}
			}
			if result[c.ID] != nil {
				break
			}
		}
	}

	return result, nil
}

// matchRule 检查容器是否匹配规则
func (m *Matcher) matchRule(rule model.GroupRule, containerName, imageName string, labels map[string]string) bool {
	switch rule.RuleType {
	case model.RuleTypeNamePrefix:
		return strings.HasPrefix(containerName, rule.Pattern)

	case model.RuleTypeNameSuffix:
		return strings.HasSuffix(containerName, rule.Pattern)

	case model.RuleTypeNameContains:
		return strings.Contains(containerName, rule.Pattern)

	case model.RuleTypeNameRegex:
		matched, err := regexp.MatchString(rule.Pattern, containerName)
		if err != nil {
			logx.Errorf("正则表达式错误[%s]: %v", rule.Pattern, err)
			return false
		}
		return matched

	case model.RuleTypeImagePrefix:
		return strings.HasPrefix(imageName, rule.Pattern)

	case model.RuleTypeImageRegex:
		matched, err := regexp.MatchString(rule.Pattern, imageName)
		if err != nil {
			logx.Errorf("正则表达式错误[%s]: %v", rule.Pattern, err)
			return false
		}
		return matched

	case model.RuleTypeLabelKey:
		_, exists := labels[rule.Pattern]
		return exists

	case model.RuleTypeLabelValue:
		// 格式: key=value
		parts := strings.SplitN(rule.Pattern, "=", 2)
		if len(parts) != 2 {
			return false
		}
		value, exists := labels[parts[0]]
		return exists && value == parts[1]

	default:
		return false
	}
}

// PreviewRuleMatches 预览规则匹配结果
func (m *Matcher) PreviewRuleMatches(ctx context.Context, ruleType model.RuleType, pattern string) ([]MatchedContainer, error) {
	containers, err := m.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	// 获取所有镜像，建立 ImageID 到镜像名称的映射
	imageMap := make(map[string]string)
	images, err := m.dockerClient.ImageList(ctx, image.ListOptions{})
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

	// 辅助函数：获取正确的镜像名称
	getImageName := func(imageID, fallback string) string {
		if name, ok := imageMap[imageID]; ok {
			return name
		}
		if fallback != "" && !strings.HasPrefix(fallback, "sha256:") {
			return fallback
		}
		if len(imageID) > 19 {
			return imageID[:19] + "..."
		}
		return imageID
	}

	rule := model.GroupRule{
		RuleType: ruleType,
		Pattern:  pattern,
	}

	var matched []MatchedContainer
	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		imageName := getImageName(c.ImageID, c.Image)

		if m.matchRule(rule, name, imageName, c.Labels) {
			matched = append(matched, MatchedContainer{
				ID:      c.ID,
				Name:    name,
				Image:   imageName,
				ImageID: c.ImageID,
				Labels:  c.Labels,
				State:   c.State,
			})
		}
	}

	return matched, nil
}

// GetMatchedImages 获取群组匹配的所有镜像（用于镜像类型群组）
func (m *Matcher) GetMatchedImages(ctx context.Context, groupID int64) ([]MatchedImage, error) {
	// 获取所有 Docker 镜像
	images, err := m.dockerClient.ImageList(ctx, image.ListOptions{All: false})
	if err != nil {
		return nil, err
	}

	// 获取手动分配的镜像
	manualImages, err := model.GetContainersByGroupID(groupID)
	if err != nil {
		return nil, err
	}

	// 构建手动分配的镜像ID映射和名称映射
	manualMapByID := make(map[string]bool)
	manualMapByName := make(map[string]bool)
	for _, mi := range manualImages {
		manualMapByID[mi.ContainerID] = true
		if mi.ContainerName != "" {
			manualMapByName[mi.ContainerName] = true
		}
	}

	var matched []MatchedImage
	matchedIDs := make(map[string]bool)

	for _, img := range images {
		// 跳过无效镜像
		if len(img.RepoTags) == 0 && len(img.RepoDigests) == 0 {
			continue
		}

		// 解析镜像名称和标签
		var imageName, imageTag, fullName string
		if len(img.RepoTags) > 0 && img.RepoTags[0] != "<none>:<none>" {
			fullName = img.RepoTags[0]
			parts := strings.Split(fullName, ":")
			imageName = parts[0]
			if len(parts) > 1 {
				imageTag = parts[1]
			}
		} else if len(img.RepoDigests) > 0 {
			// 从 digest 提取镜像名
			digest := img.RepoDigests[0]
			if idx := strings.Index(digest, "@"); idx > 0 {
				imageName = digest[:idx]
				imageTag = "latest"
				fullName = imageName + ":latest"
			}
		}

		if imageName == "" {
			continue
		}

		// 检查是否手动分配（按ID匹配）
		if manualMapByID[img.ID] {
			matched = append(matched, MatchedImage{
				ID:        img.ID,
				Name:      imageName,
				Tag:       imageTag,
				FullName:  fullName,
				RepoTags:  img.RepoTags,
				Size:      img.Size,
				MatchType: "manual",
			})
			matchedIDs[img.ID] = true
			continue
		}

		// 检查是否手动分配（按名称匹配）
		if manualMapByName[fullName] || manualMapByName[imageName] {
			if !matchedIDs[img.ID] {
				logx.Debugf("镜像[%s]通过名称匹配", fullName)
				matched = append(matched, MatchedImage{
					ID:        img.ID,
					Name:      imageName,
					Tag:       imageTag,
					FullName:  fullName,
					RepoTags:  img.RepoTags,
					Size:      img.Size,
					MatchType: "manual",
				})
				matchedIDs[img.ID] = true
			}
		}
	}

	return matched, nil
}
