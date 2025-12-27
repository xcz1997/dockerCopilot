package model

import (
	"database/sql"
	"time"
)

// UpdateStatus 更新状态
type UpdateStatus string

const (
	UpdateStatusSuccess UpdateStatus = "success"
	UpdateStatusFailed  UpdateStatus = "failed"
	UpdateStatusSkipped UpdateStatus = "skipped"
)

// UpdateHistory 更新历史记录
type UpdateHistory struct {
	ID            int64        `json:"id"`
	GroupID       *int64       `json:"groupId"`      // 可为空，群组被删除时保留历史
	ContainerID   string       `json:"containerId"`
	ContainerName string       `json:"containerName"`
	OldImage      string       `json:"oldImage"`
	NewImage      string       `json:"newImage"`
	Status        UpdateStatus `json:"status"`
	Message       string       `json:"message"`
	CreatedAt     time.Time    `json:"createdAt"`
}

// CreateHistory 创建历史记录
func CreateHistory(h *UpdateHistory) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO update_history (group_id, container_id, container_name, old_image, new_image, status, message)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, h.GroupID, h.ContainerID, h.ContainerName, h.OldImage, h.NewImage, h.Status, h.Message)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetHistoryByID 根据ID获取历史记录
func GetHistoryByID(id int64) (*UpdateHistory, error) {
	row := db.QueryRow(`
		SELECT id, group_id, container_id, container_name, old_image, new_image, status, message, created_at
		FROM update_history WHERE id = ?
	`, id)

	return scanHistory(row)
}

// GetHistoryByGroupID 获取群组的历史记录
func GetHistoryByGroupID(groupID int64, limit int) ([]UpdateHistory, error) {
	rows, err := db.Query(`
		SELECT id, group_id, container_id, container_name, old_image, new_image, status, message, created_at
		FROM update_history WHERE group_id = ? ORDER BY created_at DESC LIMIT ?
	`, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanHistories(rows)
}

// GetRecentHistory 获取最近的历史记录
func GetRecentHistory(limit int) ([]UpdateHistory, error) {
	rows, err := db.Query(`
		SELECT id, group_id, container_id, container_name, old_image, new_image, status, message, created_at
		FROM update_history ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanHistories(rows)
}

// GetHistoryByContainerID 获取容器的历史记录
func GetHistoryByContainerID(containerID string, limit int) ([]UpdateHistory, error) {
	rows, err := db.Query(`
		SELECT id, group_id, container_id, container_name, old_image, new_image, status, message, created_at
		FROM update_history WHERE container_id = ? ORDER BY created_at DESC LIMIT ?
	`, containerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanHistories(rows)
}

// GetHistoryWithPagination 分页获取历史记录
func GetHistoryWithPagination(offset, limit int) ([]UpdateHistory, int, error) {
	// 获取总数
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM update_history`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取数据
	rows, err := db.Query(`
		SELECT id, group_id, container_id, container_name, old_image, new_image, status, message, created_at
		FROM update_history ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	histories, err := scanHistories(rows)
	if err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

// DeleteHistoryBefore 删除指定时间之前的历史记录
func DeleteHistoryBefore(before time.Time) (int64, error) {
	result, err := db.Exec(`DELETE FROM update_history WHERE created_at < ?`, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteHistoryByGroupID 删除群组的所有历史记录
func DeleteHistoryByGroupID(groupID int64) error {
	_, err := db.Exec(`DELETE FROM update_history WHERE group_id = ?`, groupID)
	return err
}

// scanHistory 从单行扫描历史记录
func scanHistory(row *sql.Row) (*UpdateHistory, error) {
	var h UpdateHistory
	var groupID sql.NullInt64
	var message sql.NullString

	err := row.Scan(
		&h.ID, &groupID, &h.ContainerID, &h.ContainerName,
		&h.OldImage, &h.NewImage, &h.Status, &message, &h.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if groupID.Valid {
		h.GroupID = &groupID.Int64
	}
	if message.Valid {
		h.Message = message.String
	}

	return &h, nil
}

// scanHistories 从多行扫描历史记录
func scanHistories(rows *sql.Rows) ([]UpdateHistory, error) {
	var histories []UpdateHistory
	for rows.Next() {
		var h UpdateHistory
		var groupID sql.NullInt64
		var message sql.NullString

		err := rows.Scan(
			&h.ID, &groupID, &h.ContainerID, &h.ContainerName,
			&h.OldImage, &h.NewImage, &h.Status, &message, &h.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if groupID.Valid {
			h.GroupID = &groupID.Int64
		}
		if message.Valid {
			h.Message = message.String
		}

		histories = append(histories, h)
	}

	return histories, rows.Err()
}
