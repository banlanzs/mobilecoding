# mobilecoding React Native 移动客户端设计

- 日期：2026-06-08
- 范围：架构设计文档，不含实现代码
- 目标：为 `mc claude` 模式设计原生移动应用客户端，使手机扫码连接后可像桌面 CLI 一样正常使用，且不影响桌面端继续使用。

---

## 1. 背景与目标

当前项目已具备：

- Go 后端：负责 HTTPS、WebSocket、会话管理、消息持久化。
- Web/PWA 前端：已实现消息渲染、权限审批、会话切换、断线重连。
- `mc claude` 遥控模式：手机可接管本机 Claude 会话，但当前主要依赖浏览器端体验。

本次设计目标是新增一个 **React Native 原生客户端**，满足以下要求：

1. **桌面 CLI 与手机双持**：桌面仍按原有 CLI 使用，手机扫码接入后可以独立查看和操作同一会话。
2. **手机端完整终端体验**：不是监控面板，而是结构化终端镜像风格客户端。
3. **双端同步存储**：桌面端与手机端均保留会话和消息记录。
4. **增量同步**：复用现有 `seq` + `after_seq` 机制，服务端为权威消息源。
5. **零后端协议改造优先**：优先直连现有 REST API 与 WebSocket 协议，降低风险。

---

## 2. 已确认决策

| 维度 | 决策 |
|---|---|
| 客户端技术栈 | React Native |
| 设计范围 | 架构设计文档 |
| 产品定位 | 桌面 CLI 与手机双持，手机可完整使用，不影响桌面 |
| 数据模型 | 双端同步存储 |
| 同步策略 | 增量同步（推荐） |
| 安全等级 | 复用现有 QR token 方案 |
| 选定方案 | 方案 A：原生薄客户端 |

---

## 3. 现有系统约束

### 3.1 协议约束

移动端必须复用现有协议，不引入第二套会话协议：

- WebSocket RPC 常量定义见 [internal/protocol/protocol.go](internal/protocol/protocol.go)
- Web 前端协议对照见 [web/src/core/ws/protocol.ts](web/src/core/ws/protocol.ts)
- WebSocket 客户端行为参考 [web/src/core/ws/ws-client.ts](web/src/core/ws/ws-client.ts)
- 状态管理与事件归并逻辑参考 [web/src/core/state/ChatContext.tsx](web/src/core/state/ChatContext.tsx)

### 3.2 服务端 API 约束

移动端依赖的现有端点见 [internal/gateway/router.go](internal/gateway/router.go)：

- `GET /api/v1/messages`
- `GET /api/v1/sessions`
- `GET /api/v1/models`
- `GET /api/v1/skills`
- `GET /api/v1/search`
- `GET /version`
- `GET /api/v1/ws`

### 3.3 存储约束

服务端消息已经按 `session_id + seq` 持久化，见 [internal/store/message.go](internal/store/message.go)。

这意味着移动端无需发明新的同步算法，只需：

1. 保存本地 `last_seq`
2. 启动或重连时调用 `after_seq`
3. 收到实时事件后按 `seq` 去重写入本地 SQLite

---

## 4. 方案对比

### 方案 A：原生薄客户端（选定）

**思路**：RN 客户端直接复用现有 Go 后端的 REST + WebSocket 能力，原生端负责扫码、Keychain、SQLite、UI 渲染。

**优点**：

- 零后端协议改动
- 原生扫码、通知、手势、后台恢复能力更强
- 与现有增量同步机制天然匹配
- 长期体验最佳

**缺点**：

- 需在 RN 中重建一套前端状态与 UI 组件
- 初版开发成本高于 WebView 壳

### 方案 B：WebView 混合壳

**思路**：用 RN 原生壳承载现有 Web 前端。

**优点**：

- 开发快
- 代码复用率高

**缺点**：

- 原生体验弱
- 本地存储、扫码、后台恢复都需要桥接
- 长期维护容易形成双层复杂度

### 方案 C：共享核心包

**思路**：提取现有 Web 协议和状态逻辑为共享 core，Web 与 RN 共用。

**优点**：

- 长期维护最优
- Web/RN 行为高度一致

**缺点**：

