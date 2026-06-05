// 底部输入栏：textarea + 发送/停止合一按钮
import { useRef, useState, useEffect } from 'react';
import { useChat } from '../../core/state/ChatContext';

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
  console.log('[DEBUG] InputBar render:', { turnActive: state.turnActive, thinking: state.thinking, agentStatus: state.agentState.status, isActive, hasSession: !!state.sessionId });

  // 键盘弹出适配
  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return;
    const onResize = () => {
      const diff = window.innerHeight - vv.height;
      document.documentElement.style.setProperty('--keyboard-inset', `${Math.max(0, diff)}px`);
    };
    vv.addEventListener('resize', onResize);
    return () => vv.removeEventListener('resize', onResize);
  }, []);

  // 输入框自动缩放
  const adjustHeight = () => {
    const ta = taRef.current;
    if (!ta) return;
    ta.style.height = 'auto';
    ta.style.height = Math.min(ta.scrollHeight, 120) + 'px';
  };

  const handleSend = async () => {
    const trimmed = text.trim();
    if (!trimmed || !state.sessionId) return;

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

  const placeholder = state.sessionId
    ? (isActive ? 'AI 响应中… (Enter 中止)' : '输入消息… (Enter 发送, Shift+Enter 换行)')
    : '先点击 Start 启动会话';

  return (
    <div className="input-bar" style={{ paddingBottom: 'calc(8px + env(safe-area-inset-bottom, 0px) + var(--keyboard-inset, 0px))' }}>
      <textarea
        ref={taRef}
        value={text}
        onChange={onChange}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        rows={1}
        disabled={!state.sessionId || sending}
        aria-label="输入消息"
      />
      {state.sessionId ? (
        isStopping ? (
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
            ■
          </button>
        ) : (
          <button
            className="btn-send"
            onClick={handleSend}
            disabled={!text.trim() || sending}
            aria-label="发送"
          >
            {sending ? '…' : '→'}
          </button>
        )
      ) : (
        <button className="btn-send" disabled aria-label="发送">
          →
        </button>
      )}
    </div>
  );
}