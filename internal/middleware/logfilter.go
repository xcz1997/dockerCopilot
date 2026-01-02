package middleware

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UnifiedLogWriter 统一的日志 Writer
// 所有日志都经过统一过滤，同时输出到控制台和文件
type UnifiedLogWriter struct {
	console io.Writer  // 控制台输出
	logDir  string     // 日志目录
	logFile *os.File   // 当前日志文件
	curDate string     // 当前日期（用于日志轮转）
	mu      sync.Mutex // 保护文件写入
}

// NewUnifiedLogWriter 创建统一的日志 Writer
func NewUnifiedLogWriter(logDir string, console io.Writer) *UnifiedLogWriter {
	w := &UnifiedLogWriter{
		console: console,
		logDir:  logDir,
	}
	// 初始化日志文件
	w.rotateIfNeeded()
	return w
}

// Write 实现 io.Writer 接口
func (w *UnifiedLogWriter) Write(p []byte) (n int, err error) {
	line := string(p)

	// 统一过滤逻辑
	if shouldFilterLog(line) {
		// 过滤掉，假装写入成功
		return len(p), nil
	}

	// 写入控制台
	if w.console != nil {
		w.console.Write(p)
	}

	// 写入文件
	w.mu.Lock()
	defer w.mu.Unlock()

	w.rotateIfNeeded()
	if w.logFile != nil {
		w.logFile.Write(p)
	}

	return len(p), nil
}

// rotateIfNeeded 检查是否需要轮转日志文件
func (w *UnifiedLogWriter) rotateIfNeeded() {
	today := time.Now().Format("2006-01-02")
	if w.curDate == today && w.logFile != nil {
		return
	}

	// 关闭旧文件
	if w.logFile != nil {
		w.logFile.Close()
	}

	// 创建新文件
	logPath := filepath.Join(w.logDir, fmt.Sprintf("dockercopilot-%s.log", today))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// 文件创建失败，只输出到控制台
		w.logFile = nil
		return
	}

	w.logFile = file
	w.curDate = today

	// 清理旧日志（保留7天）
	go w.cleanOldLogs(7)
}

// cleanOldLogs 清理超过指定天数的日志文件
func (w *UnifiedLogWriter) cleanOldLogs(keepDays int) {
	cutoff := time.Now().AddDate(0, 0, -keepDays)

	files, err := os.ReadDir(w.logDir)
	if err != nil {
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasPrefix(file.Name(), "dockercopilot-") || !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(w.logDir, file.Name()))
		}
	}
}

// Close 关闭日志文件
func (w *UnifiedLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.logFile != nil {
		return w.logFile.Close()
	}
	return nil
}

// shouldFilterLog 判断日志是否应该被过滤
// 白名单模式：只保留自定义日志和 error 日志，过滤框架自动生成的 HTTP 请求日志
func shouldFilterLog(line string) bool {
	// 1. error 级别日志始终保留
	if strings.Contains(line, `"level":"error"`) {
		return false
	}

	// 2. 过滤 go-zero 框架自动生成的 HTTP 请求日志
	//    这些日志包含 [HTTP] 标记
	if strings.Contains(line, "[HTTP]") {
		return true
	}

	// 3. 其他日志（自定义的 logx.Info/Debug/Warn 等）正常输出
	return false
}

// FilteredWriter 保留向后兼容（如果其他地方用到）
type FilteredWriter struct {
	underlying io.Writer
}

// NewFilteredWriter 创建带过滤功能的 Writer
func NewFilteredWriter(w io.Writer) *FilteredWriter {
	return &FilteredWriter{underlying: w}
}

// Write 实现 io.Writer 接口
func (fw *FilteredWriter) Write(p []byte) (n int, err error) {
	line := string(p)
	if shouldFilterLog(line) {
		return len(p), nil
	}
	return fw.underlying.Write(p)
}
