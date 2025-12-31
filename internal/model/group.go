package model

import (
	"database/sql"
	"time"
)

// GroupType 群组类型常量
const (
	GroupTypeContainer = "container" // 容器维度
	GroupTypeProject   = "project"   // Compose 项目维度
	GroupTypeImage     = "image"     // 镜像维度
)

// ContainerGroup 容器群组模型
type ContainerGroup struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	GroupType          string    `json:"groupType"` // container, project, image
	CronExpr           string    `json:"cronExpr"`
	AutoUpdate         bool      `json:"autoUpdate"`
	CheckUpdate        bool      `json:"checkUpdate"`
	Priority           int       `json:"priority"`
	Enabled            bool      `json:"enabled"`
	RestartAfterUpdate bool      `json:"restartAfterUpdate"` // 更新后重启容器
	StartContainers    bool      `json:"startContainers"`    // 启动停止的容器
	StopContainers     bool      `json:"stopContainers"`     // 关闭运行中的容器
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// GroupWithDetails 群组详情（包含规则和容器）
type GroupWithDetails struct {
	ContainerGroup
	Rules      []GroupRule      `json:"rules"`
	Containers []GroupContainer `json:"containers"`
}

// CreateGroup 创建群组
func CreateGroup(group *ContainerGroup) (int64, error) {
	// 默认类型为 container
	if group.GroupType == "" {
		group.GroupType = GroupTypeContainer
	}
	result, err := db.Exec(`
		INSERT INTO container_groups (name, group_type, cron_expr, auto_update, check_update, priority, enabled, restart_after_update, start_containers, stop_containers)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, group.Name, group.GroupType, group.CronExpr, boolToInt(group.AutoUpdate), boolToInt(group.CheckUpdate), group.Priority, boolToInt(group.Enabled),
		boolToInt(group.RestartAfterUpdate), boolToInt(group.StartContainers), boolToInt(group.StopContainers))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateGroup 更新群组
func UpdateGroup(group *ContainerGroup) error {
	_, err := db.Exec(`
		UPDATE container_groups
		SET name = ?, group_type = ?, cron_expr = ?, auto_update = ?, check_update = ?, priority = ?, enabled = ?,
		    restart_after_update = ?, start_containers = ?, stop_containers = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, group.Name, group.GroupType, group.CronExpr, boolToInt(group.AutoUpdate), boolToInt(group.CheckUpdate), group.Priority, boolToInt(group.Enabled),
		boolToInt(group.RestartAfterUpdate), boolToInt(group.StartContainers), boolToInt(group.StopContainers), group.ID)
	return err
}

// DeleteGroup 删除群组
func DeleteGroup(id int64) error {
	_, err := db.Exec(`DELETE FROM container_groups WHERE id = ?`, id)
	return err
}

// GetGroupByID 根据ID获取群组
func GetGroupByID(id int64) (*ContainerGroup, error) {
	row := db.QueryRow(`
		SELECT id, name, group_type, cron_expr, auto_update, check_update, priority, enabled,
		       restart_after_update, start_containers, stop_containers, created_at, updated_at
		FROM container_groups WHERE id = ?
	`, id)

	return scanGroup(row)
}

// GetGroupByName 根据名称获取群组
func GetGroupByName(name string) (*ContainerGroup, error) {
	row := db.QueryRow(`
		SELECT id, name, group_type, cron_expr, auto_update, check_update, priority, enabled,
		       restart_after_update, start_containers, stop_containers, created_at, updated_at
		FROM container_groups WHERE name = ?
	`, name)

	return scanGroup(row)
}

// GetAllGroups 获取所有群组
func GetAllGroups() ([]ContainerGroup, error) {
	rows, err := db.Query(`
		SELECT id, name, group_type, cron_expr, auto_update, check_update, priority, enabled,
		       restart_after_update, start_containers, stop_containers, created_at, updated_at
		FROM container_groups ORDER BY priority ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []ContainerGroup
	for rows.Next() {
		group, err := scanGroupFromRows(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, *group)
	}

	return groups, rows.Err()
}

// GetEnabledGroups 获取所有启用的群组
func GetEnabledGroups() ([]ContainerGroup, error) {
	rows, err := db.Query(`
		SELECT id, name, group_type, cron_expr, auto_update, check_update, priority, enabled,
		       restart_after_update, start_containers, stop_containers, created_at, updated_at
		FROM container_groups WHERE enabled = 1 ORDER BY priority ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []ContainerGroup
	for rows.Next() {
		group, err := scanGroupFromRows(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, *group)
	}

	return groups, rows.Err()
}

// GetGroupWithDetails 获取群组详情
func GetGroupWithDetails(id int64) (*GroupWithDetails, error) {
	group, err := GetGroupByID(id)
	if err != nil {
		return nil, err
	}

	rules, err := GetRulesByGroupID(id)
	if err != nil {
		return nil, err
	}

	containers, err := GetContainersByGroupID(id)
	if err != nil {
		return nil, err
	}

	return &GroupWithDetails{
		ContainerGroup: *group,
		Rules:          rules,
		Containers:     containers,
	}, nil
}

// scanGroup 从单行扫描群组
func scanGroup(row *sql.Row) (*ContainerGroup, error) {
	var group ContainerGroup
	var autoUpdate, checkUpdate, enabled int
	var restartAfterUpdate, startContainers, stopContainers int
	var groupType sql.NullString
	err := row.Scan(
		&group.ID, &group.Name, &groupType, &group.CronExpr,
		&autoUpdate, &checkUpdate, &group.Priority, &enabled,
		&restartAfterUpdate, &startContainers, &stopContainers,
		&group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	group.GroupType = groupType.String
	if group.GroupType == "" {
		group.GroupType = GroupTypeContainer
	}
	group.AutoUpdate = autoUpdate == 1
	group.CheckUpdate = checkUpdate == 1
	group.Enabled = enabled == 1
	group.RestartAfterUpdate = restartAfterUpdate == 1
	group.StartContainers = startContainers == 1
	group.StopContainers = stopContainers == 1
	return &group, nil
}

// scanGroupFromRows 从多行扫描群组
func scanGroupFromRows(rows *sql.Rows) (*ContainerGroup, error) {
	var group ContainerGroup
	var autoUpdate, checkUpdate, enabled int
	var restartAfterUpdate, startContainers, stopContainers int
	var groupType sql.NullString
	err := rows.Scan(
		&group.ID, &group.Name, &groupType, &group.CronExpr,
		&autoUpdate, &checkUpdate, &group.Priority, &enabled,
		&restartAfterUpdate, &startContainers, &stopContainers,
		&group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	group.GroupType = groupType.String
	if group.GroupType == "" {
		group.GroupType = GroupTypeContainer
	}
	group.AutoUpdate = autoUpdate == 1
	group.CheckUpdate = checkUpdate == 1
	group.Enabled = enabled == 1
	group.RestartAfterUpdate = restartAfterUpdate == 1
	group.StartContainers = startContainers == 1
	group.StopContainers = stopContainers == 1
	return &group, nil
}

// boolToInt 将bool转换为int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
