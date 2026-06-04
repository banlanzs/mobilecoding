// WebSocket 客户端：连接管理、请求/响应关联、事件流
import type {
  ConnectionStatus,
  Envelope,
  AppEvent,
  SessionStartParams,
  SessionStartResult,
} from './types';

type EventCallback = (event: AppEvent, sessionId?: string) => void;
type StatusCallback = (status: ConnectionStatus) => void;

const RECONNECT_DELAYS = [1000, 2000, 5000, 10000, 30000];
const REQUEST_TIMEOUT = 30_000;

export class WSClient {
  private ws: WebSocket | null = null;
  private token: string = '';
  private status: ConnectionStatus = 'idle';
  private reconnectAttempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closed = false;

  private pendingRequests = new Map<
    string,
    {
      resolve: (value: unknown) => void;
      reject: (reason: Error) => void;
      timer: ReturnType<typeof setTimeout>;
    }
  >();

  private eventListeners = new Set<EventCallback>();
  private statusListeners = new Set<StatusCallback>();

  connect(token: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) return;
    this.token = token;
    this.closed = false;
    this.reconnectAttempt = 0;
    this.doConnect();
  }

  close(): void {
    this.closed = true;
    this.clearReconnect();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.ws?.close(1000);
    this.ws = null;
    this.setStatus('closed');
    this.rejectAllPending(new Error('connection closed'));
  }

  async send<T = unknown>(method: string, params?: unknown): Promise<T> {
    return new Promise<T>((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error('not connected'));
        return;
      }

      const id = crypto.randomUUID();
      const timer = setTimeout(() => {
        this.pendingRequests.delete(id);
        reject(new Error(`request ${method} timed out after ${REQUEST_TIMEOUT}ms`));
      }, REQUEST_TIMEOUT);

      this.pendingRequests.set(id, {
        resolve: resolve as (value: unknown) => void,
        reject,
        timer,
      });

      const envelope: Envelope = {
        type: 'req',
        id,
        method,
        params,
      };
      this.ws.send(JSON.stringify(envelope));
    });
  }

  async startSession(params: SessionStartParams): Promise<SessionStartResult> {
    return this.send<SessionStartResult>('session.start', params);
  }

  async sendInput(text: string): Promise<void> {
    await this.send('session.input', { text });
  }

  async answerPermission(allow: boolean, toolName: string): Promise<void> {
    await this.send('session.permission.answer', { allow, toolName });
  }

  async stopSession(): Promise<void> {
    await this.send('session.stop');
  }

  onEvent(cb: EventCallback): () => void {
    this.eventListeners.add(cb);
    return () => this.eventListeners.delete(cb);
  }

  onStatus(cb: StatusCallback): () => void {
    this.statusListeners.add(cb);
    return () => this.statusListeners.delete(cb);
  }

  getStatus(): ConnectionStatus {
    return this.status;
  }

  getToken(): string {
    return this.token;
  }

  isConnected(): boolean {
    return this.status === 'connected';
  }

  private doConnect(): void {
    if (this.closed) return;
    this.setStatus('connecting');

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${protocol}//${location.host}/api/v1/ws?token=${encodeURIComponent(this.token)}`;

    this.ws = new WebSocket(url);
    this.ws.onopen = () => {
      this.reconnectAttempt = 0;
      this.setStatus('connected');
    };

    this.ws.onmessage = (event: MessageEvent) => {
      try {
        const envelope: Envelope = JSON.parse(event.data);
        this.handleEnvelope(envelope);
      } catch {
        console.error('failed to parse ws message:', event.data);
      }
    };

    this.ws.onclose = (event: CloseEvent) => {
      if (event.code === 1000 || this.closed) {
        this.setStatus('closed');
        return;
      }
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      // onclose 会跟随 onerror，由 onclose 处理重连
    };
  }

  private handleEnvelope(env: Envelope): void {
    if (env.type === 'resp') {
      const pending = this.pendingRequests.get(env.id);
      if (!pending) return;
      this.pendingRequests.delete(env.id);
      clearTimeout(pending.timer);

      if (env.ok) {
        pending.resolve(env.result);
      } else {
        pending.reject(new Error(env.error?.message || 'rpc error'));
      }
      return;
    }

    if (env.type === 'evt') {
      this.eventListeners.forEach((cb) => {
        try {
          cb(env.event, env.sessionId);
        } catch (err) {
          console.error('event listener error:', err);
        }
      });
    }
  }

  private scheduleReconnect(): void {
    if (this.closed) return;
    const delay = RECONNECT_DELAYS[Math.min(this.reconnectAttempt, RECONNECT_DELAYS.length - 1)];
    this.reconnectAttempt++;
    this.setStatus('reconnecting');
    this.reconnectTimer = setTimeout(() => this.doConnect(), delay);
  }

  private setStatus(status: ConnectionStatus): void {
    if (this.status === status) return;
    this.status = status;
    this.statusListeners.forEach((cb) => {
      try {
        cb(status);
      } catch (err) {
        console.error('status listener error:', err);
      }
    });
  }

  private clearReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private rejectAllPending(error: Error): void {
    this.pendingRequests.forEach(({ reject, timer }) => {
      clearTimeout(timer);
      reject(error);
    });
    this.pendingRequests.clear();
  }
}