package model

import (
	"database/sql"
	"time"
)

// 环境类型常量
const (
	EnvTypeLocal  = "local"  // 本地环境
	EnvTypeRemote = "remote" // 远程环境
)

// 环境状态常量
const (
	EnvStatusOnline  = "online"  // 在线
	EnvStatusOffline = "offline" // 离线
	EnvStatusUnknown = "unknown" // 未知
	EnvStatusError   = "error"   // 错误
)

// Environment 环境模型
type Environment struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	EnvType        string     `json:"envType"`
	URL            string     `json:"url"`
	SecretKey      string     `json:"-"` // 不返回给前端
	JWTToken       string     `json:"-"` // 不返回给前端
	TokenExpiresAt *time.Time `json:"-"`
	IsDefault      bool       `json:"isDefault"`
	Status         string     `json:"status"`
	LastCheckAt    *time.Time `json:"lastCheckAt"`
	LastError      string     `json:"lastError,omitempty"`
	ContainerCount int        `json:"containerCount"`
	RunningCount   int        `json:"runningCount"`
	StoppedCount   int        `json:"stoppedCount"`
	ImageCount     int        `json:"imageCount"`
	VolumeCount    int        `json:"volumeCount"`
	CPUCores       int        `json:"cpuCores"`
	MemoryTotal    int64      `json:"memoryTotal"` // 单位：字节
	Icon           string     `json:"icon"`        // 自定义图标名称
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// EnvironmentStats 环境统计信息
type EnvironmentStats struct {
	ContainerCount int   `json:"containerCount"`
	RunningCount   int   `json:"runningCount"`
	StoppedCount   int   `json:"stoppedCount"`
	ImageCount     int   `json:"imageCount"`
	VolumeCount    int   `json:"volumeCount"`
	CPUCores       int   `json:"cpuCores"`
	MemoryTotal    int64 `json:"memoryTotal"` // 单位：字节
}

// CreateEnvironment 创建环境
func CreateEnvironment(env *Environment) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO environments (name, description, env_type, url, secret_key, is_default, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, env.Name, env.Description, env.EnvType, env.URL, env.SecretKey, boolToInt(env.IsDefault), env.Status)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateEnvironment 更新环境
func UpdateEnvironment(env *Environment) error {
	_, err := db.Exec(`
		UPDATE environments
		SET name = ?, description = ?, url = ?, icon = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, env.Name, env.Description, env.URL, env.Icon, env.ID)
	return err
}

// UpdateEnvironmentWithSecret 更新环境（包含密钥）
func UpdateEnvironmentWithSecret(env *Environment) error {
	_, err := db.Exec(`
		UPDATE environments
		SET name = ?, description = ?, url = ?, secret_key = ?, icon = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, env.Name, env.Description, env.URL, env.SecretKey, env.Icon, env.ID)
	return err
}

// DeleteEnvironment 删除环境
func DeleteEnvironment(id int64) error {
	_, err := db.Exec(`DELETE FROM environments WHERE id = ?`, id)
	return err
}

// GetEnvironmentByID 根据ID获取环境
func GetEnvironmentByID(id int64) (*Environment, error) {
	row := db.QueryRow(`
		SELECT id, name, description, env_type, url, secret_key, jwt_token, token_expires_at,
		       is_default, status, last_check_at, last_error,
		       container_count, running_count, stopped_count, image_count,
		       volume_count, cpu_cores, memory_total, icon,
		       created_at, updated_at
		FROM environments WHERE id = ?
	`, id)

	return scanEnvironment(row)
}

// GetEnvironmentByName 根据名称获取环境
func GetEnvironmentByName(name string) (*Environment, error) {
	row := db.QueryRow(`
		SELECT id, name, description, env_type, url, secret_key, jwt_token, token_expires_at,
		       is_default, status, last_check_at, last_error,
		       container_count, running_count, stopped_count, image_count,
		       volume_count, cpu_cores, memory_total, icon,
		       created_at, updated_at
		FROM environments WHERE name = ?
	`, name)

	return scanEnvironment(row)
}

