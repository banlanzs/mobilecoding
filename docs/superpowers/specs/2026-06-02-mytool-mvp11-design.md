# mytool MVP 1.1 — TLS / mTLS + projection.Stream 接入

| 字段 | 值 |
|---|---|
| 日期 | 2026-06-02 |
| 基于 | MVP 1（commit `684aae2`） |
| 状态 | 设计已确认，待实现 |
| 范围 | **只做前 2 个 followup**（其余 4 个 followup 列入远景，本 spec 不实施） |

---

## 0. 背景

MVP 1 final review 列出 6 个 followup。本 spec 聚焦前 2 个最高价值的：

1. **TLS / mTLS** — 当前 MVP 1 用 HTTP 明文，spec §5.3 / §10.1 明确要求 HTTPS
2. **projection.Stream 接入** — 当前 ws handler 手动构造事件，bypass 了 projection 层（spec §6.3 设计意图被破坏）

后 4 个 followup（ClaudeRunner/CodexRunner / files+store / stall watchdog / yourname 占位替换）保留在"远景"节，不在本次范围。

---

## 1. Followup 1：TLS / mTLS

### 1.1 目标

MVP 1.1 阶段把后端从 HTTP 升级到 HTTPS（强制），并支持 mTLS 鉴权（可选）。落地 spec §5.3 三档模式：

| 模式 | 含义 | 实施 |
|---|---|---|
| `none`（暂不开放，MVP 1.1 不实现） | 纯 HTTP | 暂不支持，避免开发时明文 |
| `optional`（**MVP 1.1 默认**） | 强制 HTTPS，但不要求客户端证书 | 自签 CA + server 证书；客户端用 Bearer 鉴权 |
| `required`（**MVP 1.1 落地**） | 强制 HTTPS + 客户端证书 | 在 `optional` 基础上加 `VerifyClientCertIfGiven` + 自签设备证书签发流程 |

### 1.2 启动时流程（参考 spec §5.1）

```
[mytool start]
  │
  ├─ 检查 ~/.mytool/auth/：
  │    ├─ ca.crt / ca.key 缺失 → 新建 CA（10 年，RSA-2048）
  │    └─ server.crt / server.key 缺失 → 用 CA 签发 server 证书
  │         SAN: <lan-ip>, localhost, 127.0.0.1
  │
  ├─ 打印：本地 URL、二维码、token
  │
  └─ 启动 HTTPS server（mTLS optional 模式）
```

### 1.3 鉴权层次（spec §5.4 + MVP 1.1 新增）

| 通道 | 鉴权 | 说明 |
|---|---|---|
| `/healthz` `/version` | mTLS 通道保护（optional 模式不要求客户端证书） | 健康检查仍可匿名 |
| `/`（SPA 静态） | mTLS 通道保护 | SPA 资源 |
| `/api/v1/ws` | mTLS 通道 + Bearer token | ws 升级 |

### 1.4 mTLS 设备证书签发（required 模式）

- 客户端首次扫码后，服务端自动签发设备证书
- 设备证书落 `~/.mytool/auth/devices/<device-id>.{crt,key}`
- key 用设备 token hash 加密保存（passphrase = SHA-256(token)）
- 客户端（PWA IndexedDB）保存设备证书，后续 WS 升级用 client cert 做 mutual TLS
- **MVP 1.1 简化**：PWA IndexedDB 持久化与 mTLS required 模式**留到 MVP 4**。本阶段只实现 `optional` 模式（强制 HTTPS + Bearer 鉴权）+ `required` 模式的服务端校验（拒无证书的连接）

### 1.5 不做什么（克制）

- ❌ 不在 MVP 1.1 实现 PWA 端 client cert 持久化
- ❌ 不做二维码配对（spec §5.1）— MVP 4 任务
- ❌ 不做 cert rotation / revocation（spec §5.3 远期）
- ❌ 不做 hostname verification custom logic（用 stdlib 默认）
- ❌ 不支持 `none` 模式（避免明文）

### 1.6 关键模块

```
internal/auth/
  ca.go             // 生成自签 CA (RSA-2048, 10 年)
  servercert.go     // 用 CA 签发 server 证书（含 SAN）
  devicecert.go     // [MVP 1.1 stub] 设备证书签发（仅骨架）
  mtls.go (扩展)    // client cert 校验（CN 提取等）

cmd/server/main.go (扩展)
  buildConfig 中按 MTLS 字段切换 server cert 路径
  run() 中 ListenAndServe → ListenAndServeTLS
  启动期一次性流程（CA + server cert 生成）
```

### 1.7 配置

```yaml
# ~/.mytool/config.yaml
port: 8443
mtls: optional  # optional | required（MVP 1.1 强制必填 optional 或 required）
```

启动参数：

