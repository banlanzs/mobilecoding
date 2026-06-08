import React, { useState, useEffect, useRef } from 'react'
import { SafeAreaView, View, TextInput, Button, Text, FlatList, Switch } from 'react-native'
import { WSClient } from '../services/network/WSClient'
import { createMessageStore } from '../stores/useMessageStore'
import { MessageCard } from '../components/terminal/MessageCard'

// ─── Mock 客户端（本地测试用） ─────────────────────────
class MockWSClient {
  private statusListeners = new Set<(status: 'idle' | 'connecting' | 'connected' | 'closed') => void>()
  private eventListeners = new Set<(event: any, sessionId?: string) => void>()
  private pendingTimers = new Set<ReturnType<typeof setTimeout>>()
  private connected = false

  connect() {
    this.setStatus('connecting')
    setTimeout(() => { this.connected = true; this.setStatus('connected') }, 300)
  }
  disconnect() {
    this.pendingTimers.forEach(t => clearTimeout(t))
    this.pendingTimers.clear()
    this.connected = false
    this.setStatus('closed')
  }
  async send(method: string, params: any): Promise<any> {
    if (!this.connected) throw new Error('未连接')
    if (method === 'send_message' && params?.text) {
      const sessionId = params.sessionId || 'default-session'
      const userText = params.text
      const timer = setTimeout(() => {
        this.pendingTimers.delete(timer)
        const seq = Date.now()
        this.eventListeners.forEach(l => l({
          type: 'text', sessionId, text: `好的，已收到：${userText}（Mock 回复）`,
          time: new Date().toISOString(), seq, messageId: `mock-${seq}`
        }, sessionId))
        this.eventListeners.forEach(l => l({
          type: 'turn_end', sessionId, text: '', message: '',
          time: new Date().toISOString(), seq: seq + 1, messageId: `mock-te-${seq}`
        }, sessionId))
      }, 1500)
      this.pendingTimers.add(timer)
    }
    return { ok: true }
  }
  onEvent(cb: (event: any, sessionId?: string) => void) { this.eventListeners.add(cb); return () => this.eventListeners.delete(cb) }
  onStatus(cb: (status: 'idle' | 'connecting' | 'connected' | 'closed') => void) { this.statusListeners.add(cb); return () => this.statusListeners.delete(cb) }
  private setStatus(status: 'idle' | 'connecting' | 'connected' | 'closed') { this.statusListeners.forEach(l => l(status)) }
}

// ─── 真实协议客户端（适配 mc claude /mobilecoding 后端） ─────────
class RealMobilecodingClient {
  private ws: WebSocket | null = null
  private statusListeners = new Set<(status: 'idle' | 'connecting' | 'connected' | 'closed') => void>()
  private eventListeners = new Set<(event: any, sessionId?: string) => void>()
  private pending = new Map<string, { resolve: (v: any) => void; reject: (e: Error) => void; timer: ReturnType<typeof setTimeout> }>()
  private queue: { method: string; params: any; resolve: (v: any) => void; reject: (e: Error) => void }[] = []
  private connected = false
  private nextId = 0
  private url = ''
  private token = ''

  connect(url: string, token: string) {
    this.url = url
    this.token = token
    this.setStatus('connecting')
    this.doConnect()
  }

  private doConnect() {
    const fullUrl = `${this.url}?token=${encodeURIComponent(this.token)}`
    try {
      this.ws = new WebSocket(fullUrl)
    } catch (e) {
      console.error('WebSocket 创建失败:', e)
      this.setStatus('closed')
      return
    }
    this.ws.onopen = () => {
      this.connected = true
      this.setStatus('connected')
      // 连接成功后发送排队的请求
      this.flushQueue()
    }
    this.ws.onmessage = (event: any) => {
      try {
        const envelope = JSON.parse(String(event.data))
        this.handleEnvelope(envelope)
      } catch {}
    }
    this.ws.onclose = () => {
      this.connected = false
      this.setStatus('closed')
    }
    this.ws.onerror = () => {
      // onclose 会跟随触发
    }
  }

  private flushQueue() {
    while (this.queue.length > 0) {
      const item = this.queue.shift()!
      this.doSend(item.method, item.params, item.resolve, item.reject)
    }
  }

  send(method: string, params: any): Promise<any> {
    return new Promise((resolve, reject) => {
      if (!this.connected) {
        this.queue.push({ method, params, resolve, reject })
        return
      }
      this.doSend(method, params, resolve, reject)
    })
  }

