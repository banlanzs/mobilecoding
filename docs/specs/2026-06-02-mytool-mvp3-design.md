# mytool MVP 3 设计 spec

| 字段 | 值 |
|---|---|
| 日期 | 2026-06-02 |
| 基于 | MVP 2（`2f4b499`） |
| 状态 | 设计已确认 |
| 范围 | 2 个功能：PWA 离线缓存 + Claude 投影深度解析 |

---

## 0. 背景

MVP 2 spec 列出 9 个后续建议。本 spec 聚焦前 2 个最高价值的功能：

1. **PWA 离线缓存** — 让手机浏览器在无网络时仍能访问最近一次会话快照
2. **Claude 投影深度解析** — 把 Claude stream-json 的 tool_use / permission_request / plan_mode 结构化为投影事件，而非当前的"透传 JSON"

---

## 1. PWA 离线缓存

### 1.1 目标

让手机浏览器在无网络时仍能：
- 打开 mytool SPA 壳
- 查看最近一次会话快照（最后 50 条事件）
- 断线重连后自动恢复

### 1.2 技术方案

使用 `vite-plugin-pwa` + Workbox：

```
web/
├── vite.config.ts
├── src/
│   ├── sw.ts              # Service Worker
│   ├── sw-sync.ts         # 后台同步
│   └── cache-strategy.ts  # 缓存策略
```

### 1.3 缓存策略

| 资源类型 | 策略 | 说明 |
|---|---|---|
| SPA 壳（index.html, main.js, style.css） | Cache First | 首次访问后缓存，离线可用 |
| /api/v1/ws | Network Only | WebSocket 不缓存 |
| /api/v1/healthz | Network Only | 健康检查不缓存 |
| 会话快照 | IndexedDB | 最后 50 条事件存 IndexedDB |

### 1.4 离线快照

Service Worker 监听 WebSocket 事件，把最近 50 条事件存入 IndexedDB：

```javascript
// sw.js
const CACHE_NAME = 'mytool-v1';
const SESSION_DB = 'mytool-sessions';

self.addEventListener('fetch', (event) => {
  if (event.request.url.includes('/api/v1/ws')) {
    // WebSocket 不缓存
    return;
  }
  event.respondWith(
    caches.match(event.request).then((response) => {
      return response || fetch(event.request);
    })
  );
});
```

### 1.5 不做什么

- ❌ 不做后台同步（MVP 3 只做离线查看，不做离线操作）
- ❌ 不做推送通知集成（MVP 4 任务）
- ❌ 不做 IndexedDB 压缩/清理（固定 50 条上限）

---

## 2. Claude 投影深度解析

### 2.1 目标

把 Claude stream-json 的 `tool_use` / `permission_request` / `plan_mode` 结构化为投影事件，而非当前的"透传 JSON"。

### 2.2 当前问题

MVP 2 的 `claude_parser.go` 只做 JSON 透传：

```go
func ParseClaudeStreamJSON(line []byte) (Event, error) {
    // 只验证 JSON，不解析内容
    return Event{Kind: EventRaw, Data: cp}, nil
}
```

导致 projection 层收到的是原始 JSON，无法区分：
- `tool_use` → 应显示为工具卡
- `permission_request` → 应显示为权限弹层
- `plan_mode` → 应显示为计划步骤列表

### 2.3 修复方案

在 `claude_parser.go` 中深度解析 JSON，输出结构化 `projection.Event`：

```go
// claude_parser.go
func ParseClaudeStreamJSON(line []byte) (Event, error) {
    var m map[string]any
    if err := json.Unmarshal(line, &m); err != nil {
        return Event{}, err
    }

    typ, _ := m["type"].(string)
    switch typ {
    case "assistant_message":
        return TextEvent("", m["message"].(string)), nil
    case "tool_use":
        return ToolUseEvent("", m["name"].(string), m["input"]), nil
    case "tool_result":
        return ToolResultEvent("", m["name"].(string), m["content"]), nil
    case "permission_request":
        return PermissionRequestEvent("", m["tool_name"].(string), m["prompt"].(string)), nil
    case "plan_mode":
        return PlanModeEvent("", m), nil
    case "context_window":
        return ContextWindowEvent("", m), nil
    case "session":
        return SessionEvent("", m), nil
    default:
        // 未知类型，透传
        return Event{Kind: EventRaw, Data: line}, nil
    }
}
```

### 2.4 新增投影事件类型

在 `projection/event.go` 中新增：

```go
const (
    EventText             EventType = "text"
    EventLifecycle        EventType = "lifecycle"
    EventToolUse          EventType = "tool_use"
    EventToolResult       EventType = "tool_result"
    EventPermissionReq    EventType = "permission_request"
    EventPlanMode         EventType = "plan_mode"
    EventContextWindow    EventType = "context_window"
    EventSession          EventType = "session"
)
```

### 2.5 ws handler 适配

`ws/handler.go` 的 `forwardSession` 需要处理新的事件类型，把它们序列化为对应的 `ws.Envelope`。

### 2.6 不做什么

- ❌ 不做 Codex 投影深度解析（MVP 4 任务）
- ❌ 不做 plan_mode 多级嵌套
- ❌ 不做 auto-accept

---

## 3. 验收标准

### 3.1 PWA 离线缓存

- [ ] `npm run build` 产出含 Service Worker
- [ ] 浏览器首次访问后，断网仍能打开 SPA 壳
- [ ] 离线时显示最近一次会话快照（最后 50 条事件）
- [ ] 重连后自动恢复实时事件流

### 3.2 Claude 投影深度解析

- [ ] `claude` command 的 `tool_use` 事件被解析为 `EventToolUse`
- [ ] `permission_request` 事件被解析为 `EventPermissionReq`
- [ ] ws handler 序列化后的 JSON 包含正确的 event type
- [ ] `go test ./...` 全部 PASS
- [ ] e2e smoke 通过

---

## 4. 不在范围

- ❌ 推送通知（APNs / Web Push）
- ❌ cert rotation / revocation
- ❌ 监控 + 日志聚合
- ❌ 配置热重载
- ❌ Skill / Memory 管理 UI
- ❌ mTLS 设备证书 PWA 持久化
- ❌ 二维码配对
- ❌ Codex 投影深度解析
