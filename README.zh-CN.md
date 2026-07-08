# mobilecoding

[English](./README.md) | [简体中文](./README.zh-CN.md)

**个人 AI CLI 远程控制台** — 把本机 Claude Code / Codex / 任意 LLM CLI 变成手机可操作的结构化聊天界面。

两种手机客户端共用同一个 Go 后端：

- **PWA（浏览器，`web/`）**：手机浏览器打开 `https://你的电脑IP:8443`，扫码或输入 token 即可连接，无需安装 App。
- **原生 App（React Native，`mobile/`）**：Android/iOS 原生客户端，本地 SQLite 持久化，扫码连接。

---

## 功能

### AI 引擎

| 功能 | 说明 |
|---|---|
| **ClaudeRunner** | 解析 `claude --output-format stream-json`，支持 assistant / tool_use / permission_request / plan_mode 等事件 |
| **CodexRunner** | `codex app-server` JSON-RPC 长连接，支持 initialize 握手、turn/interrupt 中止 |
| **PtyRunner / PipeRunner** | 通用 PTY / Pipe 模式，支持 aichat / ollama 等任意 CLI |
| **声明式 Agent 配置** | `agents.json` 声明 Agent 元数据，新增 Agent 只改 JSON |
| **多轮对话** | `--resume` 保持 Claude 会话上下文 |
| **模型切换** | 手机端下拉选择模型，通过 `--model` 传递，模型列表自动跟随当前 settings 配置刷新 |
| **权限应答** | 手机端 Allow / Deny 按钮，通过 stdin / hook 回传决策给 CLI |
| **请求中止** | 输入栏发送按钮在等待期间变为停止按钮，点击杀 CLI 子进程但保留 session |

### 前端体验

| 功能 | 说明 |
|---|---|
| **PWA** | Service Worker 离线缓存，`display: standalone` 全屏体验 |
| **流式输出** | content_block_delta 增量渲染 |
| **Thinking 折叠** | 思考过程默认折叠，一键展开/收起 |
| **Markdown 渲染** | marked 完整支持 GFM 表格、列表，代码块带复制按钮 + 语言标签 |
| **消息持久化** | localStorage + SQLite 双重持久化，页面刷新不丢失 |
| **断线重连** | WebSocket 重连后自动通过 `after_seq` 补发缺失消息 |
| **输入历史** | 上/下方向键回溯最近输入（跨会话，localStorage 持久化） |
| **草稿保留** | 按会话保存草稿，切换会话不丢失 |
| **离线排队** | 断线时输入自动入队，重连后自动发送 |
| **Unified Diff** | DiffView 解析 git diff 输出，结构化 +/ /- 渲染 |
| **会话管理** | 会话重命名、历史会话恢复续聊（resume ID 持久化） |
| **权限体验** | banner 展示工具详情（文件路径/命令）；"本次会话不再询问此工具" + "本轮全部允许" |
| **二维码配对** | 扫码自动填入 token 连接 |

### 子命令模式

| 命令 | 行为 |
|---|---|
| `mobilecoding claude` | 启动 server + Claude（遥控器模式），手机扫码后双端共存 |
| `mobilecoding codex` | 启动 server + Codex |
| `mobilecoding relay` | 连接到 relay 服务器作为 agent |
| `mobilecoding server` | 仅启动 server（默认行为） |
| `mc` | `mobilecoding` 的别名 |

**智能 settings 探测**：`mc claude` 不传 `-settings` 时，自动探测 `<CWD>/.claude/settings.local.json`；不存在则回退到全局 `~/.claude/settings.json`。显式 `-settings <path>` 优先级最高。

**智能 IP 选择**：优先局域网 IP（10.x > 172.16-31 > 192.168），跳过虚拟网卡。可用 `-ip 192.168.1.100` 或 `MOBILECODING_IP=...` 覆盖。

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

### 遥控器模式（推荐）

```bash
make build
npm link

# 自动进入遥控器模式
mobilecoding claude
mobilecoding claude --settings ~/.claude/settings.xxx.json
mobilecoding claude --model claude-opus-4-8

# mc 命令等价于 mobilecoding
mc claude
mc codex
```

### npm 安装

```bash
npm install -g @banlan/mobilecoding
mobilecoding                          # 启动 server（默认行为）
mobilecoding claude                   # 遥控器模式
mobilecoding relay -relay wss://...   # relay 模式
```

### 自定义 token

```bash
MOBILECODING_AUTH_TOKEN=mysecrettoken ./mobilecoding.exe
```

### 原生客户端（React Native）

`mobile/` 目录是 Android/iOS 原生客户端，与 PWA 平行，连接同一个 Go 后端。

