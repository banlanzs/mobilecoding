# mobilecoding 修复说明

## 问题 1：Claude CLI 启动失败 (exit status 1)

### 症状
- 手机扫描二维码连接后，选择 Claude 配置文件启动
- 日志显示 `exit status 1`，Claude CLI 启动失败
- 发送的消息没有任何回复

### 根本原因
1. **缺少 `--verbose` 参数**：Claude CLI 的 `stream-json` 格式必须配合 `--verbose` 使用
2. **Windows 平台兼容性问题**：npm 全局安装的命令无法直接通过 `exec.Command` 启动

### 解决方案
- **commit a2876f9**: 修复 Windows 上 Claude CLI 启动失败问题
  - 添加 `--verbose` 参数到 Claude CLI 启动参数
  - Windows 平台使用 `cmd /c` 包装启动 npm 命令
  - 实现 lazy start 机制：首次 Write 时才启动进程
  - 添加 `runtime` 导入以支持跨平台检测

### 修改文件
- `internal/engine/claude_runner.go`
  - 添加 `--verbose` 到启动参数列表
  - Windows 检测：`runtime.GOOS == "windows"`
  - 使用 `cmd /c claude ...` 在 Windows 上启动

---

## 问题 2：手机页面收不到 Claude 回复内容

### 症状
- Claude CLI 能够启动（修复问题 1 后）
- 服务端日志显示事件正在转发
- 但手机页面上没有显示任何回复内容

### 根本原因
**WebSocket 事件分发架构缺陷**：

```
旧架构（错误）：
┌─────────────┐
│ session.    │
│ Manager     │──┐
│ Output()    │  │
└─────────────┘  │
                 ├──> WebSocket 连接 1 (读取事件 A)
                 ├──> WebSocket 连接 2 (读取事件 B)
                 └──> WebSocket 连接 3 (读取事件 C)

问题：多个 goroutine 从同一个 channel 读取，事件被分散消费
```

当多个客户端（电脑浏览器 + 手机）同时连接时：
- 每个连接都启动一个 `forwardSession` goroutine
- 所有 goroutine 从同一个 `session.Manager.Output()` channel 读取
- Go channel 的特性：一个消息只能被一个接收者消费
- 结果：事件随机分散到不同连接，手机只能收到部分事件

### 解决方案
- **commit 68100ae**: 修复 WebSocket 事件广播架构问题

实现 **Hub 广播模式**（1 生产者 → N 订阅者）：

```
新架构（正确）：
┌─────────────┐
│ session.    │
│ Manager     │
│ Output()    │
└──────┬──────┘
       │ (单一消费者)
       ↓
┌─────────────────────────┐
│ forwardSessionEvents    │
│ (全局事件转发器)        │
└──────────┬──────────────┘
           │ Hub.Broadcast()
           ↓
    ┌──────────────┐
    │     Hub      │
    └──┬────┬────┬─┘
       │    │    │
       ↓    ↓    ↓
     连接1 连接2 连接3
     (全部收到相同的事件)
```

### 修改文件

1. **cmd/server/main.go**
   - 添加 `forwardSessionEvents()` 函数：从 session.Manager 读取事件并广播
   - 在 `run()` 中启动全局事件转发 goroutine
   - 添加必要的导入：`engine`, `projection`

2. **internal/ws/handler.go**
   - 修改 `ServeConn()`：从 Hub 订阅事件，而不是直接读取 session.Manager
   - 移除独立的 `forwardSession()` 调用
   - 使用 `hub.Subscribe()` / `hub.Unsubscribe()`

3. **internal/ws/adapter.go**
   - 导出 `ProjectionToEnvelope()` 函数供 main.go 使用
   - 保留 `projectionToEnvelope()` 作为内部别名，向后兼容

---

## 功能增强：CLI 可视化终端窗口

### 需求
当手机选择 CLI 启动后，在电脑上弹出一个终端窗口显示 CLI 的实时输出，而不是隐式在后台运行。

### 解决方案
- **commit 1ef070c**: 添加 CLI 可视化终端窗口

当手机启动 Claude CLI 时：
1. 电脑上自动弹出一个 PowerShell 窗口
2. 窗口实时显示格式化的对话内容（不是原始 JSON）
3. 用户输入用 `[USER INPUT]` 标记
4. Claude 回复用 🤖 emoji 标记
5. 工具调用用 🔧 emoji 标记

