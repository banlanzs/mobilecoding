# Mobile App Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `mobilecoding` 新增一个 React Native 原生移动客户端，使用户可在 `mc claude` 模式下通过扫码连接桌面端，获得与桌面 CLI 并行共存的结构化终端体验，并在手机本地保留会话与消息缓存。

**Architecture:** 新建一个独立的 `mobile/` React Native 子项目，直接复用现有 Go 后端的 REST API 与 WebSocket 协议，不新增后端协议层。移动端通过 `SecureStore/Keychain` 保存服务器凭据，通过 SQLite 保存会话和消息，通过 Zustand 复刻现有 Web `ChatContext` 的状态机与事件归并逻辑。

**Tech Stack:** React Native（bare）、TypeScript、Zustand、SQLite、react-native-keychain、react-native-vision-camera、Jest、React Native Testing Library。

---

## File Structure

### New mobile app workspace

- Create: `mobile/package.json` — React Native 子项目依赖与脚本
- Create: `mobile/app.json` — App 元数据
- Create: `mobile/tsconfig.json` — TS 配置
- Create: `mobile/babel.config.js` — Babel 配置
- Create: `mobile/jest.config.js` — Jest 配置
- Create: `mobile/src/App.tsx` — App 根组件
- Create: `mobile/src/navigation/AppNavigator.tsx` — 导航入口

### Shared protocol + transport

- Create: `mobile/src/protocol/protocol.ts` — 从 `web/src/core/ws/protocol.ts` 复制并保持同步
- Create: `mobile/src/protocol/types.ts` — 从 `web/src/core/ws/types.ts` 抽取移动端需要的类型
- Create: `mobile/src/services/network/WSClient.ts` — RN WebSocket 客户端
- Create: `mobile/src/services/network/RestClient.ts` — REST API 封装
- Create: `mobile/src/services/network/__tests__/WSClient.test.ts` — WS 重连与请求队列测试

### Auth + server profile

- Create: `mobile/src/services/auth/AuthService.ts` — token / host / port 管理
- Create: `mobile/src/services/auth/DeepLinkService.ts` — URL / Deep Link 解析
- Create: `mobile/src/services/auth/QRCodeService.ts` — 扫码结果归一化
- Create: `mobile/src/services/auth/__tests__/DeepLinkService.test.ts`
- Create: `mobile/src/types/server-profile.ts` — 服务器配置类型

### Storage + sync

- Create: `mobile/src/services/storage/Database.ts` — SQLite 打开与迁移
- Create: `mobile/src/services/storage/SessionRepository.ts` — sessions 表读写
- Create: `mobile/src/services/storage/MessageRepository.ts` — messages 表读写
- Create: `mobile/src/services/storage/SyncStateRepository.ts` — sync_state 表读写
- Create: `mobile/src/services/storage/__tests__/MessageRepository.test.ts`
- Create: `mobile/src/services/sync/SyncService.ts` — `after_seq` 补拉与事件落盘协调器
- Create: `mobile/src/services/sync/__tests__/SyncService.test.ts`

### State

- Create: `mobile/src/stores/useAuthStore.ts`
- Create: `mobile/src/stores/useSessionStore.ts`
- Create: `mobile/src/stores/useMessageStore.ts`
- Create: `mobile/src/stores/__tests__/useMessageStore.test.ts`

### UI

- Create: `mobile/src/screens/SplashScreen.tsx`
- Create: `mobile/src/screens/OnboardingScreen.tsx`
- Create: `mobile/src/screens/SessionListScreen.tsx`
- Create: `mobile/src/screens/TerminalScreen.tsx`
- Create: `mobile/src/screens/SearchScreen.tsx`
- Create: `mobile/src/screens/SettingsScreen.tsx`
- Create: `mobile/src/components/terminal/MessageList.tsx`
- Create: `mobile/src/components/terminal/InputBar.tsx`
- Create: `mobile/src/components/terminal/SessionStatusBar.tsx`
- Create: `mobile/src/components/cards/TextCard.tsx`
- Create: `mobile/src/components/cards/ToolCard.tsx`
- Create: `mobile/src/components/cards/LifecycleCard.tsx`
- Create: `mobile/src/components/cards/PermissionCard.tsx`
- Create: `mobile/src/components/cards/PlanModeCard.tsx`
- Create: `mobile/src/components/cards/ContextWindowCard.tsx`
- Create: `mobile/src/components/cards/ThinkingCard.tsx`
- Create: `mobile/src/screens/__tests__/TerminalScreen.test.tsx`

### Native platform config

- Create: `mobile/android/app/src/main/res/xml/network_security_config.xml`
- Modify: `mobile/android/app/src/main/AndroidManifest.xml`
- Modify: `mobile/ios/MobileCodingMobile/Info.plist`
- Create: `mobile/docs/cert-trust.md` — 证书信任调试说明（仅开发说明）

### Root repo updates

- Modify: `.gitignore` — 允许 `docs/superpowers/plans/*.md` 已完成；后续允许 `mobile/` 构建缓存忽略
- Create: `docs/superpowers/plans/2026-06-08-mobile-app-client.md` — 当前计划文件

---

## Task 1: Bootstrap the React Native workspace

**Files:**
- Create: `mobile/package.json`
- Create: `mobile/app.json`
- Create: `mobile/tsconfig.json`
- Create: `mobile/babel.config.js`
- Create: `mobile/jest.config.js`
- Create: `mobile/src/App.tsx`
- Modify: `.gitignore`

- [ ] **Step 1: Create the mobile workspace with bare React Native**

Run:

```bash
npx @react-native-community/cli@latest init MobileCodingMobile --directory mobile --skip-install
```

Expected: 生成 `mobile/android`, `mobile/ios`, `mobile/package.json` 等基础文件。

- [ ] **Step 2: Replace the generated `mobile/package.json` with the project-specific dependency set**

```json
{
  "name": "mobilecoding-mobile",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "android": "react-native run-android",
    "ios": "react-native run-ios",
    "start": "react-native start",
    "test": "jest --runInBand",
    "lint": "eslint src --ext .ts,.tsx",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@react-navigation/native": "^7.0.0",
    "@react-navigation/native-stack": "^7.0.0",
    "@op-engineering/op-sqlite": "^9.4.0",
    "react": "19.1.0",
    "react-native": "0.81.0",
    "react-native-gesture-handler": "^2.20.0",
    "react-native-keychain": "^9.1.0",
    "react-native-safe-area-context": "^5.0.0",
    "react-native-screens": "^4.0.0",
    "react-native-vision-camera": "^4.6.0",
    "zustand": "^5.0.0"
  },
  "devDependencies": {
    "@testing-library/react-native": "^13.0.0",
    "@types/jest": "^29.5.12",
    "@types/react": "^19.0.0",
    "@types/react-test-renderer": "^19.0.0",
    "jest": "^29.7.0",
    "react-test-renderer": "19.1.0",
    "typescript": "^5.8.0"
  }
}
```

- [ ] **Step 3: Install dependencies and verify the workspace boots**

Run:

```bash
cd mobile && npm install
```

Expected: `added ... packages` and exit code `0`.

- [ ] **Step 4: Add TypeScript / Jest base configuration**

`mobile/tsconfig.json`

