# mytool — 个人 AI CLI 远程控制台 设计 spec

| 字段 | 值 |
|---|---|
| 日期 | 2026-06-01 |
| 状态 | 设计已确认，待实现 |
| 替代对象 | MobileVC（参考分析见同仓库 `README.md` / `internal/`） |
| 核心目标 | 个人自用、客户端轻量化、隐私优先 |

---

## 0. 背景与定位

### 0.1 目标

写一个类似 MobileVC、但**只解决个人自用痛点**的项目：

- 把本机 Claude Code / Codex CLI / 其它 LLM CLI 的"等待态"（权限弹窗、Plan Mode、Diff 审核、会话续接、文件浏览）做成手机可操作的结构化卡片
- 不做远程终端镜像（在小屏幕上低效）
- 不做 Relay / E2EE / APNs / TTS / ADB 投屏
- 不做 Flutter 多端打包 + OTA + TestFlight

### 0.2 与 MobileVC 的差异（核心）

| 维度 | MobileVC | mytool（本项目） |
|---|---|---|
| 客户端 | Flutter iOS / Android / Web + OTA | React/Vite SPA + PWA（手机浏览器即用） |
| 网络 | LAN + 自建 Relay（含自研 P-256 ECDH + AES-GCM） | 仅 LAN；公网由用户自行用 frp/Tailscale/Cloudflared 穿透 |
| 鉴权 | 静态 AUTH_TOKEN + 跨通道透传 | 启动时随机生成 32B token + 本地 mTLS 可选 |
| 推送 | APNs | 无（实时性靠 WebSocket） |
| AI 引擎 | Claude / Codex | Claude / Codex / 通用 PTY（任意 LLM CLI） |
| 启动壳 | npm 包装 Go 预编译二进制 | 同上（沿用 MobileVC 模式，去繁就简） |
| 交互 | 结构化卡片投影 | 同上（沿用思路，前端用 React 重做） |

### 0.3 不在范围内（明确排除）

- Relay / 中继 / E2EE 自研协议
- APNs / FCM / Web Push
- TTS / 语音通话
- ADB 投屏 / 模拟器调试
- iOS / Android 原生 App
- 多用户 / 团队 / 账号体系
- 云服务 / SaaS / 遥测

---

## 1. 整体架构

```text
┌────────────────────────────────────────────────────────────┐
│           mytool 启动壳 (npm: @<scope>/mytool)              │
│  - 跨平台二进制下载/选择  - start/stop/logs/qr/token         │
│  - 随机 token 注入 + 配置文件生成                          │
└────────────────────┬───────────────────────────────────────┘
                     │  exec
                     ▼
┌────────────────────────────────────────────────────────────┐
│              Go 后端 (cmd/server) 单二进制                   │
│  HTTP/HTTPS :8443  (mTLS, go:embed SPA)                    │
│    /                 → 嵌入式 React SPA (PWA)              │
│    /api/v1/ws        → 鉴权 + WebSocket 升级               │
│    /api/v1/healthz   → 健康检查                            │
│    /api/v1/qr        → 二维码（首次扫码导入 token）         │
│    /api/v1/files/*   → 工作区文件浏览/下载（鉴权）         │
│                                                             │
│  核心包 internal/:                                          │
│    config / auth / gateway / ws / engine                    │
│    session / projection / files / store / logx             │
└────────────────────┬───────────────────────────────────────┘
                     │  PTY / JSON-RPC
                     ▼
┌────────────────────────────────────────────────────────────┐
│  本机 AI 引擎：claude | codex | 任意 LLM CLI               │
└────────────────────────────────────────────────────────────┘
                     ▲
                     │  WebSocket（结构化事件 + 输入命令）
                     │
┌────────────────────┴───────────────────────────────────────┐
│  React/Vite SPA (PWA) + 手机浏览器                         │
│   入口：扫码/手填 token → 工作台                            │
│   视图：会话列表 / 聊天卡片 / 权限审批 / Diff / 文件        │
└────────────────────────────────────────────────────────────┘
```

---

## 2. 后端 Go 模块拆分

