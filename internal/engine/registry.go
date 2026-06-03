package engine

import (
	"errors"
	"log"
	"runtime"
)

// NewRunner 根据 command 返回合适的 Runner 实现。
func NewRunner(command string, _ ExecRequest) (Runner, error) {
	if command == "" {
		return nil, errors.New("engine: command is required")
	}
	switch {
	case command == "claude" || command == "claude-code":
		log.Printf("engine: selected ClaudeRunner for command=%s", command)
		return NewClaudeRunner(), nil
	case command == "codex":
		log.Printf("engine: selected CodexRunner for command=%s", command)
		return NewCodexRunner(), nil
	default:
		if runtime.GOOS == "windows" {
			log.Printf("engine: selected PipeRunner (Windows) for command=%s", command)
			return NewPipeRunner(), nil
		}
		log.Printf("engine: selected PtyRunner for command=%s", command)
		return NewPtyRunner(), nil
	}
}