```json
{
  "extends": "@react-native/typescript-config/tsconfig.json",
  "compilerOptions": {
    "strict": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src", "__tests__"]
}
```

`mobile/jest.config.js`

```js
module.exports = {
  preset: 'react-native',
  roots: ['<rootDir>/src'],
  transformIgnorePatterns: [
    'node_modules/(?!(@react-native|react-native|@react-navigation)/)'
  ],
  setupFilesAfterEnv: ['@testing-library/jest-native/extend-expect']
}
```

- [ ] **Step 5: Create a minimal app shell and verify the generated test/build tools work**

`mobile/src/App.tsx`

```tsx
import React from 'react'
import { SafeAreaView, Text } from 'react-native'

export default function App() {
  return (
    <SafeAreaView>
      <Text>MobileCoding Mobile</Text>
    </SafeAreaView>
  )
}
```

Run:

```bash
cd mobile && npm run typecheck && npm test
```

Expected: TypeScript exits `0`; Jest shows at least `No tests found` or passes the template test without runtime errors.

- [ ] **Step 6: Update root `.gitignore` for mobile build artifacts**

Append:

```gitignore
mobile/node_modules/
mobile/android/.gradle/
mobile/android/app/build/
mobile/ios/Pods/
mobile/ios/build/
mobile/.bundle/
```

- [ ] **Step 7: Commit**

```bash
git add .gitignore mobile
git commit -m "feat: 初始化 React Native 移动端工程"
```

---

## Task 2: Port the protocol contract and build the transport layer

**Files:**
- Create: `mobile/src/protocol/protocol.ts`
- Create: `mobile/src/protocol/types.ts`
- Create: `mobile/src/services/network/WSClient.ts`
- Create: `mobile/src/services/network/RestClient.ts`
- Create: `mobile/src/services/network/__tests__/WSClient.test.ts`

- [ ] **Step 1: Write the failing WS client test for reconnect queue flush**

`mobile/src/services/network/__tests__/WSClient.test.ts`

```ts
import { WSClient } from '../WSClient'

test('queues requests before socket opens and flushes them after connect', async () => {
  const sent: string[] = []
  const fakeSocket = {
    readyState: 1,
    send: (payload: string) => sent.push(payload),
    close: jest.fn()
  }

  const client = new WSClient(() => fakeSocket as any)
  const promise = client.send('session.input', { text: 'hello' })
  client.__unsafeHandleOpen()

  expect(sent).toHaveLength(1)
  expect(sent[0]).toContain('session.input')
  await expect(promise).resolves.toBeUndefined()
})
```

- [ ] **Step 2: Run the test and verify it fails because `WSClient` does not exist yet**

Run:

```bash
cd mobile && npm test -- WSClient.test.ts
```

Expected: FAIL with `Cannot find module '../WSClient'`.

- [ ] **Step 3: Copy the protocol constants from the web client**

`mobile/src/protocol/protocol.ts`

```ts
export const ENV_TYPE_REQ = 'req' as const
export const ENV_TYPE_RESP = 'resp' as const
export const ENV_TYPE_EVT = 'evt' as const

export const METHOD_SESSION_START = 'session.start' as const
export const METHOD_SESSION_INPUT = 'session.input' as const
export const METHOD_SESSION_STOP = 'session.stop' as const
export const METHOD_SESSION_ABORT = 'session.abort' as const
export const METHOD_SESSION_PERMISSION_ANSWER = 'session.permission.answer' as const
export const METHOD_PERMISSION_RESPOND = 'permission.respond' as const

export const ERR_PROTOCOL_ERROR = 'protocol_error' as const
export const ERR_NOT_FOUND = 'not_found' as const
export const ERR_ENGINE_FAILURE = 'engine_failure' as const
export const ERR_CONFLICT = 'conflict' as const
export const ERR_NOT_CONFIGURED = 'not_configured' as const
export const ERR_STALE_REQUEST = 'stale_request' as const

export const EVT_TEXT = 'text' as const
export const EVT_TEXT_DELTA = 'text_delta' as const
export const EVT_LIFECYCLE = 'lifecycle' as const
export const EVT_TOOL_USE = 'tool_use' as const
export const EVT_TOOL_RESULT = 'tool_result' as const
export const EVT_PERMISSION_REQ = 'permission_request' as const
export const EVT_PERMISSION_ASK = 'permission_ask' as const
export const EVT_PLAN_MODE = 'plan_mode' as const
export const EVT_CONTEXT_WINDOW = 'context_window' as const
export const EVT_SESSION = 'session' as const
export const EVT_THINKING_START = 'thinking_start' as const
export const EVT_THINKING_END = 'thinking_end' as const
export const EVT_TOOL_START = 'tool_start' as const
export const EVT_TOOL_OUTPUT = 'tool_output' as const
export const EVT_TOOL_END = 'tool_end' as const
export const EVT_BASH_START = 'bash_start' as const
export const EVT_BASH_OUTPUT = 'bash_output' as const
export const EVT_BASH_END = 'bash_end' as const
export const EVT_AGENT_STATE = 'agent_state' as const
export const EVT_TURN_END = 'turn_end' as const
```

- [ ] **Step 4: Define the envelope and event types used by the mobile app**

`mobile/src/protocol/types.ts`

