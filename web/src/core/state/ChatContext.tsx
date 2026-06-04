// 全局聊天状态管理：WebSocket 连接 + 会话 + 消息流
import {
  createContext,
  useContext,
  useReducer,
  useEffect,
  useCallback,
  useRef,
  type PropsWithChildren,
} from 'react';
import { useWebSocket } from '../ws/useWebSocket';
import type { WSClient } from '../ws/ws-client';
import { RelayClient, type RelayConfig } from '../ws/relay-client';
import type {
  ConnectionStatus,
  AppEvent,
  DisplayMessage,
  UserMessage,
  TextDeltaEvent,
  SessionStartParams,
  SessionStartResult,
  PermissionRequestEvent,
  RuntimeConfig,
} from '../ws/types';

const MAX_MESSAGES = 500;
const STORED_MESSAGES_KEY = 'mobilecoding.messages';
const MAX_STORED_MESSAGES = 200;

function saveMessages(msgs: DisplayMessage[]): void {
  try {
    const toStore = msgs.slice(-MAX_STORED_MESSAGES);
    localStorage.setItem(STORED_MESSAGES_KEY, JSON.stringify(toStore));
  } catch {}
}

function loadMessages(): DisplayMessage[] {
  try {
    const raw = localStorage.getItem(STORED_MESSAGES_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.slice(-MAX_STORED_MESSAGES);
  } catch {
    return [];
  }
}

export interface AgentStateInfo {
  status: string;       // "idle" | "thinking" | "reading_files" | "editing_files" | "running_command"
  toolName?: string;
  since: number;        // Date.now() 时间戳
}

export interface ChatState {
  status: ConnectionStatus;
  sessionId: string | null;
  messages: DisplayMessage[];
  permissionPrompt: PermissionRequestEvent | null;
  lastError: string | null;
  runtime: RuntimeConfig;
  connectionMode: 'direct' | 'relay';
  thinking: boolean;
  agentState: AgentStateInfo;
}

type Action =
  | { type: 'STATUS_CHANGED'; status: ConnectionStatus }
  | { type: 'RUNTIME_LOADED'; runtime: RuntimeConfig }
  | { type: 'SESSION_STARTED'; sessionId: string }
  | { type: 'SESSION_STOPPED' }
  | { type: 'EVENT_RECEIVED'; event: AppEvent; sessionId?: string }
  | { type: 'USER_MESSAGE_SENT'; text: string; sessionId: string }
  | { type: 'PERMISSION_ANSWERED' }
  | { type: 'ABORT_TURN' }
  | { type: 'ERROR'; error: string }
  | { type: 'SET_CONNECTION_MODE'; mode: 'direct' | 'relay' };

function reducer(state: ChatState, action: Action): ChatState {
  switch (action.type) {
    case 'STATUS_CHANGED':
      return { ...state, status: action.status, lastError: action.status === 'closed' ? state.lastError : null };
    case 'SESSION_STARTED': {
      // 新会话启动，清除旧消息
      try {
        localStorage.setItem('mobilecoding.sessionId', action.sessionId);
        localStorage.removeItem('mobilecoding.messages');
      } catch {}
      return { ...state, sessionId: action.sessionId, messages: [], lastError: null };
    }
    case 'SESSION_STOPPED':
      try { localStorage.removeItem('mobilecoding.sessionId'); } catch {}
      return { ...state, sessionId: null, permissionPrompt: null, thinking: false, agentState: { status: 'idle', since: Date.now() } };
    case 'RUNTIME_LOADED':
      return { ...state, runtime: action.runtime };
    case 'SET_CONNECTION_MODE':
      return { ...state, connectionMode: action.mode };
    case 'EVENT_RECEIVED': {
      const ev = action.event;
      let messages: DisplayMessage[];

      if (ev.type === 'text_delta') {
        // 增量文本：追加到最后一张 text_delta 卡片，或创建新卡片
        const last = state.messages[state.messages.length - 1];
        if (last && last.type === 'text_delta' && (last as TextDeltaEvent).blockIndex === ev.blockIndex) {
          const lastDelta = last as TextDeltaEvent;
          const merged: TextDeltaEvent = {
            ...lastDelta,
            text: lastDelta.text + ev.text,
            thinking: lastDelta.thinking && ev.thinking
              ? lastDelta.thinking + '\n\n' + ev.thinking
              : (lastDelta.thinking || ev.thinking || undefined),
          };
          messages = [...state.messages.slice(0, -1), merged as DisplayMessage];
        } else {
          messages = [...state.messages, ev as DisplayMessage];
        }
      } else if (ev.type === 'text') {
        // 完整文本：替换同一块的 text_delta 卡片
        const last = state.messages[state.messages.length - 1];
        if (last && last.type === 'text_delta') {
          messages = [...state.messages.slice(0, -1), ev as DisplayMessage];
        } else {
          messages = [...state.messages, ev as DisplayMessage];
        }
      } else {
        messages = [...state.messages, ev as DisplayMessage];
      }

      if (messages.length > MAX_MESSAGES) {
        messages.splice(0, messages.length - MAX_MESSAGES);
      }
      saveMessages(messages);
      const next: ChatState = {
        ...state,
        messages,
        sessionId: action.sessionId || state.sessionId,
      };
      if (ev.type === 'permission_request') {
        next.permissionPrompt = ev;
      }
      // 只有实际文本内容到达才结束 thinking（lifecycle 仅是指示器，不改变状态）
      if (ev.type === 'text' || ev.type === 'text_delta') {
        next.thinking = false;
      }
      // 更新 Agent 状态
      const as = agentStateFromEvent(ev);
      if (as) {
        next.agentState = { ...state.agentState, ...as };
      }
      return next;
    }
    case 'USER_MESSAGE_SENT': {
      const userMsg: UserMessage = {
        type: 'user',
        sessionId: action.sessionId,
        time: new Date().toISOString(),
        text: action.text,
      };
      const messages = [...state.messages, userMsg];
      if (messages.length > MAX_MESSAGES) {
        messages.splice(0, messages.length - MAX_MESSAGES);
      }
      saveMessages(messages);
      return { ...state, messages, thinking: true };
    }
    case 'PERMISSION_ANSWERED':
      return { ...state, permissionPrompt: null };
    case 'ABORT_TURN':
      return { ...state, thinking: false };
    case 'ERROR':
      return { ...state, lastError: action.error };
    default:
      return state;
  }
}

// 尝试从 localStorage 恢复 sessionId（支持页面刷新）
function savedSessionId(): string | null {
  try { return localStorage.getItem('mobilecoding.sessionId'); } catch { return null; }
}

const initialState: ChatState = {
  status: 'idle',
  sessionId: savedSessionId(),
  messages: loadMessages(),
  permissionPrompt: null,
  lastError: null,
  runtime: { defaultCommand: '', defaultArgs: [], cwd: '' },
  connectionMode: 'direct',
  thinking: false,
  agentState: { status: 'idle', since: Date.now() },
};

// 根据事件类型推导 Agent 状态
function agentStateFromEvent(ev: AppEvent): Partial<AgentStateInfo> | null {
  switch (ev.type) {
    case 'thinking_start': return { status: 'thinking', since: Date.now() };
    case 'tool_start':
      if (/read|grep|glob|search|find|cat/i.test(ev.toolName)) return { status: 'reading_files', toolName: ev.toolName, since: Date.now() };
      if (/edit|write|replace|patch|create/i.test(ev.toolName)) return { status: 'editing_files', toolName: ev.toolName, since: Date.now() };
      return { status: 'running_command', toolName: ev.toolName, since: Date.now() };
    case 'bash_start': return { status: 'running_command', toolName: 'Bash', since: Date.now() };
    case 'thinking_end':
    case 'tool_end':
    case 'bash_end':
      return { status: 'idle', since: Date.now() };
    default: return null;
  }
}

interface ChatContextValue {
  state: ChatState;
  ws: WSClient;
  sendStart: (params: SessionStartParams) => Promise<SessionStartResult>;
  sendInput: (text: string) => Promise<void>;
  sendStop: () => Promise<void>;
  abortTurn: () => Promise<void>;
  answerPermission: (allow: boolean, toolName: string) => Promise<void>;
  dismissPermission: () => void;
  connectRelay: (config: RelayConfig) => void;
  disconnectRelay: () => void;
}

const ChatContext = createContext<ChatContextValue | null>(null);

export function ChatProvider({ children }: PropsWithChildren) {
  const { client, status, connect } = useWebSocket();
  const [state, dispatch] = useReducer(reducer, initialState);
  const runtimeRef = useRef<RuntimeConfig>(initialState.runtime);
  const relayClientRef = useRef<RelayClient | null>(null);

  // 同步连接状态
  useEffect(() => {
    dispatch({ type: 'STATUS_CHANGED', status });
  }, [status]);

  // 订阅 WebSocket 事件
  useEffect(() => {
    const off = client.onEvent((event, sessionId) => {
      dispatch({ type: 'EVENT_RECEIVED', event, sessionId });
    });
    return off;
  }, [client]);

  // 自动连接（direct 模式）
  useEffect(() => {
    if (state.connectionMode !== 'direct') return;
    const token = resolveToken();
    if (token) {
      connect(token);
    }
  }, [connect, state.connectionMode]);

  useEffect(() => {
    if (status !== 'connected' || state.connectionMode !== 'direct') return;
    fetch('/version')
      .then((r) => r.json())
      .then((data: { runtime?: RuntimeConfig }) => {
        if (data.runtime) {
          runtimeRef.current = data.runtime;
          dispatch({ type: 'RUNTIME_LOADED', runtime: data.runtime });
        }
      })
      .catch(() => {});
  }, [status, state.connectionMode]);

  // Relay 连接方法
  const connectRelay = useCallback((config: RelayConfig) => {
    // 断开现有连接
    if (relayClientRef.current) {
      relayClientRef.current.close();
    }

    const relayClient = new RelayClient();
    relayClientRef.current = relayClient;

    // 订阅事件
    relayClient.onEvent((event, sessionId) => {
      dispatch({ type: 'EVENT_RECEIVED', event, sessionId });
    });

    relayClient.onStatus((newStatus) => {
      dispatch({ type: 'STATUS_CHANGED', status: newStatus });
    });

    // 连接
    dispatch({ type: 'SET_CONNECTION_MODE', mode: 'relay' });
    relayClient.connect(config);
    // 注意：不自动设置 sessionId，用户需要在手机上选择 CLI 启动
  }, []);

  // 自动连接：检查 URL 参数是否包含 relay 配对信息
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const isRelay = params.get('relay');
    const session = params.get('session');
    const secret = params.get('secret');

    if (isRelay && session && secret) {
      // URL 包含 relay 参数，自动连接 relay
      const wsBase = location.protocol === 'https:'
        ? `wss://${location.host}`
        : `ws://${location.host}`;
      const relayUrl = `${wsBase}/relay`;

      connectRelay({ relayUrl, sessionId: session, pairingSecret: secret });
      // 清除 URL 参数避免重复连接
      history.replaceState(null, '', location.pathname + location.hash);
    }
  }, [connectRelay]); // eslint-disable-line react-hooks/exhaustive-deps

  // 断开 Relay 连接
  const disconnectRelay = useCallback(() => {
    if (relayClientRef.current) {
      relayClientRef.current.close();
      relayClientRef.current = null;
    }
    dispatch({ type: 'SESSION_STOPPED' });
    dispatch({ type: 'SET_CONNECTION_MODE', mode: 'direct' });
  }, []);

  const sendStart = useCallback(
    async (params: SessionStartParams): Promise<SessionStartResult> => {
      if (state.connectionMode === 'relay' && relayClientRef.current) {
        // Relay 模式：通过 relay 发送启动命令
        const payload = JSON.stringify({
          type: 'session.start',
          command: params.command,
          args: params.args || [],
          cwd: params.cwd || '',
        });
        relayClientRef.current.sendText(payload);
        const sid = 'relay_session';
        dispatch({ type: 'SESSION_STARTED', sessionId: sid });
        return { sessionId: sid };
      }
      const result = await client.startSession(params);
      if (result.sessionId) {
        dispatch({ type: 'SESSION_STARTED', sessionId: result.sessionId });
      }
      return result;
    },
    [client, state.connectionMode]
  );

  const sendInput = useCallback(
    async (text: string): Promise<void> => {
      const sid = state.sessionId;
      if (!sid) throw new Error('no active session');
      dispatch({ type: 'USER_MESSAGE_SENT', text, sessionId: sid });

      if (state.connectionMode === 'relay' && relayClientRef.current) {
        relayClientRef.current.sendText(text);
      } else {
        try {
          await client.sendInput(text);
        } catch (err) {
          // 会话已死：自动重置状态
          const msg = err instanceof Error ? err.message : String(err);
          if (msg.includes('no active runner') || msg.includes('engine_failure') || msg.includes('context canceled')) {
            dispatch({ type: 'SESSION_STOPPED' });
            dispatch({ type: 'ERROR', error: '会话已断开，请重新启动' });
          }
          throw err;
        }
      }
    },
    [client, state.sessionId, state.connectionMode]
  );

  const sendStop = useCallback(async (): Promise<void> => {
    if (state.connectionMode === 'relay') {
      disconnectRelay();
    } else {
      await client.stopSession();
      dispatch({ type: 'SESSION_STOPPED' });
    }
  }, [client, state.connectionMode, disconnectRelay]);

  const dismissPermission = useCallback(() => {
    dispatch({ type: 'PERMISSION_ANSWERED' });
  }, []);

  const abortTurn = useCallback(async (): Promise<void> => {
    dispatch({ type: 'ABORT_TURN' });
    if (state.connectionMode === 'relay' && relayClientRef.current) {
      relayClientRef.current.sendText(JSON.stringify({ type: 'session.abort' }));
    } else {
      await client.abortTurn();
    }
  }, [client, state.connectionMode]);

  const answerPermission = useCallback(
    async (allow: boolean, toolName: string): Promise<void> => {
      dispatch({ type: 'PERMISSION_ANSWERED' });
      if (state.connectionMode === 'relay' && relayClientRef.current) {
        relayClientRef.current.sendPermissionAnswer(allow, toolName);
      } else {
        await client.answerPermission(allow, toolName);
      }
    },
    [client, state.connectionMode]
  );

  const value: ChatContextValue = {
    state,
    ws: client,
    sendStart,
    sendInput,
    sendStop,
    abortTurn,
    answerPermission,
    dismissPermission,
    connectRelay,
    disconnectRelay,
  };

  return <ChatContext.Provider value={value}>{children}</ChatContext.Provider>;
}

export function useChat(): ChatContextValue {
  const ctx = useContext(ChatContext);
  if (!ctx) {
    throw new Error('useChat must be used within a ChatProvider');
  }
  return ctx;
}

function resolveToken(): string | null {
  const params = new URLSearchParams(location.search);
  const fromQuery = params.get('token');
  if (fromQuery) {
    localStorage.setItem('mobilecoding.token', fromQuery);
    history.replaceState(null, '', location.pathname + location.hash);
    return fromQuery;
  }
  return localStorage.getItem('mobilecoding.token');
}
