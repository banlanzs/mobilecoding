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
  | 'text_delta'
  | 'lifecycle'
  | 'tool_use'
  | 'tool_result'
  | 'permission_request'
  | 'permission_ask'
  | 'plan_mode'
  | 'context_window'
  | 'session'
  | 'thinking_start'
  | 'thinking_end'
  | 'tool_start'
  | 'tool_output'
  | 'tool_end'
  | 'bash_start'
  | 'bash_output'
  | 'bash_end'
  | 'agent_state'
  | 'turn_end';

export interface BaseEvent {
  type: EventType;
  sessionId: string;
  time: string; // RFC3339 ISO 8601
  messageId?: string; // 后端生成，用于去重和 requestId 关联
}

export interface TextEvent extends BaseEvent {
  type: 'text';
  text: string;
  thinking?: string;
}

export interface TextDeltaEvent extends BaseEvent {
  type: 'text_delta';
  text: string;
  thinking?: string;
  blockIndex: number;
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

export interface PermissionAskEvent extends BaseEvent {
  type: 'permission_ask';
  toolName: string;
  message: string;
  // messageId 来自 BaseEvent，承载 Claude stdio control_request 的 request_id
}

export interface TurnEndEvent extends BaseEvent {
  type: 'turn_end';
  text: string;
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

// 新增统一 Agent 事件
export interface ThinkingStartEvent extends BaseEvent { type: 'thinking_start'; }
export interface ThinkingEndEvent extends BaseEvent { type: 'thinking_end'; }
export interface ToolStartEvent extends BaseEvent {
  type: 'tool_start';
  toolId: string;
  toolName: string;
  toolInput: unknown;
}
export interface ToolOutputEvent extends BaseEvent {
  type: 'tool_output';
  toolId: string;
  toolOutput: string;
}
export interface ToolEndEvent extends BaseEvent {
  type: 'tool_end';
  toolId: string;
  toolName: string;
}
export interface BashStartEvent extends BaseEvent {
  type: 'bash_start';
  toolId: string;
  toolName: string;
  toolInput: string;
}
export interface BashOutputEvent extends BaseEvent {
  type: 'bash_output';
  toolId: string;
  toolOutput: string;
}
export interface BashEndEvent extends BaseEvent {
  type: 'bash_end';
  toolId: string;
  toolName: string;
}
export interface AgentStateEvent extends BaseEvent {
  type: 'agent_state';
  state: string;
}

export type AppEvent =
  | TextEvent
  | TextDeltaEvent
  | LifecycleEvent
  | ToolUseEvent
  | ToolResultEvent
  | PermissionRequestEvent
  | PermissionAskEvent
  | PlanModeEvent
  | ContextWindowEvent
  | SessionEvent
  | ThinkingStartEvent
  | ThinkingEndEvent
  | ToolStartEvent
  | ToolOutputEvent
  | ToolEndEvent
  | BashStartEvent
  | BashOutputEvent
  | BashEndEvent
  | AgentStateEvent
  | TurnEndEvent;

// 用户消息（前端合成，用于回显）
export interface UserMessage {
  type: 'user';
  sessionId: string;
  time: string;
  text: string;
}

export type DisplayMessage = AppEvent | UserMessage;

// RPC 方法参数
export interface RuntimeConfig {
  defaultCommand: string;
  defaultArgs: string[];
  cwd?: string;
}

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
