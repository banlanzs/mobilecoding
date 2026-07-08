// 权限请求底部弹窗 — 拇指控制区（参考 ui-template permission.html）
import { useState } from 'react';
import { useChat } from '../../core/state/ChatContext';

export function PermissionBanner() {
  const { state, answerPermission, allowToolThisSession, allowAllThisTurn } = useChat();
  const [showDetail, setShowDetail] = useState(false);
  const prompt = state.permissionPrompt;
  if (!prompt) return null;

  const requestId = state.permissionRequestId || (prompt as { messageId?: string }).messageId || undefined;
  const handleAllow = () => answerPermission(true, prompt.toolName, requestId);
  const handleDeny = () => answerPermission(false, prompt.toolName, requestId);
  const handleAllowThisSession = () => {
    allowToolThisSession(prompt.toolName);
    // 加入白名单后，自动应答 effect 会接管此 prompt
  };
  const handleAllowAllTurn = () => {
    allowAllThisTurn();
    // 开启本轮全部允许后，自动应答 effect 会接管此 prompt
  };

  // toolInput 详情：hook 路径透传的原始工具输入（文件路径/命令等）
  const toolInput = (prompt as { toolInput?: unknown }).toolInput;
  const detailLines = formatToolInput(toolInput);
  const summary = prompt.message || `请求执行 ${prompt.toolName}`;
  const autoAllowed = state.permAllowAllTurn || state.permAllowlist.includes(prompt.toolName);

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
      <div className="permission-banner-msg">{summary}</div>
      {detailLines.length > 0 && (
        <>
          <button
            type="button"
            className="permission-banner-detail-toggle"
            onClick={() => setShowDetail(!showDetail)}
          >
            {showDetail ? '收起详情' : '查看详情'}
          </button>
          {showDetail && (
            <pre className="permission-banner-detail">{detailLines.join('\n')}</pre>
          )}
        </>
      )}
      {!autoAllowed && (
        <div className="permission-banner-actions">
          <button className="btn-deny" onClick={handleDeny}>
            拒绝
          </button>
          <button className="btn-allow" onClick={handleAllow}>
            允许执行
          </button>
        </div>
      )}
      {!autoAllowed && (
        <div className="permission-banner-secondary">
          <button type="button" className="btn-secondary" onClick={handleAllowThisSession}>
            本次会话不再询问 {prompt.toolName}
          </button>
          <button type="button" className="btn-secondary" onClick={handleAllowAllTurn}>
            本轮全部允许
          </button>
        </div>
      )}
      {autoAllowed && (
        <div className="permission-banner-auto">自动允许中…</div>
      )}
    </div>
  );
}

// 把 toolInput（对象/字符串）格式化为可读行
function formatToolInput(input: unknown): string[] {
  if (!input) return [];
  if (typeof input === 'string') {
    return input.split('\n').slice(0, 30);
  }
  if (typeof input === 'object') {
    try {
      const obj = input as Record<string, unknown>;
      const lines: string[] = [];
      for (const [k, v] of Object.entries(obj)) {
        const val = typeof v === 'string' ? v : JSON.stringify(v);
        // 截断过长的值（如文件内容）
        const shown = val.length > 500 ? val.slice(0, 500) + '…' : val;
        lines.push(`${k}: ${shown}`);
      }
      return lines.slice(0, 30);
    } catch {
      return [];
    }
  }
  return [];
}
