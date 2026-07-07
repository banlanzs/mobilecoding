// 底部输入栏：快捷指令条 + textarea + 圆形发送/停止按钮
import { useRef, useState, useEffect, useCallback } from 'react';
import { useChat } from '../../core/state/ChatContext';
import { isRemoteCliNotReady } from './sessionControls';

// 快捷指令：点击填入输入框（不直接发送，保留用户编辑/确认）
const QUICK_CHIPS = ['/clear', 'git status', 'git diff', 'npm test'];

// 输入历史：跨会话共享，最近 50 条去重，localStorage 持久化
const HISTORY_KEY = 'mc-input-history';
const HISTORY_MAX = 50;
function loadHistory(): string[] {
  try {
    const raw = localStorage.getItem(HISTORY_KEY);
    if (!raw) return [];
    const arr = JSON.parse(raw);
    return Array.isArray(arr) ? arr.filter((x) => typeof x === 'string') : [];
  } catch {
    return [];
  }
}
function saveHistory(items: string[]): void {
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify(items));
  } catch {
    // ignore
  }
}

// 草稿：按 sessionId 保存，切换会话时加载/恢复
function draftKey(sessionId: string): string {
  return `mc-draft:${sessionId}`;
}
function loadDraft(sessionId: string): string {
  try {
    return sessionStorage.getItem(draftKey(sessionId)) || '';
  } catch {
    return '';
  }
}
function saveDraft(sessionId: string, text: string): void {
  try {
    if (text) sessionStorage.setItem(draftKey(sessionId), text);
    else sessionStorage.removeItem(draftKey(sessionId));
  } catch {
    // ignore
  }
}

function SendIcon() {
  return (
    <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="22" y1="2" x2="11" y2="13" />
      <polygon points="22 2 15 22 11 13 2 9 22 2" />
    </svg>
  );
}

function StopIcon() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
      <rect x="5" y="5" width="14" height="14" rx="2" />
    </svg>
  );
}

