// 终端主页面 — mobile-first 全屏对话界面
import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useChat } from '../../core/state/ChatContext';
import { ConnectionBar } from './ConnectionBar';
import { AgentStatusBar } from './AgentStatusBar';
import { SessionBar } from './SessionBar';
import { MessageList } from './MessageList';
import { InputBar } from './InputBar';
import { PermissionBanner } from './PermissionBanner';
import { Onboarding } from './Onboarding';
import { GitFilesSidebar } from '../files/GitFilesSidebar';
import './terminal.css';

export function TerminalPage() {
  const { id: sessionId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { state, viewSession } = useChat();
  const [showGitFiles, setShowGitFiles] = useState(false);

  useEffect(() => {
    if (sessionId) {
      viewSession(sessionId);
    }
  }, [sessionId, viewSession]);

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

  // 无消息且无活跃 session 时显示引导页
  const showOnboarding = state.messages.length === 0 && !state.viewedSessionId;

  const handleBack = () => {
    navigate('/sessions');
  };

  const toggleGitFiles = () => {
    setShowGitFiles(!showGitFiles);
  };

  return (
    <div className={`terminal ${showGitFiles ? 'has-git-sidebar' : ''}`}>
      <ConnectionBar />
      <AgentStatusBar />
      <SessionBar
        onBack={handleBack}
        currentSessionId={sessionId}
        onToggleFiles={toggleGitFiles}
        showFiles={showGitFiles}
      />
      {state.lastError && (
        <div className="error-msg">{state.lastError}</div>
      )}
      {showOnboarding ? (
        <Onboarding token={localStorage.getItem('mobilecoding.token') || ''} />
      ) : (
        <MessageList />
      )}
      <InputBar />
      <PermissionBanner />
      {showGitFiles && (
        <GitFilesSidebar
          cwd={state.runtime.cwd}
          onClose={() => setShowGitFiles(false)}
        />
      )}
    </div>
  );
}