- 需要先重构当前 Web 结构
- 超出本次第一阶段目标

### 结论

选择 **方案 A：原生薄客户端**。

理由：它在不改变现有后端的前提下，最大化满足“手机像桌面 CLI 一样正常使用”的目标，同时保留后续向共享 core 演进的空间。

---

## 5. 总体架构

### 5.1 分层结构

```text
UI Layer (React Native Screens + Components)
    ↓
State Layer (Zustand stores)
    ↓
Service Layer (Auth / Network / Sync / Storage)
    ↓
Infrastructure Layer (SQLite / SecureStore / WebSocket / REST)
```

### 5.2 核心模块

| 模块 | 职责 |
|---|---|
| `AuthService` | 扫码、Deep Link 解析、token 保存、当前服务器切换 |
| `NetworkService` | WebSocket 连接、RPC 调用、REST 请求、重连控制 |
| `StorageService` | SQLite 表管理、消息落盘、分页查询、去重写入 |
| `SyncService` | 会话列表同步、按 `after_seq` 补拉、实时事件接入 |
| `useAuthStore` | 当前服务器、token、连接状态 |
| `useSessionStore` | 会话列表、当前活跃会话、历史会话只读视图 |
| `useMessageStore` | 消息流、权限弹窗、thinking、turn 状态、上下文窗口 |

### 5.3 数据流

#### 启动流程

```text
App Launch
→ 读取 SecureStore 中的当前服务器配置
→ 建立 WebSocket 连接
→ 请求 /version 与 /api/v1/sessions
→ 对会话执行增量同步
→ 渲染会话列表与最后活跃会话
```

#### 消息同步流程

```text
读取本地 last_seq
→ GET /api/v1/messages?after_seq=N
→ 批量写入 SQLite
→ 建立 WebSocket 实时订阅
→ 持续写入新事件并更新 last_seq
```

---

## 6. 网络层设计

### 6.1 WebSocket 连接

连接地址直接复用当前后端：

```text
wss://{host}:{port}/api/v1/ws?token={token}
```

Envelope 结构保持不变：

```json
{
  "type": "req" | "resp" | "evt",
  "id": "uuid",
  "method": "session.start",
  "params": {},
  "event": {},
  "sessionId": "..."
}
```

### 6.2 复用的 RPC 方法

- `session.start`
- `session.input`
- `session.abort`
- `session.stop`
- `session.permission.answer`
- `permission.respond`

其中权限协议保留双通道兼容：

- 旧 `session.permission.answer`：stdin 权限协议
- 新 `permission.respond`：HTTP hook 协议

### 6.3 REST 端点使用方式

| 端点 | 用途 |
|---|---|
| `/api/v1/messages?after_seq=` | 增量补拉 |
| `/api/v1/messages?before_seq=` | 历史分页 |
| `/api/v1/sessions` | 会话列表 |
| `/api/v1/models` | 模型选择 |
| `/api/v1/skills` | Skill 列表 |
| `/api/v1/search` | 历史搜索 |
| `/version` | 运行时默认命令、launchMode、cwd |

### 6.4 重连策略

沿用现有 Web 前端策略：

```text
[1s, 2s, 5s, 10s, 30s]
```

重连成功后：

1. 读取本地 `last_seq`
2. 通过 `/api/v1/messages?after_seq=last_seq` 补发缺失消息
3. 再次进入实时订阅

### 6.5 HTTPS / 自签证书

由于当前服务端使用自签 CA，RN 需要显式信任该证书链：

- Android：自定义 `network_security_config`
- iOS：ATS 白名单或内置 CA 证书信任

此项属于实现风险最高的基础设施点，应在最早的原型阶段验证，而不是拖到发布前。

---

## 7. 本地存储设计

### 7.1 SQLite 表结构

```sql
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  name TEXT,
  agent TEXT,
  model TEXT,
  cwd TEXT,
  status TEXT,
  created_at TEXT,
  updated_at TEXT,
  message_count INTEGER DEFAULT 0,
  synced INTEGER DEFAULT 0
);

CREATE TABLE messages (
  seq INTEGER,
  session_id TEXT,
  type TEXT,
  content TEXT,
  created_at TEXT,
  PRIMARY KEY (session_id, seq)
);

CREATE TABLE sync_state (
  session_id TEXT PRIMARY KEY,
  last_seq INTEGER DEFAULT 0,
  last_sync_at TEXT
);

CREATE TABLE device_config (
  key TEXT PRIMARY KEY,
  value TEXT
);
```

