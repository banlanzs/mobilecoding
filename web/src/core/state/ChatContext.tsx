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
const STORED_LAST_SEQ_KEY = 'mobilecoding.lastSeq';
const MAX_STORED_MESSAGES = 200;

function saveLastSeq(seq: number): void {
  try { localStorage.setItem(STORED_LAST_SEQ_KEY, String(seq)); } catch {}
}
function loadLastSeq(): number {
  try { return Number(localStorage.getItem(STORED_LAST_SEQ_KEY)) || 0; } catch { return 0; }
}

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
    // 过滤掉权限请求事件（这些是一次性的，不应该在页面刷新后重新显示）
    const filtered = parsed.filter((msg: DisplayMessage) =>
      msg.type !== 'permission_request' && msg.type !== 'permission_ask'
    );
    return filtered.slice(-MAX_STORED_MESSAGES);
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
  sessionId: string | null; // 当前活跃 runner 的 session id
  viewedSessionId: string | null; // 当前页面正在查看的 session id
  readOnly: boolean; // true 表示历史会话只读，不向 active runner 发送输入
  stoppedSessionId: string | null;
  messages: DisplayMessage[];
  lastSeq: number; // 最后收到的消息 seq，用于断线重连补发
  permissionPrompt: PermissionRequestEvent | null;
  permissionRequestId: string | null; // Claude stdio protocol request_id
  lastError: string | null;
  runtime: RuntimeConfig;
  selectedCommand: string;
  connectionMode: 'direct' | 'relay';
  thinking: boolean;
  turnActive: boolean; // 整个 turn 是否在执行（用于控制"中止/发送"按钮）
  stopping: boolean; // 正在停止会话，阻止事件重新激活 turn
  agentState: AgentStateInfo;
  contextWindow: { used: number; max: number } | null; // 最近 context_window 事件的用量，供 SessionBar 进度条
}

type Action =
  | { type: 'STATUS_CHANGED'; status: ConnectionStatus }
  | { type: 'RUNTIME_LOADED'; runtime: RuntimeConfig }
  | { type: 'SESSION_STARTED'; sessionId: string }
  | { type: 'ACTIVE_SESSION_DETECTED'; sessionId: string }
  | { type: 'VIEW_SESSION_HISTORY'; sessionId: string; messages: DisplayMessage[]; lastSeq: number; readOnly: boolean }
  | { type: 'SESSION_STOPPED' }
  | { type: 'EVENT_RECEIVED'; event: AppEvent; sessionId?: string }
  | { type: 'USER_MESSAGE_SENT'; text: string; sessionId: string }
  | { type: 'PERMISSION_ANSWERED'; allowed: boolean }
  | { type: 'ABORT_TURN' }
  | { type: 'STOPPING' }
  | { type: 'ERROR'; error: string }
  | { type: 'SET_SELECTED_COMMAND'; command: string }
  | { type: 'SET_CONNECTION_MODE'; mode: 'direct' | 'relay' };

