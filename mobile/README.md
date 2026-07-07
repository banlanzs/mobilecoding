# mobilecoding — React Native 客户端

mobilecoding 的 Android/iOS 原生客户端，与 [`../web`](../web) 的 PWA 平行，连接同一个 Go 后端。

## 功能

- 扫码 / Deep Link / 手动输入连接后端（`https://电脑IP:8443`）
- WebSocket 流式输出，断线自动重连并按 `after_seq` 补发缺失消息
- 本地 SQLite（`@op-engineering/op-sqlite`）持久化会话与消息
- Zustand 复刻 Web 端 `ChatContext` 的状态机与事件归并逻辑
- 会话列表、搜索、设置、Onboarding 引导
- 权限应答、请求中止、模型切换

## 目录结构

```
src/
  navigation/    导航栈（AppNavigator）
  screens/       页面：Onboarding / QRScanner / SessionList / Terminal / Search / Settings / Splash
  components/    终端组件：MessageCard / GitDiffModal
  protocol/      协议常量与类型（与 web/src/core/ws 保持同步）
  services/
    auth/        token / host / port / Deep Link / 扫码归一化
    network/     WSClient / RestClient / ConnectionAdapter
    storage/     Database / Session / Message / SyncState 仓储
    sync/        SyncService 同步与归并
  stores/        Zustand：useAuthStore / useSessionStore / useMessageStore
  types/         server-profile
```

## 开发

```bash
npm install

# Android
npm run android

# iOS（首次需安装 Pods）
bundle exec pod install --project-directory=ios
npm run ios

# Metro 调试服务器
npm start
```

## 测试与检查

```bash
npm test           # jest
npm run lint       # eslint
npm run typecheck  # tsc --noEmit
```

## 连接后端

先在本机启动后端（见根目录 [README](../README.md)）：

```bash
make build
./dist/mobilecoding.exe claude
```

后端启动后会打印二维码与 token，App 扫码即可连接。真机需与本机同一局域网，或通过 relay 中继远程连接。
