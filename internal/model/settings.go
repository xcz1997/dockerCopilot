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
	SettingBarkEnabled = "bark_enabled"
	SettingBarkServer  = "bark_server"
	SettingBarkKey     = "bark_key"
)

// BarkConfig Bark 推送配置
type BarkConfig struct {
	Enabled bool   `json:"enabled"`
	Server  string `json:"server"`
	Key     string `json:"key"`
}

// GetBarkConfig 获取 Bark 配置
func GetBarkConfig() (*BarkConfig, error) {
	settings, err := GetSettings([]string{SettingBarkEnabled, SettingBarkServer, SettingBarkKey})
	if err != nil {
		return nil, err
	}

	return &BarkConfig{
		Enabled: settings[SettingBarkEnabled] == "true",
		Server:  settings[SettingBarkServer],
		Key:     settings[SettingBarkKey],
	}, nil
}

// SaveBarkConfig 保存 Bark 配置
func SaveBarkConfig(config *BarkConfig) error {
	enabledStr := "false"
	if config.Enabled {
		enabledStr = "true"
	}

	return SetSettings(map[string]string{
		SettingBarkEnabled: enabledStr,
		SettingBarkServer:  config.Server,
		SettingBarkKey:     config.Key,
	})
}
