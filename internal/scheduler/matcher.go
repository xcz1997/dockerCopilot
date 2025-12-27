package scheduler

import (
	"context"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types/container"
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
	// 获取所有 Docker 容器
	containers, err := m.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	// 获取群组规则
	rules, err := model.GetRulesByGroupID(groupID)
	if err != nil {
		return nil, err
	}

	// 获取手动分配的容器
	manualContainers, err := model.GetContainersByGroupID(groupID)
	if err != nil {
		return nil, err
	}

	// 构建手动分配的容器ID映射
	manualMap := make(map[string]bool)
	for _, mc := range manualContainers {
		manualMap[mc.ContainerID] = true
	}

	var matched []MatchedContainer
	matchedIDs := make(map[string]bool)

	// 首先处理手动分配的容器
	for _, c := range containers {
		if manualMap[c.ID] {
			name := strings.TrimPrefix(c.Names[0], "/")
			matched = append(matched, MatchedContainer{
				ID:        c.ID,
				Name:      name,
				Image:     c.Image,
				ImageID:   c.ImageID,
				Labels:    c.Labels,
				State:     c.State,
				MatchType: "manual",
			})
			matchedIDs[c.ID] = true
		}
	}

	// 然后处理规则匹配
	for _, c := range containers {
		// 跳过已匹配的容器
		if matchedIDs[c.ID] {
			continue
		}

		name := strings.TrimPrefix(c.Names[0], "/")
		image := c.Image

		// 检查是否匹配任何规则
		for _, rule := range rules {
			if m.matchRule(rule, name, image, c.Labels) {
				matched = append(matched, MatchedContainer{
					ID:        c.ID,
					Name:      name,
					Image:     image,
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

	rule := model.GroupRule{
		RuleType: ruleType,
		Pattern:  pattern,
	}

	var matched []MatchedContainer
	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		image := c.Image

		if m.matchRule(rule, name, image, c.Labels) {
			matched = append(matched, MatchedContainer{
				ID:      c.ID,
				Name:    name,
				Image:   image,
				ImageID: c.ImageID,
				Labels:  c.Labels,
				State:   c.State,
			})
		}
	}

	return matched, nil
}