function reducer(state: ChatState, action: Action): ChatState {
  switch (action.type) {
    case 'STATUS_CHANGED':
      // 连接断开：把 turnActive 设回 false（turn 中断，按钮切回"发送"）
      if (action.status === 'closed' || action.status === 'reconnecting') {
        return { ...state, status: action.status, turnActive: false, thinking: false, stopping: false };
      }
      return { ...state, status: action.status, lastError: null };
    case 'SESSION_STARTED': {
      // 新会话启动，清除旧消息
      try {
        localStorage.setItem('mobilecoding.sessionId', action.sessionId);
        localStorage.removeItem('mobilecoding.messages');
      } catch {}
      saveLastSeq(0);
      return {
        ...state,
        sessionId: action.sessionId,
        viewedSessionId: action.sessionId,
        readOnly: false,
        stoppedSessionId: null,
        messages: [],
        lastSeq: 0,
        lastError: null,
        thinking: false,
        turnActive: false,
        stopping: false,
        permissionPrompt: null,
        permissionRequestId: null,
        agentState: { status: 'idle', since: Date.now() },
        contextWindow: null,
      };
    }
    case 'ACTIVE_SESSION_DETECTED': {
      const viewingSameSession = !state.viewedSessionId || state.viewedSessionId === action.sessionId;
      return {
        ...state,
        sessionId: action.sessionId,
        viewedSessionId: viewingSameSession ? action.sessionId : state.viewedSessionId,
        readOnly: viewingSameSession ? false : state.readOnly,
        stoppedSessionId: null,
        lastError: null,
      };
    }
    case 'VIEW_SESSION_HISTORY':
      saveLastSeq(action.lastSeq);
      return {
        ...state,
        viewedSessionId: action.sessionId,
        readOnly: action.readOnly,
        messages: action.messages,
        lastSeq: action.lastSeq,
        lastError: null,
        thinking: false,
        turnActive: false,
        stopping: false,
        permissionPrompt: null,
        permissionRequestId: null,
        agentState: { status: 'idle', since: Date.now() },
        contextWindow: null,
      };
    case 'SESSION_STOPPED':
      try { localStorage.removeItem('mobilecoding.sessionId'); } catch {}
      return {
        ...state,
        sessionId: null,
        viewedSessionId: null,
        readOnly: false,
        stoppedSessionId: state.sessionId || state.stoppedSessionId,
        permissionPrompt: null,
        permissionRequestId: null,
        thinking: false,
        turnActive: false,
        stopping: false,
        agentState: { status: 'idle', since: Date.now() },
        contextWindow: null,
      };
    case 'RUNTIME_LOADED':
      return {
        ...state,
        runtime: action.runtime,
        selectedCommand: state.selectedCommand || action.runtime.defaultCommand || '',
      };
    case 'SET_SELECTED_COMMAND':
      return { ...state, selectedCommand: action.command };
    case 'SET_CONNECTION_MODE':
      return { ...state, connectionMode: action.mode };
    case 'EVENT_RECEIVED': {
      const ev = action.event;
      console.log('[DEBUG] EVENT_RECEIVED:', ev.type, ev);

      if (state.readOnly) {
        return state;
      }

      if (action.sessionId && action.sessionId === state.stoppedSessionId) {
        console.log('[DEBUG] Ignoring event for stopped session:', action.sessionId);
        return state;
      }

      // 按 messageId 去重（防止重连重复事件），但权限请求不去重，确保每次都能显示
      if ((ev as any).messageId &&
          ev.type !== 'permission_request' &&
          ev.type !== 'permission_ask' &&
          state.messages.some((m) => (m as any).messageId === (ev as any).messageId)) {
        console.log('[DEBUG] Event deduplicated:', ev.type);
        return state;
      }
      let messages: DisplayMessage[];

      // 权限事件去重：stdin 协议和 HTTP Hook 可能发送重复的 permission_ask，
      // 若最近一条消息已是同类权限请求（同 toolName），则只更新 permissionPrompt，不追加重复消息
      if (ev.type === 'permission_request' || ev.type === 'permission_ask') {
        const lastMsg = state.messages[state.messages.length - 1];
        const isDupPerm = lastMsg && (lastMsg.type === 'permission_request' || lastMsg.type === 'permission_ask')
          && (lastMsg as any).toolName === (ev as any).toolName;
        if (isDupPerm) {
          // 替换最后一条而不是追加新的
          messages = [...state.messages.slice(0, -1), ev as DisplayMessage];
        } else {
          messages = [...state.messages, ev as DisplayMessage];
        }
      } else if (ev.type === 'text_delta') {
        // 增量文本：追加到最后一张 text_delta 卡片，或创建新卡片
        const last = state.messages[state.messages.length - 1];
        if (last && last.type === 'text_delta' && (last as TextDeltaEvent).blockIndex === ev.blockIndex) {
          const lastDelta = last as TextDeltaEvent;
          const merged: TextDeltaEvent = {
            ...lastDelta,
            text: (lastDelta.text || '') + (ev.text || ''),
            thinking: lastDelta.thinking && ev.thinking
              ? lastDelta.thinking + '\n\n' + ev.thinking
              : (lastDelta.thinking || ev.thinking || undefined),
          };
          messages = [...state.messages.slice(0, -1), merged as DisplayMessage];
        } else {
          messages = [...state.messages, ev as DisplayMessage];
        }
      } else if (ev.type === 'text') {
        // 完整文本：替换同一块的 text_delta 卡片，保留累积的 thinking
        const last = state.messages[state.messages.length - 1];
        if (last && last.type === 'text_delta') {
          const preservedThinking = (last as TextDeltaEvent).thinking;
          const finalEvent = { ...ev };
          // 如果 text_delta 累积了 thinking 但 text 事件没有，保留之前的
          if (preservedThinking && !ev.thinking) {
            (finalEvent as any).thinking = preservedThinking;
          }
          messages = [...state.messages.slice(0, -1), finalEvent as DisplayMessage];
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
        sessionId: state.sessionId || (!state.stopping ? action.sessionId || null : null),
        viewedSessionId: state.viewedSessionId || state.sessionId || (!state.stopping ? action.sessionId || null : null),
      };

      // 追踪最新 seq（持久化到 localStorage）
      if (ev.seq && ev.seq > state.lastSeq) {
        next.lastSeq = ev.seq;
        saveLastSeq(ev.seq);
      }

      // 处理权限请求：兼容两种事件类型
      if (ev.type === 'permission_request' || ev.type === 'permission_ask') {
        console.log('[DEBUG] Permission event received:', ev);
        next.permissionPrompt = ev as PermissionRequestEvent;
        // Claude stdio control_request 协议需要回传 request_id
        if (ev.type === 'permission_ask') {
          next.permissionRequestId = (ev as any).messageId || null;
        } else {
          next.permissionRequestId = null;
        }
        if (!state.stopping) {
          next.turnActive = true; // 等待用户处理期间视为活动
        }
        console.log('[DEBUG] permissionPrompt set:', next.permissionPrompt);
      }

      // 检测 tool_result 中的权限请求（Claude hook 超时后自动拒绝，权限信息嵌在 tool_result 中）
      if (!next.permissionPrompt && ev.type === 'tool_result') {
        const rt = typeof ev.toolResult === 'string' ? ev.toolResult : '';
        if (rt.includes('requested permissions') && rt.includes("haven't granted")) {
          const toolMatch = rt.match(/permissions to (\w+)(?: (?:to|on|in) (.+?))?(?:,| but|\.|$)/);
          const toolName = toolMatch ? toolMatch[1] : 'Unknown';
          const target = toolMatch && toolMatch[2] ? toolMatch[2].trim() : '';
          const promptMsg = `请求使用 ${toolName}${target ? ' — ' + target : ''}`;
          console.log('[DEBUG] Detected permission request in tool_result:', toolName, target);
          next.permissionPrompt = {
            type: 'permission_request',
            sessionId: ev.sessionId || state.sessionId || '',
            time: ev.time,
            toolName,
            message: promptMsg,
            messageId: ev.messageId,
          } as any;
        }
      }

      // 整轮结束：把 turnActive 复位（但 thinking 由 Agent 状态决定）
      if (ev.type === 'turn_end') {
        next.turnActive = false;
        next.thinking = false;
      }
      // 兜底：lifecycle 事件中的 "turn_end" 表示旧进程退出。
      // 仅在当前没有活跃 turn 时才复位（避免旧进程的残留事件打断新 turn）
      if (ev.type === 'lifecycle' && (ev.message || '').startsWith('turn_end')) {
        if (!next.thinking && !next.turnActive && next.agentState.status === 'idle') {
          next.turnActive = false;
          next.thinking = false;
        }
      }

      // 注意：text/text_delta 不再把 thinking 置 false。
      // thinking 仅在 turn_end / abort / SESSION_STOPPED 时被清除。
      // 早期实现中"收到 text_delta 就把 thinking=false"是按钮提早变回发送的根因。

      // 更新上下文窗口用量（供 SessionBar 进度条绑定真实数据）
      if (ev.type === 'context_window') {
        const ctx = extractContextTokens((ev as any).toolInput);
        if (ctx.max > 0) next.contextWindow = ctx;
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
      return { ...state, messages, thinking: true, turnActive: true };
    }
    case 'PERMISSION_ANSWERED': {
      // 在消息列表中找到最近的权限事件，标记为已解决
      const msgs = [...state.messages];
      for (let i = msgs.length - 1; i >= 0; i--) {
        const m = msgs[i] as any;
        if (m.type === 'permission_request' || m.type === 'permission_ask') {
          msgs[i] = { ...m, resolved: action.allowed ? 'allowed' : 'denied' };
          break;
        }
      }
      return { ...state, messages: msgs, permissionPrompt: null, permissionRequestId: null };
    }
    case 'ABORT_TURN':
      return { ...state, thinking: false, turnActive: false };
    case 'STOPPING':
      return { ...state, stopping: true, thinking: false, turnActive: false };
    case 'ERROR':
      // 错误时不重置 turnActive（可能只是网络抖动，会话还在）
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
  viewedSessionId: savedSessionId(),
  readOnly: false,
  stoppedSessionId: null,
  messages: loadMessages(),
  lastSeq: loadLastSeq(),
  permissionPrompt: null,
  permissionRequestId: null,
  lastError: null,
  runtime: { defaultCommand: '', defaultArgs: [], launchMode: 'managed', cwd: '' },
  selectedCommand: '',
  connectionMode: 'direct',
  thinking: false,
  turnActive: false,
  stopping: false,
  agentState: { status: 'idle', since: Date.now() },
  contextWindow: null,
};

// 从 context_window 事件的 toolInput 提取 token 用量
function extractContextTokens(data: unknown): { used: number; max: number } {
  if (!data || typeof data !== 'object') return { used: 0, max: 0 };
  const obj = data as Record<string, unknown>;
  const pick = (v: unknown): number => {
    if (typeof v === 'number') return v;
    if (typeof v === 'string') { const n = parseInt(v, 10); return isNaN(n) ? 0 : n; }
    return 0;
  };
  return {
    used: pick(obj.usedTokens ?? obj.used ?? obj.tokens),
    max: pick(obj.maxTokens ?? obj.max ?? obj.contextWindow),
  };
}

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
  answerPermission: (allow: boolean, toolName: string, requestId?: string) => Promise<void>;
  dismissPermission: () => void;
  setSelectedCommand: (command: string) => void;
  viewSession: (sessionId: string) => Promise<void>;
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

  // 断线重连补发：连接成功后，如果有 lastSeq 且有 sessionId，通过 HTTP 补发缺失消息
  useEffect(() => {
    if (status !== 'connected' || state.connectionMode !== 'direct') return;
    if (state.readOnly || !state.sessionId || state.lastSeq <= 0) return;
    const fetchMissed = async () => {
      try {
        const token = localStorage.getItem('mobilecoding.token');
        const url = `/api/v1/messages?session_id=${encodeURIComponent(state.sessionId!)}&after_seq=${state.lastSeq}&limit=200`;
        const res = await fetch(url, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (!res.ok) return;
        const data = await res.json();
        const missed = (data.messages || []) as Array<{ seq: number; type: string; content: string }>;
        if (missed.length === 0) return;
        // 按 seq 排序后批量 dispatch，保证消息顺序
        missed.sort((a, b) => a.seq - b.seq);
        console.log('[RECONNECT] fetched', missed.length, 'missed messages after seq', state.lastSeq);
        for (const msg of missed) {
          try {
            const event = JSON.parse(msg.content) as AppEvent;
            dispatch({ type: 'EVENT_RECEIVED', event, sessionId: state.sessionId! });
          } catch {}
        }
      } catch (err) {
        console.warn('[RECONNECT] fetch missed messages failed:', err);
      }
    };
    fetchMissed();
  }, [status, state.connectionMode, state.sessionId, state.lastSeq, state.readOnly]);

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

    fetch('/api/v1/session-id')
      .then((r) => r.ok ? r.json() : null)
      .then((data: { sessionId?: string } | null) => {
        if (!data?.sessionId) return;
        localStorage.setItem('mobilecoding.sessionId', data.sessionId);
        dispatch({ type: 'ACTIVE_SESSION_DETECTED', sessionId: data.sessionId });
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
      if (state.readOnly) {
        throw new Error('历史会话只读，不能发送消息');
      }
      const sid = state.sessionId || 'remote_control';
      dispatch({ type: 'USER_MESSAGE_SENT', text, sessionId: sid });

      if (state.connectionMode === 'relay' && relayClientRef.current) {
        relayClientRef.current.sendText(text);
      } else {
        try {
          await client.sendInput(text);
        } catch (err) {
          const msg = err instanceof Error ? err.message : String(err);
          if (msg.includes('no active runner') || msg.includes('engine_failure') || msg.includes('context canceled')) {
            dispatch({ type: 'SESSION_STOPPED' });
            dispatch({ type: 'ERROR', error: '会话已断开，请重新启动' });
          }
          throw err;
        }
      }
    },
    [client, state.sessionId, state.connectionMode, state.readOnly]
  );

  const sendStop = useCallback(async (): Promise<void> => {
    if (state.stopping) return;
    dispatch({ type: 'STOPPING' }); // 立即锁定 UI，阻止事件重新激活 turn
    if (state.connectionMode === 'relay') {
      disconnectRelay();
      return;
    }

    dispatch({ type: 'SESSION_STOPPED' });
    try {
      await client.stopSession();
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      dispatch({ type: 'ERROR', error: msg });
      console.error('stop session failed:', err);
    }
  }, [client, state.connectionMode, state.stopping, disconnectRelay]);

  const dismissPermission = useCallback(() => {
    dispatch({ type: 'PERMISSION_ANSWERED', allowed: false });
  }, []);

  const setSelectedCommand = useCallback((command: string) => {
    dispatch({ type: 'SET_SELECTED_COMMAND', command });
  }, []);

  const viewSession = useCallback(async (sessionId: string): Promise<void> => {
    if (!sessionId || sessionId === 'new') {
      if (!state.sessionId) {
        dispatch({ type: 'VIEW_SESSION_HISTORY', sessionId: '', messages: [], lastSeq: 0, readOnly: false });
      }
      return;
    }
    const token = localStorage.getItem('mobilecoding.token');
    const res = await fetch(`/api/v1/messages?session_id=${encodeURIComponent(sessionId)}&after_seq=0&limit=500`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) {
      dispatch({ type: 'ERROR', error: `加载会话历史失败: ${res.statusText}` });
      return;
    }
    const data = await res.json();
    const rows = (data.messages || []) as Array<{ seq: number; content: string }>;
    rows.sort((a, b) => a.seq - b.seq);
    const messages: DisplayMessage[] = [];
    let lastSeq = 0;
    for (const row of rows) {
      try {
        messages.push(JSON.parse(row.content) as DisplayMessage);
        if (row.seq > lastSeq) lastSeq = row.seq;
      } catch {
        // 忽略无法解析的历史行，避免单条坏数据阻塞整个会话查看。
      }
    }
    dispatch({
      type: 'VIEW_SESSION_HISTORY',
      sessionId,
      messages,
      lastSeq,
      readOnly: sessionId !== state.sessionId,
    });
  }, [state.sessionId]);

  const abortTurn = useCallback(async (): Promise<void> => {
    dispatch({ type: 'ABORT_TURN' });
    if (state.connectionMode === 'relay' && relayClientRef.current) {
      relayClientRef.current.sendText(JSON.stringify({ type: 'session.abort' }));
    } else {
      await client.abortTurn();
    }
  }, [client, state.connectionMode]);

  const answerPermission = useCallback(
    async (allow: boolean, _toolName: string, requestId?: string): Promise<void> => {
      dispatch({ type: 'PERMISSION_ANSWERED', allowed: allow });
      // 没有 requestId 说明是 tool_result 中检测出的伪权限请求（Claude 已超时拒绝）
      // stdout 写入已无效，仅关闭弹窗
      if (!requestId) return;
      if (state.connectionMode === 'relay' && relayClientRef.current) {
        relayClientRef.current.sendRespondPermission(requestId, allow);
      } else {
        await client.respondPermission(requestId, allow);
      }
    }, [client, state.connectionMode]);

  const value: ChatContextValue = {
    state,
    ws: client,
    sendStart,
    sendInput,
    sendStop,
    abortTurn,
    answerPermission,
    dismissPermission,
    setSelectedCommand,
    viewSession,
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
