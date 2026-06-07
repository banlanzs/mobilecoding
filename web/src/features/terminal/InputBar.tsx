// 底部输入栏：快捷指令条 + textarea + 圆形发送/停止按钮
import { useRef, useState, useEffect } from 'react';
import { useChat } from '../../core/state/ChatContext';

// 快捷指令：点击填入输入框（不直接发送，保留用户编辑/确认）
const QUICK_CHIPS = ['/clear', 'git status', 'git diff', 'npm test'];

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
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const taRef = useRef<HTMLTextAreaElement>(null);

  // 按钮显示逻辑：整轮 turn 在执行期间一直显示"中止"。
  // turnActive 由 ChatContext 在 USER_MESSAGE_SENT 时打开，在 turn_end / abort / SESSION_STOPPED 时关闭。
  // 早期实现仅用 thinking/agentState 判断，会在收到 text_delta 后过早回到"发送"按钮。
  const isActive = state.turnActive || state.thinking || state.agentState.status !== 'idle';
  const isStopping = state.stopping;
  const disabled = state.status !== 'connected'
    || (state.runtime.launchMode === 'remote-control' && !state.sessionId);

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
  const adjustHeight = () => {
    const ta = taRef.current;
    if (!ta) return;
    ta.style.height = 'auto';
    ta.style.height = Math.min(ta.scrollHeight, 96) + 'px';
  };

  const handleSend = async () => {
    const trimmed = text.trim();
    if (!trimmed) return;

    setText('');
    setSending(true);
    try {
      await sendInput(trimmed);
    } catch (err) {
      console.error('send failed:', err);
      setText(trimmed);
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

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (isActive) {
        handleAbort();
      } else {
        handleSend();
      }
    }
  };

  const onChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setText(e.target.value);
    adjustHeight();
  };

  const applyChip = (value: string) => {
    setText(value);
    taRef.current?.focus();
    requestAnimationFrame(adjustHeight);
  };

  const placeholder = disabled
    ? '未连接...'
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
