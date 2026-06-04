package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SplitArgs 简单解析 shell 风格的参数字符串，支持单/双引号。
func SplitArgs(raw string) []string {
	var out []string
	var cur []byte
	var quote byte
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case quote != 0 && c == quote:
			quote = 0
		case quote == 0 && (c == '\'' || c == '"'):
			quote = c
		case quote == 0 && c <= ' ':
			if len(cur) > 0 {
				out = append(out, string(cur))
				cur = nil
			}
		default:
			cur = append(cur, c)
		}
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	return out
}

// Config 汇总运行时所有可配置项。字段语义见 spec §8.5。
type Config struct {
	Port          string
	AuthToken     string
	Workspace     string
	MTLS          string // none | optional | required
	LogLevel      string
	DefaultCmd    string
	DefaultArgs   []string
	Models        string // 逗号分隔的模型列表: label1:value1,label2:value2
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
		c.Workspace = filepath.Join(home, "mobilecoding-workspace")
	}
	if c.AuthDir == "" {
		home, _ := os.UserHomeDir()
		c.AuthDir = filepath.Join(home, ".mobilecoding", "auth")
	}
	if c.StoreDir == "" {
		home, _ := os.UserHomeDir()
		c.StoreDir = filepath.Join(home, ".mobilecoding", "store")
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

// Load 从环境变量重新加载配置，返回带默认值的 Config。
func Load() (Config, error) {
	env := FromEnv()
	c := Config{
		Port:        env.Port,
		AuthToken:   env.AuthToken,
		Workspace:   env.Workspace,
		MTLS:        env.MTLS,
		LogLevel:    env.LogLevel,
		DefaultCmd:  env.DefaultCmd,
		DefaultArgs: SplitArgs(os.ExpandEnv(env.DefaultArgs)),
		Models:      env.Models,
	}.WithDefaults()
	return c, nil
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
