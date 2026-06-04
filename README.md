# mobilecoding

**个人 AI CLI 远程控制台** — 把本机 Claude Code / Codex / 任意 LLM CLI 的"等待态"做成手机可操作的结构化卡片。

手机浏览器打开 https://你的电脑IP:8443 ，扫码或输入 token 即可连接。不需要安装手机 App，不需要 TestFlight 或 APK。

---

## 功能

### 后端

| 功能 | 说明 |
|---|---|
| **ClaudeRunner** | 解析 `claude --output-format stream-json` 输出，支持 assistant_message / tool_use / permission_request / plan_mode 等 7 种事件类型 |
| **CodexRunner** | 解析 `codex app-server` JSON-RPC |
| **PtyRunner** | 通用 PTY 模式，支持 aichat / crush / ollama 等任意 CLI |
| **Skill / Memory 管理** | REST API (`/api/v1/skills`, `/api/v1/memory`) |
| **文件浏览** | 工作区文件树 + 读取 + denylist（.env/*.key/*.pem 等自动拦截） |
| **会话管理** | 单活跃 session + stall watchdog（120s 沉默 → 自动 kill） |
| **Stall Watchdog** | 120s 沉默阈值 + 10min 工具执行宽限 |
| **cert 轮换** | 证书过期前 30 天自动重新签发 |
| **配置热重载** | SIGHUP 信号触发配置重载 |
| **日志轮转** | 日滚动文件，7 天保留 |

### 传输层

| 功能 | 说明 |
|---|---|
| **HTTPS** | 强制 HTTPS（自签 CA + server 证书） |
| **mTLS** | 可选客户端证书认证（`--mtls=required`） |
| **二维码配对** | 启动时终端打印二维码，手机扫码自动连接 |
| **设备证书** | 首次连接签发设备证书，IndexedDB 持久化 |
| **WebSocket** | 结构化事件流（codec/conn/hub/handler） |
| **Web Push** | PWA 推送通知 |

### 前端

| 功能 | 说明 |
|---|---|
| **PWA** | Service Worker 缓存 + 离线快照 |
| **Skill 页面** | 查看 Skill 列表 |
| **Memory 页面** | 查看/编辑 Memory |
| **二维码扫描** | 扫码自动填入 token |

### 运维

| 功能 | 说明 |
|---|---|
| **日志聚合** | 启动日志写入 `~/.mobilecoding/logs/`，7 天保留 |
| **GitHub Actions** | CI (push/PR) + Release (tag push 跨平台编译) |
| **npm 包** | `@banlan/mobilecoding`，一行命令安装 |

---

## 快速开始

### Go 构建启动

```bash
go build -o mobilecoding.exe ./cmd/server
./mobilecoding.exe

# 浏览器访问 https://127.0.0.1:8443/
```

### 二维码配对

```bash
./mobilecoding.exe

# 终端会显示:
# ==================================================
# Scan QR Code to connect:
# ████████████████████████████████████
# ██  ████  ██████  ██  ██  ██  ██████
# ██  ████  ██████  ██  ██  ██  ██████
# ...
# ==================================================
# 手机扫码即可自动连接
```

### 自定义 token

```bash
MOBILECODING_AUTH_TOKEN=mysecrettoken ./mobilecoding.exe
```

### npm 安装启动

```bash
npm install -g @banlan/mobilecoding
mobilecoding start
```

---

## 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `MOBILECODING_PORT` | `8443` | 监听端口 |
| `MOBILECODING_AUTH_TOKEN` | 自动生成 | 认证 token |
| `MOBILECODING_WORKSPACE` | `~/mobilecoding-workspace` | 工作区路径 |
| `MOBILECODING_MTLS` | `optional` | mTLS 模式 (optional / required) |
| `MOBILECODING_LOG_LEVEL` | `info` | 日志级别 (debug / info / warn / error) |
| `MOBILECODING_DEFAULT_COMMAND` | `claude` | 默认 AI 命令 |

### 命令行参数

```bash
mobilecoding.exe \
  --port 8443 \
  --auth-token "xxx" \
  --workspace "~/my-project" \
  --mtls optional \
  --log-level info \
  --default-command claude
```

配置热重载：`kill -HUP <PID>`

---

## 架构

```text
手机浏览器 (PWA)
    │
    ▼  WebSocket (wss://)
mobilecoding Go 后端
    │
    ├─ Claude CLI (--output-format stream-json)
    ├─ Codex CLI (app-server JSON-RPC)
    └─ 通用 PTY (aichat / crush / ollama ...)
```

```
后端包:
  auth/       — Token + CA + mTLS + 设备证书 + 二维码 + cert rotation
  config/     — 配置 + 环境变量 + SIGHUP 热重载
  engine/     — ClaudeRunner + CodexRunner + PtyRunner + registry
  files/      — 文件树 + 读取 + denylist
  gateway/    — HTTP 路由 + SPA + WS 升级 + API
  logx/       — 结构化日志 + 脱敏 + 轮转
  projection/ — 事件投影 + Claude 深度解析
  push/       — Web Push
  session/    — 会话管理 + stall watchdog
  store/      — 原子重命名存储
  ws/         — WebSocket 协议
```

---

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22+ |
| 路由 | chi/v5 |
| WebSocket | gorilla/websocket |
| PTY | creack/pty |
| 加密 | 标准库 crypto (TLS/mTLS/token/cert rotation) |
| 存储 | 原子重命名 JSON 文件 |
| 前端 | React + TypeScript + Vite |
| PWA | Workbox (Service Worker + IndexedDB) |
| 推送 | Web Push API |
| 发布 | npm + GitHub Actions |

---

## 安全

- ✅ 强制 HTTPS（自签 CA + server 证书）
- ✅ 可选 mTLS + 设备证书
- ✅ 32B 随机 token + 常量时间比较
- ✅ Bearer 鉴权（query + Authorization header）
- ✅ Log redaction（Authorization / api_key / token 自动脱敏）
- ✅ 0o600 文件 / 0o700 目录权限
- ✅ denylist（.env / *.key / *.pem / .git / node_modules 自动拦截）
- ✅ CheckOrigin 校验（防御 CSRF/跨源劫持）
- ✅ Cert rotation（过期前 30 天自动重新签发）

---

## 开发

```bash
# 后端测试
go test ./...

# 前端构建
cd web && npm run build

# 运行
go build -o mobilecoding.exe ./cmd/server
./mobilecoding.exe

# 日志
tail -f ~/mobilecoding/logs/mobilecoding-2026-06-02.log
```

---

## 发布

```bash
git push origin master
# CI 自动跑测试

git tag v0.1.0
git push origin v0.1.0
# Release 自动构建跨平台二进制 + 上传到 GitHub Releases
```

---

## 许可证

MIT