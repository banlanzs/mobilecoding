// WebSocket 协议类型定义（镜像 internal/ws/codec.go + internal/projection/event.go）

export type ConnectionStatus =
  | 'idle'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'closed';

// 客户端 → 服务端请求
export interface RequestEnvelope {
  type: 'req';
  id: string;
  method: string;
  params?: unknown;
}

// 服务端 → 客户端响应
export interface ResponseOkEnvelope {
  type: 'resp';
  id: string;
  ok: true;
  result?: unknown;
}

export interface ResponseErrEnvelope {
  type: 'resp';
  id: string;
  ok: false;
  error: { code: string; message: string };
}

export type ResponseEnvelope = ResponseOkEnvelope | ResponseErrEnvelope;

// 服务端 → 客户端事件
export interface EventEnvelope {
  type: 'evt';
  sessionId?: string;
  event: AppEvent;
}

export type Envelope = RequestEnvelope | ResponseEnvelope | EventEnvelope;

// 投影事件类型
export type EventType =
  | 'text'
  | 'lifecycle'
  | 'tool_use'
  | 'tool_result'
  | 'permission_request'
  | 'plan_mode'
  | 'context_window'
  | 'session';

export interface BaseEvent {
  type: EventType;
  sessionId: string;
  time: string; // RFC3339 ISO 8601
}

export interface TextEvent extends BaseEvent {
  type: 'text';
  text: string;
}

export interface LifecycleEvent extends BaseEvent {
  type: 'lifecycle';
  message: string;
}

export interface ToolUseEvent extends BaseEvent {
  type: 'tool_use';
  toolName: string;
  toolInput: unknown;
}

export interface ToolResultEvent extends BaseEvent {
  type: 'tool_result';
  toolName: string;
  toolResult: unknown;
}

export interface PermissionRequestEvent extends BaseEvent {
  type: 'permission_request';
  toolName: string;
  message: string;
}

export interface PlanModeEvent extends BaseEvent {
  type: 'plan_mode';
  toolInput: unknown;
}

export interface ContextWindowEvent extends BaseEvent {
  type: 'context_window';
  toolInput: unknown;
}

export interface SessionEvent extends BaseEvent {
  type: 'session';
  toolInput: unknown;
}

export type AppEvent =
  | TextEvent
  | LifecycleEvent
  | ToolUseEvent
  | ToolResultEvent
  | PermissionRequestEvent
  | PlanModeEvent
  | ContextWindowEvent
  | SessionEvent;

// 用户消息（前端合成，用于回显）
export interface UserMessage {
  type: 'user';
  sessionId: string;
  time: string;
  text: string;
}

export type DisplayMessage = AppEvent | UserMessage;

// RPC 方法参数
export interface SessionStartParams {
  command: string;
  args?: string[];
  cwd?: string;
}

export interface SessionStartResult {
  sessionId: string;
}

export interface SessionInputParams {
  text: string;
}

export interface RPCError {
  code: string;
  message: string;
}
