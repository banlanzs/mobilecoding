// mobilecoding CLI：claude/codex 子命令 — 启动 server + 本地 CLI，手机扫码作为遥控器。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// runClaude 处理 `mobilecoding claude [flags]` 子命令。
func runClaude(extraArgs []string) {
	fs := flag.NewFlagSet("claude", flag.ContinueOnError)
	settings := fs.String("settings", "", "Claude settings file path")
	model := fs.String("model", "", "Model override")
	port := fs.String("port", "8443", "Server port")
	resume := fs.String("resume", "", "Resume session ID")
	if err := fs.Parse(extraArgs); err != nil {
		os.Exit(1)
	}

	// 未显式指定 settings 时，优先探测项目级 .claude/settings.local.json。
	// 找到则用 --settings 显式传入（确保项目级 local 优先于全局 settings.json）；
	// 找不到则不传 --settings，由 claude 自行回退到 ~/.claude/settings.json。
	settingsSource := "全局默认"
	if *settings == "" {
		if detected := detectProjectSettings(); detected != "" {
			*settings = detected
			settingsSource = "项目级自动探测"
		}
	} else {
		settingsSource = "命令行指定"
	}

	var args []string
	if *settings != "" {
		args = append(args, "--settings", *settings)
	}
	if *model != "" {
		args = append(args, "--model", *model)
	}
	if *resume != "" {
		args = append(args, "--resume", *resume)
	}
	args = append(args, fs.Args()...)

	fmt.Fprintf(os.Stderr, "  配置文件: %s (%s)\n", displayClaudeSettings(*settings), settingsSource)
	fmt.Fprintf(os.Stderr, "  模型: %s\n", displayClaudeModel(*model))
	if *resume != "" {
		fmt.Fprintf(os.Stderr, "  恢复会话: %s\n", *resume)
	}

	session := &Session{
		Command:    "claude",
		Args:       args,
		Port:       *port,
		ServerAddr: "127.0.0.1:" + *port,
	}
	if err := runLocal(session); err != ExitLoop {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// detectProjectSettings 在当前工作目录下探测 .claude/settings.local.json。
// 返回找到的绝对路径；不存在则返回空字符串。
// 优先级：项目级 settings.local.json（本地覆盖、不入 git）。
func detectProjectSettings() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return detectProjectSettingsIn(cwd)
}

// detectProjectSettingsIn 在指定目录下探测 .claude/settings.local.json。
// 抽出 cwd 参数便于测试。
func detectProjectSettingsIn(dir string) string {
	candidate := filepath.Join(dir, ".claude", "settings.local.json")
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

func displayClaudeSettings(settings string) string {
	if settings == "" {
		return "默认配置"
	}
	return settings
}

func displayClaudeModel(model string) string {
	if model == "" {
		return "默认模型"
	}
	return model
}

// runCodex 处理 `mobilecoding codex [flags]` 子命令。
func runCodex(extraArgs []string) {
	runGeneric("codex", extraArgs)
}

func runGeneric(command string, extraArgs []string) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	port := fs.String("port", "8443", "Server port")
	if err := fs.Parse(extraArgs); err != nil {
		os.Exit(1)
	}

	session := &Session{
		Command:    command,
		Args:       fs.Args(),
		Port:       *port,
		ServerAddr: "127.0.0.1:" + *port,
	}
	if err := runLocal(session); err != ExitLoop {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// printMCUsage 打印 mc 模式子命令的用法。
func printMCUsage() {
	fmt.Fprintf(os.Stderr, "Usage: mobilecoding <command> [args...]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  claude [flags]   Start Claude Code + server (remote control mode)\n")
	fmt.Fprintf(os.Stderr, "  codex  [flags]   Start Codex + server\n")
	fmt.Fprintf(os.Stderr, "\nFlags (claude):\n")
	fmt.Fprintf(os.Stderr, "  -settings <path>   Claude settings file\n")
	fmt.Fprintf(os.Stderr, "  -model <model>     Model override\n")
	fmt.Fprintf(os.Stderr, "  -port <port>       Server port (default: 8443)\n")
}