### 窗口显示效果示例
```
=== MobileCoding Session: sess_xxx ===
Started at: 2026-06-03 19:47:04
==========================================

[USER INPUT] hello

🤖 Claude:
你好！有什么我可以帮助你的吗？

🔧 Tool: Bash
✅ Tool Result: Bash
```

### 修改文件
- `internal/engine/logwindow_windows.go` - Windows 日志窗口实现（PowerShell tail）
- `internal/engine/logwindow_other.go` - 非 Windows 平台占位符
- `internal/engine/claude_runner.go` - 集成日志窗口 + 格式化 stream-json 输出
- `internal/engine/runner.go` - ExecRequest 添加 `VisibleTerminal` 字段
- `internal/ws/handler.go` - 启动 session 时设置 `VisibleTerminal=true`

---

## 测试验证

### 测试步骤
1. 启动服务端：`./server.exe`
2. 电脑浏览器连接
3. 手机扫描二维码连接
4. 选择 Claude 配置文件
5. 发送消息 "hello"

### 预期结果
- Claude CLI 成功启动（不再有 exit status 1）
- 电脑上弹出 PowerShell 窗口，实时显示对话内容
- 电脑和手机**同时**收到相同的 Claude 回复内容
- 服务端日志显示事件广播到多个订阅者

---

## 参考

- **MobileVC 项目**：`Reference-Projects/MobileVC`
  - 参考了其 PTY Engine 和 Session Manager 的实现
  - 理解了正确的事件广播架构

- **Claude CLI 文档**：
  - `--output-format stream-json` 必须配合 `--verbose` 使用
  - `--input-format stream-json` 用于接收 JSON 格式的用户输入

---

## 架构重构：Relay 中继模式

### 背景
原架构（mobilecoding 直接启动 CLI 进程）存在以下问题：
- 终端窗口显示原始 JSON，不可读
- CLI 由 mobilecoding 管理，不自然
- 编码问题（中文显示乱码）

### 新架构
```
电脑上正常启动 CLI → 输入 /mobilecoding → 连接到 relay 服务器 ← 手机扫码连接
```

### 组件

| 组件 | 描述 | 文件 |
|------|------|------|
| Relay Server | WebSocket 中继服务器 | `internal/relay/` |
| Relay CLI | 连接到 relay 的命令行工具 | `cmd/relay/` |
| Web PWA | 手机端界面（支持 relay 模式） | `web/src/core/ws/relay-client.ts` |
| Claude Code Skill | `/mobilecoding` 命令 | `claude-skills/mobilecoding.md` |

### 工作流程

1. **启动 relay 服务器**
   ```bash
   mobilecoding
   ```

2. **在 CLI 中启动 relay 连接**
   ```bash
   mobilecoding-relay --relay ws://localhost:8443/relay/agent
   ```

3. **手机连接**
   - 打开 mobilecoding web 界面
   - 输入 session ID 和 pairing secret
   - 开始远程控制

### 使用方法

**方式 1：使用 mc 快捷命令（推荐）**
```bash
# 启动 relay 服务器
mobilecoding

# 在另一个终端运行
mc claude    # 启动 claude 并连接 relay
mc codex     # 启动 codex 并连接 relay
```

**方式 2：使用 Claude Code Skill**

在 Claude Code 中输入：
```
/mobilecoding
```

Claude 会自动运行 `mobilecoding-relay` 命令并显示配对信息。

**方式 3：手动运行 relay**
```bash
# 启动 relay 服务器
mobilecoding

# 在另一个终端运行
mobilecoding-relay --relay wss://localhost:8443/relay/agent --insecure
```

---

## 剩余工作

可能需要进一步测试和优化：
1. 多客户端并发场景
2. 网络断线重连
3. 长时间运行稳定性
4. QR 码扫描配对
5. 自动发现 relay 服务器（mDNS）

---

## npm link 兼容性

✅ **所有修复完全兼容 npm link 方式**

### 验证测试
运行 `./test-npm-link.sh` 验证：
```bash
./test-npm-link.sh
```

### 使用方式
```bash
# 1. 开发模式（使用 npm link）
npm link
mobilecoding              # 直接启动

# 2. 直接运行
npm run build
./dist/mobilecoding.exe   # 或 ./server.exe

# 3. 二进制分发
npm run build
# 将 dist/mobilecoding.exe 复制到目标机器
```

