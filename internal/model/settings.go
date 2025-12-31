package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
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

// Registry 镜像配置常量
const (
	SettingRegistryMirrorsEnabled = "registry_mirrors_enabled" // 是否启用自定义镜像
	SettingRegistryMirrors        = "registry_mirrors"         // JSON 数组，多个镜像地址
)

// 代理配置常量
const (
	SettingProxyEnabled = "proxy_enabled" // 是否启用代理
	SettingProxyType    = "proxy_type"    // http, https, socks5
	SettingProxyHost    = "proxy_host"    // 代理服务器地址
	SettingProxyPort    = "proxy_port"    // 代理端口
	SettingProxyUser    = "proxy_user"    // 代理用户名（可选）
	SettingProxyPass    = "proxy_pass"    // 代理密码（可选）
)

// 代理配置环境变量
const (
	EnvProxyEnabled = "PROXY_ENABLED"  // true/false
	EnvProxyType    = "PROXY_TYPE"     // http, https, socks5
	EnvProxyHost    = "PROXY_HOST"     // 代理服务器地址
	EnvProxyPort    = "PROXY_PORT"     // 代理端口
	EnvProxyUser    = "PROXY_USER"     // 代理用户名（可选）
	EnvProxyPass    = "PROXY_PASS"     // 代理密码（可选）
)

// RegistryMirrorsConfig Registry 镜像地址配置
type RegistryMirrorsConfig struct {
	Enabled bool     `json:"enabled"` // 是否启用自定义镜像
	Mirrors []string `json:"mirrors"` // 镜像地址列表
}

// Registry 配置环境变量
const (
	EnvRegistryMirrorsEnabled = "REGISTRY_MIRRORS_ENABLED" // true/false
	EnvRegistryMirrors        = "REGISTRY_MIRRORS"         // 逗号分隔的镜像地址列表
)

// GetRegistryMirrorsConfig 获取 Registry 镜像配置
// 优先级：环境变量 > 数据库
func GetRegistryMirrorsConfig() (*RegistryMirrorsConfig, error) {
	settings, err := GetSettings([]string{
		SettingRegistryMirrorsEnabled,
		SettingRegistryMirrors,
	})
	if err != nil {
		return nil, err
	}

	config := &RegistryMirrorsConfig{
		Enabled: settings[SettingRegistryMirrorsEnabled] == "true",
		Mirrors: []string{},
	}

	// 解析 JSON 数组
	if settings[SettingRegistryMirrors] != "" {
		if err := json.Unmarshal([]byte(settings[SettingRegistryMirrors]), &config.Mirrors); err != nil {
			// 如果解析失败，返回空数组
			config.Mirrors = []string{}
		}
	}

	// 环境变量覆盖（优先级最高）
	if envEnabled := os.Getenv(EnvRegistryMirrorsEnabled); envEnabled != "" {
		config.Enabled = strings.ToLower(envEnabled) == "true"
	}
	if envMirrors := os.Getenv(EnvRegistryMirrors); envMirrors != "" {
		// 逗号分隔的镜像地址列表
		mirrors := strings.Split(envMirrors, ",")
		config.Mirrors = make([]string, 0, len(mirrors))
		for _, m := range mirrors {
			m = strings.TrimSpace(m)
			if m != "" {
				config.Mirrors = append(config.Mirrors, m)
			}
		}
		// 如果设置了镜像地址，自动启用
		if len(config.Mirrors) > 0 {
			config.Enabled = true
		}
	}

	return config, nil
}

