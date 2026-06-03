// 终端主页面 — mobile-first 全屏对话界面
import { useChat } from '../../core/state/ChatContext';
import { ConnectionBar } from './ConnectionBar';
import { SessionBar } from './SessionBar';
import { MessageList } from './MessageList';
import { InputBar } from './InputBar';
import './terminal.css';

export function TerminalPage() {
  const { state } = useChat();

  // Relay 模式或有 token 时显示完整界面
  const hasToken = !!localStorage.getItem('mobilecoding.token');
  const isRelayMode = state.connectionMode === 'relay';

  if (!hasToken && !isRelayMode) {
    return (
      <div className="terminal">
        <div className="no-token">
          <h2>mobilecoding</h2>
          <p>
            通过电脑终端中的二维码扫码连接，
            <br />
            或在 URL 后添加 <code>?token=你的令牌</code> 连接。
            <br />
            也可在 localStorage 中设置 <code>mobilecoding.token</code>。
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="terminal">
      <ConnectionBar />
      <SessionBar />
      {state.lastError && (
        <div className="error-msg">{state.lastError}</div>
      )}
      <MessageList />
      <InputBar />
    </div>
  );
}
