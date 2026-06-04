package logx

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OpenLogFile 打开日滚动日志文件（权限 0o644）。
func OpenLogFile(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	name := "mobilecoding-" + time.Now().Format("2006-01-02") + ".log"
	return os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}

// CleanOldLogs 删除超过 maxDays 天的日志文件。
func CleanOldLogs(dir string, maxDays int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -maxDays)
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "mobilecoding-") {
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				_ = os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}
	return nil
}

// NewWithMultiWriter 创建写入多个 io.Writer 的 logger。
func NewWithMultiWriter(writers ...io.Writer) *Logger {
	return NewWithWriter(io.MultiWriter(writers...))
}