```
cmd/server/
  main.go              // 入口：装配 + 启动
  flags.go             // CLI flags (port, workspace, ...)

internal/
  config/
    config.go          // Config struct + Load + Validate
    env.go             // 环境变量解析
    secret.go          // token 持久化（0o600）

  auth/
    token.go           // NewToken / CompareToken（constant-time）
    mtls.go            // 自签 CA + 签发设备证书
    qr.go              // 生成首次配对二维码
    middleware.go      // Bearer token / mTLS 中间件

  gateway/
    router.go          // chi 路由 + 中间件链
    spa.go             // go:embed SPA handler（含 SPA fallback）
    download.go        // 工作区文件下载（鉴权后）
    handlers.go        // /api/v1/* REST handlers

  ws/
    hub.go             // 客户端连接注册表
    conn.go            // 单个 WebSocket 读写协程
    codec.go           // 消息编解码（JSON v1）
    handler.go         // 按 message.type 路由到 session/manager

  engine/
    runner.go          // interface Runner + 状态接口
    pty_runner.go      // 通用 PTY 启动 + 行读取
    claude_parser.go   // claude --output-format stream-json 解析
    codex_transport.go // codex app-server JSON-RPC
    registry.go        // runner factory（按 command 选择实现）

  session/
    manager.go         // 当前活跃 session + lifecycle
    resume.go          // --session-id 续接
    permission.go      // 权限请求/审批路由

  projection/
    events.go          // 输入：原始行 / 输出：结构化事件
    apply.go           // 事件应用到 UI 模型
    diff.go            // Diff 投影（文件级别）
    permission.go      // 权限请求投影
    plan.go            // Plan Mode 投影
    context.go         // context_window_used/max 投影

  files/
    tree.go            // 工作区文件树（限制深度、忽略 .git）
    read.go            // 单文件读取（白名单后缀）
    download.go        // 携带鉴权头的文件下载
    denylist.go        // .env / *.key / *.pem 等拒绝列表

  store/
    filestore.go       // JSON 文件 + atomic rename
    sessions.go        // 会话历史落盘
    skills.go          // Skill 仓库
    memory.go          // Memory 仓库

  logx/
    log.go             // 结构化日志
    redact.go          // token / key / Authorization 脱敏正则
    recover.go         // panic 恢复
```

**关键准则**：

- 每个 internal 包单一职责，godoc 注释在包级别
- 跨包通信用 interface，**避免** cyclic dependency
- `engine.Runner` 抽象下，claude / codex / pty 是并列实现
- `projection` 不感知 engine 细节，只接收"原始行/事件 + 上下文"
- `ws` 是唯一入口边界，REST 不暴露任何 AI 行为能力
- `store` 用 atomic rename 落盘（防断电半文件）

---

## 3. 前端 React SPA 模块拆分

```
web/                                  # Vite + React + TS
├── index.html
├── vite.config.ts
├── tsconfig.json
├── public/
│   ├── manifest.webmanifest         # PWA 清单
│   └── icons/                       # PWA 图标
├── src/
│   ├── main.tsx
│   ├── App.tsx                      # 路由 + 全局 Provider
│   ├── router.tsx                   # createBrowserRouter
│   │
│   ├── core/                        # 不含业务、纯工具
│   │   ├── api/
│   │   │   ├── ws.ts                # WebSocket 单例 + 心跳 + 重连
│   │   │   ├── rest.ts              # fetch 封装（带 token）
│   │   │   └── types.ts             # 与后端 protocol 对齐的 TS 类型
│   │   ├── state/
│   │   │   ├── store.ts             # Zustand store（slices）
│   │   │   └── events.ts            # 事件订阅 → store 更新
│   │   ├── ui/                      # 通用组件（Button/Card/Modal/...）
│   │   └── hooks/                   # useWS / useAuth / useMount
│   │
│   ├── features/                    # 业务功能模块
│   │   ├── auth/                    # LoginPage + QrScanner
│   │   ├── session/                 # SessionList + SessionDetail + ResumeBanner
│   │   ├── chat/                    # ChatView + MessageCard + StreamingText
│   │   ├── permissions/             # PermissionPrompt
│   │   ├── diff/                    # DiffView + DiffHunk
│   │   ├── plan/                    # PlanModeView
│   │   ├── files/                   # FileTree + FileViewer + FileDownload
│   │   ├── skills/                  # SkillListPage
│   │   ├── memory/                  # MemoryPage
│   │   └── settings/                # SettingsPage
│   │
│   ├── widgets/                     # 跨 feature 组合组件
│   │   ├── AppShell.tsx             # 顶栏 + 抽屉
│   │   ├── ConnectionStatus.tsx     # 在线/重连中
│   │   └── ContextWindowMeter.tsx
│   │
│   └── styles/
│       ├── tokens.css               # 设计 token
│       └── globals.css
└── dist/                            # 构建产物（go:embed 目标）
```