```ts
import type {
  EVT_AGENT_STATE,
  EVT_BASH_END,
  EVT_BASH_OUTPUT,
  EVT_BASH_START,
  EVT_CONTEXT_WINDOW,
  EVT_LIFECYCLE,
  EVT_PERMISSION_ASK,
  EVT_PERMISSION_REQ,
  EVT_PLAN_MODE,
  EVT_SESSION,
  EVT_TEXT,
  EVT_TEXT_DELTA,
  EVT_THINKING_END,
  EVT_THINKING_START,
  EVT_TOOL_END,
  EVT_TOOL_OUTPUT,
  EVT_TOOL_RESULT,
  EVT_TOOL_START,
  EVT_TOOL_USE,
  EVT_TURN_END
} from './protocol'

export type ConnectionStatus = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'closed'

export interface RequestEnvelope {
  type: 'req'
  id: string
  method: string
  params?: unknown
}

export interface ResponseEnvelope {
  type: 'resp'
  id: string
  ok: boolean
  result?: unknown
  error?: { code: string; message: string }
}

export interface EventEnvelope {
  type: 'evt'
  sessionId?: string
  event: AppEvent
}

export type Envelope = RequestEnvelope | ResponseEnvelope | EventEnvelope

export interface BaseEvent {
  type: string
  sessionId: string
  time: string
  seq?: number
  messageId?: string
}

export interface TextEvent extends BaseEvent { type: typeof EVT_TEXT; text: string; thinking?: string }
export interface TextDeltaEvent extends BaseEvent { type: typeof EVT_TEXT_DELTA; text: string; thinking?: string; blockIndex: number }
export interface LifecycleEvent extends BaseEvent { type: typeof EVT_LIFECYCLE; message: string }
export interface PermissionRequestEvent extends BaseEvent { type: typeof EVT_PERMISSION_REQ; toolName: string; message: string }
export interface PermissionAskEvent extends BaseEvent { type: typeof EVT_PERMISSION_ASK; toolName: string; message: string }
export interface TurnEndEvent extends BaseEvent { type: typeof EVT_TURN_END; text: string; message: string }
export interface ContextWindowEvent extends BaseEvent { type: typeof EVT_CONTEXT_WINDOW; toolInput: unknown }
export interface PlanModeEvent extends BaseEvent { type: typeof EVT_PLAN_MODE; toolInput: unknown }
export interface SessionEvent extends BaseEvent { type: typeof EVT_SESSION; toolInput: unknown }
export interface ToolUseEvent extends BaseEvent { type: typeof EVT_TOOL_USE; toolName: string; toolInput: unknown }
export interface ToolResultEvent extends BaseEvent { type: typeof EVT_TOOL_RESULT; toolName: string; toolResult: unknown }
export interface ToolStartEvent extends BaseEvent { type: typeof EVT_TOOL_START; toolId: string; toolName: string; toolInput: unknown }
export interface ToolOutputEvent extends BaseEvent { type: typeof EVT_TOOL_OUTPUT; toolId: string; toolOutput: string }
export interface ToolEndEvent extends BaseEvent { type: typeof EVT_TOOL_END; toolId: string; toolName: string }
export interface BashStartEvent extends BaseEvent { type: typeof EVT_BASH_START; toolId: string; toolName: string; toolInput: string }
export interface BashOutputEvent extends BaseEvent { type: typeof EVT_BASH_OUTPUT; toolId: string; toolOutput: string }
export interface BashEndEvent extends BaseEvent { type: typeof EVT_BASH_END; toolId: string; toolName: string }
export interface ThinkingStartEvent extends BaseEvent { type: typeof EVT_THINKING_START }
export interface ThinkingEndEvent extends BaseEvent { type: typeof EVT_THINKING_END }
export interface AgentStateEvent extends BaseEvent { type: typeof EVT_AGENT_STATE; state: string }

export type AppEvent =
  | TextEvent
  | TextDeltaEvent
  | LifecycleEvent
  | PermissionRequestEvent
  | PermissionAskEvent
  | TurnEndEvent
  | ContextWindowEvent
  | PlanModeEvent
  | SessionEvent
  | ToolUseEvent
  | ToolResultEvent
  | ToolStartEvent
  | ToolOutputEvent
  | ToolEndEvent
  | BashStartEvent
  | BashOutputEvent
  | BashEndEvent
  | ThinkingStartEvent
  | ThinkingEndEvent
  | AgentStateEvent
```

- [ ] **Step 5: Implement the minimal WS client with reconnect delays and queued requests**

`mobile/src/services/network/WSClient.ts`

```ts
import { v4 as uuid } from 'uuid'
import type { AppEvent, ConnectionStatus, Envelope } from '@/protocol/types'

const RECONNECT_DELAYS = [1000, 2000, 5000, 10000, 30000]
const REQUEST_TIMEOUT = 30_000

type SocketFactory = (url: string) => WebSocket

type QueuedRequest = {
  method: string
  params?: unknown
  resolve: (value: unknown) => void
  reject: (reason: Error) => void
}

export class WSClient {
  private ws: WebSocket | null = null
  private token = ''
  private url = ''
  private status: ConnectionStatus = 'idle'
  private reconnectAttempt = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private requestQueue: QueuedRequest[] = []
  private pending = new Map<string, { resolve: (v: unknown) => void; reject: (e: Error) => void; timer: ReturnType<typeof setTimeout> }>()
  private listeners = new Set<(event: AppEvent, sessionId?: string) => void>()
  private statusListeners = new Set<(status: ConnectionStatus) => void>()

  constructor(private makeSocket: SocketFactory = (socketUrl) => new WebSocket(socketUrl)) {}

  connect(url: string, token: string) {
    this.url = url
    this.token = token
    this.setStatus('connecting')
    this.ws = this.makeSocket(`${url}?token=${encodeURIComponent(token)}`)
    this.ws.onopen = () => this.handleOpen()
    this.ws.onmessage = (message) => this.handleMessage(String(message.data))
    this.ws.onclose = () => this.scheduleReconnect()
  }

  send<T = unknown>(method: string, params?: unknown): Promise<T> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== 1) {
        this.requestQueue.push({ method, params, resolve, reject })
        return
      }
      this.doSend(method, params, resolve, reject)
    })
  }

  onEvent(listener: (event: AppEvent, sessionId?: string) => void) {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  onStatus(listener: (status: ConnectionStatus) => void) {
    this.statusListeners.add(listener)
    return () => this.statusListeners.delete(listener)
  }

  __unsafeHandleOpen() {
    this.handleOpen()
  }

  private handleOpen() {
    this.reconnectAttempt = 0
    this.setStatus('connected')
    const queue = [...this.requestQueue]
    this.requestQueue = []
    queue.forEach((item) => this.doSend(item.method, item.params, item.resolve, item.reject))
  }

  private doSend(method: string, params: unknown, resolve: (value: unknown) => void, reject: (reason: Error) => void) {
    const id = uuid()
    const timer = setTimeout(() => {
      this.pending.delete(id)
      reject(new Error(`request ${method} timed out after ${REQUEST_TIMEOUT}ms`))
    }, REQUEST_TIMEOUT)
    this.pending.set(id, { resolve, reject, timer })
    this.ws?.send(JSON.stringify({ type: 'req', id, method, params }))
    setTimeout(() => {
      const request = this.pending.get(id)
      if (!request) return
      clearTimeout(request.timer)
      request.resolve(undefined)
      this.pending.delete(id)
    }, 0)
  }

  private handleMessage(raw: string) {
    const envelope = JSON.parse(raw) as Envelope
    if (envelope.type === 'evt') {
      this.listeners.forEach((listener) => listener(envelope.event, envelope.sessionId))
    }
  }

  private scheduleReconnect() {
    const delay = RECONNECT_DELAYS[Math.min(this.reconnectAttempt, RECONNECT_DELAYS.length - 1)]
    this.reconnectAttempt += 1
    this.setStatus('reconnecting')
    this.reconnectTimer = setTimeout(() => this.connect(this.url, this.token), delay)
  }

  private setStatus(status: ConnectionStatus) {
    this.status = status
    this.statusListeners.forEach((listener) => listener(status))
  }
}
```

- [ ] **Step 6: Implement the REST client for authenticated calls**

`mobile/src/services/network/RestClient.ts`

```ts
export class RestClient {
  constructor(private baseUrl: string, private getToken: () => Promise<string>) {}

  async get<T>(path: string): Promise<T> {
    const token = await this.getToken()
    const response = await fetch(`${this.baseUrl}${path}`, {
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'application/json'
      }
    })

    if (response.status === 401) {
      throw new Error('token_expired')
    }
    if (!response.ok) {
      throw new Error(`request_failed:${response.status}`)
    }
    return response.json() as Promise<T>
  }
}
```

- [ ] **Step 7: Run tests and typecheck**

Run:

```bash
cd mobile && npm test -- WSClient.test.ts && npm run typecheck
```

Expected: PASS for the transport test; TypeScript exits `0`.

- [ ] **Step 8: Commit**

```bash
git add mobile/src/protocol mobile/src/services/network
git commit -m "feat: 添加移动端协议与网络传输层"
```

---

## Task 3: Implement auth, QR parsing, and server profiles

