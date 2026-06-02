# mytool MVP 1 阶段 A：项目骨架 + logx + config + auth

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建 mytool 项目的 Go 骨架，建立日志、配置、鉴权三个基础设施层，为后续 engine / ws / gateway 等模块铺路。

**Architecture:** 单一 Go module（`github.com/<your-scope>/mytool`），按 spec §2 拆分 `internal/{logx,config,auth}` 三个包。每个包单一职责、对外暴露 interface、用 TDD 推进。先建立 `cmd/server` 之前的所有底层依赖。

**Tech Stack:**
- Go 1.22+（matches MobileVC 项目版本约束）
- 标准库 `testing`（单元测试）
- 标准库 `crypto/rand` + `crypto/sha256` + `crypto/subtle`（token）
- 标准库 `encoding/json` + `os` + `path/filepath`（配置持久化）

**Spec Reference:** `docs/superpowers/specs/2026-06-01-mytool-design.md` §2 (config / auth / logx), §5 (鉴权), §10 (隐私与安全)

**前置：** 仓库是 `D:/Documents/Dev-Repo/MobileVC`（当前 working dir），新项目 `mytool/` 将作为同级子目录建在仓库内。**实际做法：** 本阶段不直接建子目录，而是把代码写到 `mytool/` 目录下，spec 与 plan 仍在 `docs/superpowers/`。

---

## Task 1: 项目骨架

**Files:**
- Create: `mytool/go.mod`
- Create: `mytool/.gitignore`
- Create: `mytool/README.md`
- Create: `mytool/cmd/server/.gitkeep`
- Create: `mytool/internal/.gitkeep`
- Create: `mytool/scripts/.gitkeep`
- Create: `mytool/web/.gitkeep`

- [ ] **Step 1: 创建目录结构**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
mkdir -p mytool/{cmd/server,internal,scripts,web}
touch mytool/cmd/server/.gitkeep mytool/internal/.gitkeep mytool/scripts/.gitkeep mytool/web/.gitkeep
```

- [ ] **Step 2: 写 go.mod**

文件：`mytool/go.mod`

```go
module github.com/yourname/mytool

go 1.22
```

> 实际使用时把 `yourname` 替换成自己的 GitHub 用户名。

- [ ] **Step 3: 写 .gitignore**

文件：`mytool/.gitignore`

```gitignore
# 凭据（绝不入库）
.env
*.pem
*.key
*.p12
*.crt

# 构建产物
bin/
dist/
mytool-*

# IDE
.idea/
.vscode/

# OS
.DS_Store
Thumbs.db

# 测试
*.test
*.out
coverage.out
```

- [ ] **Step 4: 写 README.md（最小）**

文件：`mytool/README.md`

```markdown
# mytool

个人 AI CLI 远程控制台：把本机 Claude Code / Codex / 任意 LLM CLI 的"等待态"做成手机可操作的结构化卡片。

设计 spec：[`docs/superpowers/specs/2026-06-01-mytool-design.md`](../docs/superpowers/specs/2026-06-01-mytool-design.md)

## 当前状态

MVP 1 阶段 A：项目骨架 + logx + config + auth。

## 快速开始

```bash
cd mytool
go test ./...
```
```

- [ ] **Step 5: 验证 Go module 正确**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go version && go mod tidy
```

预期：Go 版本 ≥ 1.22，无错误（`go mod tidy` 在空 module 上是 no-op）。

- [ ] **Step 6: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/go.mod mytool/.gitignore mytool/README.md mytool/cmd/server/.gitkeep mytool/internal/.gitkeep mytool/scripts/.gitkeep mytool/web/.gitkeep
git commit -m "chore: 初始化 mytool Go module 与目录骨架"
```

---

## Task 2: internal/logx（结构化日志 + 脱敏）

**Files:**
- Create: `mytool/internal/logx/log.go`
- Create: `mytool/internal/logx/redact.go`
- Test: `mytool/internal/logx/redact_test.go`

- [ ] **Step 1: 写 redact 单元测试**

文件：`mytool/internal/logx/redact_test.go`

```go
package logx

