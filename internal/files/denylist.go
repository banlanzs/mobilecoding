package files

import (
	"path/filepath"
	"strings"
)

var deniedPatterns = []string{
	".env",
	"*.pem",
	"*.key",
	"*.p12",
	"*.crt",
	".git",
	"node_modules",
}

// IsDenied 检查路径是否匹配 denylist。
func IsDenied(path string) bool {
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	for _, pat := range deniedPatterns {
		if matched, _ := filepath.Match(pat, base); matched {
			return true
		}
	}
	if strings.Contains(clean, ".git") || strings.Contains(clean, "node_modules") {
		return true
	}
	return false
}
