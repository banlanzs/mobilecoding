// mobilecoding CLI：Local/Remote 模式切换包装器。
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
		runClaudeLoop(os.Args[2:])
	case "codex":
		runGenericLoop("codex", os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: mc <command> [args...]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  claude [flags]   Start Claude Code with local/remote switching\n")
	fmt.Fprintf(os.Stderr, "  codex  [flags]   Start Codex with local/remote switching\n")
	fmt.Fprintf(os.Stderr, "\nFlags (claude):\n")
	fmt.Fprintf(os.Stderr, "  -settings <path>   Claude settings file\n")
	fmt.Fprintf(os.Stderr, "  -model <model>     Model override\n")
	fmt.Fprintf(os.Stderr, "  -port <port>       Server port (default: 8443)\n")
}

func runClaudeLoop(extraArgs []string) {
	fs := flag.NewFlagSet("claude", flag.ContinueOnError)
	settings := fs.String("settings", "", "Claude settings file path")
	model := fs.String("model", "", "Model override")
	port := fs.String("port", "8443", "Server port")
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
	// 传递剩余非 flag 参数
	args = append(args, fs.Args()...)

	session := NewSession("claude", args, "127.0.0.1:"+*port, "")
	session.Port = *port
	if err := Run(session, ModeLocal); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runGenericLoop(command string, extraArgs []string) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	port := fs.String("port", "8443", "Server port")
	if err := fs.Parse(extraArgs); err != nil {
		os.Exit(1)
	}

	session := NewSession(command, fs.Args(), "127.0.0.1:"+*port, "")
	if err := Run(session, ModeLocal); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