- `--mtls=optional`（**MVP 1.1 默认**）
- `--mtls=required`（强制 client cert）

### 1.8 测试

- `auth/ca_test.go`：CA 生成的证书能被自己验证；不同 CA 互不信任
- `auth/servercert_test.go`：server 证书 SAN 含预期 IP/host
- `auth/servercert_test.go`：server 证书过期检测
- 端到端：`curl -k https://127.0.0.1:8443/healthz` 应返回 200
- `curl -k https://...` 加 `--cacert ca.crt` 也应返回 200
- `mtls=required` 模式下无 client cert 的 WS 升级应 4xx

### 1.9 验收

```bash
# 1. 启动
./mytool --mtls=optional
# 看到：自签 CA 生成提示 + server 证书生成提示 + 监听 :8443

# 2. 客户端：浏览器弹"证书不受信任" → 用户接受
# 3. curl -k 通过
curl -k https://127.0.0.1:8443/healthz
# → ok

# 4. mTLS=required 模式
./mytool --mtls=required
curl -k https://127.0.0.1:8443/api/v1/ws
# → 4xx（无 client cert）
```

---

## 2. Followup 2：projection.Stream 接入

### 2.1 目标

把 `internal/projection.Stream` 真正接到 ws handler 的事件流里，删除 handler.go 内的 inline `json.Marshal` 构造。落地 spec §6.3 "投影与 CLI 协议的解耦" 设计意图。

### 2.2 当前问题

`internal/ws/handler.go:48-65` 当前的 `forwardSession`：

```go
func (h *Handler) forwardSession(ctx context.Context, _ chan Envelope) {
    for {
        select {
        case <-ctx.Done():
            return
        case ev, ok := <-h.mgr.Output():
            if !ok { return }
            raw, _ := json.Marshal(map[string]any{
                "type":      "text",
                "text":      string(ev.Data),
                "sessionId": h.mgr.SessionID(),
            })
            h.hub.Broadcast(Envelope{Type: "evt", SessionID: h.mgr.SessionID(), Event: raw})
        }
    }
}
```

问题：
1. 手动 json.Marshal 构造 text event（spec §4.4 应由 projection 层负责）
2. sessionId 三处独立生成（`session.Manager.sid` / `projection.Project.sid` / `projection.Stream.sid`）—— 谁拥有真相未明
3. `projection.Stream` 是死代码（`grep projection\.Stream` 在 repo 零匹配）
4. Hub.Subscribe/Unsubscribe 调用但 channel 未使用（final review m 项）

### 2.3 修复方案

#### 2.3.1 projection 包

**新增**：让 `Stream` 接受 session id 参数：

```go
func Stream(input <-chan engine.Event, output chan<- Event, sessionID string)
```

或保留原签名 + 内部用 `mgr.SessionID()` 注入。建议方案 A：让 caller 传入 sid，因为 caller 知道 session 的真相。

**新增**：`projection.LifecycleEvent(...)` 与 `projection.TextEvent(...)` 构造器，让 event 构造逻辑集中到 projection 包。

**修正**：`Project` 接收 sid 参数（替换内部 uuid 生成）。

#### 2.3.2 session 包

**不变**：`Manager.sid` 是真相，projection 层只接收它做透传。

#### 2.3.3 ws handler

**重构** `forwardSession`：

```go
func (h *Handler) forwardSession(ctx context.Context, sub chan Envelope) {
    // 把 mgr.Output 透传到 sub
    // sub 已经在 ServeConn 中被 hub.Subscribe() 注册
    // 这里只需要把 mgr.Output 投到 sub（不是 hub.Broadcast）
    sid := h.mgr.SessionID()
    projection.Stream(h.mgr.Output(), sub, sid)
}
```

但 `projection.Stream` 当前签名是 `(input, output chan<- projection.Event)`——它输出的是 `projection.Event` 而非 `ws.Envelope`。需要 adapter：

```go
// 在 ws 包内做 adapter（保持 projection 包对 ws 包零依赖）
func projectionToEnvelope(p projection.Event) ws.Envelope {
    raw, _ := json.Marshal(p)
    return ws.Envelope{Type: "evt", SessionID: p.SessionID, Event: raw}
}
```

或修改 `projection.Stream` 的输出类型为 `json.RawMessage`（更通用）。

#### 2.3.4 实施选择

**方案 A**（推荐）：保留 `projection.Event` 类型，在 ws handler 内做 adapter
- 优点：projection 包不依赖 ws 包
- 缺点：多一个 adapter 函数

**方案 B**：修改 `projection.Stream` 输出 `json.RawMessage`
- 优点：解耦更彻底
- 缺点：失去类型安全

**方案 C**：让 projection 包直接定义 `Envelope`，与 ws 包共享
- 优点：消除重复
- 缺点：跨包耦合

