// 权限请求底部弹窗 — 拇指控制区（参考 ui-template permission.html）
import { useChat } from '../../core/state/ChatContext';

export function PermissionBanner() {
  const { state, answerPermission } = useChat();
  const prompt = state.permissionPrompt;
  if (!prompt) return null;

  const requestId = state.permissionRequestId || undefined;
  const handleAllow = () => answerPermission(true, prompt.toolName, requestId);
  const handleDeny = () => answerPermission(false, prompt.toolName, requestId);

  return (
    <div className="permission-banner">
      <div className="permission-banner-header">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z" />
          <line x1="12" y1="9" x2="12" y2="13" />
          <line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
        等待审批 · {prompt.toolName}
      </div>
      <div className="permission-banner-msg">
        {prompt.message || '请求执行工具操作'}
      </div>
      <div className="permission-banner-actions">
        <button className="btn-deny" onClick={handleDeny}>
          拒绝
        </button>
        <button className="btn-allow" onClick={handleAllow}>
          允许执行
        </button>
      </div>
    </div>
  );
}
