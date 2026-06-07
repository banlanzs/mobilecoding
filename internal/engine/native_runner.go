package engine

import (
	"log"
	"runtime"
)

// NewNativeRunner 返回原生命令桥接 runner。
// 与 NewRunner("claude") 不同，这里刻意绕过 ClaudeRunner 的 stream-json 托管模式，
// 用 PTY/Pipe 直接桥接本地 CLI 的 stdin/stdout，供 mc remote-control 模式使用。
func NewNativeRunner(command string) Runner {
	if runtime.GOOS == "windows" {
		if cfg, ok := agentRegistry[command]; ok && cfg.Runner == "claude" {
			log.Printf("engine: selected managed ClaudeRunner (Windows remote-control) for command=%s", command)
			return NewClaudeRunner()
		}
		log.Printf("engine: selected native PipeRunner (Windows) for command=%s", command)
		return NewPipeRunner()
	}
	log.Printf("engine: selected native PtyRunner for command=%s", command)
	return NewPtyRunner()
}
