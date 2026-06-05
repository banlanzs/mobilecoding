package main

import "flag"

type serverFlags struct {
	port        string
	ip          string
	workspace   string
	authToken   string
	mtls        string
	logLevel    string
	defaultCmd  string
	defaultArgs string
	showVersion bool
	showHelp    bool
}

func parseServerFlags(args []string) (serverFlags, error) {
	f := serverFlags{}
	fs := flag.NewFlagSet("mobilecoding", flag.ContinueOnError)
	fs.StringVar(&f.port, "port", "", "listen port (overrides MOBILECODING_PORT)")
	fs.StringVar(&f.ip, "ip", "", "local IP for cert & QR code (overrides MOBILECODING_IP)")
	fs.StringVar(&f.workspace, "workspace", "", "workspace root (overrides MOBILECODING_WORKSPACE)")
	fs.StringVar(&f.authToken, "auth-token", "", "auth token (overrides MOBILECODING_AUTH_TOKEN)")
	fs.StringVar(&f.mtls, "mtls", "", "mtls mode: none|optional|required")
	fs.StringVar(&f.logLevel, "log-level", "", "log level: debug|info|warn|error")
	fs.StringVar(&f.defaultCmd, "default-command", "", "default AI command (claude|codex|opencode|aichat)")
	fs.StringVar(&f.defaultArgs, "default-args", "", "default args for AI command (space-separated, quoted)")
	fs.BoolVar(&f.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&f.showHelp, "help", false, "print help and exit")

	if err := fs.Parse(args); err != nil {
		return f, err
	}
	return f, nil
}