**关键准则**：

- `core/api` 是与后端通信的唯一入口，前端其它地方不直接 fetch
- `core/state` 用 Zustand：移动端单 store 比 Redux 简单、比 Context 性能好
- `features/*` 每个目录自包含，不互相 import
- `widgets/` 跨 feature 组合；`core/ui` 是纯展示组件
- 移动端优先：默认断点基于 iPhone 14 尺寸
- 路由：`/`（登录）、`/sessions`、`/sessions/:id`（聊天详情页）
- PWA：vite-plugin-pwa + Workbox；离线时只缓存壳与最近一次会话快照

---

## 4. WebSocket 协议

### 4.1 传输

- `wss://<lan-ip>:8443/api/v1/ws`（mTLS 通道）
- 鉴权：URL query `?token=<32B base64url>` + 升级请求头 `Authorization: Bearer <token>`（两者一致）
- 编码：JSON（v1 简单优先，后续可换 MessagePack）

### 4.2 消息信封

```ts
type Envelope =
  | { type: 'req';  id: string; method: string; params?: object }
  | { type: 'resp'; id: string; ok: true;  result?: object }
  | { type: 'resp'; id: string; ok: false; error: { code: string; message: string } }
  | { type: 'evt';  event: AppEvent; sessionId?: string }
```

- `req`/`resp` 用于 RPC（命令、查询）
- `evt`  服务器主动推送（增量、状态变化）
- `id` 用 UUIDv4，客户端生成

### 4.3 客户端→服务端方法

| method | params | 说明 |
|---|---|---|
| `session.list` | `{ limit?, before? }` | 列历史会话 |
| `session.start` | `{ command: 'claude'\|'codex'\|'pty', args?, cwd?, resumeSessionId? }` | 启动新会话 |
| `session.resume` | `{ sessionId }` | 续接 |
| `session.stop` | `{ sessionId }` | 主动停止 |
| `session.input` | `{ sessionId, text }` | 输入文本（提交 prompt） |
| `session.permission.answer` | `{ sessionId, requestId, decision: 'allow'\|'deny', always? }` | 权限回复 |
| `files.list` | `{ path, depth? }` | 文件树 |
| `files.read` | `{ path, maxBytes? }` | 读文件 |
| `files.download` | `{ path }` | 触发下载（响应里给签名 URL） |
| `skills.list` | - | 列出 skills |
| `memory.list` | `{ scope: 'project'\|'user' }` | 列出 memory |

### 4.4 服务端→客户端事件

```ts
type AppEvent =
  | SessionLifecycleEvent      // started / resumed / stopped / error
  | AssistantTextDeltaEvent    // 增量文本（流式）
  | AssistantTextFinalEvent    // 完整文本
  | ToolUseEvent               // { name, target, status, preview }
  | ToolResultEvent            // { name, status, durationMs, exitCode? }
  | PermissionRequestEvent     // { requestId, kind, prompt, target, options[] }
  | DiffEvent                  // { file, status, hunks[] }
  | PlanModeEvent              // { active, steps[], planId }
  | ContextWindowEvent         // { usedTokens, maxTokens }
  | UserInputRequestEvent      // 等待用户输入（选择 / 填空）
  | ErrorEvent                 // { code, message, retriable }
```

### 4.5 错误码规范

