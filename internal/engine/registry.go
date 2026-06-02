package engine

import "errors"

// NewRunner 根据 command 返回合适的 Runner 实现。
func NewRunner(command string, _ ExecRequest) (Runner, error) {
	if command == "" {
		return nil, errors.New("engine: command is required")
	}
	switch {
	case command == "claude" || command == "claude-code":
		return NewClaudeRunner(), nil
	case command == "codex":
		return NewCodexRunner(), nil
	default:
		return NewPtyRunner(), nil
	}
}
