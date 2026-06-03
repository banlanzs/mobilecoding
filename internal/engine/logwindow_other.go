//go:build !windows

package engine

// LogWindow 非 Windows 平台的占位符
type LogWindow struct{}

// NewLogWindow 创建日志窗口（非 Windows 平台返回 nil）
func NewLogWindow(sessionID string) (*LogWindow, error) {
	return nil, nil
}

// Write 实现 io.Writer
func (w *LogWindow) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// Close 关闭日志窗口
func (w *LogWindow) Close() error {
	return nil
}

// LogPath 返回日志文件路径
func (w *LogWindow) LogPath() string {
	return ""
}
