package model

import (
	"database/sql"
	"time"
)

// Setting 设置项
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetSetting 获取单个设置
func GetSetting(key string) (string, error) {
	db := GetDB()
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting 设置单个值
func SetSetting(key, value string) error {
	db := GetDB()
	_, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

// GetSettings 获取多个设置
func GetSettings(keys []string) (map[string]string, error) {
	db := GetDB()
	result := make(map[string]string)

	for _, key := range keys {
		var value string
		err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
		if err == sql.ErrNoRows {
			result[key] = ""
		} else if err != nil {
			return nil, err
		} else {
			result[key] = value
		}
	}

	return result, nil
}

// SetSettings 批量设置
func SetSettings(settings map[string]string) error {
	db := GetDB()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range settings {
		_, err = stmt.Exec(key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Bark 配置相关常量
const (
	SettingBarkEnabled    = "bark_enabled"
	SettingBarkServer     = "bark_server"
	SettingBarkKey        = "bark_key"
	SettingBarkNotifyMode = "bark_notify_mode" // always, failure_only, success_only
	SettingBarkShowDetail = "bark_show_detail" // true/false
)

// 容器状态变化通知配置常量
const (
	SettingContainerEventEnabled = "container_event_enabled" // 是否启用容器事件通知
	SettingNotifyOnStart         = "notify_on_start"         // 容器启动时通知
	SettingNotifyOnStop          = "notify_on_stop"          // 容器停止时通知
	SettingNotifyOnDie           = "notify_on_die"           // 容器异常退出时通知
	SettingNotifyOnRestart       = "notify_on_restart"       // 容器重启时通知
	SettingNotifyOnCreate        = "notify_on_create"        // 容器创建时通知
	SettingNotifyOnDestroy       = "notify_on_destroy"       // 容器删除时通知
	SettingNotifyOnHealthy       = "notify_on_healthy"       // 容器健康检查状态变化时通知
	SettingNotifyOnUnhealthy     = "notify_on_unhealthy"     // 容器健康检查失败时通知
)

// 通知模式常量
const (
	NotifyModeAlways      = "always"       // 始终通知
	NotifyModeFailureOnly = "failure_only" // 仅失败时通知
	NotifyModeSuccessOnly = "success_only" // 仅全部成功时通知
)

// BarkConfig Bark 推送配置
type BarkConfig struct {
	Enabled    bool   `json:"enabled"`
	Server     string `json:"server"`
	Key        string `json:"key"`
	NotifyMode string `json:"notifyMode"` // always, failure_only, success_only
	ShowDetail bool   `json:"showDetail"` // 是否显示具体明细
}

// GetBarkConfig 获取 Bark 配置
func GetBarkConfig() (*BarkConfig, error) {
	settings, err := GetSettings([]string{
		SettingBarkEnabled,
		SettingBarkServer,
		SettingBarkKey,
		SettingBarkNotifyMode,
		SettingBarkShowDetail,
	})
	if err != nil {
		return nil, err
	}

	notifyMode := settings[SettingBarkNotifyMode]
	if notifyMode == "" {
		notifyMode = NotifyModeAlways // 默认始终通知
	}

	return &BarkConfig{
		Enabled:    settings[SettingBarkEnabled] == "true",
		Server:     settings[SettingBarkServer],
		Key:        settings[SettingBarkKey],
		NotifyMode: notifyMode,
		ShowDetail: settings[SettingBarkShowDetail] == "true",
	}, nil
}

// SaveBarkConfig 保存 Bark 配置
func SaveBarkConfig(config *BarkConfig) error {
	enabledStr := "false"
	if config.Enabled {
		enabledStr = "true"
	}

	showDetailStr := "false"
	if config.ShowDetail {
		showDetailStr = "true"
	}

	notifyMode := config.NotifyMode
	if notifyMode == "" {
		notifyMode = NotifyModeAlways
	}

	return SetSettings(map[string]string{
		SettingBarkEnabled:    enabledStr,
		SettingBarkServer:     config.Server,
		SettingBarkKey:        config.Key,
		SettingBarkNotifyMode: notifyMode,
		SettingBarkShowDetail: showDetailStr,
	})
}

// ContainerEventConfig 容器事件通知配置
type ContainerEventConfig struct {
	Enabled         bool `json:"enabled"`         // 是否启用容器事件通知
	NotifyOnStart   bool `json:"notifyOnStart"`   // 容器启动时通知
	NotifyOnStop    bool `json:"notifyOnStop"`    // 容器停止时通知
	NotifyOnDie     bool `json:"notifyOnDie"`     // 容器异常退出时通知
	NotifyOnRestart bool `json:"notifyOnRestart"` // 容器重启时通知
	NotifyOnCreate  bool `json:"notifyOnCreate"`  // 容器创建时通知
	NotifyOnDestroy bool `json:"notifyOnDestroy"` // 容器删除时通知
	NotifyOnHealthy bool `json:"notifyOnHealthy"` // 容器健康检查通过时通知
	NotifyOnUnhealthy bool `json:"notifyOnUnhealthy"` // 容器健康检查失败时通知
}

// GetContainerEventConfig 获取容器事件通知配置
func GetContainerEventConfig() (*ContainerEventConfig, error) {
	settings, err := GetSettings([]string{
		SettingContainerEventEnabled,
		SettingNotifyOnStart,
		SettingNotifyOnStop,
		SettingNotifyOnDie,
		SettingNotifyOnRestart,
		SettingNotifyOnCreate,
		SettingNotifyOnDestroy,
		SettingNotifyOnHealthy,
		SettingNotifyOnUnhealthy,
	})
	if err != nil {
		return nil, err
	}

	return &ContainerEventConfig{
		Enabled:         settings[SettingContainerEventEnabled] == "true",
		NotifyOnStart:   settings[SettingNotifyOnStart] == "true",
		NotifyOnStop:    settings[SettingNotifyOnStop] == "true",
		NotifyOnDie:     settings[SettingNotifyOnDie] == "true",
		NotifyOnRestart: settings[SettingNotifyOnRestart] == "true",
		NotifyOnCreate:  settings[SettingNotifyOnCreate] == "true",
		NotifyOnDestroy: settings[SettingNotifyOnDestroy] == "true",
		NotifyOnHealthy: settings[SettingNotifyOnHealthy] == "true",
		NotifyOnUnhealthy: settings[SettingNotifyOnUnhealthy] == "true",
	}, nil
}

// SaveContainerEventConfig 保存容器事件通知配置
func SaveContainerEventConfig(config *ContainerEventConfig) error {
	boolToStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	return SetSettings(map[string]string{
		SettingContainerEventEnabled: boolToStr(config.Enabled),
		SettingNotifyOnStart:         boolToStr(config.NotifyOnStart),
		SettingNotifyOnStop:          boolToStr(config.NotifyOnStop),
		SettingNotifyOnDie:           boolToStr(config.NotifyOnDie),
		SettingNotifyOnRestart:       boolToStr(config.NotifyOnRestart),
		SettingNotifyOnCreate:        boolToStr(config.NotifyOnCreate),
		SettingNotifyOnDestroy:       boolToStr(config.NotifyOnDestroy),
		SettingNotifyOnHealthy:       boolToStr(config.NotifyOnHealthy),
		SettingNotifyOnUnhealthy:     boolToStr(config.NotifyOnUnhealthy),
	})
}
