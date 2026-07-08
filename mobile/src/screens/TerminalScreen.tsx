import React, { useState, useEffect, useRef } from 'react'
import { SafeAreaView, View, TextInput, Button, Text, FlatList, KeyboardAvoidingView, Platform, BackHandler, Alert, Pressable, StatusBar, Modal, ActivityIndicator } from 'react-native'
import { Picker } from '@react-native-picker/picker'
import { createMessageStore } from '../stores/useMessageStore'
import { MessageList } from '../components/terminal/MessageList'
import { GitDiffModal } from '../components/terminal/GitDiffModal'

/** 格式化上下文用量文案，无数据返回 null */
function formatContextUsage(cw: { used?: number; total?: number; raw?: unknown } | null): string | null {
  if (!cw) return null
  if (cw.used != null && cw.total != null) {
    return `Context: ${Math.round(cw.used / 1000)}k / ${Math.round(cw.total / 1000)}k`
  }
  if (cw.used != null) {
    return `Context: ${Math.round(cw.used / 1000)}k`
  }
  return null
}

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

// 斜杠命令定义（本地处理 vs 透传给 CLI）
const SLASH_COMMANDS = [
  { cmd: '/compact', desc: '压缩对话上下文', mode: 'passthrough' as const },
  { cmd: '/clear', desc: '清空本地消息', mode: 'local' as const },
  { cmd: '/help', desc: '显示帮助信息', mode: 'passthrough' as const },
  { cmd: '/cost', desc: '显示 Token 用量', mode: 'passthrough' as const },
  { cmd: '/model', desc: '切换模型', mode: 'local' as const },
  { cmd: '/status', desc: '显示会话状态', mode: 'passthrough' as const },
  { cmd: '/context', desc: '查看上下文用量', mode: 'passthrough' as const },
  { cmd: '/memory', desc: '查看记忆文件', mode: 'passthrough' as const },
  { cmd: '/agents', desc: '查看 Agent 列表', mode: 'passthrough' as const },
  { cmd: '/bashes', desc: '查看运行中的 Bash', mode: 'passthrough' as const },
  { cmd: '/config', desc: '查看配置', mode: 'passthrough' as const },
  { cmd: '/init', desc: '初始化项目记忆文件', mode: 'passthrough' as const },
  { cmd: '/upgrade', desc: '升级 Claude Code', mode: 'passthrough' as const },
  { cmd: '/bug', desc: '提交 Bug 报告', mode: 'passthrough' as const },
  { cmd: '/doctor', desc: '安装/修复 Claude Code', mode: 'passthrough' as const },
  { cmd: '/login', desc: '登录', mode: 'passthrough' as const },
  { cmd: '/logout', desc: '登出', mode: 'passthrough' as const },
  { cmd: '/output-style', desc: '切换输出风格', mode: 'passthrough' as const },
  { cmd: '/add-dir', desc: '添加工作目录', mode: 'passthrough' as const },
  { cmd: '/resume', desc: '恢复之前的会话', mode: 'passthrough' as const },
  { cmd: '/terminal-setup', desc: '终端设置', mode: 'passthrough' as const },
  { cmd: '/todos', desc: '查看待办事项', mode: 'passthrough' as const },
  { cmd: '/export', desc: '导出对话', mode: 'passthrough' as const },
]

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
  const [showGitDiff, setShowGitDiff] = useState(false)
  const [showSlashPicker, setShowSlashPicker] = useState(false)
  const [slashFilter, setSlashFilter] = useState('')
  const [showModelPicker, setShowModelPicker] = useState(false)
  const [models, setModels] = useState<{ label: string; value: string }[]>([])
  const [selectedModel, setSelectedModel] = useState('')
  const [loadingModels, setLoadingModels] = useState(false)
  const [runtimeArgs, setRuntimeArgs] = useState<string[]>([])
  const [settingsPath, setSettingsPath] = useState<string>('')
  const [projectCwd, setProjectCwd] = useState<string>('')
  const [contextWindow, setContextWindow] = useState<{ used?: number; total?: number; raw?: unknown } | null>(null)

  const host = routeParams.host || '10.0.2.2'
  const port = routeParams.port || '8445'
  const token = routeParams.token || ''
  const path = routeParams.path || '/api/v1/ws'
  const useWss = routeParams.useWss ?? false
  const useMock = routeParams.useMock ?? false
  const initialModel = routeParams.model || ''

  const clientRef = useRef<MockWSClient | RealMobilecodingClient | null>(null)

  useEffect(() => {
    const unsubStore = messageStore.subscribe((state) => {
      setMessages([...state.messages])
      setTurnActive(state.turnActive)
      setThinking(state.thinking)
      setPermissionPrompt(state.permissionPrompt)
      setContextWindow(state.contextWindow)
    })

    // 禁用返回手势，防止误触回到 Onboarding
    if (props?.navigation) {
      props.navigation.setOptions({ gestureEnabled: false, headerBackVisible: false })
    }

    // 拦截 Android 返回键，弹窗确认退出（会同步停止桌面会话）
    const onBackPress = () => {
      Alert.alert('退出会话', '确定要退出当前会话吗？退出后桌面端会话也会停止。', [
        { text: '取消', style: 'cancel' },
        { text: '退出', style: 'destructive', onPress: async () => {
          try {
            // 先通知服务端停止会话，再断开 WebSocket
            await clientRef.current?.send('session.abort', {})
          } catch {}
          clientRef.current?.disconnect()
          messageStore.getState().resetMessages()
          props?.navigation?.goBack()
        }}
      ])
      return true
    }
    const backSub = BackHandler.addEventListener('hardwareBackPress', onBackPress)

    // 只在首次进入时连接，后续 navigate 回来不重连
    if (routeParams.host && routeParams.token && !clientRef.current) {
      setTimeout(() => handleConnect(), 100)
    }

    return () => { backSub.remove(); unsubStore() }
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
      // 构建启动参数，如果用户选择了模型则注入 --model 参数
      const args: string[] = []
      if (initialModel) {
        args.push('--model', initialModel)
      }
      const result = await clientRef.current.send('session.start', {
        command: 'claude',
        args,
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

  // 连接成功后获取 runtime 信息（含 --settings 路径）和历史消息
  useEffect(() => {
    if (!connected || !token) return
    const restScheme = useWss ? 'https' : 'http'
    const baseUrl = `${restScheme}://${host}:${port}`

    // 1. 获取 runtime 信息，提取 --settings 路径
    fetch(`${baseUrl}/version`)
      .then(res => res.ok ? res.json() : null)
      .then((data: any) => {
        if (!data?.runtime) return
        const args: string[] = data.runtime.defaultArgs || []
        setRuntimeArgs(args)
        // 从 args 中提取 --settings <path>
        const settingsIdx = args.indexOf('--settings')
        if (settingsIdx >= 0 && settingsIdx + 1 < args.length) {
          const path = args[settingsIdx + 1]
          setSettingsPath(path)
          console.log('[Terminal] settings path:', path)
        }
        // 提取工作目录作为项目名
        if (data.runtime.cwd) {
          setProjectCwd(data.runtime.cwd)
        }
      })
      .catch(err => console.warn('[Terminal] version fetch failed:', err?.message))

    // 2. 获取历史消息（通过 /api/v1/session-id 取活跃会话 ID）
    const sessionIdUrl = `${baseUrl}/api/v1/session-id`
    console.log('[Terminal] fetching active session id')
    fetch(sessionIdUrl, { headers: { 'Authorization': `Bearer ${token}` } })
      .then(res => res.ok ? res.json() : Promise.reject(new Error(`session-id HTTP ${res.status}`)))
      .then((data: { sessionId?: string }) => {
        const activeSessionId = data?.sessionId
        if (!activeSessionId) {
          console.log('[Terminal] no active session found')
          return
        }
        const msgsUrl = `${restScheme}://${host}:${port}/api/v1/messages?session_id=${activeSessionId}&limit=50`
        console.log('[Terminal] fetching history:', msgsUrl)
        return fetch(msgsUrl, { headers: { 'Authorization': `Bearer ${token}` } })
          .then(res => res.ok ? res.json() : Promise.reject(new Error(`messages HTTP ${res.status}`)))
      })
      .then((data?: { messages?: Array<{ type: string; content: string; seq: number }> }) => {
        if (!data?.messages || data.messages.length === 0) {
          console.log('[Terminal] no history messages')
          return
        }
        console.log(`[Terminal] loaded ${data.messages.length} history messages`)
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
        console.warn('[Terminal] history fetch failed:', err?.message)
      })
  }, [connected])

  const handleSend = () => {
    if (!input.trim() || !clientRef.current || !connected) return

    // 本地处理斜杠命令
    const trimmed = input.trim()
    if (trimmed === '/clear') {
      messageStore.getState().resetMessages()
      setInput('')
      return
    }
    if (trimmed === '/model') {
      fetchModelsAndShowPicker()
      setInput('')
      return
    }

    // 其他命令透传给 CLI
    messageStore.getState().addUserMessage(input, 'default-session')
    clientRef.current.send('session.input', { text: input })
      .catch(err => console.error('发送失败:', err))
    setInput('')
    setShowSlashPicker(false)
  }

  const handleInputChange = (text: string) => {
    setInput(text)
    // 检测斜杠命令输入
    if (text.startsWith('/') && !text.includes(' ')) {
      setSlashFilter(text)
      setShowSlashPicker(true)
    } else {
      setShowSlashPicker(false)
    }
  }

  const selectSlashCommand = (cmd: string) => {
    setInput(cmd + ' ')
    setShowSlashPicker(false)
  }

  const filteredCommands = SLASH_COMMANDS.filter(c =>
    c.cmd.toLowerCase().includes(slashFilter.toLowerCase())
  )

  const fetchModelsAndShowPicker = async () => {
    setLoadingModels(true)
    setShowModelPicker(true)
    try {
      const scheme = useWss ? 'https' : 'http'
      // 带 settings 参数获取当前配置商的模型列表
      let url = `${scheme}://${host}:${port}/api/v1/models`
      if (settingsPath) {
        url += `?settings=${encodeURIComponent(settingsPath)}`
      }
      console.log('[Models] fetching:', url)
      const res = await fetch(url)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: { label: string; value: string }[] = await res.json()
      setModels(data)
      if (data.length > 0) setSelectedModel(data[0].value)
    } catch (err) {
      console.warn('[Models] 获取失败:', err)
      setModels([
        { label: '默认模型', value: '' },
        { label: 'Sonnet 4.6', value: 'claude-sonnet-4-6' },
        { label: 'Opus 4.8', value: 'claude-opus-4-8' },
      ])
      setSelectedModel('')
    } finally {
      setLoadingModels(false)
    }
  }

  const applyModelSwitch = async () => {
    if (!clientRef.current) return
    try {
      // 保留原始 args（含 --settings 等），只替换 --model
      const args: string[] = []
      // 注入原始 runtime args（关键是 --settings <path>）
      for (let i = 0; i < runtimeArgs.length; i++) {
        if (runtimeArgs[i] === '--model') { i++; continue }
        args.push(runtimeArgs[i])
      }
      if (selectedModel) {
        args.push('--model', selectedModel)
      }
      console.log('[Terminal] restarting with args:', args)
      await clientRef.current.send('session.start', {
        command: 'claude',
        args,
        cwd: '',
        restart: true
      })
      Alert.alert('模型切换成功', `已切换到：${models.find(m => m.value === selectedModel)?.label || '默认模型'}`)
      setShowModelPicker(false)
    } catch (err: any) {
      Alert.alert('切换失败', err?.message || '未知错误')
    }
  }

  const handleAbort = () => {
    if (!clientRef.current) return
    clientRef.current.send('session.abort', {})
      .catch(err => console.error('停止失败:', err))
    // 本地立即重置 turnActive，不等待服务端 turn_end 事件
    setTurnActive(false)
    setThinking(false)
  }

  const statusBarHeight = Platform.OS === 'android' ? (StatusBar.currentHeight || 24) : 0

  // 派生：项目名（cwd 的 basename）、上下文用量文案
  const projectName = projectCwd ? projectCwd.replace(/[\\/]+$/, '').split(/[\\/]/).pop() || projectCwd : ''
  const contextUsage = formatContextUsage(contextWindow)

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: '#ededed', paddingTop: statusBarHeight }}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        keyboardVerticalOffset={0}
      >
        {/* 顶栏 */}
        <View style={{ paddingHorizontal: 16, paddingVertical: 12, backgroundColor: '#ededed', borderBottomWidth: 1, borderBottomColor: '#d9d9d9' }}>
          <View style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' }}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8, flexShrink: 1 }}>
              <Text style={{ fontSize: 17, fontWeight: '600', color: '#000' }}>Claude Code</Text>
              {projectName ? (
                <View style={{ flexDirection: 'row', alignItems: 'center', gap: 3, flexShrink: 1 }}>
                  <Text style={{ fontSize: 13, color: '#888' }}>📂</Text>
                  <Text style={{ fontSize: 13, color: '#666' }} numberOfLines={1}>{projectName}</Text>
                </View>
              ) : null}
            </View>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              {initialModel ? (
                <Text style={{ fontSize: 11, color: '#999' }} numberOfLines={1}>{initialModel}</Text>
              ) : null}
              {/* Git Diff 按钮 */}
              <Pressable
                onPress={() => setShowGitDiff(true)}
                style={({ pressed }) => ({
                  padding: 6,
                  borderRadius: 8,
                  backgroundColor: pressed ? '#d0d0d0' : '#f0f0f0',
                })}
              >
                <Text style={{ fontSize: 16 }}>📁</Text>
              </Pressable>
              <View style={{ flexDirection: 'row', alignItems: 'center', gap: 5 }}>
                <View style={{ width: 8, height: 8, borderRadius: 4, backgroundColor: connected ? '#34c759' : '#f44336' }} />
                <Text style={{ fontSize: 12, color: '#666' }}>{connected ? '已连接' : '未连接'}</Text>
              </View>
            </View>
          </View>
          {/* 上下文用量（有数据时显示） */}
          {contextUsage ? (
            <Text style={{ fontSize: 11, color: '#999', marginTop: 4 }}>{contextUsage}</Text>
          ) : null}
        </View>

        {/* 消息列表 */}
        <MessageList messages={messages} />

        {/* 斜杠命令下拉 */}
        {showSlashPicker && filteredCommands.length > 0 && (
          <View style={{
            maxHeight: 200,
            backgroundColor: '#fff',
            borderTopWidth: 1, borderTopColor: '#d9d9d9',
            borderBottomWidth: 1, borderBottomColor: '#d9d9d9',
          }}>
            <FlatList
              data={filteredCommands}
              keyExtractor={(item) => item.cmd}
              renderItem={({ item }) => (
                <Pressable
                  onPress={() => selectSlashCommand(item.cmd)}
                  style={({ pressed }) => ({
                    paddingHorizontal: 16, paddingVertical: 10,
                    backgroundColor: pressed ? '#f0f0f0' : '#fff',
                    borderBottomWidth: 0.5, borderBottomColor: '#eee',
                  })}
                >
                  <Text style={{ fontSize: 14, color: '#333', fontWeight: '500', marginBottom: 2 }}>
                    {item.cmd}
                  </Text>
                  <Text style={{ fontSize: 12, color: '#999' }}>{item.desc}</Text>
                </Pressable>
              )}
            />
          </View>
        )}

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
        <View style={{ paddingHorizontal: 12, paddingTop: 8, paddingBottom: 12, backgroundColor: '#f7f7f7', borderTopWidth: 1, borderTopColor: '#d9d9d9', flexDirection: 'row', gap: 10, alignItems: 'center' }}>
          <TextInput
            value={input}
            onChangeText={handleInputChange}
            placeholder={connected ? '输入消息，/ 打开命令...' : '连接中...'}
            placeholderTextColor="#bbb"
            editable={connected}
            multiline
            style={{ flex: 1, backgroundColor: '#fff', borderRadius: 21, borderWidth: 1, borderColor: '#d9d9d9', paddingHorizontal: 16, paddingVertical: 10, maxHeight: 100, color: '#000', fontSize: 15 }}
          />
          {turnActive ? (
            <Pressable
              onPress={handleAbort}
              hitSlop={8}
              style={{ width: 42, height: 42, borderRadius: 21, backgroundColor: '#f44336', alignItems: 'center', justifyContent: 'center' }}
            >
              <Text style={{ color: '#fff', fontSize: 13, fontWeight: '600' }}>停止</Text>
            </Pressable>
          ) : (
            <Pressable
              onPress={handleSend}
              disabled={!connected || !input.trim()}
              hitSlop={8}
              style={({ pressed }) => ({
                width: 42,
                height: 42,
                borderRadius: 21,
                backgroundColor: !connected || !input.trim() ? '#ccc' : (pressed ? '#256029' : '#2e7d32'),
                alignItems: 'center',
                justifyContent: 'center',
              })}
            >
              <Text style={{ color: '#fff', fontSize: 20, fontWeight: '600' }}>➜</Text>
            </Pressable>
          )}
        </View>
      </KeyboardAvoidingView>

      {/* Git Diff Modal */}
      <GitDiffModal
        visible={showGitDiff}
        onClose={() => setShowGitDiff(false)}
        host={host}
        port={port}
        token={token}
        useWss={useWss}
      />

      {/* 模型选择 Modal */}
      <Modal visible={showModelPicker} animationType="slide" onRequestClose={() => setShowModelPicker(false)}>
        <SafeAreaView style={{ flex: 1, backgroundColor: '#fff', paddingTop: statusBarHeight }}>
          <View style={{
            paddingHorizontal: 16, paddingVertical: 12,
            borderBottomWidth: 1, borderBottomColor: '#e5e5e5',
            flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center',
            backgroundColor: '#f7f7f7'
          }}>
            <Text style={{ fontSize: 17, fontWeight: '600', color: '#333' }}>选择模型</Text>
            <Pressable onPress={() => setShowModelPicker(false)} style={{ padding: 8 }}>
              <Text style={{ fontSize: 20, color: '#666' }}>✕</Text>
            </Pressable>
          </View>

          <View style={{ flex: 1, justifyContent: 'space-between', padding: 16 }}>
            {loadingModels ? (
              <ActivityIndicator size="large" style={{ marginTop: 40 }} />
            ) : (
              <Picker
                selectedValue={selectedModel}
                onValueChange={(value) => setSelectedModel(value)}
                style={{ flexGrow: 0 }}
              >
                {models.map((m) => (
                  <Picker.Item key={m.value} label={m.label} value={m.value} />
                ))}
              </Picker>
            )}

            <View style={{ flexDirection: 'row', gap: 12, marginTop: 20 }}>
              <View style={{ flex: 1 }}>
                <Button
                  title="取消"
                  onPress={() => setShowModelPicker(false)}
                  color="#666"
                />
              </View>
              <View style={{ flex: 1 }}>
                <Button
                  title="应用模型"
                  onPress={applyModelSwitch}
                  disabled={loadingModels}
                />
              </View>
            </View>
          </View>
        </SafeAreaView>
      </Modal>
    </SafeAreaView>
  )
}
