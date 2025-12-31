package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Auth struct { // JWT 认证需要的密钥和过期时间配置
		AccessSecret string
		AccessExpire int64
	}
	Performance PerformanceConfig `json:",optional"` // 性能配置
}

// PerformanceConfig 性能相关配置
// 用于低性能设备优化，减少资源占用
type PerformanceConfig struct {
	// LowPowerMode 低性能模式
	// 启用后会禁用所有并发操作，改为顺序执行
	// 适用于 NAS、树莓派等低性能设备
	LowPowerMode bool `json:",default=false"`

	// MaxConcurrentChecks 镜像检查最大并发数
	// 默认 10，低性能模式下自动设为 1
	// 范围: 1-20
	MaxConcurrentChecks int `json:",default=10,range=[1:20]"`

	// CheckIntervalMinutes 镜像自动检查间隔（分钟）
	// 默认 30 分钟，设为 0 禁用自动检查
	// 范围: 0-1440 (最大24小时)
	CheckIntervalMinutes int `json:",default=30,range=[0:1440]"`

	// DisableAutoCheck 禁用启动时自动检查镜像更新
	// 默认 false，启用后启动时不检查更新
	DisableAutoCheck bool `json:",default=false"`
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

var (
	Version   string
	BuildDate string
)