### 7.2 同步策略

#### 首次同步

- 拉取会话列表
- 对每个会话执行 `after_seq=0`
- 本地保存最新 500 条消息
- 写入 `sync_state.last_seq`

#### 增量同步

- 启动时对每个会话按 `last_seq` 增量拉取
- 重连成功后也执行同样流程

#### 实时同步

- 每个 `evt` 事件附带 `seq`
- 若 `seq <= last_seq`，则忽略
- 否则落盘并刷新 UI

### 7.3 容量管理

| 项目 | 限制 |
|---|---|
| 单会话本地缓存 | 2000 条 |
| 全局消息缓存 | 10,000 条 |
| 首次同步单会话上限 | 500 条 |

超过阈值时，优先清理最旧会话或最早消息，不影响最近活跃会话。

### 7.4 搜索策略

- **在线搜索**：调用服务端 `/api/v1/search`
- **离线搜索**：本地 `messages` 表执行轻量匹配

第一版优先做在线搜索，离线搜索作为增强能力保留扩展点。

---

## 8. 认证与扫码流程

### 8.1 现有二维码来源

二维码由服务端生成，逻辑见 [internal/auth/qr.go](internal/auth/qr.go)。当前内容本质上是带 token 的 URL。

### 8.2 移动端接入方式

扫码后分两种入口：

1. **直接解析 HTTPS URL**：提取 `token`、`host`、`port`
2. **转换为 App Deep Link**：统一走 `mobilecoding://connect?...`

推荐设计：扫码页同时支持两种格式，避免桌面端必须改变二维码内容。

### 8.3 token 存储

token 不存 SQLite，改存：

- iOS：Keychain
- Android：Keystore

字段：

- 当前服务器 token
- host
- port
- 最近连接服务器列表

### 8.4 多服务器支持

App 必须支持多个桌面实例切换：

- 保存多个服务器配置
- 每次只能有一个当前活跃连接
- 切换服务器时，断开当前 WS，切换到目标实例，并加载对应本地缓存

---

## 9. 状态管理设计

### 9.1 Store 划分

#### `useAuthStore`

负责：

- token
- host / port
- 连接状态
- 当前服务器切换

#### `useSessionStore`

负责：

- 会话列表
- 当前活跃会话 `activeSessionId`
- 当前查看会话 `viewedSessionId`
- 是否只读 `readOnly`

#### `useMessageStore`

负责：

- 消息列表
- `lastSeq`
- `permissionPrompt`
- `thinking`
- `turnActive`
- `agentState`
- `contextWindow`

### 9.2 与现有 Web 状态逻辑的映射

RN 不重新发明事件状态机，而是迁移以下关键逻辑：

- `text_delta` 按 `blockIndex` 归并
- `text` 替换最后一条 `text_delta`
- `permission_request` / `permission_ask` 去重
- `turn_end` 后复位 `turnActive`
- `context_window` 提取 token 使用量
- `agentState` 由事件类型推导

这部分现有逻辑来源于 [web/src/core/state/ChatContext.tsx](web/src/core/state/ChatContext.tsx)，应视为行为基准。

### 9.3 去重策略

双重去重：

1. `session_id + seq`：数据库级去重
2. `messageId`：内存级事件去重

这可覆盖：

- 重连补发
- 重复广播
- 页面恢复后的重复渲染

---

## 10. UI 结构设计

### 10.1 页面

- `SplashScreen`
- `OnboardingScreen`
- `SessionListScreen`
- `TerminalScreen`
- `SearchScreen`
- `SettingsScreen`

### 10.2 核心终端页面

`TerminalScreen` 结构：

1. Header：当前会话、连接状态、模型信息
2. Session Status Bar：Agent 状态、Context Window 进度
3. Message List：结构化消息卡片
4. Input Bar：输入框 + 发送 / 停止按钮

### 10.3 UI 原则

移动端是 **第二终端**，不是简化聊天 App，因此保留这些结构化元素：

