package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config 汇总运行时所有可配置项。字段语义见 spec §8.5。
type Config struct {
	Port          string
	AuthToken     string
	Workspace     string
	MTLS          string // none | optional | required
	LogLevel      string
	DefaultCmd    string
	DefaultArgs   []string
	AuthDir       string
	StoreDir      string
	WatchdogWarn1 string
	WatchdogWarn2 string
	WatchdogAbort string
}

// WithDefaults 返回 c，未设置的字段填入 spec §8.5 里的默认值。
func (c Config) WithDefaults() Config {
	if c.Port == "" {
		c.Port = "8443"
	}
	if c.MTLS == "" {
		c.MTLS = "optional"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.DefaultCmd == "" {
		c.DefaultCmd = "claude"
	}
	if c.Workspace == "" {
		home, _ := os.UserHomeDir()
		c.Workspace = filepath.Join(home, "mytool-workspace")
	}
	if c.AuthDir == "" {
		home, _ := os.UserHomeDir()
		c.AuthDir = filepath.Join(home, ".mytool", "auth")
	}
	if c.StoreDir == "" {
		home, _ := os.UserHomeDir()
		c.StoreDir = filepath.Join(home, ".mytool", "store")
	}
	if c.WatchdogWarn1 == "" {
		c.WatchdogWarn1 = "60s"
	}
	if c.WatchdogWarn2 == "" {
		c.WatchdogWarn2 = "90s"
	}
	if c.WatchdogAbort == "" {
		c.WatchdogAbort = "120s"
	}
	return c
}

// Validate 检查必填项。
func (c Config) Validate() error {
	if c.Port == "" {
		return errors.New("port is required")
	}
	if c.AuthToken == "" {
		return errors.New("auth token is required")
	}
	if c.Workspace == "" {
		return errors.New("workspace is required")
	}
	if c.MTLS != "" && c.MTLS != "optional" && c.MTLS != "required" {
		return fmt.Errorf("mtls must be one of optional|required, got %q", c.MTLS)
	}
	return nil
}
