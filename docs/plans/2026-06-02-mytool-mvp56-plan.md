# mytool MVP 5+6 实施计划

## Task 40: Web Push 基础 (subscription)

**Files:** Create: `web/src/core/push/push-service.ts`, Modify: `web/src/sw.ts`

### Step 1: push-service.ts

```typescript
export async function registerPush(): Promise<PushSubscription | null> {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) return null;
  const reg = await navigator.serviceWorker.ready;
  return await reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(PUBLIC_VAPID_KEY),
  });
}

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = window.atob(base64);
  return new Uint8Array([...rawData].map(char => char.charCodeAt(0)));
}
```

### Step 2: sw.ts add push handler

```typescript
self.addEventListener('push', (event) => {
  const data = event.data?.json() ?? { title: 'mytool', body: 'New notification' };
  event.waitUntil(
    self.registration.showNotification(data.title, { body: data.body })
  );
});
```

### Step 3: Submit

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && git add web/ && git commit -m "feat(push): Web Push subscription + handler"
```

---

## Task 41: Cert Rotation

**Files:** Modify: `mytool/cmd/server/main.go`

### Step 1: Add cert rotation logic

```go
func checkCertExpiry(certPath string) bool {
    raw, err := os.ReadFile(certPath)
    if err != nil { return false }
    block, _ := pem.Decode(raw)
    if block == nil { return false }
    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil { return false }
    return time.Until(cert.NotAfter) < 30*24*time.Hour
}
```

在 `main()` 启动时调用此函数，expired → 重新签发 server 证书。

### Step 2: Submit

```bash
git add mytool/cmd/server/ && git commit -m "feat(cert): 证书过期自动重新签发"
```

---

## Task 42: 日志聚合

**Files:** Create: `mytool/internal/logx/rotate.go`, Modify: `mytool/cmd/server/main.go`

### Step 1: rotate.go

```go
package logx

import (
    "os"
    "path/filepath"
    "time"
)

func OpenLogFile(dir string) (*os.File, error) {
    os.MkdirAll(dir, 0o755)
    name := "mytool-" + time.Now().Format("2006-01-02") + ".log"
    return os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}

func CleanOldLogs(dir string, maxDays int) error {
    cutoff := time.Now().AddDate(0, 0, -maxDays)
    entries, _ := os.ReadDir(dir)
    for _, e := range entries {
        if !e.IsDir() && strings.HasPrefix(e.Name(), "mytool-") {
            if info, _ := e.Info(); info != nil && info.ModTime().Before(cutoff) {
                os.Remove(filepath.Join(dir, e.Name()))
            }
        }
    }
    return nil
}
```

### Step 2: main.go write logs to file

```go
logFile, err := logx.OpenLogFile(filepath.Join(home, ".mytool", "logs"))
if err == nil {
    logger = logx.NewWithWriter(io.MultiWriter(os.Stderr, logFile))
}
```

### Step 3: Submit

```bash
git add mytool/internal/logx/ mytool/cmd/server/ && git commit -m "feat(logx): 日志文件轮转（7 天保留）"
```

---

## Task 43: 配置热重载

**Files:** Modify: `mytool/cmd/server/main.go`

### Step 1: SIGHUP handler

```go
func handleReload(cfg *config.Config, logger *logx.Logger) {
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGHUP)
    for range sig {
        newCfg, err := config.Load()
        if err == nil {
            *cfg = newCfg
            logger.SetLevel(parseLevel(cfg.LogLevel))
            logger.Info("reload", "config reloaded")
        }
    }
}
```

### Step 2: Submit

```bash
git add mytool/cmd/server/ && git commit -m "feat(config): 配置热重载（SIGHUP）"
```

---

## Task 44: npm 包发布

**Files:** Create: `package.json` (root), `bin/mytool.js`

### Step 1: package.json

```json
{
  "name": "@banlan/mytool",
  "version": "0.1.0",
  "description": "AI CLI 远程控制台",
  "bin": { "mytool": "bin/mytool.js" },
  "files": ["bin/", "dist/"]
}
```

### Step 2: bin/mytool.js

```javascript
#!/usr/bin/env node
const { execSync } = require('child_process');
const os = require('os');
const path = require('path');

const platform = os.platform();
const arch = os.arch();
const ext = platform === 'win32' ? '.exe' : '';
const binary = path.join(__dirname, '..', 'dist', `mytool-${platform}-${arch}${ext}`);

try {
  execSync(binary, { stdio: 'inherit' });
} catch (e) {
  console.error('mytool binary not found:', binary);
  process.exit(1);
}
```

### Step 3: Submit

```bash
git add package.json bin/ && git commit -m "feat(npm): npm 包发布配置"
```

---

## Task 45: GitHub Actions CI/CD

**Files:** Create: `.github/workflows/ci.yml`, `.github/workflows/release.yml`

### Step 1: ci.yml

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test ./...
      - run: go build ./cmd/server
```

### Step 2: release.yml

```yaml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - name: Build matrix
        run: |
          mkdir -p dist
          for pair in "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64"; do
            GOOS=${pair%/*} GOARCH=${pair#*/}
            out="dist/mytool-${GOOS}-${GOARCH}"
            [ "$GOOS" = "windows" ] && out="${out}.exe"
            GOOS=$GOOS GOARCH=$GOARCH go build -o "$out" ./cmd/server
          done
      - uses: softprops/action-gh-release@v1
        with:
          files: dist/*
```

### Step 3: Submit

```bash
git add .github/ && git commit -m "ci: GitHub Actions CI + release workflow"
```

---

## Task 46: 整体回归

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1
```

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool"
git add -A && git commit -m "test: MVP 5+6 整体回归"
```