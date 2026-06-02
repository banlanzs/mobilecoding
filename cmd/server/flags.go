package main

import "flag"

type serverFlags struct {
	port        string
	workspace   string
	authToken   string
	mtls        string
	logLevel    string
	defaultCmd  string
	showVersion bool
	showHelp    bool
}

func parseServerFlags(args []string) (serverFlags, error) {
	f := serverFlags{}
	fs := flag.NewFlagSet("mytool", flag.ContinueOnError)
	fs.StringVar(&f.port, "port", "", "listen port (overrides MYTOOL_PORT)")
	fs.StringVar(&f.workspace, "workspace", "", "workspace root (overrides MYTOOL_WORKSPACE)")
	fs.StringVar(&f.authToken, "auth-token", "", "auth token (overrides MYTOOL_AUTH_TOKEN)")
	fs.StringVar(&f.mtls, "mtls", "", "mtls mode: none|optional|required")
	fs.StringVar(&f.logLevel, "log-level", "", "log level: debug|info|warn|error")
	fs.StringVar(&f.defaultCmd, "default-command", "", "default AI command")
	fs.BoolVar(&f.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&f.showHelp, "help", false, "print help and exit")

	if err := fs.Parse(args); err != nil {
		return f, err
	}
	return f, nil
}