  private doSend(method: string, params: any, resolve: (v: any) => void, reject: (e: Error) => void) {
    const id = `req-${++this.nextId}`
    const timer = setTimeout(() => {
      this.pending.delete(id)
      reject(new Error(`请求超时: ${method}`))
    }, 30000)
    this.pending.set(id, { resolve, reject, timer })
    const envelope = { type: 'req', id, method, params }
    try {
      this.ws!.send(JSON.stringify(envelope))
    } catch (err) {
      clearTimeout(timer)
      this.pending.delete(id)
      reject(err as Error)
    }
  }

  private handleEnvelope(envelope: any) {
    if (envelope.type === 'evt') {
      this.eventListeners.forEach(l => {
        try { l(envelope.event, envelope.sessionId) } catch {}
      })
      return
    }
    if (envelope.type === 'resp') {
      const entry = this.pending.get(envelope.id)
      if (!entry) return
      clearTimeout(entry.timer)
      this.pending.delete(envelope.id)
      envelope.ok
        ? entry.resolve(envelope.result)
        : entry.reject(new Error(envelope.error?.message ?? 'RPC 错误'))
    }
  }

  disconnect() {
    this.connected = false
    this.ws?.close(1000)
    this.ws = null
    this.setStatus('closed')
    this.pending.forEach(({ reject, timer }) => { clearTimeout(timer); reject(new Error('连接已断开')) })
    this.pending.clear()
    this.queue.forEach(q => q.reject(new Error('连接已断开')))
    this.queue = []
  }

  onEvent(cb: (event: any, sessionId?: string) => void) { this.eventListeners.add(cb); return () => this.eventListeners.delete(cb) }
  onStatus(cb: (status: 'idle' | 'connecting' | 'connected' | 'closed') => void) { this.statusListeners.add(cb); return () => this.statusListeners.delete(cb) }
  private setStatus(status: 'idle' | 'connecting' | 'connected' | 'closed') { this.statusListeners.forEach(l => l(status)) }
}

const messageStore = createMessageStore()