// GetAllEnvironments 获取所有环境
func GetAllEnvironments() ([]Environment, error) {
	rows, err := db.Query(`
		SELECT id, name, description, env_type, url, secret_key, jwt_token, token_expires_at,
		       is_default, status, last_check_at, last_error,
		       container_count, running_count, stopped_count, image_count,
		       volume_count, cpu_cores, memory_total, icon,
		       created_at, updated_at
		FROM environments ORDER BY is_default DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var environments []Environment
	for rows.Next() {
		env, err := scanEnvironmentFromRows(rows)
		if err != nil {
			return nil, err
		}
		environments = append(environments, *env)
	}

	return environments, rows.Err()
}

// GetDefaultEnvironment 获取默认环境
func GetDefaultEnvironment() (*Environment, error) {
	row := db.QueryRow(`
		SELECT id, name, description, env_type, url, secret_key, jwt_token, token_expires_at,
		       is_default, status, last_check_at, last_error,
		       container_count, running_count, stopped_count, image_count,
		       volume_count, cpu_cores, memory_total, icon,
		       created_at, updated_at
		FROM environments WHERE is_default = 1 LIMIT 1
	`)

	return scanEnvironment(row)
}

// GetLocalEnvironment 获取本地环境
func GetLocalEnvironment() (*Environment, error) {
	row := db.QueryRow(`
		SELECT id, name, description, env_type, url, secret_key, jwt_token, token_expires_at,
		       is_default, status, last_check_at, last_error,
		       container_count, running_count, stopped_count, image_count,
		       volume_count, cpu_cores, memory_total, icon,
		       created_at, updated_at
		FROM environments WHERE env_type = 'local' LIMIT 1
	`)

	return scanEnvironment(row)
}

// SetDefaultEnvironment 设置默认环境
func SetDefaultEnvironment(id int64) error {
	// 先取消所有默认
	_, err := db.Exec(`UPDATE environments SET is_default = 0`)
	if err != nil {
		return err
	}
	// 设置新的默认
	_, err = db.Exec(`UPDATE environments SET is_default = 1 WHERE id = ?`, id)
	return err
}

// UpdateEnvironmentStats 更新环境统计信息
func UpdateEnvironmentStats(id int64, stats *EnvironmentStats) error {
	_, err := db.Exec(`
		UPDATE environments
		SET container_count = ?, running_count = ?, stopped_count = ?, image_count = ?,
		    volume_count = ?, cpu_cores = ?, memory_total = ?,
		    status = ?, last_check_at = CURRENT_TIMESTAMP, last_error = '', updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, stats.ContainerCount, stats.RunningCount, stats.StoppedCount, stats.ImageCount,
		stats.VolumeCount, stats.CPUCores, stats.MemoryTotal, EnvStatusOnline, id)
	return err
}

// UpdateEnvironmentStatus 更新环境状态
func UpdateEnvironmentStatus(id int64, status, lastError string) error {
	_, err := db.Exec(`
		UPDATE environments
		SET status = ?, last_error = ?, last_check_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, lastError, id)
	return err
}

// UpdateEnvironmentToken 更新环境的 JWT Token
func UpdateEnvironmentToken(id int64, token string, expiresAt time.Time) error {
	_, err := db.Exec(`
		UPDATE environments
		SET jwt_token = ?, token_expires_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, token, expiresAt, id)
	return err
}

// EnsureLocalEnvironment 确保本地环境存在
func EnsureLocalEnvironment() error {
	// 检查是否已存在本地环境
	env, err := GetLocalEnvironment()
	if err == nil && env != nil {
		return nil
	}

	// 创建本地环境
	localEnv := &Environment{
		Name:        "Local",
		Description: "本地 Docker 环境",
		EnvType:     EnvTypeLocal,
		IsDefault:   true,
		Status:      EnvStatusOnline,
	}

	_, err = CreateEnvironment(localEnv)
	return err
}

// scanEnvironment 从单行扫描环境
func scanEnvironment(row *sql.Row) (*Environment, error) {
	var env Environment
	var isDefault int
	var tokenExpiresAt, lastCheckAt sql.NullTime
	var jwtToken, lastError, icon sql.NullString

	err := row.Scan(
		&env.ID, &env.Name, &env.Description, &env.EnvType, &env.URL, &env.SecretKey,
		&jwtToken, &tokenExpiresAt, &isDefault, &env.Status, &lastCheckAt, &lastError,
		&env.ContainerCount, &env.RunningCount, &env.StoppedCount, &env.ImageCount,
		&env.VolumeCount, &env.CPUCores, &env.MemoryTotal, &icon,
		&env.CreatedAt, &env.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	env.IsDefault = isDefault == 1
	if tokenExpiresAt.Valid {
		env.TokenExpiresAt = &tokenExpiresAt.Time
	}
	if lastCheckAt.Valid {
		env.LastCheckAt = &lastCheckAt.Time
	}
	if jwtToken.Valid {
		env.JWTToken = jwtToken.String
	}
	if lastError.Valid {
		env.LastError = lastError.String
	}
	if icon.Valid {
		env.Icon = icon.String
	}

	return &env, nil
}

// scanEnvironmentFromRows 从多行扫描环境
func scanEnvironmentFromRows(rows *sql.Rows) (*Environment, error) {
	var env Environment
	var isDefault int
	var tokenExpiresAt, lastCheckAt sql.NullTime
	var jwtToken, lastError, icon sql.NullString

	err := rows.Scan(
		&env.ID, &env.Name, &env.Description, &env.EnvType, &env.URL, &env.SecretKey,
		&jwtToken, &tokenExpiresAt, &isDefault, &env.Status, &lastCheckAt, &lastError,
		&env.ContainerCount, &env.RunningCount, &env.StoppedCount, &env.ImageCount,
		&env.VolumeCount, &env.CPUCores, &env.MemoryTotal, &icon,
		&env.CreatedAt, &env.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	env.IsDefault = isDefault == 1
	if tokenExpiresAt.Valid {
		env.TokenExpiresAt = &tokenExpiresAt.Time
	}
	if lastCheckAt.Valid {
		env.LastCheckAt = &lastCheckAt.Time
	}
	if jwtToken.Valid {
		env.JWTToken = jwtToken.String
	}
	if lastError.Valid {
		env.LastError = lastError.String
	}
	if icon.Valid {
		env.Icon = icon.String
	}

	return &env, nil
}
