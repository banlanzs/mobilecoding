# mytool MVP 4 设计 spec

| 字段 | 值 |
|---|---|
| 日期 | 2026-06-02 |
| 基于 | MVP 3（`38e45d5`） |
| 状态 | 设计已确认 |
| 范围 | 3 个功能：Skill/Memory 管理 UI + mTLS 设备证书 PWA 持久化 + 二维码配对 |

---

## 0. 背景

MVP 2 spec 列出 9 个后续建议。本 spec 聚焦功能类 3-5：

1. **Skill / Memory 管理 UI** — 前端 SPA 实现 Skill/Memory 的查看、编辑、删除
2. **mTLS 设备证书 PWA 持久化** — 在 PWA IndexedDB 中持久化设备证书，实现 mTLS required 模式下的自动认证
3. **二维码配对** — 启动时生成二维码，手机扫码即可连接（替代手动输入 token）

---

## 1. Skill / Memory 管理 UI

### 1.1 目标

在前端 SPA 中实现 Skill/Memory 的管理界面，让用户可以：
- 查看 Skill 列表
- 查看 Memory 列表
- 编辑 Memory 内容
- 删除 Memory

### 1.2 后端 API

新增 REST API：

```
GET  /api/v1/skills          — 列出 Skills
GET  /api/v1/memory          — 列出 Memory
PUT  /api/v1/memory/:name    — 更新 Memory
DELETE /api/v1/memory/:name  — 删除 Memory
```

### 1.3 前端页面

```
web/src/features/
├── skills/
│   ├── SkillListPage.tsx
│   └── SkillCard.tsx
├── memory/
│   ├── MemoryListPage.tsx
│   ├── MemoryCard.tsx
│   └── MemoryEditor.tsx
```

### 1.4 不做什么

- ❌ 不做 Skill 编辑（Skill 是只读的）
- ❌ 不做 Memory 创建（Memory 由 Claude 自动生成）

---

## 2. mTLS 设备证书 PWA 持久化

### 2.1 目标

在 PWA IndexedDB 中持久化设备证书，实现 mTLS required 模式下的自动认证。

### 2.2 流程

1. 首次连接：服务器签发设备证书
2. 客户端把设备证书 + 私钥存入 IndexedDB
3. 后续连接：客户端自动使用设备证书进行 mTLS 认证

### 2.3 实现

```
web/src/core/
├── mtls/
│   ├── device-cert.ts      — 设备证书管理
│   ├── cert-storage.ts     — IndexedDB 存储
│   └── mtls-provider.ts    — mTLS 上下文 Provider
```

### 2.4 不做什么

- ❌ 不做证书轮换（MVP 4 只做首次签发 + 持久化）
- ❌ 不做证书撤销（远期）

---

## 3. 二维码配对

### 3.1 目标

启动时生成二维码，手机扫码即可连接（替代手动输入 token）。

### 3.2 流程

1. 服务器启动时生成二维码（包含 `https://<ip>:<port>/?token=<token>`）
2. 终端打印二维码（ASCII art）
3. 手机扫码 → 自动跳转到 mytool SPA → 自动填入 token → 连接

### 3.3 实现

```
internal/auth/
├── qr.go              — 二维码生成（ASCII art）
├── qr_test.go

cmd/server/
├── main.go            — 启动时打印二维码
```

### 3.4 不做什么

- ❌ 不做动态二维码刷新（token 不变）
- ❌ 不做二维码图片文件生成（只打印 ASCII art）

---

## 4. 验收标准

### 4.1 Skill / Memory 管理 UI

- [ ] `/api/v1/skills` 返回 Skill 列表
- [ ] `/api/v1/memory` 返回 Memory 列表
- [ ] `/api/v1/memory/:name` PUT 更新 Memory
- [ ] `/api/v1/memory/:name` DELETE 删除 Memory
- [ ] 前端页面显示 Skill/Memory 列表
- [ ] 前端页面支持编辑 Memory

### 4.2 mTLS 设备证书 PWA 持久化

- [ ] 首次连接时服务器签发设备证书
- [ ] 设备证书存入 IndexedDB
- [ ] 后续连接自动使用设备证书
- [ ] mTLS required 模式下自动认证成功

### 4.3 二维码配对

- [ ] 启动时终端打印二维码（ASCII art）
- [ ] 二维码包含 `https://<ip>:<port>/?token=<token>`
- [ ] 手机扫码可跳转到 mytool SPA

---

## 5. 不在范围

- ❌ 推送通知（APNs / Web Push）
- ❌ cert rotation / revocation
- ❌ 监控 + 日志聚合
- ❌ 配置热重载
- ❌ npm 包发布
- ❌ GitHub Actions CI/CD
- ❌ 跨平台预编译二进制
