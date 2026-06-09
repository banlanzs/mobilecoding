// mobilecoding CLI：启动 server + 本地 Claude，手机扫码作为遥控器共存。
// 用法：mc claude [--settings xxx.json]
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "claude":
		runClaude(os.Args[2:])
	case "codex":
		runGeneric("codex", os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: mc <command> [args...]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  claude [flags]   Start Claude Code + server (remote control mode)\n")
	fmt.Fprintf(os.Stderr, "  codex  [flags]   Start Codex + server\n")
	fmt.Fprintf(os.Stderr, "\nFlags (claude):\n")
	fmt.Fprintf(os.Stderr, "  -settings <path>   Claude settings file\n")
	fmt.Fprintf(os.Stderr, "  -model <model>     Model override\n")
	fmt.Fprintf(os.Stderr, "  -port <port>       Server port (default: 8443)\n")
}

func runClaude(extraArgs []string) {
	fs := flag.NewFlagSet("claude", flag.ContinueOnError)
	settings := fs.String("settings", "", "Claude settings file path")
	model := fs.String("model", "", "Model override")
	port := fs.String("port", "8443", "Server port")
	resume := fs.String("resume", "", "Resume session ID")
	if err := fs.Parse(extraArgs); err != nil {
		os.Exit(1)
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

	fmt.Fprintf(os.Stderr, "  配置文件: %s\n", displayClaudeSettings(*settings))
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
