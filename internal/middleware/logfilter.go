package middleware

import (
	"io"
	"strings"
)

// FilteredWriter 包装 io.Writer，过滤 go-zero 框架的 HTTP 请求日志
// 只保留：
// 1. 自定义的 logx 日志（不包含 [HTTP] 标记）
// 2. error 级别日志
type FilteredWriter struct {
	underlying io.Writer
}

// NewFilteredWriter 创建带过滤功能的 Writer
func NewFilteredWriter(w io.Writer) *FilteredWriter {
	return &FilteredWriter{underlying: w}
}

// Write 实现 io.Writer 接口，使用白名单模式过滤日志
func (fw *FilteredWriter) Write(p []byte) (n int, err error) {
	line := string(p)

	// 白名单模式：只保留以下日志
	// 1. error 级别日志始终保留
	if strings.Contains(line, `"level":"error"`) {
		return fw.underlying.Write(p)
	}

	// 2. 过滤 go-zero 框架自动生成的 HTTP 请求日志
	//    这些日志包含 [HTTP] 标记，格式如：
	//    {"content":"[HTTP] 200 - GET /api/tasks?status=current - ...","level":"info"}
	if strings.Contains(line, "[HTTP]") {
		// 过滤掉 HTTP 请求日志，假装写入成功
		return len(p), nil
	}

	// 3. 其他日志（自定义的 logx.Info/Debug/Warn 等）正常输出
	return fw.underlying.Write(p)
}
