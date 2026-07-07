// WebSocket 协议类型定义
// 事件类型常量来自 protocol.ts（与 internal/protocol/protocol.go 同步）

import type {
  EVT_TEXT, EVT_TEXT_DELTA, EVT_LIFECYCLE, EVT_TOOL_USE, EVT_TOOL_RESULT,
  EVT_PERMISSION_REQ, EVT_PERMISSION_ASK, EVT_PLAN_MODE, EVT_CONTEXT_WINDOW,
  EVT_SESSION, EVT_THINKING_START, EVT_THINKING_END, EVT_TOOL_START,
  EVT_TOOL_OUTPUT, EVT_TOOL_END, EVT_BASH_START, EVT_BASH_OUTPUT,
  EVT_BASH_END, EVT_AGENT_STATE, EVT_TURN_END,
} from './protocol';

export type ConnectionStatus =
  | 'idle'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'closed';

// ─── Envelope（WebSocket 帧） ─────────────────────────────────────────────────

export interface RequestEnvelope {
  type: 'req';
  id: string;
  method: string;
  params?: unknown;
}

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

export interface EventEnvelope {
  type: 'evt';
  sessionId?: string;
  event: AppEvent;
}

export type Envelope = RequestEnvelope | ResponseEnvelope | EventEnvelope;

// ─── 投影事件类型 ─────────────────────────────────────────────────────────────

export type EventType =
  | typeof EVT_TEXT
  | typeof EVT_TEXT_DELTA
  | typeof EVT_LIFECYCLE
  | typeof EVT_TOOL_USE
  | typeof EVT_TOOL_RESULT
  | typeof EVT_PERMISSION_REQ
  | typeof EVT_PERMISSION_ASK
  | typeof EVT_PLAN_MODE
  | typeof EVT_CONTEXT_WINDOW
  | typeof EVT_SESSION
  | typeof EVT_THINKING_START
  | typeof EVT_THINKING_END
  | typeof EVT_TOOL_START
  | typeof EVT_TOOL_OUTPUT
  | typeof EVT_TOOL_END
  | typeof EVT_BASH_START
  | typeof EVT_BASH_OUTPUT
  | typeof EVT_BASH_END
  | typeof EVT_AGENT_STATE
  | typeof EVT_TURN_END;

// ─── 基础事件接口 ─────────────────────────────────────────────────────────────

export interface BaseEvent {
  type: EventType;
  sessionId: string;
  time: string; // RFC3339 ISO 8601
  seq?: number; // 消息序列号，用于断线重连补发
  messageId?: string; // 后端生成，用于去重和 requestId 关联
}

// ─── 具体事件接口（保留类型窄化能力） ───────────────────────────────────────

export interface TextEvent extends BaseEvent {
  type: typeof EVT_TEXT;
  text: string;
  thinking?: string;
}

export interface TextDeltaEvent extends BaseEvent {
  type: typeof EVT_TEXT_DELTA;
  text: string;
  thinking?: string;
  blockIndex: number;
}

export interface LifecycleEvent extends BaseEvent {
  type: typeof EVT_LIFECYCLE;
  message: string;
}

export interface ToolUseEvent extends BaseEvent {
  type: typeof EVT_TOOL_USE;
  toolName: string;
  toolInput: unknown;
}

export interface ToolResultEvent extends BaseEvent {
  type: typeof EVT_TOOL_RESULT;
  toolName: string;
  toolResult: unknown;
}

export interface PermissionRequestEvent extends BaseEvent {
  type: typeof EVT_PERMISSION_REQ;
  toolName: string;
  message: string;
}

export interface PermissionAskEvent extends BaseEvent {
  type: typeof EVT_PERMISSION_ASK;
  toolName: string;
  message: string;
}

export interface TurnEndEvent extends BaseEvent {
  type: typeof EVT_TURN_END;
  text: string;
  message: string;
}

export interface PlanModeEvent extends BaseEvent {
  type: typeof EVT_PLAN_MODE;
  toolInput: unknown;
}

export interface ContextWindowEvent extends BaseEvent {
  type: typeof EVT_CONTEXT_WINDOW;
  toolInput: unknown;
}

export interface SessionEvent extends BaseEvent {
  type: typeof EVT_SESSION;
  toolInput: unknown;
}

export interface ThinkingStartEvent extends BaseEvent { type: typeof EVT_THINKING_START; }
export interface ThinkingEndEvent extends BaseEvent { type: typeof EVT_THINKING_END; }

export interface ToolStartEvent extends BaseEvent {
  type: typeof EVT_TOOL_START;
  toolId: string;
  toolName: string;
  toolInput: unknown;
}

export interface ToolOutputEvent extends BaseEvent {
  type: typeof EVT_TOOL_OUTPUT;
  toolId: string;
  toolOutput: string;
}

export interface ToolEndEvent extends BaseEvent {
  type: typeof EVT_TOOL_END;
  toolId: string;
  toolName: string;
}

export interface BashStartEvent extends BaseEvent {
  type: typeof EVT_BASH_START;
  toolId: string;
  toolName: string;
  toolInput: string;
}

export interface BashOutputEvent extends BaseEvent {
  type: typeof EVT_BASH_OUTPUT;
  toolId: string;
  toolOutput: string;
}

export interface BashEndEvent extends BaseEvent {
  type: typeof EVT_BASH_END;
  toolId: string;
  toolName: string;
}

export interface AgentStateEvent extends BaseEvent {
  type: typeof EVT_AGENT_STATE;
  state: string;
}

// ─── 联合类型 ─────────────────────────────────────────────────────────────────

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

// ─── 前端内部类型 ─────────────────────────────────────────────────────────────

export interface UserMessage {
  type: 'user';
  sessionId: string;
  time: string;
  text: string;
}

export type DisplayMessage = AppEvent | UserMessage;

// ─── RPC 方法参数 ─────────────────────────────────────────────────────────────

export interface RuntimeConfig {
  defaultCommand: string;
  defaultArgs: string[];
  launchMode?: 'managed' | 'remote-control';
  cwd?: string;
}

export interface SessionStartParams {
  command: string;
  args?: string[];
  cwd?: string;
  restart?: boolean;
  resumeSessionId?: string; // Claude 内部 session_id，用于恢复历史会话
}

export interface SessionStartResult {
  sessionId: string;
}

export interface SessionInputParams {
  text: string;
}

export interface SessionListParams {}

export interface SessionListResult {
  sessions: SessionMeta[];
}

export interface SessionMeta {
  id: string;
  name: string;
  agent: string;
  model?: string;
  cwd?: string;
  status: string;
  resumeSessionId?: string; // Claude 内部 session_id，用于 --resume 续聊
  command?: string;         // 启动命令，用于恢复
  args?: string[];          // 启动参数，用于恢复
  createdAt: string;
  updatedAt: string;
  lastActiveAt: string;
  messageCount: number;
}

export interface RPCError {
  code: string;
  message: string;
}