### 工作原理
- `bin/mobilecoding.js` 是 Node.js 启动器
- 自动查找并启动 Go 编译的二进制文件
- 支持 `dist/mobilecoding.exe` 和 `dist/mobilecoding-{platform}-{arch}` 多种命名
- 所有修复都在 Go 二进制中，与启动方式无关

### 测试结果
```
✓ mobilecoding 命令可用
✓ 版本: 0.1.0
✓ 帮助信息正常
✓ dist/mobilecoding.exe 存在 (大小: 11M)
✓ Claude CLI 启动失败问题已修复
✓ WebSocket 事件广播问题已修复
```

---
---

## 问题 4：权限申请弹窗无法拦截（HTTP Hook 修复）

### 症状
- 修复问题 2/3 后，工具过程和按钮切换都正常，但权限申请弹窗仍然不显示
- 之前的 commit 0ba7885 假设 Claude Code v2.x 支持 --permission-prompt-tool stdio flag + control_request 协议，但实测 v2.1.161 已移除该能力

### 根本原因
- v2.1.161 已移除 --permission-prompt-tool flag：旧的 stdio 权限协议废弃
- v2.1.161 不再发出 control_request 事件：Claude 不再以 stream-json 事件方式传递权限请求
- 之前实现的 ControlResponse 协议帧虽然格式正确，但永远不会收到对应的 control_request，等形同虚设

### 解决方案：Claude Code HTTP Hook（v2.1+ Feb 2026）
Claude Code v2.1.161 引入 type: http 的原生 hook（无需额外脚本），由 Claude CLI 直接 POST 到后端 URL。

#### 架构
[Claude CLI]
  -> POST /v1/hooks/permission-request ->
[mobilecoding 后端]
  -> broadcast hook.Event -> 所有 WS 客户端 ->
[手机端弹窗]
  -> 用户 Allow/Deny -> WS permission.respond ->
[mobilecoding 后端 hook.Registry.Respond()]
  -> HTTP 响应回 Claude CLI（含 permissionDecision）

#### 关键改动
- 新增 internal/hook/handler.go：HTTP 端点 /v1/hooks/permission-request
- 新增 internal/hook/settings.go：启动时把 hook 注入 ~/.claude/settings.json（幂等、可还原）
- 新增 WS 方法 permission.respond：手机端用 requestId 回应（与旧 session.permission.answer 并存）
- 后端 wsHandler.SetHookRegistry(reg)：桥接 HTTP handler 和 WS handler
- gateway.NewRouter 挂载 hook 端点：Bearer token 鉴权
- 去掉 --permission-prompt-tool stdio：该 flag 已不存在
- 前端 ChatContext.answerPermission：优先调 respondPermission，回退旧 answerPermission
- 前端 ws-client/relay-client：新增 respondPermission/sendRespondPermission 方法

#### 验证
- go test ./internal/hook/ 11/11 通过
- go test ./internal/projection/ 全过
- go test ./internal/engine/ 全过
- go build ./... 通过
- npx tsc --noEmit 通过
- npm run build 通过

---

## 问题 5：多行消息截断 + 工具调用/结果不可见 + 权限弹窗链路断

### 症状
修复问题 4（HTTP hook）后用户反馈三个新问题：
1. 手机发送多行消息（如 GitHub Actions 日志含 `[Pasted ~5 lines]`），CLI 只收到首行 `Run go test ./...`
2. 工具调用过程和结果在手机端完全不可见（只看到 `🧠 思考中…` 这类 lifecycle）
3. 权限弹窗仍不显示，且 Claude CLI 报告"未收到 allow/deny 响应"

### 根本原因
**问题 5.1（多行截断）**：`claude_runner.go` 把用户消息作为 argv 传给 `claude --print "..."`。Windows `CreateProcess` 把 `\n` 当作命令行分隔符 → 多行消息被截到第一行。

**问题 5.2（工具不可见）**：`internal/projection/raw.go` 的 `parseClaudeEvent` 假设 Claude 顶层发出 `tool_use`/`tool_result` 事件，但真实 Claude Code stream-json（Anthropic API Streams 格式）把工具调用作为 `content_block`（`type:"tool_use"`）嵌入 assistant 消息，工具结果作为 `user` 消息的 content 块。结果：
- `content_block_start` with `cbType: "tool_use"` → 在 line 290-292 被 `errors.New("skip ...")` 静默丢弃
- `content_block_delta` with `deltaType: "input_json_delta"` → 被丢弃
- `user` 消息完全无 case 处理，tool_result 永远到不了前端

