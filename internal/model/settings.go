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
