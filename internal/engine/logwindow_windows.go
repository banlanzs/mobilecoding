// +build windows

package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// LogWindow 在 Windows 上创建一个实时日志窗口
type LogWindow struct {
	logFile *os.File
	logPath string
}

// NewLogWindow 创建一个新的日志窗口
// 它会：
// 1. 创建一个临时日志文件
// 2. 启动一个 PowerShell 窗口实时 tail 这个文件
func NewLogWindow(sessionID string) (*LogWindow, error) {
	// 创建临时日志文件
	tempDir := os.TempDir()
	logPath := filepath.Join(tempDir, fmt.Sprintf("mobilecoding-%s.log", sessionID))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	// 启动 PowerShell 窗口实时显示日志
	// Get-Content -Path <file> -Wait 相当于 tail -f
	psScript := fmt.Sprintf(
		`Get-Content -Path '%s' -Wait`,
		logPath,
	)

	cmd := exec.Command("powershell.exe", "-NoExit", "-Command", psScript)
	if err := cmd.Start(); err != nil {
		logFile.Close()
		os.Remove(logPath)
		return nil, fmt.Errorf("start powershell window: %w", err)
	}

	// 写入欢迎消息
	fmt.Fprintf(logFile, "=== MobileCoding Session: %s ===\n", sessionID)
	fmt.Fprintf(logFile, "Started at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(logFile, "==========================================\n\n")
	logFile.Sync()

	return &LogWindow{
		logFile: logFile,
		logPath: logPath,
	}, nil
}

// Write 写入日志内容（同时会显示在窗口中）
func (w *LogWindow) Write(p []byte) (n int, err error) {
	n, err = w.logFile.Write(p)
	w.logFile.Sync() // 立即刷新，确保窗口实时更新
	return
}

// Close 关闭日志窗口（但保留日志文件供查看）
func (w *LogWindow) Close() error {
	if w.logFile != nil {
		fmt.Fprintf(w.logFile, "\n=== Session Ended at %s ===\n", time.Now().Format("2006-01-02 15:04:05"))
		w.logFile.Sync()
		return w.logFile.Close()
	}
	return nil
}

// LogPath 返回日志文件路径
func (w *LogWindow) LogPath() string {
	return w.logPath
}