import "testing"

func TestRedactAuthorizationBearer(t *testing.T) {
	in := "Authorization: Bearer abc.def.ghi"
	out := Redact(in)
	want := "Authorization: Bearer <redacted>"
	if out != want {
		t.Errorf("Redact() = %q, want %q", out, want)
	}
}

func TestRedactTokenKeyValue(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"api_key=sk-live-12345", "api_key=<redacted>"},
		{"token: my-secret-token", "token=<redacted>"},
		{"PASSWORD=hunter2", "PASSWORD=<redacted>"},
		{"auth_token: t=abc", "auth_token=<redacted>"},
	}
	for _, c := range cases {
		got := Redact(c.in)
		if got != c.want {
			t.Errorf("Redact(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRedactNoMatch(t *testing.T) {
	in := "session started, no secrets here"
	out := Redact(in)
	if out != in {
		t.Errorf("Redact() should leave non-secret text untouched, got %q", out)
	}
}

func TestRedactEmpty(t *testing.T) {
	if got := Redact(""); got != "" {
		t.Errorf("Redact(\"\") = %q, want empty", got)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/logx/...
```

预期：FAIL，`undefined: Redact`。

- [ ] **Step 3: 实现 redact.go**

文件：`mytool/internal/logx/redact.go`

```go
package logx

import "regexp"

// 凭据脱敏：把 Authorization Bearer、api_key/token/password/secret/auth_token
// 后面的值替换为 <redacted>。空字符串直接返回。
var redactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)[^\s"'\\]+`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|token|password|secret|auth[_-]?token)\s*[:=]\s*)[^\s"'\\]+`),
	regexp.MustCompile(`(?i)((?:--(?:api-key|token|password|secret|auth-token))(?:=|\s+))[^\s"'\\]+`),
}

// Redact 返回 s 脱敏后的副本。
func Redact(s string) string {
	if s == "" {
		return ""
	}
	out := s
	for _, re := range redactPatterns {
		out = re.ReplaceAllString(out, `${1}<redacted>`)
	}
	return out
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/logx/... -v -run TestRedact
```

预期：4 个测试全部 PASS。

- [ ] **Step 5: 实现 log.go（结构化日志器）**

文件：`mytool/internal/logx/log.go`

```go
package logx

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level 表示日志级别。
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String 返回 level 的小写字符串。
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	}
	return "unknown"
}

// Logger 是极简结构化日志器。所有消息输出前会经过 Redact 脱敏。
type Logger struct {
	mu    sync.Mutex
	w     io.Writer
	level Level
}

// New 返回写入 stderr 的 Logger。
func New() *Logger { return NewWithWriter(os.Stderr) }

// NewWithWriter 返回写入 w 的 Logger，便于测试。
func NewWithWriter(w io.Writer) *Logger {
	return &Logger{w: w, level: LevelInfo}
}

// SetLevel 设置最低输出级别。
func (l *Logger) SetLevel(lv Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = lv
}

// logf 内部实现：组件名 + 消息（脱敏）+ 可选 kv。
func (l *Logger) logf(lv Level, component, format string, args ...any) {
	if lv < l.level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = Redact(msg)
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	line := fmt.Sprintf("%s %s %s %s\n", ts, lv.String(), component, msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = io.WriteString(l.w, line)
}

// Info 输出 info 级日志。
func (l *Logger) Info(component, format string, args ...any) { l.logf(LevelInfo, component, format, args...) }

// Warn 输出 warn 级日志。
func (l *Logger) Warn(component, format string, args ...any) { l.logf(LevelWarn, component, format, args...) }

// Error 输出 error 级日志。
func (l *Logger) Error(component, format string, args ...any) { l.logf(LevelError, component, format, args...) }

// Debug 输出 debug 级日志。
func (l *Logger) Debug(component, format string, args ...any) { l.logf(LevelDebug, component, format, args...) }

// Default 是全局共享的 Logger，输出到 stderr。
var Default = New()
```

- [ ] **Step 6: 补充 log_test.go（验证脱敏应用到日志输出）**

文件：`mytool/internal/logx/log_test.go`

```go
package logx

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerRedactsMessage(t *testing.T) {
	var buf bytes.Buffer
	lg := NewWithWriter(&buf)
	lg.Info("auth", "user logged in with api_key=sk-live-12345")
	out := buf.String()
	if !strings.Contains(out, "api_key=<redacted>") {
		t.Errorf("logger should redact api_key in output, got: %s", out)
	}
	if strings.Contains(out, "sk-live-12345") {
		t.Errorf("logger should NOT contain raw secret, got: %s", out)
	}
}

func TestLoggerRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	lg := NewWithWriter(&buf)
	lg.SetLevel(LevelWarn)
	lg.Info("c", "info line")
	lg.Warn("c", "warn line")
	out := buf.String()
	if strings.Contains(out, "info line") {
		t.Errorf("info should be filtered out at warn level, got: %s", out)
	}
	if !strings.Contains(out, "warn line") {
		t.Errorf("warn line should be present, got: %s", out)
	}
}
```

- [ ] **Step 7: 跑全部 logx 测试**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/logx/... -v
```

预期：所有测试 PASS（5 个 redact + 2 个 log = 7 个）。

- [ ] **Step 8: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/logx/
git commit -m "feat(logx): 结构化日志 + 凭据脱敏"
```

---

## Task 3: internal/config（配置 + token 持久化）

**Files:**
- Create: `mytool/internal/config/config.go`
- Create: `mytool/internal/config/env.go`
- Create: `mytool/internal/config/secret.go`
- Test: `mytool/internal/config/config_test.go`
- Test: `mytool/internal/config/secret_test.go`

- [ ] **Step 1: 写 secret_test.go（先测 token 持久化）**

文件：`mytool/internal/config/secret_test.go`

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth", "token")

	tok, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken failed: %v", err)
	}
	if err := SaveToken(path, tok); err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	// 验证文件权限
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if perm := st.Mode().Perm(); perm != 0o600 {
		t.Errorf("token file perm = %o, want 0o600", perm)
	}

	got, err := LoadToken(path)
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}
	if got != tok {
		t.Errorf("LoadToken = %q, want %q", got, tok)
	}
}

func TestLoadTokenMissing(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadToken(filepath.Join(dir, "nope")); err == nil {
		t.Errorf("LoadToken on missing file should fail")
	}
}

func TestSaveTokenRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := SaveToken(filepath.Join(dir, "tok"), ""); err == nil {
		t.Errorf("SaveToken with empty token should fail")
	}
}

func TestNewTokenIsRandom(t *testing.T) {
	a, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	b, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if a == b {
		t.Errorf("two NewToken() calls should produce different tokens")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/config/...
```

预期：FAIL（包内还没有任何文件）。

- [ ] **Step 3: 实现 secret.go**

文件：`mytool/internal/config/secret.go`

```go
package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// tokenBytes 是 NewToken 生成的随机字节数（>= 32 bytes，符合 spec §10.2）。
const tokenBytes = 32

// NewToken 生成 32 字节随机 token 并以 base64url 编码返回。
func NewToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SaveToken 把 token 写入 path 指定的文件，权限 0o600。父目录权限 0o700。
// token 为空返回错误。
func SaveToken(path, token string) error {
	if token == "" {
		return errors.New("token is empty")
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir auth dir: %w", err)
		}
		_ = os.Chmod(dir, 0o700)
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return fmt.Errorf("write token: %w", err)
	}
	return os.Chmod(path, 0o600)
}

// LoadToken 从 path 读取并 trim 末尾换行返回。文件不存在返回错误。
func LoadToken(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read token: %w", err)
	}
	token := string(raw)
	// 去掉末尾所有空白
	for len(token) > 0 && (token[len(token)-1] == '\n' || token[len(token)-1] == ' ' || token[len(token)-1] == '\r') {
		token = token[:len(token)-1]
	}
	if token == "" {
		return "", errors.New("token file is empty")
	}
	return token, nil
}
```

- [ ] **Step 4: 跑 secret 测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/config/... -v -run "TestSave|TestLoad|TestNewToken"
```

预期：4 个 secret 测试 PASS。

- [ ] **Step 5: 写 config_test.go**

文件：`mytool/internal/config/config_test.go`

```go
package config

import (
	"testing"
)

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"ok", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws"}, false},
		{"missing port", Config{AuthToken: "abc", Workspace: "/tmp/ws"}, true},
		{"missing token", Config{Port: "8443", Workspace: "/tmp/ws"}, true},
		{"missing workspace", Config{Port: "8443", AuthToken: "abc"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cfg.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	c := Config{}.WithDefaults()
	if c.Port != "8443" {
		t.Errorf("default port = %q, want 8443", c.Port)
	}
	if c.LogLevel != "info" {
		t.Errorf("default log level = %q, want info", c.LogLevel)
	}
	if c.MTLS != "optional" {
		t.Errorf("default mtls = %q, want optional", c.MTLS)
	}
}
```

- [ ] **Step 6: 跑 config 测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/config/... -v -run TestConfig
```

预期：FAIL（Config 类型未定义）。

- [ ] **Step 7: 实现 env.go**

文件：`mytool/internal/config/env.go`

```go
package config

import (
	"os"
	"strconv"
	"strings"
)

// EnvOverrides 记录从环境变量派生的覆盖项。
type EnvOverrides struct {
	Port        string
	AuthToken   string
	Workspace   string
	MTLS        string
	LogLevel    string
	DefaultCmd  string
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

// envInt 解析 int 环境变量；空或解析失败返回 fallback。
func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
```

- [ ] **Step 8: 实现 config.go**

文件：`mytool/internal/config/config.go`

```go
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
	if c.MTLS != "" && c.MTLS != "none" && c.MTLS != "optional" && c.MTLS != "required" {
		return fmt.Errorf("mtls must be one of none|optional|required, got %q", c.MTLS)
	}
	return nil
}
```

- [ ] **Step 9: 跑全部 config 测试**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/config/... -v
```

预期：所有 config 测试 PASS（4 secret + 2 config = 6 个）。

- [ ] **Step 10: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/config/
git commit -m "feat(config): Config + FromEnv + token 持久化 0o600"
```

---

## Task 4: internal/auth（token 校验 + Bearer middleware）

**Files:**
- Create: `mytool/internal/auth/token.go`
- Create: `mytool/internal/auth/middleware.go`
- Test: `mytool/internal/auth/token_test.go`
- Test: `mytool/internal/auth/middleware_test.go`

- [ ] **Step 1: 写 token_test.go**

文件：`mytool/internal/auth/token_test.go`

```go
package auth

import "testing"

func TestTokenConstantTimeCompare(t *testing.T) {
	hash, err := HashToken("super-secret-token")
	if err != nil {
		t.Fatalf("HashToken: %v", err)
	}
	if !VerifyToken(hash, "super-secret-token") {
		t.Errorf("VerifyToken with correct secret should pass")
	}
	if VerifyToken(hash, "wrong-token") {
		t.Errorf("VerifyToken with wrong secret should fail")
	}
}

func TestHashTokenRejectsEmpty(t *testing.T) {
	if _, err := HashToken(""); err == nil {
		t.Errorf("HashToken(\"\") should fail")
	}
}

func TestHashTokenFormat(t *testing.T) {
	h, err := HashToken("abc")
	if err != nil {
		t.Fatalf("HashToken: %v", err)
	}
	// base64.RawURLEncoding(SHA-256) = 43 字符
	if len(h) != 43 {
		t.Errorf("hash length = %d, want 43", len(h))
	}
}
```

- [ ] **Step 2: 写 middleware_test.go**

文件：`mytool/internal/auth/middleware_test.go`

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerMiddleware_AcceptsValidToken(t *testing.T) {
	const token = "valid-token-abc"
	handler := BearerMiddleware(token, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestBearerMiddleware_RejectsMissingToken(t *testing.T) {
	handler := BearerMiddleware("valid", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestBearerMiddleware_RejectsWrongToken(t *testing.T) {
	handler := BearerMiddleware("valid", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestBearerMiddleware_RejectsMalformedHeader(t *testing.T) {
	handler := BearerMiddleware("valid", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestBearerMiddleware_QueryToken(t *testing.T) {
	const token = "valid-token-abc"
	handler := BearerMiddleware(token, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}
```

- [ ] **Step 3: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/...
```

预期：FAIL（包内还没有文件）。

- [ ] **Step 4: 实现 token.go**

文件：`mytool/internal/auth/token.go`

```go
package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
)

// HashToken 返回 token 的 SHA-256 摘要（base64url）。空 token 返回错误。
// 用于在内存中保存 token 摘要，避免 token 明文驻留。
func HashToken(token string) (string, error) {
	if token == "" {
		return "", errors.New("token is empty")
	}
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// VerifyToken 用常量时间比较验证 token 是否匹配 hash。
func VerifyToken(hash, token string) bool {
	want, err := base64.RawURLEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	sum := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(want, sum[:]) == 1
}
```

- [ ] **Step 5: 实现 middleware.go**

文件：`mytool/internal/auth/middleware.go`

```go
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// BearerMiddleware 返回一个 HTTP 中间件：从 Authorization 头（Bearer）或
// URL query (?token=...) 读取 token，与 expected 做常量时间比较。
// 不匹配一律 401，不进入下游 handler。
func BearerMiddleware(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := extractToken(r)
		if got == "" || !equalConstantTime(got, expected) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractToken 按顺序从 query、Authorization 头提取 token。
func extractToken(r *http.Request) string {
	if q := strings.TrimSpace(r.URL.Query().Get("token")); q != "" {
		return q
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[len("bearer "):])
	}
	return ""
}

// equalConstantTime 用 crypto/subtle 比较两个字符串，避免时序泄露。
func equalConstantTime(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
```

- [ ] **Step 6: 跑全部 auth 测试**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/auth/... -v
```

预期：3 token + 5 middleware = 8 个测试全部 PASS。

- [ ] **Step 7: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/auth/
git commit -m "feat(auth): token 哈希 + Bearer middleware（常量时间比较）"
```

---

## 阶段 A 完成标准

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -v
```

预期输出（精简）：

```
--- PASS: TestRedactAuthorizationBearer
--- PASS: TestRedactTokenKeyValue
--- PASS: TestRedactNoMatch
--- PASS: TestRedactEmpty
--- PASS: TestLoggerRedactsMessage
--- PASS: TestLoggerRespectsLevel
--- PASS: TestSaveAndLoadToken
--- PASS: TestLoadTokenMissing
--- PASS: TestSaveTokenRejectsEmpty
--- PASS: TestNewTokenIsRandom
--- PASS: TestConfigValidate
--- PASS: TestConfigDefaults
--- PASS: TestTokenConstantTimeCompare
--- PASS: TestHashTokenRejectsEmpty
--- PASS: TestHashTokenFormat
--- PASS: TestBearerMiddleware_AcceptsValidToken
--- PASS: TestBearerMiddleware_RejectsMissingToken
--- PASS: TestBearerMiddleware_RejectsWrongToken
--- PASS: TestBearerMiddleware_RejectsMalformedHeader
--- PASS: TestBearerMiddleware_QueryToken
PASS
ok  	github.com/yourname/mytool/internal/auth
ok  	github.com/yourname/mytool/internal/config
ok  	github.com/yourname/mytool/internal/logx
```

git log 应包含 4 个新 commit：

```
chore: 初始化 mytool Go module 与目录骨架
feat(logx): 结构化日志 + 凭据脱敏
feat(config): Config + FromEnv + token 持久化 0o600
feat(auth): token 哈希 + Bearer middleware（常量时间比较）
```