**问题 5.3（权限链路断）**：`installClaudeHook` 写入 settings.json 的 URL 是 `http://127.0.0.1:8443/...`，但主服务器是 HTTPS-only（`ListenAndServeTLS`），Claude CLI 的 HTTP POST 在 HTTPS 端口上要么连接失败要么 TLS 错误，permission_request 永远到不了后端。

### 解决方案

#### Fix 5.1：消息写入 stdin（避免 Windows CreateProcess 截断）
- `claude_runner.go` 启动参数增加 `--input-format stream-json`，**不再**把 prompt 拼到 argv
- `Write(p)` 启动进程后用 `formatClaudeInput` 把消息封装为 `{"type":"user","message":{"role":"user","content":"..."}}` JSON 行写入 stdin
- `formatClaudeInput` 修正为 Claude --input-format stream-json 期望的标准格式

#### Fix 5.2：完整解析 Claude stream-json（content_block + user message）
- `raw.go` 把 `parseClaudeEvent` 重构为 `parseClaudeEventWithTracker(data, sid, pt) []Event`：
  - `content_block_start` with `cbType: "tool_use"` → 登记到 `pt.pendingToolUses[blockIndex]`，不立即 emit
  - `content_block_delta` with `deltaType: "input_json_delta"` → 累积 `partial_json` 到 pending
  - `content_block_stop` → 把累积的 input 解析为 JSON，emit `ToolUseEvent` + 在 `pt.toolUseIDs[id] = name` 建立映射
  - `user` message → 从 `message.content` 数组提取 `tool_result` 块，按 `tool_use_id` 反查工具名，emit `ToolResultEvent`
  - `parseClaudeEvent(data, sid)` 保留为无状态包装，兼容旧调用
- `PhaseTracker` 新增字段：`pendingToolUses map[int]*pendingToolUse`、`toolUseIDs map[string]string`
- 兼容旧 top-level `{"type":"tool_use",...}` 格式（部分老版本 Claude CLI 仍这样发）

#### Fix 5.3：Hook 改用独立 HTTP 监听器（避开 HTTPS 端口）
- `cmd/server/main.go` 新增 `startHookListener(cfg, hookHandler, logger)`：单独启动一个绑定 `127.0.0.1:<port>` 的 HTTP 监听器
  - 端口优先级：`MOBILECODING_HOOK_PORT` 环境变量 > 主端口 + 1（默认 8444）
  - 仅 127.0.0.1 可达 → 本地 IPC 足够安全，无需 TLS
  - Bearer 鉴权与主服务器共用 `cfg.AuthToken`
- `installClaudeHook(cfg, hookURL, logger)` 把 `startHookListener` 返回的真实 URL 写入 `~/.claude/settings.json`
- `internal/gateway/router.go` 移除 `/v1/hooks/permission-request` 路由（避免在 HTTPS 主路由上重复挂载）
- `Dependencies.HookHandler` 字段删除

#### 调试端点
- `GET /api/v1/hook-status` 返回 hook 注入状态：`{ installed, settingsPath, hookURL, hooks[] }`，便于手机端排查"权限弹窗不显示"问题

### 修改文件
- `internal/engine/claude_runner.go`：加 `--input-format stream-json`，Write 写 stdin
- `internal/engine/claude_runner_test.go`：测试新 JSON 格式 + 多行消息保留
- `internal/projection/raw.go`：重构 parseClaudeEvent 追踪 tool_use，添加 handleContentBlockStart/Delta/Stop + handleUserMessage
- `internal/projection/raw_test.go`：4 个新测试覆盖真实 stream-json 序列
- `internal/ws/handler_test.go` + `internal/session/manager_test.go`：补 mockRunner/fakeRunner 的 Abort + SendToStdin（之前预存在的 build 失败）
- `cmd/server/main.go`：startHookListener + pickHookPort + installClaudeHook 接收真实 URL
- `internal/gateway/router.go`：移除 HookHandler 字段 + /v1/hooks 路由；新增 /api/v1/hook-status

### 验证
- `go test ./...` 全部通过（含 4 个新增 projection 测试）
- `go build ./...` 通过
- `npx tsc --noEmit` 通过
- `npm run build` 通过

