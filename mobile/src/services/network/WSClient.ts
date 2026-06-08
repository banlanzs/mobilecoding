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
  private nextId = 0

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
    return new Promise<T>((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== 1) {
        this.requestQueue.push({ method, params, resolve: resolve as (v: unknown) => void, reject })
        return
      }
      this.doSend(method, params, resolve as (v: unknown) => void, reject)
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

  getStatus(): ConnectionStatus {
    return this.status
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
    this.ws = null
    this.setStatus('closed')
  }

  __unsafeHandleOpen() {
    if (!this.ws) {
      this.ws = this.makeSocket('')
    }
    this.handleOpen()
  }

  __unsafeHandleMessage(raw: string) {
    this.handleMessage(raw)
  }

  private handleOpen() {
    this.reconnectAttempt = 0
    this.setStatus('connected')
    const queue = [...this.requestQueue]
    this.requestQueue = []
    queue.forEach((item) => this.doSend(item.method, item.params, item.resolve, item.reject))
  }

  private makeId(): string {
    return `req-${++this.nextId}-${Date.now()}`
  }

  private doSend(method: string, params: unknown, resolve: (value: unknown) => void, reject: (reason: Error) => void) {
    const id = this.makeId()
    const timer = setTimeout(() => {
      this.pending.delete(id)
      reject(new Error(`request ${method} timed out after ${REQUEST_TIMEOUT}ms`))
    }, REQUEST_TIMEOUT)
    this.pending.set(id, { resolve, reject, timer })
    this.ws?.send(JSON.stringify({ type: 'req', id, method, params }))
  }

  private handleMessage(raw: string) {
    let envelope: Envelope
    try {
      envelope = JSON.parse(raw) as Envelope
    } catch {
      return
    }
    if (envelope.type === 'evt') {
      this.listeners.forEach((listener) => listener(envelope.event, envelope.sessionId))
    } else if (envelope.type === 'resp') {
      const entry = this.pending.get(envelope.id)
      if (!entry) return
      clearTimeout(entry.timer)
      this.pending.delete(envelope.id)
      if (envelope.ok) {
        entry.resolve(envelope.result)
      } else {
        entry.reject(new Error(envelope.error?.message ?? 'unknown error'))
      }
    }
  }

  private scheduleReconnect() {
    if (this.status === 'closed') return
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