export function InputBar() {
  const { state, sendInput, abortTurn } = useChat();
  const sessionId = state.sessionId || 'default';
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const taRef = useRef<HTMLTextAreaElement>(null);
  const historyRef = useRef<string[]>(loadHistory());
  // 历史导航游标：null 表示未在历史中浏览，否则指向 historyRef 的索引
  const historyCursorRef = useRef<number | null>(null);
  // 进入历史浏览前保存的当前输入，便于按"下"回到原值
  const savedDraftRef = useRef<string>('');

  // 切换会话时加载对应草稿
  useEffect(() => {
    setText(loadDraft(sessionId));
    historyCursorRef.current = null;
    requestAnimationFrame(() => {
      const ta = taRef.current;
      if (ta) {
        ta.style.height = 'auto';
        ta.style.height = Math.min(ta.scrollHeight, 96) + 'px';
      }
    });
  }, [sessionId]);

  // 按钮显示逻辑：整轮 turn 在执行期间一直显示"中止"。
  // turnActive 由 ChatContext 在 USER_MESSAGE_SENT 时打开，在 turn_end / abort / SESSION_STOPPED 时关闭。
  // 早期实现仅用 thinking/agentState 判断，会在收到 text_delta 后过早回到"发送"按钮。
  const isActive = state.turnActive || state.thinking || state.agentState.status !== 'idle';
  const isStopping = state.stopping;
  const remoteCliNotReady = isRemoteCliNotReady(state.connectionMode, state.runtime.launchMode, state.sessionId);
  // 断线时不禁用输入（允许排队），仅 readOnly / CLI 未就绪时禁用
  const disabled = state.readOnly || remoteCliNotReady;

  // 键盘弹出适配
  useEffect(() => {
    const vv = window.visualViewport;
    const root = document.documentElement;
    const updateInsets = () => {
      const bottomInset = vv
        ? Math.max(0, window.innerHeight - vv.height - vv.offsetTop)
        : 0;
      const standalone = window.matchMedia('(display-mode: standalone)').matches
        || Boolean((navigator as Navigator & { standalone?: boolean }).standalone);
      const touchBrowser = window.matchMedia('(pointer: coarse)').matches && !standalone;
      root.style.setProperty('--keyboard-inset', `${bottomInset}px`);
      root.style.setProperty('--browser-chrome-inset', touchBrowser ? '20px' : '0px');
    };
    updateInsets();
    vv?.addEventListener('resize', updateInsets);
    vv?.addEventListener('scroll', updateInsets);
    window.addEventListener('orientationchange', updateInsets);
    return () => {
      vv?.removeEventListener('resize', updateInsets);
      vv?.removeEventListener('scroll', updateInsets);
      window.removeEventListener('orientationchange', updateInsets);
    };
  }, []);

  // 输入框自动缩放
  const adjustHeight = useCallback(() => {
    const ta = taRef.current;
    if (!ta) return;
    ta.style.height = 'auto';
    ta.style.height = Math.min(ta.scrollHeight, 96) + 'px';
  }, []);

  const handleSend = async () => {
    const trimmed = text.trim();
    if (!trimmed) return;

    // 记入历史（去重、置顶、限长）
    const hist = historyRef.current.filter((h) => h !== trimmed);
    hist.unshift(trimmed);
    if (hist.length > HISTORY_MAX) hist.length = HISTORY_MAX;
    historyRef.current = hist;
    saveHistory(hist);
    historyCursorRef.current = null;

    setText('');
    saveDraft(sessionId, '');
    setSending(true);
    try {
      await sendInput(trimmed);
    } catch (err) {
      console.error('send failed:', err);
      setText(trimmed);
      saveDraft(sessionId, trimmed);
    } finally {
      setSending(false);
      if (taRef.current) {
        taRef.current.style.height = 'auto';
      }
      taRef.current?.focus();
    }
  };

  const handleAbort = async () => {
    try {
      await abortTurn();
    } catch (err) {
      console.error('abort failed:', err);
    }
  };

  // 历史导航：单行输入时上/下方向键回溯历史。多行时交给默认换行行为。
  const navigateHistory = (direction: 'up' | 'down'): boolean => {
    const hist = historyRef.current;
    if (hist.length === 0) return false;

    if (direction === 'up') {
      // 首次进入历史浏览，记下当前输入
      if (historyCursorRef.current === null) {
        savedDraftRef.current = text;
        historyCursorRef.current = 0;
      } else if (historyCursorRef.current < hist.length - 1) {
        historyCursorRef.current += 1;
      } else {
        return true; // 已到最旧，停留
      }
    } else {
      if (historyCursorRef.current === null) return true;
      if (historyCursorRef.current > 0) {
        historyCursorRef.current -= 1;
      } else {
        // 回到当前输入
        historyCursorRef.current = null;
        setText(savedDraftRef.current);
        requestAnimationFrame(adjustHeight);
        return true;
      }
    }
    const value = hist[historyCursorRef.current] ?? '';
    setText(value);
    // 放到末尾
    requestAnimationFrame(() => {
      const ta = taRef.current;
      if (ta) {
        ta.value = value;
        ta.style.height = 'auto';
        ta.style.height = Math.min(ta.scrollHeight, 96) + 'px';
        const end = value.length;
        ta.setSelectionRange(end, end);
      }
    });
    return true;
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (isActive) {
        handleAbort();
      } else {
        handleSend();
      }
      return;
    }
    // 单行时方向键回溯历史；多行时不拦截
    if ((e.key === 'ArrowUp' || e.key === 'ArrowDown') && !e.shiftKey) {
      const ta = taRef.current;
      const multiLine = (ta?.value || '').includes('\n');
      if (!multiLine) {
        e.preventDefault();
        navigateHistory(e.key === 'ArrowUp' ? 'up' : 'down');
      }
    }
  };

  const onChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setText(e.target.value);
    saveDraft(sessionId, e.target.value);
    historyCursorRef.current = null; // 编辑后重置历史游标
    adjustHeight();
  };

  const applyChip = (value: string) => {
    setText(value);
    saveDraft(sessionId, value);
    historyCursorRef.current = null;
    taRef.current?.focus();
    requestAnimationFrame(adjustHeight);
  };

  const placeholder = state.readOnly
    ? '历史会话只读'
    : remoteCliNotReady
    ? '桌面 CLI 未就绪，请确认 mc claude 会话仍在运行'
    : isActive
    ? 'AI 响应中… (Enter 中止)'
    : '输入消息… (Enter 发送, Shift+Enter 换行)';

  return (
    <div className="input-bar">
      <div className="quick-chips">
        {QUICK_CHIPS.map((chip) => (
          <button key={chip} type="button" className="chip" onClick={() => applyChip(chip)}>
            {chip}
          </button>
        ))}
      </div>
      <div className="input-row">
        <div className="input-wrapper">
          <textarea
            ref={taRef}
            value={text}
            onChange={onChange}
            onKeyDown={onKeyDown}
            placeholder={placeholder}
            rows={1}
            disabled={disabled || sending}
            aria-label="输入消息"
          />
        </div>
        {disabled ? (
          <button className="btn-send" disabled aria-label="发送">
            <SendIcon />
          </button>
        ) : isStopping ? (
          <button className="btn-stop-action stopping" disabled aria-label="停止中…">
            …
          </button>
        ) : isActive ? (
          <button
            className="btn-stop-action"
            onClick={handleAbort}
            aria-label="中止请求"
            title="中止当前 AI 请求"
          >
            <StopIcon />
          </button>
        ) : (
          <button
            className="btn-send"
            onClick={handleSend}
            disabled={!text.trim() || sending}
            aria-label="发送"
          >
            <SendIcon />
          </button>
        )}
      </div>
    </div>
  );
}
