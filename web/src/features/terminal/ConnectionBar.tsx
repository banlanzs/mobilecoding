// 连接状态栏
import { useChat } from '../../core/state/ChatContext';

const STATUS_LABELS: Record<string, string> = {
  idle: '未连接',
  connecting: '连接中…',
  connected: '已连接',
  reconnecting: '重连中…',
  closed: '已断开',
};

export function ConnectionBar() {
  const { state } = useChat();

  const shortCwd = state.runtime.cwd
    ? state.runtime.cwd.replace(/^[A-Z]:/, '').replace(/\\/g, '/').split('/').slice(-2).join('/')
    : '';

  return (
    <div className="conn-bar" data-status={state.status}>
      <span className="conn-dot" />
      <span className="conn-label">
        {STATUS_LABELS[state.status] || state.status}
      </span>
      {shortCwd && (
        <span className="conn-label" style={{ color: '#565f89' }}>
          {shortCwd}
        </span>
      )}
      {state.sessionId && (
        <span className="conn-sid">session: {state.sessionId.slice(0, 12)}…</span>
      )}
    </div>
  );
}