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
  SessionStartParams,
  SessionStartResult,
  PermissionRequestEvent,
  RuntimeConfig,
} from '../ws/types';

const MAX_MESSAGES = 500;

export interface ChatState {
  status: ConnectionStatus;
  sessionId: string | null;
  messages: DisplayMessage[];
  permissionPrompt: PermissionRequestEvent | null;
  lastError: string | null;
  runtime: RuntimeConfig;
  connectionMode: 'direct' | 'relay';
}

type Action =
  | { type: 'STATUS_CHANGED'; status: ConnectionStatus }
  | { type: 'RUNTIME_LOADED'; runtime: RuntimeConfig }
  | { type: 'SESSION_STARTED'; sessionId: string }
  | { type: 'SESSION_STOPPED' }
  | { type: 'EVENT_RECEIVED'; event: AppEvent; sessionId?: string }
  | { type: 'USER_MESSAGE_SENT'; text: string; sessionId: string }
  | { type: 'PERMISSION_ANSWERED' }
  | { type: 'ERROR'; error: string }
  | { type: 'SET_CONNECTION_MODE'; mode: 'direct' | 'relay' };

function reducer(state: ChatState, action: Action): ChatState {
  switch (action.type) {
    case 'STATUS_CHANGED':
      return { ...state, status: action.status, lastError: action.status === 'closed' ? state.lastError : null };
    case 'SESSION_STARTED':
      return { ...state, sessionId: action.sessionId, lastError: null };
    case 'SESSION_STOPPED':
      return { ...state, sessionId: null, permissionPrompt: null };
    case 'RUNTIME_LOADED':
      return { ...state, runtime: action.runtime };
    case 'SET_CONNECTION_MODE':
      return { ...state, connectionMode: action.mode };
    case 'EVENT_RECEIVED': {
      const ev = action.event;
      const messages = [...state.messages, ev as DisplayMessage];
      if (messages.length > MAX_MESSAGES) {
        messages.splice(0, messages.length - MAX_MESSAGES);
      }
      const next: ChatState = {
        ...state,
        messages,
        sessionId: action.sessionId || state.sessionId,
      };
      if (ev.type === 'permission_request') {
        next.permissionPrompt = ev;
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
      return { ...state, messages };
    }
    case 'PERMISSION_ANSWERED':
      return { ...state, permissionPrompt: null };
    case 'ERROR':
      return { ...state, lastError: action.error };
    default:
      return state;
  }
}

const initialState: ChatState = {
  status: 'idle',
  sessionId: null,
  messages: [],
  permissionPrompt: null,
  lastError: null,
  runtime: { defaultCommand: '', defaultArgs: [] },
  connectionMode: 'direct',
};

interface ChatContextValue {
  state: ChatState;
  ws: WSClient;
  sendStart: (params: SessionStartParams) => Promise<SessionStartResult>;
  sendInput: (text: string) => Promise<void>;
  sendStop: () => Promise<void>;
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

  // 自动连接（仅 direct 模式）
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

    // 在 relay 模式下，session 由 CLI 创建，自动设置 sessionId
    dispatch({ type: 'SESSION_STARTED', sessionId: config.sessionId });
  }, []);

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
      if (state.connectionMode === 'relay') {
        // Relay 模式下不需要启动 session（CLI 已启动）
        return { sessionId: state.sessionId || '' };
      }
      const result = await client.startSession(params);
      if (result.sessionId) {
        dispatch({ type: 'SESSION_STARTED', sessionId: result.sessionId });
      }
      return result;
    },
    [client, state.connectionMode, state.sessionId]
  );

  const sendInput = useCallback(
    async (text: string): Promise<void> => {
      const sid = state.sessionId;
      if (!sid) throw new Error('no active session');
      dispatch({ type: 'USER_MESSAGE_SENT', text, sessionId: sid });

      if (state.connectionMode === 'relay' && relayClientRef.current) {
        relayClientRef.current.sendText(text);
      } else {
        await client.sendInput(text);
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

  const value: ChatContextValue = {
    state,
    ws: client,
    sendStart,
    sendInput,
    sendStop,
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