| code | 含义 |
|---|---|
| `unauthorized` | token 错或缺失 |
| `forbidden` | 已鉴权但无权限 |
| `not_found` | 资源不存在 |
| `conflict` | 已有活跃 session / resume 冲突 |
| `rate_limited` | 客户端动作去重触发 |
| `internal` | 后端 bug |
| `engine_failure` | AI 引擎崩溃 / 不可用 |
| `protocol_error` | 客户端发非法消息 |

### 4.6 流控与可靠性

- **心跳**：服务端每 15s 推 `ping`，客户端 30s 内未收到则触发重连
- **重连**：客户端按 1s/2s/5s/10s/30s 退避；重连后带 `lastEventId` 触发 `session.replay`
- **消息去重**：客户端动作 24h 滑窗去重（与 MobileVC 同思路）
- **大消息**：单条 > 64KB 的 diff 拆为 `DiffHunk` 增量推送

---

## 5. 鉴权与 mTLS

### 5.1 启动期一次性流程

```
[mytool start]
   │
   ├─ 若 ~/.mytool/auth/ 不存在：
   │    1. 随机生成 32B token，base64url
   │    2. 生成自签 CA（10 年），落到 ~/.mytool/auth/ca.{crt,key} (0o600)
   │    3. 用 CA 签发 server 证书（SAN: <lan-ip>, localhost）
   │    4. 把 token + server 证书 + 端口打包成 JSON 写 ~/.mytool/auth/pair.json (0o600)
   │
   ├─ 打印本地 URL、二维码、token（首次只打印一次，之后 mytool token show）
   │
   └─ 启动 HTTPS（mTLS 可选）+ HTTP/2 服务器
```

### 5.2 客户端首次接入

- 手机浏览器打开 `https://<lan-ip>:8443/`
- 浏览器弹"证书不受信任"提示（自签 CA），用户手动信任一次（一次性）
- 登录页提供两种方式：
  - **扫码**：手机相机扫启动时打印的二维码 → 解析出 `lanIp:port + token`
  - **手填**：在终端跑 `mytool qr` 或 `mytool token` 把 token 拷过来
- token 落 `localStorage`（PWA scope 内）+ `IndexedDB` 的会话表（首次）

### 5.3 mTLS 策略

- **默认**：`MTLS_OPTIONAL`（强制 HTTPS，不要求客户端证书）
- 启动参数 `--mtls=required` 时：
  - 客户端首次扫码后服务端自动签发设备证书
  - 设备证书落 `~/.mytool/auth/devices/<device-id>.{crt,key}`，key 加密保存（passphrase = 设备 token hash）
  - 客户端把设备证书 + key 落到 PWA IndexedDB
  - 后续 WS 升级用 `clientCert` 进行 mutual TLS

### 5.4 鉴权层次

| 通道 | 鉴权 | 说明 |
|---|---|---|
| `/api/v1/healthz` | 无 | 仅健康检查 |
| `/api/v1/qr` | 无 | 仅启动期生成二维码内容；运行期不暴露 |
| 静态 SPA | token | Bearer token 用于 fetch；mTLS 通道保护 |
| `/api/v1/ws` | token + (mTLS 可选) | query + Authorization 双带 |
| `/api/v1/files/*` | token + 路径白名单 | workspace 之外禁止访问 |

### 5.5 路径白名单

- workspace 路径由 `MYTOOL_WORKSPACE` 指定（默认 `~/mytool-workspace`）
- 路径校验：必须 `filepath.Clean + EvalSymlinks` 落在 workspace 内
- 禁止读：`.git/`、`node_modules/`、`.env`、`*.pem`、`*.key`、`*.p12`（保护凭据）

---

## 6. AI 引擎抽象

### 6.1 Runner 接口

```go
type Runner interface {
    Start(ctx context.Context, req ExecRequest) error
    Write(p []byte) error                              // 写入 stdin / PTY
    Resize(cols, rows int) error                       // PTY 终端大小
    Close() error
    Events() <-chan RawEvent                          // 原始行 / 协议事件
    Errors() <-chan error
    Done() <-chan struct{}

    SessionID() string
    InteractiveStateProvider
    TurnStateProvider
}

type InteractiveStateProvider interface {
    CanAcceptInteractiveInput() bool
}
type TurnStateProvider interface {
    HasActiveTurn() bool
}
```

