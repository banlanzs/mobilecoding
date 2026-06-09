package engine

import (
	"log"
	"runtime"
)

// NewNativeRunner 返回 remote-control runner。
// Windows 上 Claude Code 交互式 CLI 需要真实终端，普通 stdin/stdout pipe 会启动后退出；
// 因此 Windows + Claude 继续使用 ClaudeRunner 的托管 stream-json 模式。
func NewNativeRunner(command string) Runner {
	if runtime.GOOS == "windows" {
		if cfg, ok := agentRegistry[command]; ok && cfg.Runner == "claude" {
			log.Printf("engine: selected managed ClaudeRunner (Windows remote-control) for command=%s", command)
			return NewClaudeRunner()
		}
		log.Printf("engine: selected native PipeRunner (Windows remote-control) for command=%s", command)
		return NewPipeRunner()
	}
	log.Printf("engine: selected native PtyRunner for command=%s", command)
	return NewPtyRunner()
}