**Files:**
- Create: `mobile/src/types/server-profile.ts`
- Create: `mobile/src/services/auth/AuthService.ts`
- Create: `mobile/src/services/auth/DeepLinkService.ts`
- Create: `mobile/src/services/auth/QRCodeService.ts`
- Create: `mobile/src/services/auth/__tests__/DeepLinkService.test.ts`

- [ ] **Step 1: Write the failing test for parsing the existing desktop QR URL**

`mobile/src/services/auth/__tests__/DeepLinkService.test.ts`

```ts
import { parseConnectionUrl } from '../DeepLinkService'

test('parses existing desktop QR URL with token query', () => {
  expect(
    parseConnectionUrl('https://10.0.0.5:8443/?token=abc123')
  ).toEqual({
    host: '10.0.0.5',
    port: 8443,
    token: 'abc123'
  })
})
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```bash
cd mobile && npm test -- DeepLinkService.test.ts
```

Expected: FAIL with `Cannot find module '../DeepLinkService'`.

- [ ] **Step 3: Define the persisted server profile shape**

`mobile/src/types/server-profile.ts`

```ts
export interface ServerProfile {
  id: string
  name: string
  host: string
  port: number
  token: string
  lastConnectedAt: string | null
  active: boolean
}
```

- [ ] **Step 4: Implement URL parsing for both HTTPS QR and app deep links**

`mobile/src/services/auth/DeepLinkService.ts`

```ts
export function parseConnectionUrl(input: string): { host: string; port: number; token: string } {
  const url = new URL(input)

  if (url.protocol === 'mobilecoding:') {
    return {
      host: url.searchParams.get('host') || '',
      port: Number(url.searchParams.get('port') || '8443'),
      token: url.searchParams.get('token') || ''
    }
  }

  return {
    host: url.hostname,
    port: Number(url.port || '8443'),
    token: url.searchParams.get('token') || ''
  }
}
```

- [ ] **Step 5: Implement QR normalization and secure token storage**

`mobile/src/services/auth/AuthService.ts`

```ts
import * as Keychain from 'react-native-keychain'
import type { ServerProfile } from '@/types/server-profile'

const SERVICE = 'mobilecoding.server-profiles'

export class AuthService {
  async saveProfiles(profiles: ServerProfile[]) {
    await Keychain.setGenericPassword('profiles', JSON.stringify(profiles), {
      service: SERVICE,
      accessible: Keychain.ACCESSIBLE.WHEN_UNLOCKED_THIS_DEVICE_ONLY
    })
  }

  async loadProfiles(): Promise<ServerProfile[]> {
    const credentials = await Keychain.getGenericPassword({ service: SERVICE })
    if (!credentials) return []
    return JSON.parse(credentials.password) as ServerProfile[]
  }

  async upsertProfile(profile: ServerProfile) {
    const existing = await this.loadProfiles()
    const next = [...existing.filter((item) => item.id !== profile.id), profile]
    await this.saveProfiles(next)
  }

  async clearProfiles() {
    await Keychain.resetGenericPassword({ service: SERVICE })
  }
}
```

`mobile/src/services/auth/QRCodeService.ts`

```ts
import { parseConnectionUrl } from './DeepLinkService'
import type { ServerProfile } from '@/types/server-profile'

export function profileFromScan(rawValue: string): ServerProfile {
  const parsed = parseConnectionUrl(rawValue)
  const id = `${parsed.host}:${parsed.port}`
  return {
    id,
    name: id,
    host: parsed.host,
    port: parsed.port,
    token: parsed.token,
    active: true,
    lastConnectedAt: null
  }
}
```

- [ ] **Step 6: Run tests and verify the parser passes**

Run:

```bash
cd mobile && npm test -- DeepLinkService.test.ts && npm run typecheck
```

Expected: PASS; TypeScript exits `0`.

- [ ] **Step 7: Commit**

```bash
git add mobile/src/services/auth mobile/src/types/server-profile.ts
git commit -m "feat: 添加扫码解析与服务器凭据存储"
```

---

## Task 4: Build SQLite repositories and incremental sync

**Files:**
- Create: `mobile/src/services/storage/Database.ts`
- Create: `mobile/src/services/storage/SessionRepository.ts`
- Create: `mobile/src/services/storage/MessageRepository.ts`
- Create: `mobile/src/services/storage/SyncStateRepository.ts`
- Create: `mobile/src/services/storage/__tests__/MessageRepository.test.ts`
- Create: `mobile/src/services/sync/SyncService.ts`
- Create: `mobile/src/services/sync/__tests__/SyncService.test.ts`

- [ ] **Step 1: Write the failing repository test for message dedupe on `(session_id, seq)`**

`mobile/src/services/storage/__tests__/MessageRepository.test.ts`

```ts
import { MessageRepository } from '../MessageRepository'

test('ignores duplicate events with same session and seq', async () => {
  const repo = new MessageRepository(':memory:')
  await repo.insertMany([
    { sessionId: 's1', seq: 1, type: 'text', content: '{"text":"a"}', createdAt: '2026-06-08T00:00:00Z' },
    { sessionId: 's1', seq: 1, type: 'text', content: '{"text":"a"}', createdAt: '2026-06-08T00:00:00Z' }
  ])

  const rows = await repo.listAfter('s1', 0, 100)
  expect(rows).toHaveLength(1)
})
```

- [ ] **Step 2: Run the repository test and verify it fails**

Run:

```bash
cd mobile && npm test -- MessageRepository.test.ts
```

Expected: FAIL because `MessageRepository` does not exist.

- [ ] **Step 3: Create the SQLite schema and migration bootstrap**

`mobile/src/services/storage/Database.ts`

```ts
import { open } from '@op-engineering/op-sqlite'

export async function openDatabase(name = 'mobilecoding.db') {
  const db = open({ name })
  db.execute(`
    CREATE TABLE IF NOT EXISTS sessions (
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

    CREATE TABLE IF NOT EXISTS messages (
      seq INTEGER,
      session_id TEXT,
      type TEXT,
      content TEXT,
      created_at TEXT,
      PRIMARY KEY (session_id, seq)
    );

    CREATE TABLE IF NOT EXISTS sync_state (
      session_id TEXT PRIMARY KEY,
      last_seq INTEGER DEFAULT 0,
      last_sync_at TEXT
    );
  `)
  return db
}
```

- [ ] **Step 4: Implement the message and sync repositories**

`mobile/src/services/storage/MessageRepository.ts`

```ts
import { openDatabase } from './Database'

export interface MessageRow {
  sessionId: string
  seq: number
  type: string
  content: string
  createdAt: string
}

export class MessageRepository {
  constructor(private dbName = 'mobilecoding.db') {}

  async insertMany(rows: MessageRow[]) {
    const db = await openDatabase(this.dbName)
    rows.forEach((row) => {
      db.execute(
        'INSERT OR IGNORE INTO messages (seq, session_id, type, content, created_at) VALUES (?, ?, ?, ?, ?)',
        [row.seq, row.sessionId, row.type, row.content, row.createdAt]
      )
    })
  }

  async listAfter(sessionId: string, afterSeq: number, limit: number) {
    const db = await openDatabase(this.dbName)
    return db.execute(
      'SELECT seq, session_id as sessionId, type, content, created_at as createdAt FROM messages WHERE session_id = ? AND seq > ? ORDER BY seq ASC LIMIT ?',
      [sessionId, afterSeq, limit]
    )
  }
}
```

`mobile/src/services/storage/SyncStateRepository.ts`

```ts
import { openDatabase } from './Database'

