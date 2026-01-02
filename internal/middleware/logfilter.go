package middleware

import (
	"io"
	"strings"
	"sync"
)

// 需要过滤的日志路径模式（包含这些字符串的日志行会被过滤）
var logFilterPatterns = []string{
	"/api/environment/",          // 环境刷新等高频请求
	"/api/container/",            // 容器日志等高频请求
	"/api/environment/self/logs", // 自身日志 SSE
}

// logFilterPatternsMu 保护 logFilterPatterns 的读写
var logFilterPatternsMu sync.RWMutex

// FilteredWriter 包装 io.Writer，过滤包含特定模式的日志
type FilteredWriter struct {
	underlying io.Writer
}

// NewFilteredWriter 创建带过滤功能的 Writer
func NewFilteredWriter(w io.Writer) *FilteredWriter {
	return &FilteredWriter{underlying: w}
}

// Write 实现 io.Writer 接口，过滤匹配模式的日志
func (fw *FilteredWriter) Write(p []byte) (n int, err error) {
	line := string(p)

	logFilterPatternsMu.RLock()
	patterns := logFilterPatterns
	logFilterPatternsMu.RUnlock()

	// 检查是否匹配任意过滤模式
	for _, pattern := range patterns {
		if strings.Contains(line, pattern) {
			// 匹配到过滤模式，假装写入成功但实际不输出
			return len(p), nil
		}
	}

	// 不匹配，正常写入
	return fw.underlying.Write(p)
}

// AddLogFilterPattern 添加日志过滤模式
func AddLogFilterPattern(pattern string) {
	logFilterPatternsMu.Lock()
	defer logFilterPatternsMu.Unlock()
	logFilterPatterns = append(logFilterPatterns, pattern)
}

// SetLogFilterPatterns 设置日志过滤模式列表
func SetLogFilterPatterns(patterns []string) {
	logFilterPatternsMu.Lock()
	defer logFilterPatternsMu.Unlock()
	logFilterPatterns = patterns
}

// GetLogFilterPatterns 获取当前日志过滤模式列表
func GetLogFilterPatterns() []string {
	logFilterPatternsMu.RLock()
	defer logFilterPatternsMu.RUnlock()
	result := make([]string, len(logFilterPatterns))
	copy(result, logFilterPatterns)
	return result
}