**选定**：方案 A。

### 2.4 关键模块

```
internal/projection/
  event.go (扩展)  // 添加 TextEvent / LifecycleEvent 构造器
  raw.go (重构)   // Project 接受 sid 参数；Stream 输出 chan Event（不变）
  raw_test.go (扩展)  // 补 Project 与 sid 的测试

internal/ws/
  handler.go (重构)  // 用 projection.Stream 替换 inline json.Marshal
  adapter.go (新增)  // projection.Event → ws.Envelope 转换
  adapter_test.go (新增)  // 转换正确性测试
```

### 2.5 测试

- `projection/raw_test.go`：补 `Project(in, sid)` 测试，验证事件使用传入 sid
- `ws/adapter_test.go`：验证 text / lifecycle projection event 转换正确
- `ws/handler_test.go`（已有）：dispatch 已覆盖 session.start/input/stop，新增 `TestForwardSessionUsesProjection` 验证事件流走向 projection 层

### 2.6 验收

```bash
# 启动后用 websocat 测：
websocat "ws://127.0.0.1:8443/api/v1/ws?token=$TOKEN"
# → 发送 {"type":"req","id":"r1","method":"session.start","params":{"command":"echo","args":["hi"]}}
# → 期望收到 {"type":"resp","id":"r1","ok":true,"result":{"sessionId":"sess_xxx"}}
# → 然后收到 {"type":"evt","sessionId":"sess_xxx","event":{"type":"text","text":"hi\n","sessionId":"sess_xxx"}}
# → 然后收到 {"type":"evt","sessionId":"sess_xxx","event":{"type":"lifecycle","message":"exited"}}
```

文本事件结构必须由 projection 包产生，而非 handler inline 构造。

---

## 3. 验收标准（合并两项 followup）

### 3.1 功能

- [ ] `mytool start` 第一次启动生成自签 CA + server 证书
- [ ] `curl -k https://127.0.0.1:8443/healthz` 返回 200
- [ ] `curl --cacert ca.crt https://...` 也能验证
- [ ] `--mtls=required` 模式下无 client cert 的 WS 升级被 4xx 拒
- [ ] ws handler 内的 `json.Marshal` 内联 event 构造被删除
- [ ] `projection.Stream` 真正被消费（grep 有命中）
- [ ] `Stream` 输出的 event 在 `event.sessionId` 与 `Envelope.sessionId` 上一致

### 3.2 安全

- [ ] 强 HTTPS（无明文 HTTP 监听）
- [ ] server 证书 SAN 含运行主机 IP + localhost
- [ ] CA 与 server 私钥文件权限 0o600
- [ ] 自签 CA 不被分发到生产（仅个人本机）
- [ ] `mtls=optional` 模式不依赖客户端证书，浏览器可用

### 3.3 代码

- [ ] 6 个包测试包（auth / config / engine / gateway / logx / projection / session / ws）全 PASS
- [ ] `go vet ./...` 干净
- [ ] `go build ./cmd/server` 产出 9.7MB+ 二进制
- [ ] e2e smoke 通过（升级到 https 后需要 `curl -k`）

### 3.4 文档

- [ ] README 更新：启动示例用 https URL
- [ ] 启动日志清晰说明自签 CA 路径
- [ ] 提交信息符合 Conventional Commits

---

## 4. 不在范围

明确排除，避免范围漂移：

- ❌ ClaudeRunner / CodexRunner（MVP 2/3 任务）
- ❌ files / store 包（MVP 2 任务）
- ❌ Stall watchdog timer（MVP 2 任务）
- ❌ 替换 `yourname` 占位 module path（用户发布前手动做）
- ❌ mTLS 设备证书 PWA 持久化（MVP 4 任务）
- ❌ 二维码配对（MVP 4 任务）
- ❌ cert rotation / revocation（远期）

---

## 5. 实施阶段（建议）

1. **阶段 D-A（4-6 task）**：TLS / mTLS
   - Task 14: internal/auth/ca.go（CA 生成 + 加载/创建）
   - Task 15: internal/auth/servercert.go（用 CA 签发 server 证书）
   - Task 16: cmd/server 启动流程集成（ListenAndServeTLS + 配置切换）
   - Task 17: mTLS required 模式（VerifyClientCertIfGiven）
   - Task 18: e2e smoke 升级到 https

2. **阶段 D-B（2-3 task）**：projection.Stream 接入
   - Task 19: projection 重构（Project 接受 sid；新增 TextEvent / LifecycleEvent 构造器）
   - Task 20: ws handler 重构（用 projection.Stream + adapter）
   - Task 21: 验证 grep 与 e2e 行为

预计 8-9 task，约 60-80 step，约 25-30 测试用例。