### 6.2 三种实现

| Runner | 适用 command | 实现 |
|---|---|---|
| `ClaudeRunner` | `claude`, `claude --resume <id>` | 启动 `claude --output-format stream-json --verbose`，按行解析 stream-json 事件；权限弹窗通过解析 `permission_request` 拦截 |
| `CodexRunner` | `codex` | 启动 `codex app-server`，JSON-RPC over stdio；订阅 `thread/started`、`item/agentMessage/delta`、`toolCall` 等 |
| `PtyRunner` | 其他任意 LLM CLI（`aichat`、`crush`、`llm`、`ollama run` …） | 通用 PTY + 行解析；不做协议理解；只暴露"输入 + 输出文本"最简能力 |

### 6.3 投影流水线

```
[Runner Events]
   ├─ claude stream-json line ──┐
   ├─ codex JSON-RPC message ──┼─► projection.Normalize
   └─ pty raw line ─────────────┘            │
                                              ▼
                                  projection.Event (统一 schema)
                                              │
                            ┌─────────────────┼─────────────────┐
                            ▼                 ▼                 ▼
                       DiffEvent    PermissionRequestEvent   ToolUseEvent
                            │                 │                 │
                            └──────► ws.hub.broadcast ◄───────┘
                                              │
                                              ▼
                                  React features/* 订阅
```

### 6.4 关键约束

- **claude 必须 `--output-format stream-json`**：用结构化事件，不用正则瞎猜 ANSI
- **codex 必须用 app-server 模式**：不用 `codex exec` 一次性（不支持会话续接）
- **pty runner 不参与投影**：只透传"最近 200 行 raw output"作为兜底；权限/plan/diff 由用户在通用 CLI 里自己解释
- **stall watchdog**：所有 runner 共用 60/90/120s 沉默阈值 + 工具执行 10min 宽限

### 6.5 引擎选择策略

- 显式命令优先；空命令时取 `MYTOOL_DEFAULT_COMMAND`（默认 `claude`）
- 解析规则（`engine/registry.go`）：

```
"claude"          → ClaudeRunner
"claude-code"     → ClaudeRunner
"claude ..."      → ClaudeRunner（带额外 args）
"codex"           → CodexRunner
其它                → PtyRunner
```

### 6.6 投影与 CLI 协议的解耦

- `projection` 包**不** import `engine`
- 反过来：engine 通过 `event_sink` 接口把原始事件推给 projection
- 这样将来换 CLI（如 Cursor CLI、opencode）只新增一个 Runner，不动 projection

---

## 7. 投影与卡片化（核心 UX）

### 7.1 投影目标

把"AI 引擎在终端的输出"翻译成"手机上可以按按钮的结构化卡片"。

### 7.2 事件分类与卡片

| 投影事件 | UI 卡片 | 交互 |
|---|---|---|
| `AssistantTextDelta` | 聊天气泡（流式打字机） | 阅读 |
| `AssistantTextFinal` | 聊天气泡（完整版） | 阅读、复制 |
| `ToolUse` | 工具卡（图标 + 目标 + 状态） | 点击展开 raw input/output |
| `ToolResult` | 工具卡（耗时 + 退出码） | 同上 |
| `PermissionRequest` | 权限弹层（优先级最高） | **Allow / Deny / Always** 三个按钮 |
| `Diff` | 文件 diff 卡片（折叠） | 接受 / 回滚 / 单文件查看 |
| `PlanMode` | 计划步骤列表 | Approve / Reject |
| `ContextWindow` | 顶栏上下文进度条 | 只读 |
| `UserInputRequest` | 选择/填空表单 | 单选 / 文本 / 提交 |
| `Error` | 错误条 | 重试按钮 |

### 7.3 卡片优先级与渲染

