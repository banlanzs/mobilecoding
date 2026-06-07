package engine

import (
	"log"
	"runtime"
)

// NewNativeRunner 返回原生命令桥接 runner。
// remote-control 必须是真实交互式 CLI：Windows 使用 PipeRunner，非 Windows 使用 PtyRunner。
// 这里刻意绕过 ClaudeRunner 的 --print/stream-json 托管模式，因为它不支持交互式 /model 热切换。
func NewNativeRunner(command string) Runner {
	if runtime.GOOS == "windows" {
		log.Printf("engine: selected native PipeRunner (Windows remote-control) for command=%s", command)
		return NewPipeRunner()
	}
	log.Printf("engine: selected native PtyRunner for command=%s", command)
	return NewPtyRunner()
}
