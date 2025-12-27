package model

import (
	"database/sql"
	"time"
)

// ContainerGroup 容器群组模型
type ContainerGroup struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	CronExpr    string    `json:"cronExpr"`
	AutoUpdate  bool      `json:"autoUpdate"`
	CheckUpdate bool      `json:"checkUpdate"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GroupWithDetails 群组详情（包含规则和容器）
type GroupWithDetails struct {
	ContainerGroup
	Rules      []GroupRule      `json:"rules"`
	Containers []GroupContainer `json:"containers"`
}

// CreateGroup 创建群组
func CreateGroup(group *ContainerGroup) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO container_groups (name, cron_expr, auto_update, check_update, priority, enabled)
		VALUES (?, ?, ?, ?, ?, ?)
	`, group.Name, group.CronExpr, boolToInt(group.AutoUpdate), boolToInt(group.CheckUpdate), group.Priority, boolToInt(group.Enabled))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateGroup 更新群组
func UpdateGroup(group *ContainerGroup) error {
	_, err := db.Exec(`
		UPDATE container_groups
		SET name = ?, cron_expr = ?, auto_update = ?, check_update = ?, priority = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, group.Name, group.CronExpr, boolToInt(group.AutoUpdate), boolToInt(group.CheckUpdate), group.Priority, boolToInt(group.Enabled), group.ID)
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
		SELECT id, name, cron_expr, auto_update, check_update, priority, enabled, created_at, updated_at
		FROM container_groups WHERE id = ?
	`, id)

	return scanGroup(row)
}

// GetGroupByName 根据名称获取群组
func GetGroupByName(name string) (*ContainerGroup, error) {
	row := db.QueryRow(`
		SELECT id, name, cron_expr, auto_update, check_update, priority, enabled, created_at, updated_at
		FROM container_groups WHERE name = ?
	`, name)

	return scanGroup(row)
}

// GetAllGroups 获取所有群组
func GetAllGroups() ([]ContainerGroup, error) {
	rows, err := db.Query(`
		SELECT id, name, cron_expr, auto_update, check_update, priority, enabled, created_at, updated_at
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
		SELECT id, name, cron_expr, auto_update, check_update, priority, enabled, created_at, updated_at
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
	err := row.Scan(
		&group.ID, &group.Name, &group.CronExpr,
		&autoUpdate, &checkUpdate, &group.Priority, &enabled,
		&group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	group.AutoUpdate = autoUpdate == 1
	group.CheckUpdate = checkUpdate == 1
	group.Enabled = enabled == 1
	return &group, nil
}

// scanGroupFromRows 从多行扫描群组
func scanGroupFromRows(rows *sql.Rows) (*ContainerGroup, error) {
	var group ContainerGroup
	var autoUpdate, checkUpdate, enabled int
	err := rows.Scan(
		&group.ID, &group.Name, &group.CronExpr,
		&autoUpdate, &checkUpdate, &group.Priority, &enabled,
		&group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	group.AutoUpdate = autoUpdate == 1
	group.CheckUpdate = checkUpdate == 1
	group.Enabled = enabled == 1
	return &group, nil
}

// boolToInt 将bool转换为int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