export function TerminalScreen(_props?: any) {
  const [input, setInput] = useState('')
  const [messages, setMessages] = useState<any[]>([])
  const [turnActive, setTurnActive] = useState(false)
  const [connected, setConnected] = useState(false)
  const [thinking, setThinking] = useState(false)
  const [permissionPrompt, setPermissionPrompt] = useState<any>(null)

  // 连接配置
  const [useMock, setUseMock] = useState(false)  // 默认关闭 Mock
  const [host, setHost] = useState('10.0.2.2')
  const [port, setPort] = useState('8443')
  const [token, setToken] = useState('')
  const [path, setPath] = useState('/api/v1/ws')
  const [useWss, setUseWss] = useState(false)
  const [sessionStarted, setSessionStarted] = useState(false)

  const clientRef = useRef<MockWSClient | RealMobilecodingClient | null>(null)

  useEffect(() => {
    const unsubStore = messageStore.subscribe((state) => {
      setMessages(state.messages)
      setTurnActive(state.turnActive)
      setThinking(state.thinking)
      setPermissionPrompt(state.permissionPrompt)
    })
    return () => { unsubStore(); clientRef.current?.disconnect(); clientRef.current = null }
  }, [])

  const handleConnect = () => {
    if (clientRef.current) { clientRef.current.disconnect(); clientRef.current = null }
    messageStore.getState().resetMessages()
    setSessionStarted(false)

    if (useMock) {
      const mock = new MockWSClient()
      clientRef.current = mock
      mock.onEvent((event, sid) => messageStore.getState().handleEvent(event, sid))
      mock.onStatus((s) => setConnected(s === 'connected'))
      mock.connect()
    } else {
      const real = new RealMobilecodingClient()
      clientRef.current = real
      real.onEvent((event, sid) => messageStore.getState().handleEvent(event, sid))
      real.onStatus((s) => setConnected(s === 'connected'))
      const scheme = useWss ? 'wss' : 'ws'
      const url = `${scheme}://${host}:${port}${path}`
      real.connect(url, token)
    }
  }

  // 连接后自动启动 session（mc claude 需要先 session.start）
  const handleStartSession = async () => {
    if (!clientRef.current || sessionStarted) return
    try {
      const result = await clientRef.current.send('session.start', {
        command: 'claude',
        args: [],
        cwd: ''
      })
      console.log('Session started:', result)
      setSessionStarted(true)
    } catch (err) {
      console.error('Session start failed:', err)
    }
  }

  const handleSend = () => {
    if (!input.trim() || !clientRef.current || !connected) return
    messageStore.getState().addUserMessage(input, 'default-session')
    // mc claude 真实协议：session.input
    clientRef.current.send('session.input', { text: input })
      .catch(err => console.error('发送失败:', err))
    setInput('')
  }

  const handleAbort = () => {
    if (!clientRef.current) return
    clientRef.current.send('session.abort', {})
      .catch(err => console.error('停止失败:', err))
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: '#f5f5f5' }}>
      {/* 标题栏 */}
      <View style={{ padding: 12, backgroundColor: '#e0e0e0', flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text>Terminal</Text>
        <View style={{ flexDirection: 'row', alignItems: 'center', gap: 6 }}>
          <View style={{ width: 8, height: 8, borderRadius: 4, backgroundColor: connected ? '#4caf50' : '#f44336' }} />
          <Text style={{ fontSize: 12 }}>{connected ? '已连接' : '未连接'}</Text>
        </View>
      </View>

      {/* 连接设置 */}
      <View style={{ padding: 8, backgroundColor: '#fafafa', borderBottomWidth: 1, borderBottomColor: '#ddd' }}>
        <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
          <Switch value={useMock} onValueChange={setUseMock} />
          <Text>Mock 模式</Text>

          {!useMock && (
            <View style={{ flexDirection: 'row', alignItems: 'center', marginLeft: 16 }}>
              <Switch value={useWss} onValueChange={setUseWss} />
              <Text> WSS</Text>
            </View>
          )}
        </View>
        <View style={{ marginTop: 4 }}>
          <TextInput value={host} onChangeText={setHost} placeholder="Host (10.0.2.2 / 局域网IP)" style={inputStyle} />
          <TextInput value={port} onChangeText={setPort} placeholder="Port (8443)" keyboardType="numeric" style={inputStyle} />
          {!useMock && (
            <>
              <TextInput value={token} onChangeText={setToken} placeholder="Token（从服务器日志复制）" style={inputStyle} />
              <TextInput value={path} onChangeText={setPath} placeholder="WS 路径（/api/v1/ws）" style={inputStyle} />
            </>
          )}
        </View>
        <Button title={connected ? '已连接（点此重连）' : '连接'} onPress={handleConnect} />
      </View>

      {/* 消息列表 */}
      <FlatList
        data={messages}
        keyExtractor={(_, idx) => String(idx)}
        renderItem={({ item }) => <MessageCard message={item} />}
        style={{ flex: 1 }}
      />

      {/* Thinking 指示器 */}
      {thinking && (
        <View style={{ padding: 8, alignItems: 'center', backgroundColor: '#e8f5e9' }}>
          <Text style={{ color: '#666', fontStyle: 'italic' }}>思考中...</Text>
        </View>
      )}

      {/* 权限审批卡片 */}
      {permissionPrompt && (
        <View style={{ padding: 12, backgroundColor: '#fff9c4', borderTopWidth: 1, borderTopColor: '#fbc02d' }}>
          <Text style={{ fontWeight: '600', marginBottom: 8 }}>
            权限请求: {permissionPrompt.toolName}
          </Text>
          <Text style={{ marginBottom: 8 }}>{permissionPrompt.message}</Text>
          <View style={{ flexDirection: 'row', gap: 12 }}>
            <Button
              title="允许"
              onPress={() => {
                messageStore.getState().answerPermission(true)
                clientRef.current?.send('permission.respond', {
                  requestId: messageStore.getState().permissionRequestId,
                  allow: true
                }).catch(() => {})
              }}
            />
            <Button
              title="拒绝"
              color="#f44336"
              onPress={() => {
                messageStore.getState().answerPermission(false)
                clientRef.current?.send('permission.respond', {
                  requestId: messageStore.getState().permissionRequestId,
                  allow: false
                }).catch(() => {})
              }}
            />
          </View>
        </View>
      )}

      {/* 输入栏 */}
      <View style={{ padding: 12, flexDirection: 'row', gap: 8 }}>
        <TextInput
          value={input}
          onChangeText={setInput}
          placeholder={connected ? '输入消息...' : '请先连接'}
          editable={connected}
          style={{ flex: 1, borderWidth: 1, borderColor: '#ccc', borderRadius: 8, paddingHorizontal: 12, height: 40, backgroundColor: connected ? '#fff' : '#f5f5f5' }}
        />
        {turnActive ? (
          <Button title="停止" onPress={handleAbort} />
        ) : (
          <Button title="发送" onPress={handleSend} disabled={!connected || !input.trim()} />
        )}
      </View>
    </SafeAreaView>
  )
}

const inputStyle = {
  borderWidth: 1,
  borderColor: '#ccc',
  borderRadius: 6,
  paddingHorizontal: 8,
  paddingVertical: 4,
  marginVertical: 2,
  fontSize: 12
}
