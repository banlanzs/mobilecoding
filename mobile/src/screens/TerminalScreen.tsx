import React, { useState, useEffect, useRef } from 'react'
import { SafeAreaView, View, TextInput, Button, Text, FlatList, KeyboardAvoidingView, Platform } from 'react-native'
import { createMessageStore } from '../stores/useMessageStore'
import { MessageCard } from '../components/terminal/MessageCard'

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
    if ((method === 'send_message' || method === 'session.input') && params?.text) {
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
    console.log('[WS] connecting to', fullUrl)
    try {
      this.ws = new WebSocket(fullUrl)
    } catch (e) {
      console.error('[WS] 创建失败:', e)
      this.setStatus('closed')
      return
    }
    this.ws.onopen = () => {
      console.log('[WS] connected')
      this.connected = true
      this.setStatus('connected')
      this.flushQueue()
    }
    this.ws.onmessage = (event: any) => {
      try {
        const envelope = JSON.parse(String(event.data))
        if (envelope.type === 'evt') {
          console.log('[WS] evt:', envelope.event?.type, 'text:', (envelope.event?.text || '').substring(0, 80))
        }
        this.handleEnvelope(envelope)
      } catch (e) {
        console.error('[WS] parse error:', e)
      }
    }
    this.ws.onclose = () => {
      console.log('[WS] closed')
      this.connected = false
      this.setStatus('closed')
    }
    this.ws.onerror = (e: any) => {
      console.error('[WS] error:', e?.message || e)
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
        try { l(envelope.event, envelope.sessionId) } catch (e) { console.error('[WS] event listener error:', e) }
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

export function TerminalScreen(props?: any) {
  const routeParams = props?.route?.params || {}
  const [input, setInput] = useState('')
  const [messages, setMessages] = useState<any[]>([])
  const [turnActive, setTurnActive] = useState(false)
  const [connected, setConnected] = useState(false)
  const [thinking, setThinking] = useState(false)
  const [permissionPrompt, setPermissionPrompt] = useState<any>(null)
  const [sessionStarted, setSessionStarted] = useState(false)

  const host = routeParams.host || '10.0.2.2'
  const port = routeParams.port || '8445'
  const token = routeParams.token || ''
  const path = routeParams.path || '/api/v1/ws'
  const useWss = routeParams.useWss ?? false
  const useMock = routeParams.useMock ?? false

  const clientRef = useRef<MockWSClient | RealMobilecodingClient | null>(null)

  useEffect(() => {
    const unsubStore = messageStore.subscribe((state) => {
      console.log('[Store] messages:', state.messages.length, 'turnActive:', state.turnActive, 'thinking:', state.thinking)
      setMessages([...state.messages])
      setTurnActive(state.turnActive)
      setThinking(state.thinking)
      setPermissionPrompt(state.permissionPrompt)
    })

    if (routeParams.host && routeParams.token) {
      setTimeout(() => handleConnect(), 100)
    }

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
      return
    }

    const real = new RealMobilecodingClient()
    clientRef.current = real
    real.onEvent((event, sid) => {
      console.log('[Terminal] event received:', event?.type)
      messageStore.getState().handleEvent(event, sid)
    })
    real.onStatus((s) => {
      console.log('[Terminal] status:', s)
      setConnected(s === 'connected')
    })
    const scheme = useWss ? 'wss' : 'ws'
    const url = `${scheme}://${host}:${port}${path}`
    real.connect(url, token)
  }

  const handleStartSession = async () => {
    if (!clientRef.current || sessionStarted) return
    try {
      const result = await clientRef.current.send('session.start', {
        command: 'claude',
        args: [],
        cwd: ''
      })
      console.log('[Terminal] Session started:', result)
      setSessionStarted(true)
    } catch (err: any) {
      console.warn('[Terminal] session.start failed (expected if native session already active):', err?.message)
      // session.start 失败通常是因为 native session 已在运行，
      // 但 session.input 仍然可以写入已有 session，所以标记为已启动以解锁 UI
      setSessionStarted(true)
    }
  }

  useEffect(() => {
    if (!connected || sessionStarted || !clientRef.current) return
    handleStartSession()
  }, [connected, sessionStarted])

  // 连接成功后自动拉取历史消息（支持 --resume 恢复会话）
  useEffect(() => {
    if (!connected || !token) return
    const restScheme = useWss ? 'https' : 'http'
    const restUrl = `${restScheme}://${host}:${port}/api/v1/messages?session_id=default-session&limit=50`
    console.log('[Terminal] fetching history:', restUrl)
    fetch(restUrl, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(res => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        return res.json()
      })
      .then((data: { messages?: Array<{ type: string; content: string; seq: number }> }) => {
        if (!data.messages || data.messages.length === 0) {
          console.log('[Terminal] no history messages')
          return
        }
        console.log(`[Terminal] loaded ${data.messages.length} history messages`)
        // 按 seq 正序排列，逐条喂给 messageStore
        const sorted = [...data.messages].sort((a, b) => (a.seq || 0) - (b.seq || 0))
        for (const msg of sorted) {
          try {
            const event = JSON.parse(msg.content)
            messageStore.getState().handleEvent(event)
          } catch (e) {
            console.warn('[Terminal] failed to parse history message:', e)
          }
        }
      })
      .catch(err => {
        console.warn('[Terminal] history fetch failed (expected for new sessions):', err?.message)
      })
  }, [connected])

  const handleSend = () => {
    if (!input.trim() || !clientRef.current || !connected) return
    messageStore.getState().addUserMessage(input, 'default-session')
    clientRef.current.send('session.input', { text: input })
      .catch(err => console.error('发送失败:', err))
    setInput('')
  }

  const handleAbort = () => {
    if (!clientRef.current) return
    clientRef.current.send('session.abort', {})
      .catch(err => console.error('停止失败:', err))
    // 本地立即重置 turnActive，不等待服务端 turn_end 事件
    setTurnActive(false)
    setThinking(false)
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: '#ededed' }}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        keyboardVerticalOffset={0}
      >
        {/* 顶栏 */}
        <View style={{ paddingHorizontal: 16, paddingVertical: 14, backgroundColor: '#ededed', borderBottomWidth: 1, borderBottomColor: '#d9d9d9', flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' }}>
          <Text style={{ fontSize: 17, fontWeight: '600', color: '#000' }}>Claude</Text>
          <View style={{ flexDirection: 'row', alignItems: 'center', gap: 6 }}>
            <View style={{ width: 8, height: 8, borderRadius: 4, backgroundColor: connected ? '#34c759' : '#f44336' }} />
            <Text style={{ fontSize: 12, color: '#666' }}>{connected ? '已连接' : '未连接'}</Text>
          </View>
        </View>

        {/* 消息列表 */}
        <FlatList
          data={messages}
          keyExtractor={(_, idx) => String(idx)}
          renderItem={({ item }) => <MessageCard message={item} />}
          style={{ flex: 1 }}
          contentContainerStyle={{ paddingVertical: 8 }}
        />

        {/* Thinking 指示器 */}
        {thinking && (
          <View style={{ paddingHorizontal: 12, paddingBottom: 4 }}>
            <View style={{ alignSelf: 'flex-start', backgroundColor: '#ffffff', borderRadius: 12, borderWidth: 1, borderColor: '#e5e5e5', paddingHorizontal: 14, paddingVertical: 10 }}>
              <Text style={{ color: '#666', fontStyle: 'italic' }}>思考中...</Text>
            </View>
          </View>
        )}

        {/* 权限审批卡片 */}
        {permissionPrompt && (
          <View style={{ paddingHorizontal: 12, paddingBottom: 8 }}>
            <View style={{ backgroundColor: '#fff9c4', borderRadius: 12, borderWidth: 1, borderColor: '#fbc02d', padding: 12 }}>
              <Text style={{ fontWeight: '600', marginBottom: 6 }}>权限请求</Text>
              <Text style={{ color: '#000', marginBottom: 4 }}>{permissionPrompt.toolName}</Text>
              <Text style={{ marginBottom: 10 }}>{permissionPrompt.message}</Text>
              <View style={{ flexDirection: 'row', gap: 12 }}>
                <Button
                  title="允许"
                  onPress={() => {
                    const reqId = messageStore.getState().permissionRequestId
                    messageStore.getState().answerPermission(true)
                    clientRef.current?.send('permission.respond', {
                      requestId: reqId,
                      allow: true
                    }).catch((err) => console.error('[Perm] allow failed:', err))
                  }}
                />
                <Button
                  title="拒绝"
                  color="#f44336"
                  onPress={() => {
                    const reqId = messageStore.getState().permissionRequestId
                    messageStore.getState().answerPermission(false)
                    clientRef.current?.send('permission.respond', {
                      requestId: reqId,
                      allow: false
                    }).catch((err) => console.error('[Perm] deny failed:', err))
                  }}
                />
              </View>
            </View>
          </View>
        )}

        {/* 输入栏 */}
        <View style={{ paddingHorizontal: 12, paddingTop: 8, paddingBottom: 12, backgroundColor: '#f7f7f7', borderTopWidth: 1, borderTopColor: '#d9d9d9', flexDirection: 'row', gap: 8, alignItems: 'center' }}>
          <TextInput
            value={input}
            onChangeText={setInput}
            placeholder={connected ? '输入消息...' : '连接中...'}
            editable={connected}
            style={{ flex: 1, backgroundColor: '#fff', borderRadius: 6, borderWidth: 1, borderColor: '#d9d9d9', paddingHorizontal: 12, height: 42, color: '#000' }}
          />
          {turnActive ? (
            <Button title="停止" onPress={handleAbort} color="#f44336" />
          ) : (
            <Button title="发送" onPress={handleSend} disabled={!connected || !input.trim()} color="#2e7d32" />
          )}
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  )
}
