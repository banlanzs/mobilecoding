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
}

type Action =
  | { type: 'STATUS_CHANGED'; status: ConnectionStatus }
  | { type: 'RUNTIME_LOADED'; runtime: RuntimeConfig }
  | { type: 'SESSION_STARTED'; sessionId: string }
  | { type: 'SESSION_STOPPED' }
  | { type: 'EVENT_RECEIVED'; event: AppEvent; sessionId?: string }
  | { type: 'USER_MESSAGE_SENT'; text: string; sessionId: string }
  | { type: 'PERMISSION_ANSWERED' }
  | { type: 'ERROR'; error: string };

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
};

interface ChatContextValue {
  state: ChatState;
  ws: WSClient;
  sendStart: (params: SessionStartParams) => Promise<SessionStartResult>;
  sendInput: (text: string) => Promise<void>;
  sendStop: () => Promise<void>;
  dismissPermission: () => void;
}

const ChatContext = createContext<ChatContextValue | null>(null);

export function ChatProvider({ children }: PropsWithChildren) {
  const { client, status, connect } = useWebSocket();
  const [state, dispatch] = useReducer(reducer, initialState);
  const runtimeRef = useRef<RuntimeConfig>(initialState.runtime);

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

  // 自动连接 + 连接后拉取 runtime 配置并自动启动默认 CLI
  useEffect(() => {
    const token = resolveToken();
    if (token) {
      connect(token);
    }
  }, [connect]);

  useEffect(() => {
    if (status !== 'connected') return;
    fetch('/version')
      .then((r) => r.json())
      .then((data: { runtime?: RuntimeConfig }) => {
        if (data.runtime) {
          runtimeRef.current = data.runtime;
          dispatch({ type: 'RUNTIME_LOADED', runtime: data.runtime });
        }
      })
      .catch(() => {});
  }, [status]);

  // 连接成功且有 defaultCommand 时自动启动
  const autoStartedRef = useRef(false);
  useEffect(() => {
    if (status !== 'connected') {
      autoStartedRef.current = false;
      return;
    }
    if (state.sessionId || autoStartedRef.current) return;
    const rc = runtimeRef.current;
    if (!rc.defaultCommand) return;
    autoStartedRef.current = true;
    client.startSession({ command: rc.defaultCommand, args: rc.defaultArgs })
      .then((r) => dispatch({ type: 'SESSION_STARTED', sessionId: r.sessionId }))
      .catch(() => {
        autoStartedRef.current = false;
      });
  }, [status, state.sessionId, state.runtime, client]);

  const sendStart = useCallback(
    async (params: SessionStartParams): Promise<SessionStartResult> => {
      const result = await client.startSession(params);
      if (result.sessionId) {
        dispatch({ type: 'SESSION_STARTED', sessionId: result.sessionId });
      }
      return result;
    },
    [client]
  );

  const sendInput = useCallback(
    async (text: string): Promise<void> => {
      const sid = state.sessionId;
      if (!sid) throw new Error('no active session');
      dispatch({ type: 'USER_MESSAGE_SENT', text, sessionId: sid });
      await client.sendInput(text);
    },
    [client, state.sessionId]
  );

  const sendStop = useCallback(async (): Promise<void> => {
    await client.stopSession();
    dispatch({ type: 'SESSION_STOPPED' });
  }, [client]);

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