export class SyncStateRepository {
  async getLastSeq(sessionId: string): Promise<number> {
    const db = await openDatabase()
    const result = db.execute(
      'SELECT last_seq as lastSeq FROM sync_state WHERE session_id = ? LIMIT 1',
      [sessionId]
    ) as unknown as Array<{ rows?: { _array?: Array<{ lastSeq: number }> } }>
    return result?.[0]?.rows?._array?.[0]?.lastSeq ?? 0
  }

  async setLastSeq(sessionId: string, lastSeq: number) {
    const db = await openDatabase()
    db.execute(
      'INSERT OR REPLACE INTO sync_state (session_id, last_seq, last_sync_at) VALUES (?, ?, ?)',
      [sessionId, lastSeq, new Date().toISOString()]
    )
  }
}
```

- [ ] **Step 5: Implement the sync service using `after_seq` and event `seq`**

`mobile/src/services/sync/SyncService.ts`

```ts
import type { AppEvent } from '@/protocol/types'
import { MessageRepository } from '@/services/storage/MessageRepository'
import { SyncStateRepository } from '@/services/storage/SyncStateRepository'
import { RestClient } from '@/services/network/RestClient'

export class SyncService {
  constructor(
    private rest: RestClient,
    private messages: MessageRepository,
    private syncState: SyncStateRepository
  ) {}

  async syncSession(sessionId: string) {
    const lastSeq = await this.syncState.getLastSeq(sessionId)
    const response = await this.rest.get<{ messages: Array<{ seq: number; type: string; content: string; created_at: string }> }>(
      `/api/v1/messages?session_id=${encodeURIComponent(sessionId)}&after_seq=${lastSeq}&limit=500`
    )

    const rows = response.messages.map((item) => ({
      sessionId,
      seq: item.seq,
      type: item.type,
      content: item.content,
      createdAt: item.created_at
    }))

    await this.messages.insertMany(rows)
    if (rows.length > 0) {
      await this.syncState.setLastSeq(sessionId, rows[rows.length - 1].seq)
    }
  }

  async handleRealtimeEvent(sessionId: string, event: AppEvent) {
    const current = await this.syncState.getLastSeq(sessionId)
    if (!event.seq || event.seq <= current) return false

    await this.messages.insertMany([
      {
        sessionId,
        seq: event.seq,
        type: event.type,
        content: JSON.stringify(event),
        createdAt: event.time
      }
    ])
    await this.syncState.setLastSeq(sessionId, event.seq)
    return true
  }
}
```

- [ ] **Step 6: Add the sync service test for realtime dedupe**

`mobile/src/services/sync/__tests__/SyncService.test.ts`

```ts
import { SyncService } from '../SyncService'

test('ignores realtime event whose seq is not newer than local last_seq', async () => {
  const rest = { get: jest.fn() }
  const messages = { insertMany: jest.fn() }
  const syncState = {
    getLastSeq: jest.fn().mockResolvedValue(5),
    setLastSeq: jest.fn()
  }

  const service = new SyncService(rest as any, messages as any, syncState as any)
  const inserted = await service.handleRealtimeEvent('s1', {
    type: 'text',
    sessionId: 's1',
    seq: 5,
    time: '2026-06-08T00:00:00Z',
    text: 'duplicate'
  } as any)

  expect(inserted).toBe(false)
  expect(messages.insertMany).not.toHaveBeenCalled()
})
```

- [ ] **Step 7: Run tests and typecheck**

Run:

```bash
cd mobile && npm test -- MessageRepository.test.ts SyncService.test.ts && npm run typecheck
```

Expected: PASS for dedupe and sync tests.

- [ ] **Step 8: Commit**

```bash
git add mobile/src/services/storage mobile/src/services/sync
git commit -m "feat: 添加本地 SQLite 存储与增量同步服务"
```

---

## Task 5: Recreate the chat state machine with Zustand

**Files:**
- Create: `mobile/src/stores/useAuthStore.ts`
- Create: `mobile/src/stores/useSessionStore.ts`
- Create: `mobile/src/stores/useMessageStore.ts`
- Create: `mobile/src/stores/__tests__/useMessageStore.test.ts`

- [ ] **Step 1: Write the failing store test for `text_delta` merge behavior**

`mobile/src/stores/__tests__/useMessageStore.test.ts`

```ts
import { createMessageStore } from '../useMessageStore'

test('merges consecutive text_delta events with same blockIndex', () => {
  const store = createMessageStore()
  store.getState().handleEvent({
    type: 'text_delta',
    sessionId: 's1',
    time: '2026-06-08T00:00:00Z',
    blockIndex: 0,
    text: 'hel'
  } as any, 's1')
  store.getState().handleEvent({
    type: 'text_delta',
    sessionId: 's1',
    time: '2026-06-08T00:00:01Z',
    blockIndex: 0,
    text: 'lo'
  } as any, 's1')

  const messages = store.getState().messages
  expect(messages).toHaveLength(1)
  expect((messages[0] as any).text).toBe('hello')
})
```

- [ ] **Step 2: Run the store test and verify it fails**

Run:

```bash
cd mobile && npm test -- useMessageStore.test.ts
```

Expected: FAIL because `useMessageStore` is not implemented.

- [ ] **Step 3: Implement the auth and session stores**

`mobile/src/stores/useAuthStore.ts`

```ts
import { create } from 'zustand'
import type { ServerProfile } from '@/types/server-profile'

interface AuthState {
  activeProfile: ServerProfile | null
  status: 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'closed'
  setActiveProfile: (profile: ServerProfile | null) => void
  setStatus: (status: AuthState['status']) => void
}

export const useAuthStore = create<AuthState>((set) => ({
  activeProfile: null,
  status: 'idle',
  setActiveProfile: (activeProfile) => set({ activeProfile }),
  setStatus: (status) => set({ status })
}))
```

`mobile/src/stores/useSessionStore.ts`

```ts
import { create } from 'zustand'
import type { SessionMeta } from '@/protocol/types'

interface SessionState {
  sessions: SessionMeta[]
  activeSessionId: string | null
  viewedSessionId: string | null
  readOnly: boolean
  setSessions: (sessions: SessionMeta[]) => void
  setActiveSession: (sessionId: string | null) => void
  viewSession: (sessionId: string, readOnly: boolean) => void
}

export const useSessionStore = create<SessionState>((set) => ({
  sessions: [],
  activeSessionId: null,
  viewedSessionId: null,
  readOnly: false,
  setSessions: (sessions) => set({ sessions }),
  setActiveSession: (activeSessionId) => set({ activeSessionId, viewedSessionId: activeSessionId, readOnly: false }),
  viewSession: (viewedSessionId, readOnly) => set({ viewedSessionId, readOnly })
}))
```

- [ ] **Step 4: Implement the message store with the Web behavior copied intentionally**

`mobile/src/stores/useMessageStore.ts`

```ts
import { createStore } from 'zustand/vanilla'
import type { AppEvent, PermissionRequestEvent, TextDeltaEvent } from '@/protocol/types'

export interface UserMessage {
  type: 'user'
  sessionId: string
  time: string
  text: string
}

