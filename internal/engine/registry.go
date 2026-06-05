package engine

import (
	_ "embed"
	"encoding/json"
	"errors"
	"log"
	"runtime"
)

//go:embed agents.json
var agentsJSON []byte

// AgentConfig 声明式 Agent 配置。
type AgentConfig struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	Aliases     []string `json:"aliases"`
	Runner      string   `json:"runner"`
	Description string   `json:"description"`
}

// agentRegistry 从 JSON 加载的 Agent 配置表。
var agentRegistry map[string]AgentConfig

func init() {
	var cfg struct {
		Agents []AgentConfig `json:"agents"`
	}
	if err := json.Unmarshal(agentsJSON, &cfg); err != nil {
		log.Printf("engine: failed to load agents.json: %v", err)
		return
	}
	agentRegistry = make(map[string]AgentConfig)
	for _, a := range cfg.Agents {
		agentRegistry[a.Command] = a
		for _, alias := range a.Aliases {
			agentRegistry[alias] = a
		}
	}
	log.Printf("engine: loaded %d agent configs", len(cfg.Agents))
}

// ListAgents 返回所有可用 Agent 配置。
func ListAgents() []AgentConfig {
	var list []AgentConfig
	seen := map[string]bool{}
	for _, a := range agentRegistry {
		if !seen[a.ID] {
			list = append(list, a)
			seen[a.ID] = true
		}
	}
	return list
}

// NewRunner 根据 command 返回合适的 Runner 实现。
func NewRunner(command string, _ ExecRequest) (Runner, error) {
	if command == "" {
		return nil, errors.New("engine: command is required")
	}

	// 从声明式配置查找
	if cfg, ok := agentRegistry[command]; ok {
		return newRunnerByType(cfg.Runner, command)
	}

	// 回退到 generic
	return newRunnerByType("generic", command)
}

func newRunnerByType(runnerType, command string) (Runner, error) {
	switch runnerType {
	case "claude":
		log.Printf("engine: selected ClaudeRunner for command=%s", command)
		return NewClaudeRunner(), nil
	case "codex":
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
