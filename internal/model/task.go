package model

import (
	"database/sql"
	"encoding/json"
	"time"
)

// SubTask 子任务
type SubTask struct {
	Name       string     `json:"name"`
	Status     string     `json:"status"` // pending, in_progress, completed, failed
	Message    string     `json:"message"`
	DetailMsg  string     `json:"detailMsg,omitempty"`  // 详细进度信息
	Percentage int        `json:"percentage,omitempty"` // 子任务进度百分比 0-100
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// Task 任务
type Task struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Message    string     `json:"message"`
	DetailMsg  string     `json:"detailMsg"`
	Percentage int        `json:"percentage"`
	IsDone     bool       `json:"isDone"`
	TaskType   string     `json:"taskType"`   // container_update, group_check, group_update
	TargetID   string     `json:"targetId"`   // 容器ID或群组ID
	TargetName string     `json:"targetName"` // 容器名或群组名
	SubTasks   []SubTask  `json:"subTasks"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// SaveTask 保存任务（插入或更新）
func SaveTask(task *Task) error {
	db := GetDB()

	subTasksJSON, err := json.Marshal(task.SubTasks)
	if err != nil {
		return err
	}

	isDone := 0
	if task.IsDone {
		isDone = 1
	}

	_, err = db.Exec(`
		INSERT INTO tasks (id, name, message, detail_msg, percentage, is_done, task_type, target_id, target_name, sub_tasks, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			message = excluded.message,
			detail_msg = excluded.detail_msg,
			percentage = excluded.percentage,
			is_done = excluded.is_done,
			task_type = excluded.task_type,
			target_id = excluded.target_id,
			target_name = excluded.target_name,
			sub_tasks = excluded.sub_tasks,
			finished_at = excluded.finished_at
	`, task.ID, task.Name, task.Message, task.DetailMsg, task.Percentage, isDone,
		task.TaskType, task.TargetID, task.TargetName, string(subTasksJSON),
		task.StartedAt, task.FinishedAt)
	return err
}

// GetTask 获取单个任务
func GetTask(id string) (*Task, error) {
	db := GetDB()
	var task Task
	var isDone int
	var subTasksJSON string
	var finishedAt sql.NullTime

	err := db.QueryRow(`
		SELECT id, name, message, detail_msg, percentage, is_done, task_type, target_id, target_name, sub_tasks, started_at, finished_at
		FROM tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.Name, &task.Message, &task.DetailMsg, &task.Percentage, &isDone,
		&task.TaskType, &task.TargetID, &task.TargetName, &subTasksJSON, &task.StartedAt, &finishedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	task.IsDone = isDone == 1
	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	}

	if err := json.Unmarshal([]byte(subTasksJSON), &task.SubTasks); err != nil {
		task.SubTasks = []SubTask{}
	}

	return &task, nil
}

// GetAllTasks 获取所有任务
func GetAllTasks() ([]Task, error) {
	db := GetDB()
	rows, err := db.Query(`
		SELECT id, name, message, detail_msg, percentage, is_done, task_type, target_id, target_name, sub_tasks, started_at, finished_at
		FROM tasks ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var isDone int
		var subTasksJSON string
		var finishedAt sql.NullTime

		err := rows.Scan(&task.ID, &task.Name, &task.Message, &task.DetailMsg, &task.Percentage, &isDone,
			&task.TaskType, &task.TargetID, &task.TargetName, &subTasksJSON, &task.StartedAt, &finishedAt)
		if err != nil {
			return nil, err
		}

		task.IsDone = isDone == 1
		if finishedAt.Valid {
			task.FinishedAt = &finishedAt.Time
		}

		if err := json.Unmarshal([]byte(subTasksJSON), &task.SubTasks); err != nil {
			task.SubTasks = []SubTask{}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetCurrentTasks 获取进行中的任务
func GetCurrentTasks() ([]Task, error) {
	db := GetDB()
	rows, err := db.Query(`
		SELECT id, name, message, detail_msg, percentage, is_done, task_type, target_id, target_name, sub_tasks, started_at, finished_at
		FROM tasks WHERE is_done = 0 ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var isDone int
		var subTasksJSON string
		var finishedAt sql.NullTime

		err := rows.Scan(&task.ID, &task.Name, &task.Message, &task.DetailMsg, &task.Percentage, &isDone,
			&task.TaskType, &task.TargetID, &task.TargetName, &subTasksJSON, &task.StartedAt, &finishedAt)
		if err != nil {
			return nil, err
		}

		task.IsDone = isDone == 1
		if finishedAt.Valid {
			task.FinishedAt = &finishedAt.Time
		}

		if err := json.Unmarshal([]byte(subTasksJSON), &task.SubTasks); err != nil {
			task.SubTasks = []SubTask{}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetHistoryTasks 获取历史任务（已完成）
func GetHistoryTasks(limit int) ([]Task, error) {
	db := GetDB()
	rows, err := db.Query(`
		SELECT id, name, message, detail_msg, percentage, is_done, task_type, target_id, target_name, sub_tasks, started_at, finished_at
		FROM tasks WHERE is_done = 1 ORDER BY finished_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var isDone int
		var subTasksJSON string
		var finishedAt sql.NullTime

		err := rows.Scan(&task.ID, &task.Name, &task.Message, &task.DetailMsg, &task.Percentage, &isDone,
			&task.TaskType, &task.TargetID, &task.TargetName, &subTasksJSON, &task.StartedAt, &finishedAt)
		if err != nil {
			return nil, err
		}

		task.IsDone = isDone == 1
		if finishedAt.Valid {
			task.FinishedAt = &finishedAt.Time
		}

		if err := json.Unmarshal([]byte(subTasksJSON), &task.SubTasks); err != nil {
			task.SubTasks = []SubTask{}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// DeleteTask 删除任务
func DeleteTask(id string) error {
	db := GetDB()
	_, err := db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

// CleanOldTasks 清理旧任务（保留最近 N 条已完成任务）
func CleanOldTasks(keepCount int) error {
	db := GetDB()
	_, err := db.Exec(`
		DELETE FROM tasks WHERE is_done = 1 AND id NOT IN (
			SELECT id FROM tasks WHERE is_done = 1 ORDER BY finished_at DESC LIMIT ?
		)
	`, keepCount)
	return err
}