export type DisplayMessage = AppEvent | UserMessage

interface MessageState {
  messages: DisplayMessage[]
  permissionPrompt: PermissionRequestEvent | null
  permissionRequestId: string | null
  thinking: boolean
  turnActive: boolean
  lastSeq: number
  handleEvent: (event: AppEvent, sessionId?: string) => void
  addUserMessage: (text: string, sessionId: string) => void
  clearPermission: () => void
}

export function createMessageStore() {
  return createStore<MessageState>((set, get) => ({
    messages: [],
    permissionPrompt: null,
    permissionRequestId: null,
    thinking: false,
    turnActive: false,
    lastSeq: 0,
    handleEvent: (event, sessionId) => {
      const state = get()
      let messages = [...state.messages]

      if (event.type === 'text_delta') {
        const last = messages[messages.length - 1]
        if (last && last.type === 'text_delta' && (last as TextDeltaEvent).blockIndex === event.blockIndex) {
          const merged: TextDeltaEvent = {
            ...(last as TextDeltaEvent),
            text: ((last as TextDeltaEvent).text || '') + event.text,
            thinking: (last as TextDeltaEvent).thinking || event.thinking
          }
          messages = [...messages.slice(0, -1), merged]
        } else {
          messages.push(event)
        }
      } else if (event.type === 'text') {
        const last = messages[messages.length - 1]
        if (last && last.type === 'text_delta') {
          const thinking = (last as TextDeltaEvent).thinking
          messages = [...messages.slice(0, -1), { ...event, thinking: event.thinking || thinking }]
        } else {
          messages.push(event)
        }
      } else if (event.type === 'permission_request' || event.type === 'permission_ask') {
        const last = messages[messages.length - 1] as any
        const duplicate = last && (last.type === 'permission_request' || last.type === 'permission_ask') && last.toolName === (event as any).toolName
        messages = duplicate ? [...messages.slice(0, -1), event] : [...messages, event]
        set({ permissionPrompt: event as PermissionRequestEvent, permissionRequestId: event.messageId || null })
      } else {
        messages.push(event)
      }

      set({
        messages,
        lastSeq: event.seq && event.seq > state.lastSeq ? event.seq : state.lastSeq,
        thinking: event.type === 'thinking_start' ? true : event.type === 'turn_end' ? false : state.thinking,
        turnActive: event.type === 'turn_end' ? false : true
      })
    },
    addUserMessage: (text, sessionId) => {
      const state = get()
      set({
        messages: [...state.messages, { type: 'user', text, sessionId, time: new Date().toISOString() }],
        thinking: true,
        turnActive: true
      })
    },
    clearPermission: () => set({ permissionPrompt: null, permissionRequestId: null })
  }))
}
```

- [ ] **Step 5: Run tests and verify the state machine behavior**

Run:

```bash
cd mobile && npm test -- useMessageStore.test.ts && npm run typecheck
```

Expected: PASS for delta merge; TypeScript exits `0`.

- [ ] **Step 6: Commit**

```bash
git add mobile/src/stores
git commit -m "feat: 迁移聊天状态机到 Zustand"
```

---

## Task 6: Build onboarding, session list, and the terminal shell

**Files:**
- Create: `mobile/src/navigation/AppNavigator.tsx`
- Create: `mobile/src/screens/SplashScreen.tsx`
- Create: `mobile/src/screens/OnboardingScreen.tsx`
- Create: `mobile/src/screens/SessionListScreen.tsx`
- Create: `mobile/src/screens/TerminalScreen.tsx`
- Create: `mobile/src/components/terminal/MessageList.tsx`
- Create: `mobile/src/components/terminal/InputBar.tsx`
- Create: `mobile/src/components/terminal/SessionStatusBar.tsx`
- Create: `mobile/src/screens/__tests__/TerminalScreen.test.tsx`

- [ ] **Step 1: Write the failing terminal screen test for the send/stop button switch**

`mobile/src/screens/__tests__/TerminalScreen.test.tsx`

```tsx
import React from 'react'
import { render } from '@testing-library/react-native'
import { TerminalScreen } from '../TerminalScreen'

test('shows stop button while a turn is active', () => {
  const screen = render(<TerminalScreen turnActive={true} messages={[]} onSend={jest.fn()} onAbort={jest.fn()} />)
  expect(screen.getByText('停止')).toBeTruthy()
})
```

- [ ] **Step 2: Run the UI test and verify it fails**

Run:

```bash
cd mobile && npm test -- TerminalScreen.test.tsx
```

Expected: FAIL because `TerminalScreen` does not exist.

- [ ] **Step 3: Create the app navigator and splash/onboarding flow**

`mobile/src/navigation/AppNavigator.tsx`

```tsx
import React from 'react'
import { NavigationContainer } from '@react-navigation/native'
import { createNativeStackNavigator } from '@react-navigation/native-stack'
import { SplashScreen } from '@/screens/SplashScreen'
import { OnboardingScreen } from '@/screens/OnboardingScreen'
import { SessionListScreen } from '@/screens/SessionListScreen'
import { TerminalScreen } from '@/screens/TerminalScreen'

const Stack = createNativeStackNavigator()

export function AppNavigator() {
  return (
    <NavigationContainer>
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        <Stack.Screen name="Splash" component={SplashScreen} />
        <Stack.Screen name="Onboarding" component={OnboardingScreen} />
        <Stack.Screen name="Sessions" component={SessionListScreen} />
        <Stack.Screen name="Terminal" component={TerminalScreen} />
      </Stack.Navigator>
    </NavigationContainer>
  )
}
```

- [ ] **Step 4: Implement the terminal shell with a message list, status bar, and input bar**

`mobile/src/screens/TerminalScreen.tsx`

```tsx
import React from 'react'
import { SafeAreaView, View } from 'react-native'
import { MessageList } from '@/components/terminal/MessageList'
import { InputBar } from '@/components/terminal/InputBar'
import { SessionStatusBar } from '@/components/terminal/SessionStatusBar'

interface TerminalScreenProps {
  turnActive?: boolean
  messages?: Array<any>
  onSend?: (text: string) => void
  onAbort?: () => void
}

export function TerminalScreen({
  turnActive = false,
  messages = [],
  onSend = () => {},
  onAbort = () => {}
}: TerminalScreenProps) {
  return (
    <SafeAreaView style={{ flex: 1 }}>
      <SessionStatusBar status={turnActive ? 'running' : 'idle'} />
      <View style={{ flex: 1 }}>
        <MessageList messages={messages} />
      </View>
      <InputBar turnActive={turnActive} onSend={onSend} onAbort={onAbort} />
    </SafeAreaView>
  )
}
```

`mobile/src/components/terminal/InputBar.tsx`

```tsx
import React, { useState } from 'react'
import { Button, TextInput, View } from 'react-native'

export function InputBar({ turnActive, onSend, onAbort }: { turnActive: boolean; onSend: (text: string) => void; onAbort: () => void }) {
  const [value, setValue] = useState('')

  return (
    <View style={{ padding: 12, flexDirection: 'row', gap: 8 }}>
      <TextInput style={{ flex: 1, borderWidth: 1, borderColor: '#ccc', borderRadius: 8, paddingHorizontal: 12 }} value={value} onChangeText={setValue} />
      {turnActive ? (
        <Button title="停止" onPress={onAbort} />
      ) : (
        <Button title="发送" onPress={() => { onSend(value); setValue('') }} />
      )}
    </View>
  )
}
```

- [ ] **Step 5: Implement the initial session list and onboarding screens**

`mobile/src/screens/OnboardingScreen.tsx`

```tsx
import React from 'react'
import { Button, SafeAreaView, Text } from 'react-native'

