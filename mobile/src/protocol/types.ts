import type {
  EVT_AGENT_STATE,
  EVT_BASH_END,
  EVT_BASH_OUTPUT,
  EVT_BASH_START,
  EVT_CONTEXT_WINDOW,
  EVT_LIFECYCLE,
  EVT_PERMISSION_ASK,
  EVT_PERMISSION_REQ,
  EVT_PLAN_MODE,
  EVT_SESSION,
  EVT_TEXT,
  EVT_TEXT_DELTA,
  EVT_THINKING_END,
  EVT_THINKING_START,
  EVT_TOOL_END,
  EVT_TOOL_OUTPUT,
  EVT_TOOL_RESULT,
  EVT_TOOL_START,
  EVT_TOOL_USE,
  EVT_TURN_END
} from './protocol'

export type ConnectionStatus = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'closed'

export interface SessionMeta {
  id: string
  name: string
  agent: string
  model?: string
  status: string
  lastActivity?: string
}

export interface RequestEnvelope {
  type: 'req'
  id: string
  method: string
  params?: unknown
}

export interface ResponseEnvelope {
  type: 'resp'
  id: string
  ok: boolean
  result?: unknown
  error?: { code: string; message: string }
}

export interface EventEnvelope {
  type: 'evt'
  sessionId?: string
  event: AppEvent
}

export type Envelope = RequestEnvelope | ResponseEnvelope | EventEnvelope

export interface BaseEvent {
  type: string
  sessionId: string
  time: string
  seq?: number
  messageId?: string
}

export interface TextEvent extends BaseEvent { type: typeof EVT_TEXT; text: string; thinking?: string }
export interface TextDeltaEvent extends BaseEvent { type: typeof EVT_TEXT_DELTA; text: string; thinking?: string; blockIndex: number }
export interface LifecycleEvent extends BaseEvent { type: typeof EVT_LIFECYCLE; message: string }
export interface PermissionRequestEvent extends BaseEvent { type: typeof EVT_PERMISSION_REQ; toolName: string; message: string }
export interface PermissionAskEvent extends BaseEvent { type: typeof EVT_PERMISSION_ASK; toolName: string; message: string }
export interface TurnEndEvent extends BaseEvent { type: typeof EVT_TURN_END; text: string; message: string }
export interface ContextWindowEvent extends BaseEvent { type: typeof EVT_CONTEXT_WINDOW; toolInput: unknown }
export interface PlanModeEvent extends BaseEvent { type: typeof EVT_PLAN_MODE; toolInput: unknown }
export interface SessionEvent extends BaseEvent { type: typeof EVT_SESSION; toolInput: unknown }
export interface ToolUseEvent extends BaseEvent { type: typeof EVT_TOOL_USE; toolName: string; toolInput: unknown }
export interface ToolResultEvent extends BaseEvent { type: typeof EVT_TOOL_RESULT; toolName: string; toolResult: unknown }
export interface ToolStartEvent extends BaseEvent { type: typeof EVT_TOOL_START; toolId: string; toolName: string; toolInput: unknown }
export interface ToolOutputEvent extends BaseEvent { type: typeof EVT_TOOL_OUTPUT; toolId: string; toolOutput: string }
export interface ToolEndEvent extends BaseEvent { type: typeof EVT_TOOL_END; toolId: string; toolName: string }
export interface BashStartEvent extends BaseEvent { type: typeof EVT_BASH_START; toolId: string; toolName: string; toolInput: string }
export interface BashOutputEvent extends BaseEvent { type: typeof EVT_BASH_OUTPUT; toolId: string; toolOutput: string }
export interface BashEndEvent extends BaseEvent { type: typeof EVT_BASH_END; toolId: string; toolName: string }
export interface ThinkingStartEvent extends BaseEvent { type: typeof EVT_THINKING_START }
export interface ThinkingEndEvent extends BaseEvent { type: typeof EVT_THINKING_END }
export interface AgentStateEvent extends BaseEvent { type: typeof EVT_AGENT_STATE; state: string }

export type AppEvent =
  | TextEvent
  | TextDeltaEvent
  | LifecycleEvent
  | PermissionRequestEvent
  | PermissionAskEvent
  | TurnEndEvent
  | ContextWindowEvent
  | PlanModeEvent
  | SessionEvent
  | ToolUseEvent
  | ToolResultEvent
  | ToolStartEvent
  | ToolOutputEvent
  | ToolEndEvent
  | BashStartEvent
  | BashOutputEvent
  | BashEndEvent
  | ThinkingStartEvent
  | ThinkingEndEvent
  | AgentStateEvent
