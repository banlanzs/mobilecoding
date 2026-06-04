package config

import "os"

// EnvOverrides 记录从环境变量派生的覆盖项。
type EnvOverrides struct {
	Port        string
	AuthToken   string
	Workspace   string
	MTLS        string
	LogLevel    string
	DefaultCmd  string
	DefaultArgs string
	Models      string // 逗号分隔的模型列表: label1:value1,label2:value2
}

// FromEnv 从环境变量读取覆盖项。空值表示未设置。
// 优先使用 MOBILECODING_* 前缀，兼容旧的 MYTOOL_* 前缀。
func FromEnv() EnvOverrides {
	return EnvOverrides{
		Port:        firstEnv("MOBILECODING_PORT", "MYTOOL_PORT"),
		AuthToken:   firstEnv("MOBILECODING_AUTH_TOKEN", "MYTOOL_AUTH_TOKEN"),
		Workspace:   firstEnv("MOBILECODING_WORKSPACE", "MYTOOL_WORKSPACE"),
		MTLS:        firstEnv("MOBILECODING_MTLS", "MYTOOL_MTLS"),
		LogLevel:    firstEnv("MOBILECODING_LOG_LEVEL", "MYTOOL_LOG_LEVEL"),
		DefaultCmd:  firstEnv("MOBILECODING_DEFAULT_COMMAND", "MYTOOL_DEFAULT_COMMAND"),
		DefaultArgs: firstEnv("MOBILECODING_DEFAULT_ARGS", "MYTOOL_DEFAULT_ARGS"),
		Models:      firstEnv("MOBILECODING_MODELS", "MYTOOL_MODELS"),
	}
}

func firstEnv(names ...string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}