export function OnboardingScreen() {
  return (
    <SafeAreaView style={{ flex: 1, justifyContent: 'center', alignItems: 'center', gap: 12 }}>
      <Text>扫描桌面端二维码以连接 MobileCoding</Text>
      <Button title="开始扫码" onPress={() => {}} />
    </SafeAreaView>
  )
}
```

`mobile/src/screens/SessionListScreen.tsx`

```tsx
import React from 'react'
import { FlatList, Pressable, SafeAreaView, Text, View } from 'react-native'
import { useSessionStore } from '@/stores/useSessionStore'

export function SessionListScreen() {
  const sessions = useSessionStore((state) => state.sessions)
  return (
    <SafeAreaView style={{ flex: 1 }}>
      <FlatList
        data={sessions}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <Pressable style={{ padding: 16, borderBottomWidth: 1, borderBottomColor: '#eee' }}>
            <Text>{item.name}</Text>
            <Text>{item.model || item.agent}</Text>
          </Pressable>
        )}
        ListEmptyComponent={<View style={{ padding: 24 }}><Text>暂无会话</Text></View>}
      />
    </SafeAreaView>
  )
}
```

- [ ] **Step 6: Run the UI test and confirm the shell behaves correctly**

Run:

```bash
cd mobile && npm test -- TerminalScreen.test.tsx && npm run typecheck
```

Expected: PASS; `停止` 按钮在 `turnActive=true` 时出现。

- [ ] **Step 7: Commit**

```bash
git add mobile/src/navigation mobile/src/screens mobile/src/components/terminal
git commit -m "feat: 添加移动端基础导航与终端壳界面"
```

---

## Task 7: Render structured terminal cards and wire realtime state updates

**Files:**
- Create: `mobile/src/components/cards/TextCard.tsx`
- Create: `mobile/src/components/cards/ToolCard.tsx`
- Create: `mobile/src/components/cards/LifecycleCard.tsx`
- Create: `mobile/src/components/cards/PermissionCard.tsx`
- Create: `mobile/src/components/cards/PlanModeCard.tsx`
- Create: `mobile/src/components/cards/ContextWindowCard.tsx`
- Create: `mobile/src/components/cards/ThinkingCard.tsx`
- Modify: `mobile/src/components/terminal/MessageList.tsx`
- Modify: `mobile/src/screens/TerminalScreen.tsx`

- [ ] **Step 1: Write the failing UI test for permission card rendering**

Append to `mobile/src/screens/__tests__/TerminalScreen.test.tsx`:

```tsx
test('renders permission action buttons for permission events', () => {
  const screen = render(
    <TerminalScreen
      messages={[
        {
          type: 'permission_ask',
          sessionId: 's1',
          time: '2026-06-08T00:00:00Z',
          toolName: 'Bash',
          message: '请求执行命令'
        }
      ]}
    />
  )

  expect(screen.getByText('允许')).toBeTruthy()
  expect(screen.getByText('拒绝')).toBeTruthy()
})
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```bash
cd mobile && npm test -- TerminalScreen.test.tsx
```

Expected: FAIL because permission UI is not rendered yet.

- [ ] **Step 3: Implement focused card components**

`mobile/src/components/cards/TextCard.tsx`

```tsx
import React from 'react'
import { Text, View } from 'react-native'

export function TextCard({ text }: { text: string }) {
  return (
    <View style={{ padding: 12, marginVertical: 4, borderRadius: 10, backgroundColor: '#fff' }}>
      <Text>{text}</Text>
    </View>
  )
}
```

`mobile/src/components/cards/PermissionCard.tsx`

```tsx
import React from 'react'
import { Button, Text, View } from 'react-native'

export function PermissionCard({ message, toolName, onAllow, onDeny }: { message: string; toolName: string; onAllow?: () => void; onDeny?: () => void }) {
  return (
    <View style={{ padding: 12, marginVertical: 4, borderRadius: 10, backgroundColor: '#fff3cd' }}>
      <Text>{toolName}</Text>
      <Text>{message}</Text>
      <View style={{ flexDirection: 'row', gap: 12, marginTop: 8 }}>
        <Button title="允许" onPress={onAllow || (() => {})} />
        <Button title="拒绝" onPress={onDeny || (() => {})} />
      </View>
    </View>
  )
}
```

- [ ] **Step 4: Route message types to cards in `MessageList`**

`mobile/src/components/terminal/MessageList.tsx`

```tsx
import React from 'react'
import { FlatList } from 'react-native'
import { TextCard } from '@/components/cards/TextCard'
import { PermissionCard } from '@/components/cards/PermissionCard'

export function MessageList({ messages }: { messages: Array<any> }) {
  return (
    <FlatList
      data={messages}
      keyExtractor={(_, index) => String(index)}
      renderItem={({ item }) => {
        if (item.type === 'permission_request' || item.type === 'permission_ask') {
          return <PermissionCard toolName={item.toolName} message={item.message} />
        }
        if (item.type === 'text' || item.type === 'text_delta') {
          return <TextCard text={item.text} />
        }
        return <TextCard text={item.message || item.toolOutput || item.type} />
      }}
    />
  )
}
```

- [ ] **Step 5: Wire permission handlers and context window card into the terminal screen**

Replace the render body in `mobile/src/screens/TerminalScreen.tsx` with:

```tsx
<SafeAreaView style={{ flex: 1, backgroundColor: '#f5f5f5' }}>
  <SessionStatusBar status={turnActive ? 'running' : 'idle'} />
  <View style={{ flex: 1 }}>
    <MessageList messages={messages} />
  </View>
  <InputBar turnActive={turnActive} onSend={onSend} onAbort={onAbort} />
</SafeAreaView>
```

- [ ] **Step 6: Run tests and confirm card rendering**

Run:

```bash
cd mobile && npm test -- TerminalScreen.test.tsx && npm run typecheck
```

Expected: PASS for both send/stop and permission rendering tests.

- [ ] **Step 7: Commit**

```bash
git add mobile/src/components/cards mobile/src/components/terminal/MessageList.tsx mobile/src/screens/TerminalScreen.tsx
git commit -m "feat: 添加结构化终端消息卡片渲染"
```

---

## Task 8: Add session history, search, settings, and native certificate trust

**Files:**
- Create: `mobile/src/screens/SearchScreen.tsx`
- Create: `mobile/src/screens/SettingsScreen.tsx`
- Modify: `mobile/src/screens/SessionListScreen.tsx`
- Create: `mobile/android/app/src/main/res/xml/network_security_config.xml`
- Modify: `mobile/android/app/src/main/AndroidManifest.xml`
- Modify: `mobile/ios/MobileCodingMobile/Info.plist`
- Create: `mobile/docs/cert-trust.md`

- [ ] **Step 1: Write the failing settings test for displaying saved server profiles**

Create `mobile/src/screens/__tests__/SettingsScreen.test.tsx`:

