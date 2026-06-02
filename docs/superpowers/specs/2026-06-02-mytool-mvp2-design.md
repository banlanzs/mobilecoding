# mytool MVP 2 设计 spec

| 字段 | 值 |
|---|---|
| 日期 | 2026-06-02 |
| 基于 | MVP 1.1（`815c021`） |
| 状态 | ✅ 已完成（MVP 2 合并到 main） |
| 范围 | 4 个 followup：ClaudeRunner + CodexRunner + files/store + stall watchdog + yourname 替换 |

---

## 0. 背景

MVP 1.1 final review 列出 6 个 followup，前 2 个（TLS + projection）已完成。本 spec 覆盖剩余 4 个：

1. **ClaudeRunner** — 解析 `claude --output-format stream-json --verbose` 输出，提取结构化事件
2. **CodexRunner** — 解析 `codex app-server` JSON-RPC，提取结构化事件
3. **files / store** — 工作区文件浏览、下载、denylist、原子重命名存储
4. **Stall watchdog** — session.Manager.forward 中的沉默计时器（120s kill）
5. **yourname 替换** — 替换 `github.com/yourname/mytool` 为真实 module path

---

## 1. ClaudeRunner

### 1.1 目标

替代当前 registry 中的 PtyRunner fallback，当 command 是 `claude` / `claude-code` / `claude --resume <id>` 时使用 ClaudeRunner。

### 1.2 协议

`claude --output-format stream-json --verbose` 每行输出一个 JSON 对象：

```json
{"type":"assistant_message","message":"..."}
{"type":"tool_use","name":"Bash","input":{"command":"ls"}}
{"type":"tool_result","name":"Bash","content":"file1\nfile2","exit_code":0}
{"type":"permission_request","tool_name":"Bash","prompt":"Allow this bash command?"}
{"type":"context_window","used_tokens":1234,"max_tokens":200000}
{"type":"session","session_id":"sess_abc123"}
```

### 1.3 实现

```
internal/engine/claude_parser.go   — 行解析器：JSON line → engine.Event
internal/engine/claude_runner.go   — ClaudeRunner：PTY 启动 + stream-json 解析
internal/engine/claude_parser_test.go
internal/engine/claude_runner_test.go
```

关键：`ClaudeRunner.Start()` 启动 `claude --output-format stream-json --verbose`，`readLoop` 按行解析 JSON，转为 `engine.Event`。

### 1.4 与 projection 的关系

ClaudeRunner 的 `Events()` 通道输出 `engine.Event`（与 PtyRunner 一致），projection.Stream 负责转为结构化事件。

### 1.5 不做什么

- ❌ 不解析 `claude --output-format json`（非 stream 版本）
- ❌ 不解析 `claude --output-format text`（纯文本）
- ❌ 不做 skill/memory 管理（MVP 3）

---

## 2. CodexRunner

### 2.1 目标

当 command 是 `codex` 时使用 CodexRunner，通过 JSON-RPC over stdio 与 codex app-server 交互。

### 2.2 协议

`codex app-server` JSON-RPC：

```json
{"method":"thread/started","params":{"thread_id":"..."}}
{"method":"item/agentMessage/delta","params":{"delta":"..."}}
{"method":"toolCall","params":{"name":"bash","input":"ls"}}
{"method":"item/toolResult","params":{"content":"file1\nfile2"}}
```

### 2.3 实现

```
internal/engine/codex_transport.go — JSON-RPC over stdio
internal/engine/codex_runner.go    — CodexRunner
internal/engine/codex_model_catalog.go — 模型目录（可选）
internal/engine/codex_transport_test.go
internal/engine/codex_runner_test.go
```

### 2.4 不做什么

- ❌ 不做 app-server daemon 模式
- ❌ 不做 remote-control

---

## 3. files / store

### 3.1 目标

提供工作区文件浏览（tree）、文件读取、文件下载，以及会话历史的原子重命名存储。

### 3.2 实现

```
internal/files/tree.go      — 文件树（限制深度、忽略 .git）
internal/files/read.go      — 单文件读取（白名单后缀）
internal/files/download.go  — 携带鉴权的文件下载
internal/files/denylist.go  — .env / *.key / *.pem / *.p12 / *.crt / .git/ / node_modules/

internal/store/filestore.go — 原子重命名存储
internal/store/sessions.go  — 会话历史落盘
internal/store/skills.go    — Skill 仓库
internal/store/memory.go    — Memory 仓库
```

### 3.3 denylist

显式拒绝：

- `.env`
- `*.pem`
- `*.key`
- `*.p12`
- `*.crt`
- `.git/`
- `node_modules/`

### 3.4 不做什么

- ❌ 不做文件上传
- ❌ 不做文件搜索（仅 tree + read）

---

## 4. Stall watchdog

### 4.1 目标

在 session.Manager.forward 中加计时器：120s 沉默 → kill runner，防止引擎挂死。

### 4.2 实现

在 `session/manager.go` 的 `forward` 中：

```go
func (m *Manager) forward(run engine.Runner) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    lastActivity := time.Now()

    for {
        select {
        case ev, ok := <-run.Events():
            if !ok { return }
            lastActivity = time.Now()
            // ... existing event forwarding ...
        case <-ticker.C:
            if time.Since(lastActivity) > 120*time.Second {
                // Kill runner
                run.Close()
                return
            }
        }
    }
}
```

### 4.3 不做什么

- ❌ 不做可配置阈值（硬编码 120s）
- ❌ 不做工具执行宽限（10min，MVP 2 不支持）

---

## 5. yourname 占位替换

### 5.1 目标

把 `github.com/yourname/mytool` 替换为真实 org/repo。

### 5.2 实现

机械替换：
- `go.mod` 中的 module path
- 所有 `.go` 文件中的 import 路径
- 测试文件中的 import 路径

### 5.3 真实 module path

待用户确认。暂定 `github.com/jaycrl/mytool`（基于 MobileVC 的 org `JayCRL`）。

---

## 6. 验收标准

- [ ] `claude` command 启动 ClaudeRunner
- [ ] `codex` command 启动 CodexRunner
- [ ] `go test ./...` 全部 PASS
- [ ] stall watchdog 120s kill 有效
- [ ] yourname 占位替换完成
- [ ] e2e smoke 通过