- 权限请求与 Plan Mode 抢顶层（toast + modal）
- Diff 折叠进"修改列表"抽屉，不抢聊天气泡流
- 工具调用是聊天气泡的子项（行内），可展开
- 错误条持久在底部，重试成功后消失

### 7.4 投影的状态机

```
{ Idle } ──(用户输入)──► { Streaming } ──(tool_use)──► { AwaitingPermission }
   ▲                          │                              │
   │                          ▼                              ▼
   │                     { Idle } ◄──(tool_result)──── { Running }
   │                          │
   └──────(session.stop)──────┘
```

- 状态变化通过 `SessionLifecycleEvent` 推送
- 前端 `core/state` 维护 currentSession + lifecycle 状态机

### 7.5 不做什么（克制）

- ❌ 不做 plan 多级嵌套
- ❌ 不做"自动继续" / "auto-accept"
- ❌ 不做 sub-agent / worktree 视图
- ❌ 不做 voice / TTS
- ❌ 不做 ADB 投屏

这样投影层保持 1000 行内 Go，前端 features 12 个以内。

---

## 8. 部署与启动

### 8.1 仓库结构

```
mytool/
├── package.json                # @<scope>/mytool 启动壳
├── README.md
├── LICENSE
├── .gitignore
├── cmd/server/                 # Go 入口
│   ├── main.go
│   └── flags.go
├── internal/                   # Go 业务（见 §2）
├── web/                        # Vite + React SPA（见 §3）
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
├── web/dist/                   # 构建产物（go:embed 目标）
├── scripts/
│   ├── build_web.sh            # 构建 SPA
│   ├── embed_web.sh            # 把 dist/ 复制到 cmd/server/web/
│   └── release.sh              # 跨平台编译 + npm pack
├── docs/
├── tests/                      # 集成测试
└── .github/workflows/ci.yml
```

### 8.2 npm 启动壳

```json
{
  "name": "@<scope>/mytool",
  "version": "0.1.0",
  "bin": { "mytool": "bin/mytool.js" },
  "scripts": {
    "build": "npm run build:web && npm run build:go",
    "build:web": "cd web && npm ci && npm run build && cd ..",
    "build:go": "bash scripts/embed_web.sh && go build -o bin/mytool ./cmd/server"
  }
}
```

`bin/mytool.js` 关键命令：

- `mytool start` — 选平台二进制、exec、转发 SIGTERM
- `mytool stop` — 读 pidfile、`kill -TERM`
- `mytool logs` / `mytool logs -f`
- `mytool status` — 查 healthz
- `mytool token` / `mytool qr` — 打印配对凭据
- `mytool config` — 打印 ~/.mytool/config.yaml
- `mytool uninstall` — 清理

### 8.3 跨平台发布

- Go build matrix：linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- npm `optionalDependencies` 选对应预编译二进制
- GitHub Actions：tag push → 编译 + npm publish

### 8.4 启动时序

```
mytool start
   │
   ├─ 1. 加载 ~/.mytool/config.yaml + 环境变量
   ├─ 2. 检查 workspace 路径存在性
   ├─ 3. 检查 ~/.mytool/auth/，缺失则首次启动流程
   ├─ 4. 初始化 store、加载 session 列表
   ├─ 5. 启动 engine registry（无副作用，只注册）
   ├─ 6. 启动 HTTPS + HTTP/2 server
   ├─ 7. 启动 WebSocket hub
   ├─ 8. 打印：URL + 二维码 + token（仅首次）
   └─ 9. 等待 SIGTERM
```

### 8.5 配置项

```yaml
# ~/.mytool/config.yaml
port: 8443
workspace: ~/mytool-workspace
default_command: claude
default_args: []
mtls: optional  # none | optional | required
log_level: info
log_redact: true
auth_dir: ~/.mytool/auth
store_dir: ~/.mytool/store
stall_watchdog:
  warn1: 60s
  warn2: 90s
  abort: 120s
  tool_abort: 10m
```

环境变量优先级最高：`PORT`、`MYTOOL_WORKSPACE`、`MYTOOL_DEFAULT_COMMAND`、`MYTOOL_MTLS`、`MYTOOL_LOG_LEVEL`

