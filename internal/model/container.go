package model

import (
	"database/sql"
	"time"
)

// GroupContainer 手动分配的容器
type GroupContainer struct {
	ID            int64     `json:"id"`
	GroupID       int64     `json:"groupId"`
	ContainerID   string    `json:"containerId"`
	ContainerName string    `json:"containerName"`
	CreatedAt     time.Time `json:"createdAt"`
}

// CreateGroupContainer 添加容器到群组
func CreateGroupContainer(gc *GroupContainer) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO group_containers (group_id, container_id, container_name)
		VALUES (?, ?, ?)
	`, gc.GroupID, gc.ContainerID, gc.ContainerName)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// DeleteGroupContainer 从群组移除容器
func DeleteGroupContainer(id int64) error {
	_, err := db.Exec(`DELETE FROM group_containers WHERE id = ?`, id)
	return err
}

// DeleteGroupContainerByContainerID 根据容器ID从群组移除
func DeleteGroupContainerByContainerID(groupID int64, containerID string) error {
	_, err := db.Exec(`DELETE FROM group_containers WHERE group_id = ? AND container_id = ?`, groupID, containerID)
	return err
}

// DeleteContainersByGroupID 删除群组下的所有容器
func DeleteContainersByGroupID(groupID int64) error {
	_, err := db.Exec(`DELETE FROM group_containers WHERE group_id = ?`, groupID)
	return err
}

// GetGroupContainerByID 根据ID获取容器分配
func GetGroupContainerByID(id int64) (*GroupContainer, error) {
	row := db.QueryRow(`
		SELECT id, group_id, container_id, container_name, created_at
		FROM group_containers WHERE id = ?
	`, id)

	return scanGroupContainer(row)
}

// GetContainersByGroupID 获取群组下的所有容器
func GetContainersByGroupID(groupID int64) ([]GroupContainer, error) {
	rows, err := db.Query(`
		SELECT id, group_id, container_id, container_name, created_at
		FROM group_containers WHERE group_id = ? ORDER BY id ASC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []GroupContainer
	for rows.Next() {
		gc, err := scanGroupContainerFromRows(rows)
		if err != nil {
			return nil, err
		}
		containers = append(containers, *gc)
	}

	return containers, rows.Err()
}

// GetAllGroupContainers 获取所有容器分配
func GetAllGroupContainers() ([]GroupContainer, error) {
	rows, err := db.Query(`
		SELECT id, group_id, container_id, container_name, created_at
		FROM group_containers ORDER BY group_id ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []GroupContainer
	for rows.Next() {
		gc, err := scanGroupContainerFromRows(rows)
		if err != nil {
			return nil, err
		}
		containers = append(containers, *gc)
	}

	return containers, rows.Err()
}

// GetContainerGroupByContainerID 根据容器ID获取其所属群组（按优先级排序，返回第一个）
func GetContainerGroupByContainerID(containerID string) (*ContainerGroup, error) {
	row := db.QueryRow(`
		SELECT g.id, g.name, g.cron_expr, g.auto_update, g.check_update, g.priority, g.enabled, g.created_at, g.updated_at
		FROM container_groups g
		INNER JOIN group_containers gc ON g.id = gc.group_id
		WHERE gc.container_id = ? AND g.enabled = 1
		ORDER BY g.priority ASC
		LIMIT 1
	`, containerID)

	return scanGroup(row)
}

// IsContainerInGroup 检查容器是否在指定群组中
func IsContainerInGroup(groupID int64, containerID string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM group_containers
		WHERE group_id = ? AND container_id = ?
	`, groupID, containerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// scanGroupContainer 从单行扫描容器分配
func scanGroupContainer(row *sql.Row) (*GroupContainer, error) {
	var gc GroupContainer
	err := row.Scan(&gc.ID, &gc.GroupID, &gc.ContainerID, &gc.ContainerName, &gc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &gc, nil
}

// scanGroupContainerFromRows 从多行扫描容器分配
func scanGroupContainerFromRows(rows *sql.Rows) (*GroupContainer, error) {
	var gc GroupContainer
	err := rows.Scan(&gc.ID, &gc.GroupID, &gc.ContainerID, &gc.ContainerName, &gc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &gc, nil
}
