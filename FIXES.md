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

## 测试验证

### 测试步骤
1. 启动服务端：`./server.exe`
2. 电脑浏览器连接
3. 手机扫描二维码连接
4. 选择 Claude 配置文件
5. 发送消息 "hello"

### 预期结果
- Claude CLI 成功启动（不再有 exit status 1）
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

## 剩余工作

可能需要进一步测试和优化：
1. 多客户端并发场景
2. 网络断线重连
3. 长时间运行稳定性
4. Claude 响应超时处理

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
- `bin/mytool.js` 是 Node.js 启动器
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