### 8.6 systemd 单元（可选）

```ini
[Unit]
Description=mytool
After=network.target

[Service]
Type=simple
Environment=MYTOOL_PORT=8443
Environment=MYTOOL_WORKSPACE=%h/mytool-workspace
ExecStart=%h/.local/bin/mytool start
Restart=on-failure

[Install]
WantedBy=default.target
```

---

## 9. 测试策略

### 9.1 后端 Go 测试

| 层级 | 工具 | 覆盖目标 |
|---|---|---|
| 单元 | 标准 `testing` + 表驱动 | `internal/projection`、`internal/auth`、`internal/store` 等纯函数包 |
| 集成 | `httptest` + `gorilla/websocket` | `internal/gateway`、`internal/ws` 端到端 |
| 端到端 | bash + `websocat` | `scripts/e2e_smoke.sh` |
| 协议 | golden file | 引擎流事件→投影事件的可重现快照 |

### 9.2 前端测试

| 层级 | 工具 | 覆盖目标 |
|---|---|---|
| 单元 | Vitest | `core/api/types` schema 校验、reducer |
| 组件 | Vitest + Testing Library | 关键卡片（Permission、Diff、Chat） |
| E2E | Playwright | PWA 登录 → 启动会话 → 权限审批 |

### 9.3 契约测试

- `core/api/types.ts` 与 `internal/protocol/event.go` 共享同一份 JSON schema（手动同步）
- 关键事件有 `protocol-contract.test.ts` / `protocol_contract_test.go` 双向校验
- CI 跑 golden file diff：引擎输出变 → 必须 review 才允许合并

### 9.4 安全测试

- `auth/token_test.go`：常量时间比较、token 强度（≥ 32B）
- `gateway/router_test.go`：CSRF 拒绝跨源 Origin
- `files/path_test.go`：路径穿越（`..`、符号链接）拒绝
- `files/denylist_test.go`：`.env` / `.git/` / `*.pem` 拦截

### 9.5 性能基准（可选）

- WS 单连接吞吐：1000 evt/s 不丢包
- 大 diff（10MB）首屏 < 1s
- 冷启动 < 1s（Go 二进制 + 嵌入式 SPA）

### 9.6 不做什么

- ❌ 不写 e2e 模拟"AI 真在思考"的脚本（无法稳定）
- ❌ 不做 fuzzing（个人项目成本不匹配）
- ❌ 不做 chaos test（同上）

---

## 10. 隐私与安全要点

### 10.1 与 MobileVC 风险对照表

| MobileVC 风险 | 本设计对策 | 实施位置 |
|---|---|---|
| `.env` 提交真实凭据 | 凭据目录 `~/.mytool/auth/` 加入 `.gitignore`；文档明确只放占位；首次启动脚本生成而非读取 | `scripts/embed_web.sh`；README 启动段 |
| `CheckOrigin` 永真 | 仅 `https://<lan-ip>:8443` 与 `https://localhost:8443` 通过；其余 403 | `internal/gateway/router.go` |
| 静态 token 多通道透传 | token 仅出现在 WS upgrade 阶段；TTS/文件下载不复用（这两个模块在本设计里直接不存在） | `internal/auth/middleware.go` |
| ws:// 直连（明文） | 全 HTTPS + mTLS 可选强制 | `cmd/server/main.go` |
| Relay 元数据泄露 | 没有 Relay | — |
| APNs 推送凭据泄露 | 没有推送通道；状态变化靠 WebSocket 实时 | — |
| ADB 暴露 | 没有 ADB 模块 | — |
| 文件路径穿越 | `filepath.Clean + EvalSymlinks` 强校验 | `internal/files/*` |
| `.env`/`.pem` 误读 | 显式 deny list | `internal/files/denylist.go` |
| 日志泄露 token | `logx/redact` 强制脱敏 api_key/token/Authorization | `internal/logx/redact.go` |
| 文件路径含 PII | 配置文件、日志输出只用 `~/` 缩写 | `cmd/server/main.go` 启动打印 |

### 10.2 强制安全不变量（CI 检查）

