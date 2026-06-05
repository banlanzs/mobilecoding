// 权限请求底部弹窗 — 固定在页面底部，仅在有权限请求时显示
import { useChat } from '../../core/state/ChatContext';

export function PermissionBanner() {
  const { state, answerPermission } = useChat();
  const prompt = state.permissionPrompt;
  if (!prompt) return null;

  return (
    <div className="permission-banner">
      <div className="permission-banner-header">⚠ 权限请求</div>
      <div className="permission-banner-tool">{prompt.toolName}</div>
      <div className="permission-banner-msg">
        {prompt.message || '请求执行工具操作'}
      </div>
      <div className="permission-banner-actions">
        <button
          className="btn-allow"
          onClick={() =>
            answerPermission(true, prompt.toolName, state.permissionRequestId || undefined)
          }
        >
          Allow
        </button>
        <button
          className="btn-deny"
          onClick={() =>
            answerPermission(false, prompt.toolName, state.permissionRequestId || undefined)
          }
        >
          Deny
        </button>
      </div>
    </div>
  );
}
