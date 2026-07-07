// 连接状态栏 — 状态(中文标签 + 脉冲点) + 工作目录 + 主题切换 + 断线横幅
import { useState } from 'react';
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
  const [theme, setTheme] = useState<string>(
    () => document.documentElement.getAttribute('data-theme') || 'dark'
  );

  const shortCwd = state.runtime.cwd
    ? state.runtime.cwd.replace(/^[A-Z]:/, '').replace(/\\/g, '/').split('/').slice(-2).join('/')
    : '';

  const dotClass = state.status === 'connected'
    ? 'status-dot connected pulse'
    : `status-dot ${state.status}`;

  const toggleTheme = () => {
    // 循环切换：dark → light → terminal → dark
    const next = theme === 'dark' ? 'light' : theme === 'light' ? 'terminal' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    try { localStorage.setItem('mc-theme', next); } catch {}
    setTheme(next);
  };

  // 主题图标
  const themeIcon = theme === 'dark' ? '☀️' : theme === 'light' ? '💻' : '🌙';
  const themeLabel = theme === 'dark' ? '亮色' : theme === 'light' ? '终端' : '暗色';

  const isOffline = state.status === 'reconnecting' || state.status === 'closed';
  const pendingCount = state.pendingQueue.length;

  return (
    <div className="connection-bar">
      <div className="connection-status">
        <span className={dotClass} />
        <span className="conn-state">{STATUS_LABELS[state.status] || state.status}</span>
        {isOffline && pendingCount > 0 && (
          <span className="pending-badge">{pendingCount} 条待发送</span>
        )}
      </div>
      <div className="connection-info">
        {shortCwd && <span className="cwd">{shortCwd}</span>}
        <button
          className="theme-toggle"
          onClick={toggleTheme}
          aria-label={`切换到${themeLabel}模式`}
          title={`切换到${themeLabel}模式`}
        >
          {themeIcon}
        </button>
      </div>
      {isOffline && (
        <div className="offline-banner" role="status">
          {state.status === 'reconnecting' ? '正在重连…' : '连接已断开'}
          {pendingCount > 0 && `，输入将在重连后自动发送`}
        </div>
      )}
    </div>
  );
}