- 仓库根 `.gitignore` 必须包含 `.env`、`*.pem`、`*.key`、`*.p12`
- 启动脚本会校验 `~/.mytool/auth/` 权限为 0o700
- token 必须 ≥ 32 字节随机（启动期 `crypto/rand` 生成）
- `auth/middleware` 拒绝任何 `ws://` 升级（即便开发模式）
- 路径白名单 deny list 必须含 `.env`、`*.key`、`*.pem`、`*.p12`、`*.crt`、`.git/`、`node_modules/`

### 10.3 隐私不收集原则

- ❌ 不收集任何使用数据
- ❌ 不上传任何 AI 引擎输出/输入
- ❌ 不发请求到外部（除非用户自己接 TTS 等）
- ❌ 不接 npm/匿名遥测
- ❌ 不在文档里暗示有云服务

### 10.4 用户可控的"最严模式"（自用场景）

```bash
mytool start --port 8443 --mtls required --no-qr --bind 127.0.0.1
```

- `--mtls required`：必须客户端证书
- `--no-qr`：不打印二维码（防肩窥）
- `--bind 127.0.0.1`：只本地回环（外部穿透由用户自己 frp）

### 10.5 失败模式

- 凭据目录被误删 → 启动失败并打印"运行 `mytool reset-auth` 重新生成"（不自动重置）
- token 泄露（用户怀疑）→ `mytool rotate-token` 重生成
- 引擎崩溃 → stall watchdog 120s kill + ErrorEvent 推送
- 客户端 WS 断开 → 引擎继续运行，用户回来重连看到最新投影

---

## 11. 验收标准

### 11.1 功能完成

- [ ] `mytool start` 启动后浏览器扫码可登录
- [ ] 可启动 Claude Code / Codex / 通用 PTY 三种 Runner
- [ ] 权限弹窗在手机上以卡片呈现，Allow/Deny/Always 可点
- [ ] Diff 以文件级折叠卡片呈现，单文件可展开
- [ ] Plan Mode 显示步骤列表并支持 Approve/Reject
- [ ] 会话列表可加载、可续接
- [ ] 文件树、文件查看、文件下载工作
- [ ] Skills / Memory 列表工作
- [ ] stall watchdog 120s kill 引擎
- [ ] `mytool stop` 正常退出

### 11.2 安全完成

- [ ] 任何 `ws://` 升级被拒绝
- [ ] 任何跨源 Origin 被 403
- [ ] 路径穿越请求被拒
- [ ] deny list 拦截 `.env`/`.git/`/`*.pem`
- [ ] token 长度 ≥ 32B 且用 `crypto/rand` 生成
- [ ] `~/.mytool/auth/` 权限 0o700
- [ ] 日志中 token / Authorization 自动脱敏
- [ ] `npm pack` 后的发布包不含任何用户凭据

### 11.3 体验完成

- [ ] 冷启动 < 1s
- [ ] 手机浏览器冷启动 SPA 首屏 < 1.5s
- [ ] 离线重启 PWA 可加载最近一次会话快照
- [ ] 大 diff（10MB）首屏 < 1s
- [ ] iPhone Safari 14+ / Android Chrome 90+ 可用

### 11.4 文档完成

- [ ] README 含：快速开始、配置说明、命令清单、隐私声明
- [ ] 每个 internal 包有 godoc 注释
- [ ] `docs/security.md` 列明威胁模型与对策
- [ ] 故障排查：`docs/troubleshooting.md`

---

## 12. 实施阶段（建议顺序）

1. **MVP 1（1-2 周）**：Go 后端骨架 + PTY 通用 Runner + 鉴权 + 静态 SPA 占位 + WebSocket 协议
2. **MVP 2（1-2 周）**：ClaudeRunner + 投影 + React 聊天/权限/Diff 卡片
3. **MVP 3（1 周）**：CodexRunner + 会话续接 + Skills/Memory
4. **MVP 4（1 周）**：PWA + mTLS + 二维码配对 + npm 启动壳
5. **发布（持续）**：跨平台编译 + CI + npm publish

每个 MVP 结束都是一个可用的发布版本。
