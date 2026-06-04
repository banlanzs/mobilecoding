// Relay WebSocket 客户端：通过 relay 服务器连接到 CLI
import type { ConnectionStatus, AppEvent } from './types';

type EventCallback = (event: AppEvent, sessionId?: string) => void;
type StatusCallback = (status: ConnectionStatus) => void;

const RECONNECT_DELAYS = [1000, 2000, 5000, 10000, 30000];

export interface RelayConfig {
  relayUrl: string;
  sessionId: string;
  pairingSecret: string;
}

export class RelayClient {
  private ws: WebSocket | null = null;
  private status: ConnectionStatus = 'idle';
  private reconnectAttempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closed = false;
  private config: RelayConfig | null = null;
  private clientId: string = '';

  private eventListeners = new Set<EventCallback>();
  private statusListeners = new Set<StatusCallback>();

  connect(config: RelayConfig): void {
    this.config = config;
    this.closed = false;
    this.reconnectAttempt = 0;
    this.doConnect();
  }

  close(): void {
    this.closed = true;
    this.clearReconnect();
    this.ws?.close(1000);
    this.ws = null;
    this.setStatus('closed');
  }

  sendText(text: string): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('not connected');
    }
    this.sendRelayForward(JSON.stringify({ text }));
  }

  sendPermissionAnswer(allow: boolean, toolName: string, requestId?: string): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('not connected');
    }
    // 优先使用 control_response 协议（带 requestId），否则用 permission_answer
    if (requestId) {
      this.sendRelayForward(JSON.stringify({
        type: 'control_response',
        response: { request_id: requestId, allow },
      }));
    } else {
      this.sendRelayForward(JSON.stringify({
        type: 'permission_answer',
        allow,
        toolName,
      }));
    }
  }

  // sendRespondPermission 走新协议（HTTP hook），与 sendPermissionAnswer 并存
  sendRespondPermission(requestId: string, allow: boolean, reason?: string): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('not connected');
    }
    this.sendRelayForward(JSON.stringify({
      type: 'permission.respond',
      requestId,
      allow,
      reason,
    }));
  }

  private sendRelayForward(payload: string): void {
    const envelope = {
      type: 'relay.forward',
      version: 1,
      sessionId: this.config?.sessionId,
      clientId: this.clientId,
      direction: 'client_to_agent',
      messageId: `msg_${Date.now()}`,
      contentType: 'mobilecoding.ws.v1',
      payload,
    };
    this.ws!.send(JSON.stringify(envelope));
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

  isConnected(): boolean {
    return this.status === 'connected';
  }

  private doConnect(): void {
    if (this.closed || !this.config) return;
    this.setStatus('connecting');

    const url = `${this.config.relayUrl}/relay/client`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectAttempt = 0;
      // 发送配对帧
      const pairFrame = {
        type: 'client.pair',
        version: 1,
        sessionId: this.config!.sessionId,
        pairingSecret: this.config!.pairingSecret,
        deviceName: navigator.userAgent,
      };
      this.ws!.send(JSON.stringify(pairFrame));
    };

    this.ws.onmessage = (event: MessageEvent) => {
      try {
        const frame = JSON.parse(event.data);
        this.handleFrame(frame);
      } catch {
        console.error('failed to parse relay message:', event.data);
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
      // onclose 会跟随 onerror
    };
  }

  private handleFrame(frame: any): void {
    switch (frame.type) {
      case 'client.paired':
        this.clientId = frame.clientId;
        this.setStatus('connected');
        break;

      case 'relay.forward':
        if (frame.direction === 'agent_to_client') {
          try {
            const payload = JSON.parse(frame.payload);
            // 转换为 AppEvent 格式
            const event: AppEvent = {
              type: payload.type || 'text',
              sessionId: frame.sessionId,
              time: new Date().toISOString(),
              ...payload,
            };
            this.eventListeners.forEach((cb) => {
              try {
                cb(event, frame.sessionId);
              } catch (err) {
                console.error('event listener error:', err);
              }
            });
          } catch {
            console.error('failed to parse relay payload:', frame.payload);
          }
        }
        break;

      case 'relay.error':
        console.error('relay error:', frame.code, frame.message);
        if (frame.code === 'agent_disconnected') {
          // Agent 断开，等待重连
          console.log('agent disconnected, waiting for reconnect...');
        }
        break;

      default:
        console.log('unknown relay frame:', frame.type);
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
}
