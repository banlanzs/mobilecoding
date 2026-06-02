package config

import (
	"os"
)

// EnvOverrides 记录从环境变量派生的覆盖项。
type EnvOverrides struct {
	Port       string
	AuthToken  string
	Workspace  string
	MTLS       string
	LogLevel   string
	DefaultCmd string
}

// FromEnv 从环境变量读取覆盖项。空值表示未设置。
func FromEnv() EnvOverrides {
	return EnvOverrides{
		Port:       os.Getenv("MYTOOL_PORT"),
		AuthToken:  os.Getenv("MYTOOL_AUTH_TOKEN"),
		Workspace:  os.Getenv("MYTOOL_WORKSPACE"),
		MTLS:       os.Getenv("MYTOOL_MTLS"),
		LogLevel:   os.Getenv("MYTOOL_LOG_LEVEL"),
		DefaultCmd: os.Getenv("MYTOOL_DEFAULT_COMMAND"),
	}
}