// SaveRegistryMirrorsConfig 保存 Registry 镜像配置
func SaveRegistryMirrorsConfig(config *RegistryMirrorsConfig) error {
	enabledStr := "false"
	if config.Enabled {
		enabledStr = "true"
	}

	// 序列化镜像列表为 JSON
	mirrorsJSON, err := json.Marshal(config.Mirrors)
	if err != nil {
		return err
	}

	return SetSettings(map[string]string{
		SettingRegistryMirrorsEnabled: enabledStr,
		SettingRegistryMirrors:        string(mirrorsJSON),
	})
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`  // 是否启用代理
	Type     string `json:"type"`     // http, https, socks5
	Host     string `json:"host"`     // 代理服务器地址
	Port     int    `json:"port"`     // 代理端口
	Username string `json:"username"` // 代理用户名（可选）
	Password string `json:"password"` // 代理密码（可选）
}

// GetProxyURL 获取代理 URL
func (p *ProxyConfig) GetProxyURL() string {
	if !p.Enabled || p.Host == "" || p.Port == 0 {
		return ""
	}
	auth := ""
	if p.Username != "" {
		if p.Password != "" {
			auth = url.QueryEscape(p.Username) + ":" + url.QueryEscape(p.Password) + "@"
		} else {
			auth = url.QueryEscape(p.Username) + "@"
		}
	}
	return fmt.Sprintf("%s://%s%s:%d", p.Type, auth, p.Host, p.Port)
}

// GetProxyConfig 获取代理配置
// 优先级：环境变量 > 数据库
func GetProxyConfig() (*ProxyConfig, error) {
	settings, err := GetSettings([]string{
		SettingProxyEnabled,
		SettingProxyType,
		SettingProxyHost,
		SettingProxyPort,
		SettingProxyUser,
		SettingProxyPass,
	})
	if err != nil {
		return nil, err
	}

	port := 0
	if settings[SettingProxyPort] != "" {
		port, _ = strconv.Atoi(settings[SettingProxyPort])
	}

	proxyType := settings[SettingProxyType]
	if proxyType == "" {
		proxyType = "http" // 默认 HTTP 代理
	}

	config := &ProxyConfig{
		Enabled:  settings[SettingProxyEnabled] == "true",
		Type:     proxyType,
		Host:     settings[SettingProxyHost],
		Port:     port,
		Username: settings[SettingProxyUser],
		Password: settings[SettingProxyPass],
	}

	// 环境变量覆盖（优先级最高）
	if envEnabled := os.Getenv(EnvProxyEnabled); envEnabled != "" {
		config.Enabled = strings.ToLower(envEnabled) == "true"
	}
	if envType := os.Getenv(EnvProxyType); envType != "" {
		config.Type = envType
	}
	if envHost := os.Getenv(EnvProxyHost); envHost != "" {
		config.Host = envHost
		// 如果设置了代理地址，自动启用
		config.Enabled = true
	}
	if envPort := os.Getenv(EnvProxyPort); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			config.Port = p
		}
	}
	if envUser := os.Getenv(EnvProxyUser); envUser != "" {
		config.Username = envUser
	}
	if envPass := os.Getenv(EnvProxyPass); envPass != "" {
		config.Password = envPass
	}

	return config, nil
}

// SaveProxyConfig 保存代理配置
func SaveProxyConfig(config *ProxyConfig) error {
	enabledStr := "false"
	if config.Enabled {
		enabledStr = "true"
	}

	proxyType := config.Type
	if proxyType == "" {
		proxyType = "http"
	}

	return SetSettings(map[string]string{
		SettingProxyEnabled: enabledStr,
		SettingProxyType:    proxyType,
		SettingProxyHost:    config.Host,
		SettingProxyPort:    strconv.Itoa(config.Port),
		SettingProxyUser:    config.Username,
		SettingProxyPass:    config.Password,
	})
}

// Performance 配置常量
const (
	SettingPerformanceLowPowerMode         = "performance_low_power_mode"
	SettingPerformanceMaxConcurrentChecks  = "performance_max_concurrent_checks"
	SettingPerformanceCheckIntervalMinutes = "performance_check_interval_minutes"
	SettingPerformanceDisableAutoCheck     = "performance_disable_auto_check"
)

// 性能配置环境变量
const (
	EnvPerformanceLowPowerMode         = "LOW_POWER_MODE"          // true/false
	EnvPerformanceMaxConcurrentChecks  = "MAX_CONCURRENT_CHECKS"   // 数字，1-20
	EnvPerformanceCheckIntervalMinutes = "CHECK_INTERVAL_MINUTES"  // 数字，0-1440
	EnvPerformanceDisableAutoCheck     = "DISABLE_AUTO_CHECK"      // true/false
)

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	LowPowerMode         bool `json:"lowPowerMode"`         // 低性能模式
	MaxConcurrentChecks  int  `json:"maxConcurrentChecks"`  // 镜像检查最大并发数
	CheckIntervalMinutes int  `json:"checkIntervalMinutes"` // 镜像自动检查间隔（分钟）
	DisableAutoCheck     bool `json:"disableAutoCheck"`     // 禁用启动时自动检查
}

// GetEffectiveMaxConcurrent 获取实际生效的最大并发数
func (p *PerformanceConfig) GetEffectiveMaxConcurrent() int {
	if p.LowPowerMode {
		return 1
	}
	if p.MaxConcurrentChecks <= 0 {
		return 10
	}
	if p.MaxConcurrentChecks > 20 {
		return 20
	}
	return p.MaxConcurrentChecks
}

// GetEffectiveCheckInterval 获取实际生效的检查间隔
func (p *PerformanceConfig) GetEffectiveCheckInterval() int {
	if p.CheckIntervalMinutes < 0 {
		return 30
	}
	return p.CheckIntervalMinutes
}

// GetPerformanceConfig 获取性能配置
// 优先级：环境变量 > 数据库 > 默认值
func GetPerformanceConfig() (*PerformanceConfig, error) {
	settings, err := GetSettings([]string{
		SettingPerformanceLowPowerMode,
		SettingPerformanceMaxConcurrentChecks,
		SettingPerformanceCheckIntervalMinutes,
		SettingPerformanceDisableAutoCheck,
	})
	if err != nil {
		return nil, err
	}

	maxConcurrent := 10
	if settings[SettingPerformanceMaxConcurrentChecks] != "" {
		maxConcurrent, _ = strconv.Atoi(settings[SettingPerformanceMaxConcurrentChecks])
	}

	checkInterval := 30
	if settings[SettingPerformanceCheckIntervalMinutes] != "" {
		checkInterval, _ = strconv.Atoi(settings[SettingPerformanceCheckIntervalMinutes])
	}

	config := &PerformanceConfig{
		LowPowerMode:         settings[SettingPerformanceLowPowerMode] == "true",
		MaxConcurrentChecks:  maxConcurrent,
		CheckIntervalMinutes: checkInterval,
		DisableAutoCheck:     settings[SettingPerformanceDisableAutoCheck] == "true",
	}

	// 环境变量覆盖（优先级最高）
	if envLowPower := os.Getenv(EnvPerformanceLowPowerMode); envLowPower != "" {
		config.LowPowerMode = strings.ToLower(envLowPower) == "true"
	}
	if envMaxConcurrent := os.Getenv(EnvPerformanceMaxConcurrentChecks); envMaxConcurrent != "" {
		if v, err := strconv.Atoi(envMaxConcurrent); err == nil {
			config.MaxConcurrentChecks = v
		}
	}
	if envCheckInterval := os.Getenv(EnvPerformanceCheckIntervalMinutes); envCheckInterval != "" {
		if v, err := strconv.Atoi(envCheckInterval); err == nil {
			config.CheckIntervalMinutes = v
		}
	}
	if envDisableAutoCheck := os.Getenv(EnvPerformanceDisableAutoCheck); envDisableAutoCheck != "" {
		config.DisableAutoCheck = strings.ToLower(envDisableAutoCheck) == "true"
	}

	return config, nil
}

// SavePerformanceConfig 保存性能配置
func SavePerformanceConfig(config *PerformanceConfig) error {
	boolToStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	// 验证范围
	maxConcurrent := config.MaxConcurrentChecks
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if maxConcurrent > 20 {
		maxConcurrent = 20
	}

	checkInterval := config.CheckIntervalMinutes
	if checkInterval < 0 {
		checkInterval = 0
	}
	if checkInterval > 1440 {
		checkInterval = 1440
	}

	return SetSettings(map[string]string{
		SettingPerformanceLowPowerMode:         boolToStr(config.LowPowerMode),
		SettingPerformanceMaxConcurrentChecks:  strconv.Itoa(maxConcurrent),
		SettingPerformanceCheckIntervalMinutes: strconv.Itoa(checkInterval),
		SettingPerformanceDisableAutoCheck:     boolToStr(config.DisableAutoCheck),
	})
}

// EnvOverride 环境变量覆盖状态
type EnvOverride struct {
	HasOverride bool   `json:"hasOverride"` // 是否有环境变量覆盖
	Message     string `json:"message"`     // 提示信息
}

// RegistryEnvOverride Registry 配置的环境变量覆盖状态
type RegistryEnvOverride struct {
	Enabled EnvOverride `json:"enabled"`
	Mirrors EnvOverride `json:"mirrors"`
}

// GetRegistryEnvOverride 获取 Registry 配置的环境变量覆盖状态
func GetRegistryEnvOverride() RegistryEnvOverride {
	result := RegistryEnvOverride{}
	if os.Getenv(EnvRegistryMirrorsEnabled) != "" {
		result.Enabled = EnvOverride{HasOverride: true, Message: "已通过环境变量 REGISTRY_MIRRORS_ENABLED 配置"}
	}
	if os.Getenv(EnvRegistryMirrors) != "" {
		result.Mirrors = EnvOverride{HasOverride: true, Message: "已通过环境变量 REGISTRY_MIRRORS 配置"}
	}
	return result
}

// HasRegistryEnvOverride 检查是否有任何 Registry 环境变量覆盖
func HasRegistryEnvOverride() bool {
	return os.Getenv(EnvRegistryMirrorsEnabled) != "" || os.Getenv(EnvRegistryMirrors) != ""
}

// ProxyEnvOverride Proxy 配置的环境变量覆盖状态
type ProxyEnvOverride struct {
	Enabled  EnvOverride `json:"enabled"`
	Type     EnvOverride `json:"type"`
	Host     EnvOverride `json:"host"`
	Port     EnvOverride `json:"port"`
	Username EnvOverride `json:"username"`
	Password EnvOverride `json:"password"`
}

// GetProxyEnvOverride 获取 Proxy 配置的环境变量覆盖状态
func GetProxyEnvOverride() ProxyEnvOverride {
	result := ProxyEnvOverride{}
	if os.Getenv(EnvProxyEnabled) != "" {
		result.Enabled = EnvOverride{HasOverride: true, Message: "已通过环境变量 PROXY_ENABLED 配置"}
	}
	if os.Getenv(EnvProxyType) != "" {
		result.Type = EnvOverride{HasOverride: true, Message: "已通过环境变量 PROXY_TYPE 配置"}
	}
	if os.Getenv(EnvProxyHost) != "" {
		result.Host = EnvOverride{HasOverride: true, Message: "已通过环境变量 PROXY_HOST 配置"}
	}
	if os.Getenv(EnvProxyPort) != "" {
		result.Port = EnvOverride{HasOverride: true, Message: "已通过环境变量 PROXY_PORT 配置"}
	}
	if os.Getenv(EnvProxyUser) != "" {
		result.Username = EnvOverride{HasOverride: true, Message: "已通过环境变量 PROXY_USER 配置"}
	}
	if os.Getenv(EnvProxyPass) != "" {
		result.Password = EnvOverride{HasOverride: true, Message: "已通过环境变量 PROXY_PASS 配置"}
	}
	return result
}

// HasProxyEnvOverride 检查是否有任何 Proxy 环境变量覆盖
func HasProxyEnvOverride() bool {
	return os.Getenv(EnvProxyEnabled) != "" ||
		os.Getenv(EnvProxyType) != "" ||
		os.Getenv(EnvProxyHost) != "" ||
		os.Getenv(EnvProxyPort) != "" ||
		os.Getenv(EnvProxyUser) != "" ||
		os.Getenv(EnvProxyPass) != ""
}

// PerformanceEnvOverride Performance 配置的环境变量覆盖状态
type PerformanceEnvOverride struct {
	LowPowerMode         EnvOverride `json:"lowPowerMode"`
	MaxConcurrentChecks  EnvOverride `json:"maxConcurrentChecks"`
	CheckIntervalMinutes EnvOverride `json:"checkIntervalMinutes"`
	DisableAutoCheck     EnvOverride `json:"disableAutoCheck"`
}

// GetPerformanceEnvOverride 获取 Performance 配置的环境变量覆盖状态
func GetPerformanceEnvOverride() PerformanceEnvOverride {
	result := PerformanceEnvOverride{}
	if os.Getenv(EnvPerformanceLowPowerMode) != "" {
		result.LowPowerMode = EnvOverride{HasOverride: true, Message: "已通过环境变量 LOW_POWER_MODE 配置"}
	}
	if os.Getenv(EnvPerformanceMaxConcurrentChecks) != "" {
		result.MaxConcurrentChecks = EnvOverride{HasOverride: true, Message: "已通过环境变量 MAX_CONCURRENT_CHECKS 配置"}
	}
	if os.Getenv(EnvPerformanceCheckIntervalMinutes) != "" {
		result.CheckIntervalMinutes = EnvOverride{HasOverride: true, Message: "已通过环境变量 CHECK_INTERVAL_MINUTES 配置"}
	}
	if os.Getenv(EnvPerformanceDisableAutoCheck) != "" {
		result.DisableAutoCheck = EnvOverride{HasOverride: true, Message: "已通过环境变量 DISABLE_AUTO_CHECK 配置"}
	}
	return result
}

// HasPerformanceEnvOverride 检查是否有任何 Performance 环境变量覆盖
func HasPerformanceEnvOverride() bool {
	return os.Getenv(EnvPerformanceLowPowerMode) != "" ||
		os.Getenv(EnvPerformanceMaxConcurrentChecks) != "" ||
		os.Getenv(EnvPerformanceCheckIntervalMinutes) != "" ||
		os.Getenv(EnvPerformanceDisableAutoCheck) != ""
}