- thinking 折叠块
- tool_use / tool_result 卡片
- lifecycle 事件
- plan mode 卡片
- permission prompt 卡片
- context window 指示

### 10.4 第一版非目标

- mTLS 设备证书完整接入
- Push Notification 远程唤醒
- 离线编辑回放
- 多账号体系
- 文件浏览器 / Git 变更浏览

---

## 11. 错误处理与边界

### 11.1 网络错误

| 场景 | 处理 |
|---|---|
| WS 断线 | 自动重连 + 增量补拉 |
| REST 失败 | 最多 3 次退避重试 |
| token 失效 | 清空凭据并返回扫码页 |
| 服务端离线 | 5 次重试后标记离线 |

### 11.2 双端并发

由于服务端是唯一写入源，桌面和手机都只是消费者，因此不存在消息写冲突。

唯一竞态点是 **权限审批**：

- 双端都可能显示同一个权限请求
- 服务端只接受第一个响应
- 后续响应收到 `stale_request`
- 客户端收到后应清空本地权限卡片

### 11.3 边界情况

- 单会话历史过大：首屏仅拉最新 500 条
- 事件乱序：统一按 `seq` 排序渲染
- App 被系统杀掉：下次启动走增量恢复
- SQLite 写入失败：降级为仅内存显示，并提醒用户缓存受限

---

## 12. 测试策略

### 12.1 单元测试

- `StorageService`：主键去重、分页、容量清理
- `NetworkService`：Envelope 编解码、RPC 错误映射
- `SyncService`：`after_seq` 补拉、实时合并
- `useMessageStore`：状态机与消息归并

### 12.2 集成测试

- Mock REST + WebSocket
- 断线重连与 missed messages 补发
- 权限竞争
- 会话切换

### 12.3 手工测试

- 扫码连接
- 桌面 CLI + 手机双持消息一致
- 手机审批权限后桌面立即继续
- 后台恢复后自动补拉
- 历史分页与搜索
- iOS / Android 自签证书信任

---

## 13. 目录结构建议

```text
src/
  screens/
    Onboarding/
    SessionList/
    Terminal/
    Search/
    Settings/
  stores/
    useAuthStore.ts
    useSessionStore.ts
    useMessageStore.ts
  services/
    AuthService.ts
    NetworkService.ts
    StorageService.ts
    SyncService.ts
  protocol/
    protocol.ts
    types.ts
  components/
    MessageCards/
    PermissionPrompt/
    StatusBar/
  utils/
    deeplink.ts
    logger.ts
```

---

## 14. 实施路线图

### Milestone 1：基础连接

- RN 工程初始化
- 扫码 / Deep Link
- Keychain / Keystore
- 基本 WS 连接

### Milestone 2：同步闭环

- SQLite schema
- 会话列表同步
- `after_seq` 增量同步
- 重连补发

### Milestone 3：终端体验

- TerminalScreen
- 消息卡片
- 输入与中止
- 权限审批

### Milestone 4：完善能力

- 搜索
- 多服务器切换
- 模型/Skill 查看
- 性能优化

### Milestone 5：平台验证

- iOS / Android 证书信任
- 后台恢复
- 内测分发

---

## 15. 风险与缓解

| 风险 | 影响 | 缓解 |
|---|---|---|
| iOS 自签证书信任困难 | 高 | 最早验证，必要时内置 CA 或改证书分发流程 |
| 大量消息渲染卡顿 | 中 | FlatList 虚拟化 + 分页 + 轻量卡片 |
| 后台 WebSocket 易断 | 中 | 重连后强制增量补拉，保证一致性 |
| Web 与 RN 行为偏差 | 中 | 以现有 `ChatContext` 行为为基线编写测试 |

---

## 16. 最终结论

本设计建议新增一个 **React Native 原生薄客户端**，直接复用现有 Go 后端的 WebSocket 协议与 REST API，通过本地 SQLite + `after_seq` 增量同步机制，在手机端提供接近桌面 CLI 的完整远程终端体验。

该方案：

- 满足桌面与手机双持
- 满足双端同步存储
- 避免后端大改
- 兼容现有 `mc claude` 使用方式
- 为后续抽取共享 core 留出空间
