// 协议常量 — 与 internal/protocol/protocol.go 保持同步
// 这是前后端的唯一协议契约

// Envelope 类型
export const ENV_TYPE_REQ = 'req' as const;
export const ENV_TYPE_RESP = 'resp' as const;
export const ENV_TYPE_EVT = 'evt' as const;

// RPC 方法名
export const METHOD_SESSION_START = 'session.start' as const;
export const METHOD_SESSION_INPUT = 'session.input' as const;
export const METHOD_SESSION_STOP = 'session.stop' as const;
export const METHOD_SESSION_ABORT = 'session.abort' as const;
export const METHOD_SESSION_PERMISSION_ANSWER = 'session.permission.answer' as const;
export const METHOD_PERMISSION_RESPOND = 'permission.respond' as const;

// RPC 错误码
export const ERR_PROTOCOL_ERROR = 'protocol_error' as const;
export const ERR_NOT_FOUND = 'not_found' as const;
export const ERR_ENGINE_FAILURE = 'engine_failure' as const;
export const ERR_CONFLICT = 'conflict' as const;
export const ERR_NOT_CONFIGURED = 'not_configured' as const;
export const ERR_STALE_REQUEST = 'stale_request' as const;

// 投影事件类型
export const EVT_TEXT = 'text' as const;
export const EVT_TEXT_DELTA = 'text_delta' as const;
export const EVT_LIFECYCLE = 'lifecycle' as const;
export const EVT_TOOL_USE = 'tool_use' as const;
export const EVT_TOOL_RESULT = 'tool_result' as const;
export const EVT_PERMISSION_REQ = 'permission_request' as const;
export const EVT_PERMISSION_ASK = 'permission_ask' as const;
export const EVT_PLAN_MODE = 'plan_mode' as const;
export const EVT_CONTEXT_WINDOW = 'context_window' as const;
export const EVT_SESSION = 'session' as const;
export const EVT_THINKING_START = 'thinking_start' as const;
export const EVT_THINKING_END = 'thinking_end' as const;
export const EVT_TOOL_START = 'tool_start' as const;
export const EVT_TOOL_OUTPUT = 'tool_output' as const;
export const EVT_TOOL_END = 'tool_end' as const;
export const EVT_BASH_START = 'bash_start' as const;
export const EVT_BASH_OUTPUT = 'bash_output' as const;
export const EVT_BASH_END = 'bash_end' as const;
export const EVT_AGENT_STATE = 'agent_state' as const;
export const EVT_TURN_END = 'turn_end' as const;

// 事件类型联合
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
