# mobilecoding

**个人 AI CLI 远程控制台** — 把本机 Claude Code / Codex / 任意 LLM CLI 变成手机可操作的结构化聊天界面。

手机浏览器打开 `https://你的电脑IP:8443`，扫码或输入 token 即可连接。无需安装 App。

---

## 功能

### AI 引擎

| 功能 | 说明 |
|---|---|
| **ClaudeRunner** | 解析 `claude --output-format stream-json`，支持 assistant / tool_use / permission_request / plan_mode 等事件 |
| **CodexRunner** | `codex app-server` JSON-RPC 长连接 |
| **PtyRunner / PipeRunner** | 通用 PTY / Pipe 模式，支持 aichat / ollama 等任意 CLI |
| **多轮对话** | `--resume` 保持 Claude 会话上下文 |
| **模型切换** | 手机端下拉选择模型，通过 `--model` 传递，模型列表自动跟随 settings 配置刷新 |
| **权限应答** | 手机端 Allow / Deny 按钮，通过 stdin 回传决策给 CLI 进程 |
| **请求中止** | 输入栏发送按钮在等待期间变为停止按钮，点击杀 CLI 子进程但保留 session |

### 前端体验

| 功能 | 说明 |
|---|---|
| **PWA** | Service Worker 离线缓存，`display: standalone` 全屏体验 |
| **流式输出** | content_block_delta 增量渲染 + 打字机逐字揭示动画 |
| **Thinking 折叠** | 思考过程默认折叠，紫色高亮区域，一键展开/收起 |
| **Markdown 渲染** | marked 库完整支持 GFM 表格、列表、标题层级、代码高亮 |
| **消息持久化** | localStorage 保存最近 200 条消息，页面刷新不丢失 |
| **移动端适配** | 三级响应式（480/768/769+）、安全区域、键盘弹出适配、表格横向滚动 |
| **项目目录显示** | 连接栏显示服务器当前工作目录 |
| **Skill / Memory** | REST API（`/api/v1/skills`、`/api/v1/memory`）查看和编辑 |
| **二维码扫描** | 扫码自动填入 token 连接 |

### 传输与安全

| 功能 | 说明 |
|---|---|
| **HTTPS** | 自签 CA + server 证书 |
| **mTLS** | 可选客户端证书认证（`--mtls=required`） |
| **二维码配对** | 启动时终端打印二维码，手机扫码自动连接 |
| **WebSocket** | 结构化 RPC 协议（codec/conn/hub/handler），指数退避重连 |
| **Relay 中继** | 跨网络远程连接（agent ↔ relay ↔ client） |
| **认证** | 32B 随机 Bearer token + 常量时间比较 + 日志脱敏 |
| **证书轮换** | 过期前 30 天自动重新签发 |
| **配置热重载** | SIGHUP 触发 |

---

## 快速开始

### 构建启动

```bash
make build
./dist/mobilecoding.exe

# 浏览器访问 https://127.0.0.1:8443/
```

### npm 安装

```bash
npm install -g @banlan/mobilecoding
mobilecoding start
```

### 自定义 token

```bash
MOBILECODING_AUTH_TOKEN=mysecrettoken ./mobilecoding.exe
```

---

## 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `MOBILECODING_PORT` | `8443` | 监听端口 |
| `MOBILECODING_AUTH_TOKEN` | 自动生成 | 认证 token |
| `MOBILECODING_WORKSPACE` | `~/mobilecoding-workspace` | 工作区路径 |
| `MOBILECODING_MTLS` | `optional` | mTLS 模式（optional / required） |
| `MOBILECODING_LOG_LEVEL` | `info` | 日志级别 |
| `MOBILECODING_DEFAULT_COMMAND` | `claude` | 默认 AI 命令 |
| `MOBILECODING_DEFAULT_ARGS` | — | 默认 CLI 参数 |
| `MOBILECODING_MODELS` | 内置默认列表 | 自定义模型列表（`标签:模型名,标签:模型名`） |

### 模型配置

手机端模型选择器默认显示服务端 `/api/v1/models` 返回的列表。可通过以下方式自定义：

1. **环境变量**：`MOBILECODING_MODELS=Haiku:claude-haiku-4-5,Sonnet:claude-sonnet-4-6`
2. **Settings 文件**：在 `~/.claude/settings.*.json` 的 `env` 中配置 `ANTHROPIC_*_MODEL` 变量，切换配置时自动刷新模型列表

---

## 架构

```
手机浏览器 (PWA)
    │
    ▼  WebSocket (wss://)
mobilecoding Go 后端
    │
    ├─ Claude CLI (--output-format stream-json, stdin 双向通信)
    ├─ Codex CLI (app-server JSON-RPC)
    └─ 通用 PTY / Pipe (aichat / ollama ...)
```

```
内部包:
  auth/       — Token + CA + mTLS + 设备证书 + 二维码 + 证书轮换
  config/     — 配置 + 环境变量 + SIGHUP 热重载
  engine/     — ClaudeRunner + CodexRunner + PtyRunner + PipeRunner + 注册表
  gateway/    — HTTP 路由 + SPA + WS 升级 + REST API
  projection/ — 事件投影：stream-json → 结构化事件（text_delta / tool_use / permission_request …）
  relay/      — WebSocket 中继（agent ↔ relay ↔ client）
  session/    — 单活跃会话管理
  ws/         — WebSocket 协议（codec/conn/hub/handler）
```

---

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22+ |
| 路由 | chi/v5 |
| WebSocket | gorilla/websocket |
| PTY | creack/pty |
| 加密 | 标准库 crypto (TLS / mTLS / token / cert) |
| 存储 | 原子重命名 JSON 文件 |
| 前端 | React 18 + TypeScript + Vite |
| Markdown | marked (GFM) |
| PWA | Workbox (Service Worker + IndexedDB) |
| 发布 | npm + GitHub Actions |

---

## 安全

- ✅ 强制 HTTPS（自签 CA + server 证书）
- ✅ 可选 mTLS + 设备证书
- ✅ 32B 随机 token + 常量时间比较
- ✅ Bearer 鉴权（query + Authorization header）
- ✅ 日志脱敏（Authorization / api_key / token）
- ✅ CheckOrigin 校验（防御 CSRF / 跨源劫持）
- ✅ 证书过期前 30 天自动轮换

---

## 开发

```bash
# 后端
go test ./...
go build -o dist/mobilecoding.exe ./cmd/server

# 前端
cd web && npm run build

# 一键构建
make build
```

---

## 许可证

MIT