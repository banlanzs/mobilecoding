// 统一连接接口：支持直接连接和 relay 连接
import type { ConnectionStatus, AppEvent, SessionStartParams, SessionStartResult } from './types';

export interface Connection {
  connect(params: any): void;
  close(): void;
  send(method: string, params?: unknown): Promise<unknown>;
  sendText(text: string): void;
  onEvent(cb: (event: AppEvent, sessionId?: string) => void): () => void;
  onStatus(cb: (status: ConnectionStatus) => void): () => void;
  getStatus(): ConnectionStatus;
  isConnected(): boolean;
}