```bash
cd mobile
npm install
npm run android   # 或：npm run ios（首次需 bundle exec pod install --project-directory=ios）
```

启动后扫码或输入后端地址（`https://电脑IP:8443`）与 token 连接。

---

## 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `MOBILECODING_PORT` | `8443` | 监听端口 |
| `MOBILECODING_IP` | 自动检测 | 本机 IP（覆盖自动检测） |
| `MOBILECODING_AUTH_TOKEN` | 自动生成 | 认证 token |
| `MOBILECODING_WORKSPACE` | `~/mobilecoding-workspace` | 工作区路径 |
| `MOBILECODING_MTLS` | `optional` | mTLS 模式（optional / required） |
| `MOBILECODING_LOG_LEVEL` | `info` | 日志级别 |
| `MOBILECODING_DEFAULT_COMMAND` | `claude` | 默认 AI 命令 |
| `MOBILECODING_DEFAULT_ARGS` | — | 默认 CLI 参数 |
| `MOBILECODING_MODELS` | 内置默认列表 | 自定义模型列表（`标签:模型名,标签:模型名`） |
| `MOBILECODING_LAUNCH_MODE` | — | `managed` / `remote-control` |

### 模型配置

手机端模型选择器显示服务端 `/api/v1/models` 返回的列表。可通过以下方式自定义：

1. **环境变量**：`MOBILECODING_MODELS=Haiku:claude-haiku-4-5,Sonnet:claude-sonnet-4-6`
2. **Settings 文件**：在 `~/.claude/settings.*.json` 或 `<项目>/.claude/settings.local.json` 的 `env` 块中配置 `ANTHROPIC_*_MODEL` 变量。下拉列表显示**实际模型名**（如 `minimax-m3[1m]`），而非档位标签。切换 settings 时自动刷新模型列表。

---

## 架构

```
手机浏览器 (PWA) / React Native App
    │
    ▼  WebSocket (wss://)
mobilecoding Go 后端
    │
    ├─ Claude CLI (--output-format stream-json, stdin 双向通信)
    ├─ Codex CLI (app-server JSON-RPC, initialize 握手)
    └─ 通用 PTY / Pipe (aichat / ollama ...)
```

```
内部包:
  auth/       — Token + CA + mTLS + 设备证书 + 二维码 + 证书轮换
  config/     — 配置 + 环境变量 + SIGHUP 热重载
  engine/     — ClaudeRunner + CodexRunner + PtyRunner + PipeRunner + 声明式 Agent 注册表
  files/      — Git status/diff + 文件树 + 文件读取（路径越界防护）
  gateway/    — HTTP 路由 + SPA + WS 升级 + REST API
  hook/       — Claude HTTP hook 端点（PermissionRequest）
  logx/       — 日志 + 脱敏 + 轮换
  projection/ — 事件投影：stream-json / Codex JSON-RPC → 结构化事件
  protocol/   — Wire 协议常量（事件类型 / RPC 方法 / 错误码）
  relay/      — WebSocket 中继（agent ↔ relay ↔ client）
  session/    — 会话生命周期 + 元数据持久化 + resume ID
  store/      — SQLite 消息持久化 + 序列号 + 搜索 + 清理
  ws/         — WebSocket 协议（codec/conn/hub/handler + Replay 缓冲）
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
| 存储 | modernc.org/sqlite (WAL 模式) |
| 前端 | React 18 + TypeScript + Vite |
| Markdown | marked (GFM) |
| PWA | Workbox (Service Worker) |
| 移动端 | React Native 0.81（`mobile/`，Android/iOS，op-sqlite + Zustand） |
| 发布 | npm + GitHub Actions |

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

## 参考项目

mobilecoding 参考了以下开源项目的设计理念：

- **[Happy](https://github.com/slopus/happy)** — Claude Code & Codex 的移动/Web 客户端。端到端加密、Local/Remote 模式、Session Scanner。→ 消息持久化 + 序列号、V3 消息 API、Wire 协议统一。
- **[VibeAround](https://github.com/jazzenchen/VibeAround)** — AI Agent 管理平台。多 Agent、Local API Bridge、Onboarding。→ 声明式 Agent 配置、首次使用引导。
- **[MindFS](https://github.com/a9gent/mindfs)** — AI Agent 远程网关 + 结果可视化。StreamHub Replay、会话搜索、单二进制。→ StreamHub Replay、会话搜索。
- **[EasyCodex](https://github.com/Ryan-Laws/easycodex)** — Codex 远程控制。JSON-RPC、消息规范化、流式批处理。→ Codex JSON-RPC 协议、消息规范化、投影层。

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

## 许可证

MIT