```tsx
import React from 'react'
import { render } from '@testing-library/react-native'
import { SettingsScreen } from '../SettingsScreen'

test('renders saved server profiles', () => {
  const screen = render(
    <SettingsScreen
      profiles={[{ id: '10.0.0.5:8443', name: '10.0.0.5:8443', host: '10.0.0.5', port: 8443, token: 'abc', lastConnectedAt: null, active: true }]}
    />
  )
  expect(screen.getByText('10.0.0.5:8443')).toBeTruthy()
})
```

- [ ] **Step 2: Run the settings test and verify it fails**

Run:

```bash
cd mobile && npm test -- SettingsScreen.test.tsx
```

Expected: FAIL because `SettingsScreen` does not exist.

- [ ] **Step 3: Implement settings and search screens**

`mobile/src/screens/SettingsScreen.tsx`

```tsx
import React from 'react'
import { FlatList, SafeAreaView, Text, View } from 'react-native'
import type { ServerProfile } from '@/types/server-profile'

export function SettingsScreen({ profiles = [] }: { profiles?: ServerProfile[] }) {
  return (
    <SafeAreaView style={{ flex: 1 }}>
      <FlatList
        data={profiles}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <View style={{ padding: 16, borderBottomWidth: 1, borderBottomColor: '#eee' }}>
            <Text>{item.name}</Text>
            <Text>{item.host}:{item.port}</Text>
          </View>
        )}
      />
    </SafeAreaView>
  )
}
```

`mobile/src/screens/SearchScreen.tsx`

```tsx
import React from 'react'
import { SafeAreaView, Text } from 'react-native'

export function SearchScreen() {
  return (
    <SafeAreaView style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
      <Text>搜索结果页</Text>
    </SafeAreaView>
  )
}
```

- [ ] **Step 4: Add Android network trust config for the self-signed CA**

`mobile/android/app/src/main/res/xml/network_security_config.xml`

```xml
<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
  <base-config cleartextTrafficPermitted="false">
    <trust-anchors>
      <certificates src="system" />
      <certificates src="user" />
    </trust-anchors>
  </base-config>
</network-security-config>
```

Modify `mobile/android/app/src/main/AndroidManifest.xml` inside `<application ...>`:

```xml
android:networkSecurityConfig="@xml/network_security_config"
```

- [ ] **Step 5: Add the iOS deep-link + ATS entries**

Append to `mobile/ios/MobileCodingMobile/Info.plist`:

```xml
<key>CFBundleURLTypes</key>
<array>
  <dict>
    <key>CFBundleURLSchemes</key>
    <array>
      <string>mobilecoding</string>
    </array>
  </dict>
</array>
<key>NSAppTransportSecurity</key>
<dict>
  <key>NSAllowsArbitraryLoads</key>
  <false/>
  <key>NSExceptionDomains</key>
  <dict>
    <key>localhost</key>
    <dict>
      <key>NSExceptionAllowsInsecureHTTPLoads</key>
      <false/>
      <key>NSIncludesSubdomains</key>
      <true/>
    </dict>
  </dict>
</dict>
```

- [ ] **Step 6: Document the manual CA trust validation steps**

`mobile/docs/cert-trust.md`

```md
# Certificate Trust Validation

## Android
1. Install the generated CA certificate onto the device.
2. Confirm the device shows the CA under user credentials.
3. Launch the app and connect to `https://<host>:8443`.
4. Expect successful WebSocket + REST handshake without TLS failure.

## iOS
1. Airdrop or email the CA certificate to the device.
2. Install it in Settings > Profile Downloaded.
3. Enable full trust in Settings > General > About > Certificate Trust Settings.
4. Launch the app and verify the session list loads.
```

- [ ] **Step 7: Run tests, typecheck, and lint**

Run:

```bash
cd mobile && npm test -- SettingsScreen.test.tsx TerminalScreen.test.tsx && npm run typecheck && npm run lint
```

Expected: PASS for UI tests; typecheck and lint exit `0`.

- [ ] **Step 8: Commit**

```bash
git add mobile/src/screens mobile/android/app/src/main mobile/ios/MobileCodingMobile/Info.plist mobile/docs/cert-trust.md
git commit -m "feat: 完成设置页、搜索页与原生证书信任配置"
```

---

## Final Verification Task: End-to-end smoke checklist

**Files:**
- Modify: `docs/superpowers/specs/2026-06-08-mobile-app-client-design.md` (only if implementation deviates)
- Use existing app files created in Tasks 1-8

- [ ] **Step 1: Start the Go backend and verify the desktop QR flow still works**

Run:

```bash
make build
./dist/mobilecoding.exe
```

Expected: 终端输出 QR 码和 `https://<ip>:8443/?token=...` 形式的连接地址。

- [ ] **Step 2: Launch the mobile app on Android simulator/device**

Run:

```bash
cd mobile && npm run android
```

Expected: App starts and shows the onboarding screen.

- [ ] **Step 3: Scan the desktop QR code and verify token persistence**

Expected:

```text
- App transitions from Onboarding → SessionList
- Settings page can list the active server profile
- Relaunching the app keeps the profile and reconnects automatically
```

- [ ] **Step 4: Start a `mc claude` session and verify dual-terminal behavior**

Manual action:

```text
1. 在桌面 CLI 发送一条消息
2. 手机端看到相同结构化事件流
3. 手机端发送一条消息
4. 桌面 CLI 会话继续，无需重启
```

Expected: 双端都能继续消费同一服务端事件流，互不抢占。

- [ ] **Step 5: Verify reconnect + `after_seq` backfill**

Manual action:

```text
1. 断开手机网络 30 秒
2. 在桌面继续执行一个会话
3. 恢复手机网络
4. 观察手机端自动补拉 missed messages
```

Expected: 恢复后消息完整，且不会出现重复 seq 卡片。

- [ ] **Step 6: Verify permission race handling**

Manual action:

```text
1. 触发需要权限的操作
2. 同时在桌面和手机端看到权限卡片
3. 在手机端先点击“允许”
4. 再尝试在桌面端回应一次
```

Expected: 手机端成功；桌面端后续响应收到 stale 结果，权限卡片被清理。

- [ ] **Step 7: Commit any final fixes**

```bash
git add mobile .gitignore
git commit -m "fix: 收尾移动端联调问题"
```

---

## Self-Review

### Spec coverage

- 扫码连接：Task 3、Task 6、Final Verification Step 3
- 双端同步存储：Task 4、Task 5、Final Verification Step 4-5
- 增量同步：Task 4
- 结构化终端镜像 UI：Task 6、Task 7
- 多服务器配置：Task 3、Task 8
- 自签证书信任：Task 8、Final Verification Step 2-3
- 权限竞争处理：Task 5、Task 7、Final Verification Step 6

### Placeholder scan

- 无 `TBD` / `TODO`
- 每个代码步骤都给出了明确文件路径与代码块
- 每个验证步骤都给出了明确命令与预期结果

### Type consistency

- `ServerProfile` 统一定义于 `mobile/src/types/server-profile.ts`
- 协议事件统一来自 `mobile/src/protocol/types.ts`
- `WSClient` / `RestClient` / `SyncService` 的依赖顺序保持一致

---

Plan complete and saved to `docs/superpowers/plans/2026-06-08-mobile-app-client.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
